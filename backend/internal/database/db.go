package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/edgelink/backend/internal/config"
	"github.com/edgelink/backend/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewPostgresDB 创建数据库连接（Fx兼容）
func NewPostgresDB(cfg *config.Config) (*gorm.DB, error) {
	return New(&cfg.Database)
}

// New 创建数据库连接
func New(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 获取底层sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// 配置连接池 - 生产级调优
	// 基于负载测试和最佳实践的配置
	maxOpenConns := cfg.MaxOpenConns
	if maxOpenConns == 0 {
		// 默认值：基于CPU核心数
		// 公式: (CPU cores * 2) + effective_spindle_count
		// 对于云环境，建议 25-100
		maxOpenConns = 100
	}

	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns == 0 {
		// 空闲连接数应为最大连接数的25-50%
		maxIdleConns = maxOpenConns / 2
	}

	connMaxLifetime := cfg.ConnMaxLifetime
	if connMaxLifetime == 0 {
		// 连接最大生命周期：5分钟
		// 防止连接泄漏和数据库端超时
		connMaxLifetime = 5 * time.Minute
	}

	// 连接最大空闲时间
	connMaxIdleTime := 10 * time.Minute

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	log.Printf("Database connection pool configured: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v, MaxIdleTime=%v",
		maxOpenConns, maxIdleConns, connMaxLifetime, connMaxIdleTime)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 预热连接池 - 创建初始空闲连接
	log.Println("Warming up connection pool...")
	for i := 0; i < maxIdleConns/2; i++ {
		if err := sqlDB.Ping(); err != nil {
			log.Printf("Warning: failed to warm up connection %d: %v", i, err)
		}
	}

	// 启动连接池监控
	go monitorConnectionPool(sqlDB)

	// 自动运行数据库迁移
	log.Println("Running database migrations...")
	
	// 首先创建ENUM类型
	if err := createEnumTypes(db); err != nil {
		return nil, fmt.Errorf("failed to create enum types: %w", err)
	}
	
	// 然后运行GORM AutoMigrate
	if err := AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Println("Database migrations completed successfully")

	return db, nil
}

// createEnumTypes 创建PostgreSQL ENUM类型
func createEnumTypes(gormDB *gorm.DB) error {
	// 定义所有需要的ENUM类型（不使用IF NOT EXISTS，改用DO块）
	enums := []struct {
		name   string
		values string
	}{
		{"platform_enum", "'desktop_linux', 'desktop_windows', 'desktop_macos', 'mobile_ios', 'mobile_android', 'iot', 'container'"},
		{"nat_type_enum", "'none', 'full_cone', 'restricted_cone', 'port_restricted_cone', 'symmetric', 'unknown'"},
		{"key_status_enum", "'active', 'pending_rotation', 'revoked', 'expired'"},
		{"connection_type_enum", "'p2p_direct', 'turn_relay'"},
		{"severity_enum", "'critical', 'high', 'medium', 'low'"},
		{"alert_type_enum", "'device_offline', 'high_latency', 'failed_auth', 'key_expiration', 'tunnel_failure'"},
		{"alert_status_enum", "'active', 'acknowledged', 'resolved'"},
		{"role_enum", "'super_admin', 'admin', 'network_operator', 'auditor', 'readonly'"},
		{"diagnostic_status_enum", "'requested', 'collecting', 'uploaded', 'failed', 'expired'"},
		{"resource_type_enum", "'device', 'virtual_network', 'pre_shared_key', 'alert', 'organization'"},
	}
	
	// 使用DO块创建ENUM类型（如果不存在）
	for _, enum := range enums {
		doBlock := fmt.Sprintf(`
			DO $$
			BEGIN
				IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = '%s') THEN
					CREATE TYPE %s AS ENUM (%s);
				END IF;
			END$$;
		`, enum.name, enum.name, enum.values)
		
		if err := gormDB.Exec(doBlock).Error; err != nil {
			log.Printf("Warning: failed to create enum %s: %v", enum.name, err)
			// 继续执行，因为错误可能是ENUM已存在
		}
	}
	
	return nil
}

// AutoMigrate 自动迁移所有模型（仅用于开发环境）
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.Organization{},
		&domain.VirtualNetwork{},
		&domain.Device{},
		&domain.DeviceKey{},
		&domain.PreSharedKey{},
		&domain.PeerConfiguration{},
		&domain.Session{},
		&domain.Alert{},
		&domain.AuditLog{},
		&domain.DiagnosticBundle{},
		&domain.AdminUser{},
	)
}



// Close 关闭数据库连接
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// monitorConnectionPool 监控数据库连接池状态
// 定期记录连接池指标，用于性能分析和问题排查
func monitorConnectionPool(sqlDB *sql.DB) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := sqlDB.Stats()

		log.Printf("Connection Pool Stats: "+
			"OpenConnections=%d, "+
			"InUse=%d, "+
			"Idle=%d, "+
			"WaitCount=%d, "+
			"WaitDuration=%v, "+
			"MaxIdleClosed=%d, "+
			"MaxIdleTimeClosed=%d, "+
			"MaxLifetimeClosed=%d",
			stats.OpenConnections,
			stats.InUse,
			stats.Idle,
			stats.WaitCount,
			stats.WaitDuration,
			stats.MaxIdleClosed,
			stats.MaxIdleTimeClosed,
			stats.MaxLifetimeClosed,
		)

		// 警告：如果等待连接的请求过多
		if stats.WaitCount > 100 {
			log.Printf("WARNING: High connection wait count (%d). Consider increasing MaxOpenConns.", stats.WaitCount)
		}

		// 警告：如果空闲连接不足
		if stats.Idle == 0 && stats.InUse > 0 {
			log.Printf("WARNING: No idle connections available. All %d connections are in use.", stats.InUse)
		}

		// 警告：如果连接池接近饱和
		if float64(stats.InUse)/float64(stats.OpenConnections) > 0.9 && stats.OpenConnections > 0 {
			log.Printf("WARNING: Connection pool is %.0f%% saturated (%d/%d).",
				float64(stats.InUse)/float64(stats.OpenConnections)*100,
				stats.InUse,
				stats.OpenConnections,
			)
		}
	}
}
