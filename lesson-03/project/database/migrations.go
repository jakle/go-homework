package database

import (
	"fmt"
	"log"

	"gin-examples/project/models"

	"gorm.io/gorm"
)

type Migration struct {
	db *gorm.DB
}

func NewMigration(db *gorm.DB) *Migration {
	return &Migration{db: db}
}

func (m *Migration) Run() error {
	log.Println("Running database migrations...")

	// 自动迁移基础表结构
	if err := m.db.AutoMigrate(
		&models.User{},
	); err != nil {
		return fmt.Errorf("failed to auto migrate: %v", err)
	}

	// 为不同数据库执行特定的迁移
	switch m.db.Dialector.Name() {
	case "mysql":
		return m.migrateMySQL()
	case "postgres":
		return m.migratePostgreSQL()
	case "sqlite":
		return m.migrateSQLite()
	default:
		return nil
	}
}

func (m *Migration) migrateMySQL() error {
	log.Println("Running MySQL specific migrations...")
	// MySQL特定的迁移逻辑
	return nil
}

func (m *Migration) migratePostgreSQL() error {
	log.Println("Running PostgreSQL specific migrations...")
	// 确保ENUM类型存在
	if err := m.db.Exec(`DO $$ BEGIN
		CREATE TYPE user_role AS ENUM ('user', 'admin');
	EXCEPTION
		WHEN duplicate_object THEN null;
	END $$;`).Error; err != nil {
		return err
	}
	return nil
}

func (m *Migration) migrateSQLite() error {
	log.Println("Running SQLite specific migrations...")
	// SQLite不需要特殊处理
	return nil
}

// CreateAdminUser 创建默认管理员用户
func (m *Migration) CreateAdminUser() error {
	var count int64
	m.db.Model(&models.User{}).Count(&count)

	if count == 0 {
		// 这里应该使用服务层来创建用户，确保密码正确加密
		log.Println("No users found, would create admin user here")
		// 实际实现中应该调用UserService.CreateUser
	}

	return nil
}
