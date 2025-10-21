package migrations

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var migrations embed.FS

// Migrator 数据库迁移器
type Migrator struct {
	db *sql.DB
	m  *migrate.Migrate
}

// NewMigrator 创建新的迁移器
func NewMigrator(db *sql.DB) (*Migrator, error) {
	// 创建嵌入式文件系统源
	sourceDriver, err := iofs.New(migrations, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to create source driver: %w", err)
	}

	// 创建 PostgreSQL 驱动
	dbDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}

	// 创建 migrate 实例
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbDriver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	return &Migrator{
		db: db,
		m:  m,
	}, nil
}

// Up 执行所有待执行的迁移
func (migrator *Migrator) Up() error {
	if err := migrator.m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to migrate up: %w", err)
	}
	return nil
}

// Down 回滚所有迁移
func (migrator *Migrator) Down() error {
	if err := migrator.m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to migrate down: %w", err)
	}
	return nil
}

// Steps 执行指定步数的迁移
func (migrator *Migrator) Steps(n int) error {
	if err := migrator.m.Steps(n); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to migrate steps: %w", err)
	}
	return nil
}

// Version 获取当前迁移版本
func (migrator *Migrator) Version() (uint, bool, error) {
	version, dirty, err := migrator.m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("failed to get version: %w", err)
	}
	return version, dirty, nil
}

// Close 关闭迁移器
func (migrator *Migrator) Close() error {
	sourceErr, dbErr := migrator.m.Close()
	if sourceErr != nil {
		return sourceErr
	}
	if dbErr != nil {
		return dbErr
	}
	return nil
}
