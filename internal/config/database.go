package config

import (
	"fmt"
	"os"
)

// LoadDatabaseDSNFromEnv 为 Composition Root 统一读取并校验数据库 DSN。
func LoadDatabaseDSNFromEnv() (string, error) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		return "", fmt.Errorf("DATABASE_DSN is required")
	}
	return dsn, nil
}
