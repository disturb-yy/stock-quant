package mysql

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	migrationLockWaitSeconds = 30
	migrationTable           = "t_schema_migrations"
)

var migrationFilename = regexp.MustCompile(`^([0-9]+)\.(up|down)\.sql$`)
var createTableStatement = regexp.MustCompile(`(?i)^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?` + "`?([a-z0-9_]+)`?")

// Migration stores one immutable, versioned up/down schema change.
type Migration struct {
	Version int
	Name    string
	Up      string
	Down    string
}

// Migrator applies or rolls back embedded schema migrations while holding a MySQL advisory lock.
type Migrator struct {
	DB         *sql.DB
	Migrations []Migration
}

type migrationState struct {
	Version   int
	Name      string
	Checksum  string
	Dirty     bool
	Direction string
}

// LoadMigrations reads paired, contiguous NNNN.up.sql / NNNN.down.sql files.
func LoadMigrations(source fs.FS) ([]Migration, error) {
	upFiles, err := fs.Glob(source, "*.up.sql")
	if err != nil {
		return nil, fmt.Errorf("list up migrations: %w", err)
	}
	downFiles, err := fs.Glob(source, "*.down.sql")
	if err != nil {
		return nil, fmt.Errorf("list down migrations: %w", err)
	}
	versions := make(map[int]*Migration, len(upFiles))
	for _, file := range upFiles {
		match := migrationFilename.FindStringSubmatch(path.Base(file))
		if match == nil || match[2] != "up" {
			return nil, fmt.Errorf("invalid up migration filename %q", file)
		}
		version, err := strconv.Atoi(match[1])
		if err != nil || version < 1 {
			return nil, fmt.Errorf("invalid migration version in %q", file)
		}
		if file != fmt.Sprintf("%04d.up.sql", version) {
			return nil, fmt.Errorf("migration filename %q must use four-digit version padding", file)
		}
		if _, exists := versions[version]; exists {
			return nil, fmt.Errorf("duplicate migration version %d", version)
		}
		contents, err := fs.ReadFile(source, file)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", file, err)
		}
		versions[version] = &Migration{Version: version, Name: fmt.Sprintf("%04d", version), Up: string(contents)}
	}
	if len(versions) == 0 {
		return nil, errors.New("no up migrations found")
	}
	for _, file := range downFiles {
		match := migrationFilename.FindStringSubmatch(path.Base(file))
		if match == nil || match[2] != "down" {
			return nil, fmt.Errorf("invalid down migration filename %q", file)
		}
		version, err := strconv.Atoi(match[1])
		if err != nil || version < 1 {
			return nil, fmt.Errorf("invalid migration version in %q", file)
		}
		if file != fmt.Sprintf("%04d.down.sql", version) {
			return nil, fmt.Errorf("migration filename %q must use four-digit version padding", file)
		}
		if _, exists := versions[version]; !exists {
			return nil, fmt.Errorf("down migration %q has no matching up migration", file)
		}
	}
	for version, migration := range versions {
		file := fmt.Sprintf("%04d.down.sql", version)
		contents, err := fs.ReadFile(source, file)
		if err != nil {
			return nil, fmt.Errorf("read paired down migration %s: %w", file, err)
		}
		migration.Down = string(contents)
	}
	ordered := make([]Migration, 0, len(versions))
	for _, migration := range versions {
		ordered = append(ordered, *migration)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Version < ordered[j].Version })
	for i, migration := range ordered {
		if migration.Version != i+1 {
			return nil, fmt.Errorf("migration versions must be contiguous from 1: found %d at position %d", migration.Version, i+1)
		}
	}
	return ordered, nil
}

// Up applies every unapplied migration in ascending version order. A dirty up can be retried
// only with the same migration checksum, allowing idempotent DDL to finish after repair.
func (m *Migrator) Up(ctx context.Context) error {
	if err := m.validate(); err != nil {
		return err
	}
	return m.withLock(ctx, func(conn *sql.Conn) error {
		if err := ensureMigrationTable(ctx, conn); err != nil {
			return err
		}
		states, err := readMigrationStates(ctx, conn)
		if err != nil {
			return err
		}
		known := make(map[int]Migration, len(m.Migrations))
		for _, migration := range m.Migrations {
			known[migration.Version] = migration
		}
		if err := validateStoredStates(states, known); err != nil {
			return err
		}
		for _, migration := range m.Migrations {
			checksum := migrationChecksum(migration)
			state, exists := states[migration.Version]
			if exists {
				if state.Checksum != checksum {
					return fmt.Errorf("migration %d checksum mismatch: stored %s, current %s", migration.Version, state.Checksum, checksum)
				}
				if !state.Dirty && state.Direction == "up" {
					continue
				}
				if state.Direction != "up" {
					return fmt.Errorf("migration %d is dirty in %s direction; run that direction to recover", migration.Version, state.Direction)
				}
			} else {
				if err := ensureMigrationObjectsAbsent(ctx, conn, migration); err != nil {
					return err
				}
				if err := insertMigrationState(ctx, conn, migration, checksum, "up"); err != nil {
					return err
				}
			}
			if err := markMigrationDirty(ctx, conn, migration.Version, "up"); err != nil {
				return err
			}
			if err := executeSQL(ctx, conn, migration.Up); err != nil {
				return fmt.Errorf("apply migration %d: %w", migration.Version, err)
			}
			if err := markMigrationClean(ctx, conn, migration.Version); err != nil {
				return err
			}
			states[migration.Version] = migrationState{Version: migration.Version, Name: migration.Name, Checksum: checksum, Direction: "up"}
		}
		return nil
	})
}

// Down rolls back exactly the latest applied migration. It never removes the migration ledger.
func (m *Migrator) Down(ctx context.Context) error {
	if err := m.validate(); err != nil {
		return err
	}
	return m.withLock(ctx, func(conn *sql.Conn) error {
		if err := ensureMigrationTable(ctx, conn); err != nil {
			return err
		}
		states, err := readMigrationStates(ctx, conn)
		if err != nil {
			return err
		}
		known := make(map[int]Migration, len(m.Migrations))
		for _, migration := range m.Migrations {
			known[migration.Version] = migration
		}
		if err := validateStoredStates(states, known); err != nil {
			return err
		}
		var latest *Migration
		for i := range m.Migrations {
			migration := &m.Migrations[i]
			if _, exists := states[migration.Version]; exists && (latest == nil || migration.Version > latest.Version) {
				latest = migration
			}
		}
		if latest == nil {
			return nil
		}
		state := states[latest.Version]
		if state.Checksum != migrationChecksum(*latest) {
			return fmt.Errorf("migration %d checksum mismatch: stored %s, current %s", latest.Version, state.Checksum, migrationChecksum(*latest))
		}
		if state.Dirty && state.Direction != "down" {
			return fmt.Errorf("migration %d is dirty in up direction; finish up before down", latest.Version)
		}
		if !state.Dirty && state.Direction != "up" {
			return fmt.Errorf("migration %d has unexpected clean direction %q", latest.Version, state.Direction)
		}
		if err := markMigrationDirty(ctx, conn, latest.Version, "down"); err != nil {
			return err
		}
		if err := executeSQL(ctx, conn, latest.Down); err != nil {
			return fmt.Errorf("rollback migration %d: %w", latest.Version, err)
		}
		if _, err := conn.ExecContext(ctx, "DELETE FROM "+migrationTable+" WHERE version = ?", latest.Version); err != nil {
			return fmt.Errorf("remove migration %d record: %w", latest.Version, err)
		}
		return nil
	})
}

func (m *Migrator) validate() error {
	if m == nil || m.DB == nil {
		return errors.New("migration database is required")
	}
	if len(m.Migrations) == 0 {
		return errors.New("at least one migration is required")
	}
	previous := 0
	for _, migration := range m.Migrations {
		if migration.Version != previous+1 {
			return fmt.Errorf("migration versions must be contiguous from 1: expected %d, got %d", previous+1, migration.Version)
		}
		if strings.TrimSpace(migration.Name) == "" {
			return fmt.Errorf("migration %d requires a name", migration.Version)
		}
		if strings.TrimSpace(migration.Up) == "" || strings.TrimSpace(migration.Down) == "" {
			return fmt.Errorf("migration %d requires up and down SQL", migration.Version)
		}
		previous = migration.Version
	}
	return nil
}

func (m *Migrator) withLock(ctx context.Context, action func(*sql.Conn) error) (result error) {
	conn, err := m.DB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration lock connection: %w", err)
	}
	defer conn.Close()
	var databaseName sql.NullString
	if err := conn.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&databaseName); err != nil {
		return fmt.Errorf("identify migration database: %w", err)
	}
	if !databaseName.Valid || databaseName.String == "" {
		return errors.New("migration requires a selected database")
	}
	lockDigest := sha256.Sum256([]byte(databaseName.String))
	lockName := "stock_quant_migrate_" + hex.EncodeToString(lockDigest[:20])
	var acquired sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", lockName, migrationLockWaitSeconds).Scan(&acquired); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		return fmt.Errorf("acquire migration lock: timed out after %d seconds", migrationLockWaitSeconds)
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var released sql.NullInt64
		releaseErr := conn.QueryRowContext(releaseCtx, "SELECT RELEASE_LOCK(?)", lockName).Scan(&released)
		if releaseErr == nil && (!released.Valid || released.Int64 != 1) {
			releaseErr = errors.New("MySQL did not release the advisory lock")
		}
		if releaseErr != nil {
			releaseErr = fmt.Errorf("release migration lock: %w", releaseErr)
			result = errors.Join(result, releaseErr)
		}
	}()
	return action(conn)
}

func ensureMigrationTable(ctx context.Context, conn *sql.Conn) error {
	statement := `CREATE TABLE IF NOT EXISTS t_schema_migrations (
  version INT NOT NULL PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  checksum CHAR(64) NOT NULL,
  dirty BOOLEAN NOT NULL,
  direction VARCHAR(8) NOT NULL,
  applied_at DATETIME(6) NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`
	if _, err := conn.ExecContext(ctx, statement); err != nil {
		return fmt.Errorf("ensure schema migration ledger: %w", err)
	}
	return nil
}

func readMigrationStates(ctx context.Context, conn *sql.Conn) (map[int]migrationState, error) {
	rows, err := conn.QueryContext(ctx, "SELECT version, name, checksum, dirty, direction FROM "+migrationTable+" ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("read migration versions: %w", err)
	}
	defer rows.Close()
	states := make(map[int]migrationState)
	for rows.Next() {
		var state migrationState
		if err := rows.Scan(&state.Version, &state.Name, &state.Checksum, &state.Dirty, &state.Direction); err != nil {
			return nil, fmt.Errorf("scan migration version: %w", err)
		}
		states[state.Version] = state
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migration versions: %w", err)
	}
	return states, nil
}

func validateStoredStates(states map[int]migrationState, known map[int]Migration) error {
	for version := 1; version <= len(states); version++ {
		if _, exists := states[version]; !exists {
			return fmt.Errorf("database migration history is not contiguous: missing version %d", version)
		}
	}
	for version, state := range states {
		migration, exists := known[version]
		if !exists {
			return fmt.Errorf("database contains unknown migration version %d", version)
		}
		if state.Name != migration.Name {
			return fmt.Errorf("migration %d name mismatch: stored %q, current %q", version, state.Name, migration.Name)
		}
		if state.Checksum != migrationChecksum(migration) {
			return fmt.Errorf("migration %d checksum mismatch: stored %s, current %s", version, state.Checksum, migrationChecksum(migration))
		}
		if state.Direction != "up" && state.Direction != "down" {
			return fmt.Errorf("migration %d has invalid direction %q", version, state.Direction)
		}
	}
	return nil
}

func ensureMigrationObjectsAbsent(ctx context.Context, conn *sql.Conn, migration Migration) error {
	statements, err := splitStatements(migration.Up)
	if err != nil {
		return fmt.Errorf("inspect migration %d objects: %w", migration.Version, err)
	}
	for _, statement := range statements {
		if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(statement)), "CREATE TABLE") {
			continue
		}
		match := createTableStatement.FindStringSubmatch(strings.TrimSpace(statement))
		if match == nil {
			return fmt.Errorf("migration %d contains an unsupported CREATE TABLE statement", migration.Version)
		}
		name := match[1]
		var count int
		if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", name).Scan(&count); err != nil {
			return fmt.Errorf("check migration %d object %s: %w", migration.Version, name, err)
		}
		if count != 0 {
			return fmt.Errorf("migration %d refuses to adopt pre-existing table %q", migration.Version, name)
		}
	}
	return nil
}

func insertMigrationState(ctx context.Context, conn *sql.Conn, migration Migration, checksum, direction string) error {
	_, err := conn.ExecContext(ctx, "INSERT INTO "+migrationTable+" (version, name, checksum, dirty, direction) VALUES (?, ?, ?, TRUE, ?)", migration.Version, migration.Name, checksum, direction)
	if err != nil {
		return fmt.Errorf("record migration %d start: %w", migration.Version, err)
	}
	return nil
}

func markMigrationDirty(ctx context.Context, conn *sql.Conn, version int, direction string) error {
	_, err := conn.ExecContext(ctx, "UPDATE "+migrationTable+" SET dirty = TRUE, direction = ?, applied_at = NULL WHERE version = ?", direction, version)
	if err != nil {
		return fmt.Errorf("mark migration %d dirty: %w", version, err)
	}
	return nil
}

func markMigrationClean(ctx context.Context, conn *sql.Conn, version int) error {
	_, err := conn.ExecContext(ctx, "UPDATE "+migrationTable+" SET dirty = FALSE, direction = 'up', applied_at = CURRENT_TIMESTAMP(6) WHERE version = ?", version)
	if err != nil {
		return fmt.Errorf("mark migration %d applied: %w", version, err)
	}
	return nil
}

func executeSQL(ctx context.Context, conn *sql.Conn, source string) error {
	statements, err := splitStatements(source)
	if err != nil {
		return err
	}
	for index, statement := range statements {
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("statement %d: %w", index+1, err)
		}
	}
	return nil
}

func splitStatements(source string) ([]string, error) {
	statements := make([]string, 0)
	var current strings.Builder
	var quote byte
	lineComment := false
	blockComment := false
	for index := 0; index < len(source); {
		char := source[index]
		if lineComment {
			if char == '\n' {
				lineComment = false
				current.WriteByte(' ')
			}
			index++
			continue
		}
		if blockComment {
			if char == '*' && index+1 < len(source) && source[index+1] == '/' {
				blockComment = false
				current.WriteByte(' ')
				index += 2
				continue
			}
			index++
			continue
		}
		if quote != 0 {
			current.WriteByte(char)
			if char == '\\' && quote != '`' && index+1 < len(source) {
				current.WriteByte(source[index+1])
				index += 2
				continue
			}
			if char == quote {
				if index+1 < len(source) && source[index+1] == quote {
					current.WriteByte(source[index+1])
					index += 2
					continue
				}
				quote = 0
			}
			index++
			continue
		}
		if char == '\'' || char == '"' || char == '`' {
			quote = char
			current.WriteByte(char)
			index++
			continue
		}
		if char == '#' || (char == '-' && index+1 < len(source) && source[index+1] == '-' && (index+2 == len(source) || source[index+2] <= ' ')) {
			lineComment = true
			index++
			if char == '-' {
				index++
			}
			continue
		}
		if char == '/' && index+1 < len(source) && source[index+1] == '*' {
			blockComment = true
			index += 2
			continue
		}
		if char == ';' {
			if statement := strings.TrimSpace(current.String()); statement != "" {
				statements = append(statements, statement)
			}
			current.Reset()
			index++
			continue
		}
		current.WriteByte(char)
		index++
	}
	if quote != 0 {
		return nil, fmt.Errorf("migration SQL contains an unterminated quoted value")
	}
	if blockComment {
		return nil, fmt.Errorf("migration SQL contains an unterminated block comment")
	}
	if statement := strings.TrimSpace(current.String()); statement != "" {
		statements = append(statements, statement)
	}
	return statements, nil
}

func migrationChecksum(migration Migration) string {
	payload := migration.Name + "\x00" + migration.Up + "\x00" + migration.Down
	digest := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(digest[:])
}
