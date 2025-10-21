package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/edgelink/backend/internal/migrations"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// 从环境变量获取数据库连接字符串
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://edgelink:edgelink_dev_password@localhost:5432/edgelink?sslmode=disable"
		log.Printf("DATABASE_URL not set, using default: %s", dbURL)
	}

	// 连接数据库
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// 创建迁移器
	migrator, err := migrations.NewMigrator(db)
	if err != nil {
		log.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	// 执行命令
	command := os.Args[1]
	switch command {
	case "up":
		log.Println("Running migrations up...")
		if err := migrator.Up(); err != nil {
			log.Fatalf("Failed to migrate up: %v", err)
		}
		log.Println("Migrations completed successfully")

	case "down":
		log.Println("Rolling back all migrations...")
		if err := migrator.Down(); err != nil {
			log.Fatalf("Failed to migrate down: %v", err)
		}
		log.Println("Rollback completed successfully")

	case "version":
		version, dirty, err := migrator.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		if dirty {
			log.Printf("Current version: %d (dirty)", version)
		} else {
			log.Printf("Current version: %d", version)
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: migrate <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  up       - Run all pending migrations")
	fmt.Println("  down     - Rollback all migrations")
	fmt.Println("  version  - Print current migration version")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  DATABASE_URL - PostgreSQL connection string (optional)")
	fmt.Println("                 Default: postgres://edgelink:edgelink_dev_password@localhost:5432/edgelink?sslmode=disable")
}
