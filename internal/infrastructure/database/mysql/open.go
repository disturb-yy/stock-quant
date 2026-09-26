package mysql

import (
	"context"
	"fmt"

	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Open 创建并验证 GORM MySQL 连接，调用方负责在生命周期结束时关闭底层连接。
func Open(ctx context.Context, dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_DSN is required")
	}
	db, err := gorm.Open(gormmysql.New(gormmysql.Config{
		DSN:                       dsn,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		return nil, fmt.Errorf("open mysql with gorm: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get mysql sql database: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return db, nil
}
