package storage

import (
	"fmt"
)

// DatabaseType 表示数据库类型
type DatabaseType string

const (
	// MySQL 数据库类型
	MySQL DatabaseType = "mysql"
	// PostgreSQL 数据库类型
	PostgreSQL DatabaseType = "postgresql"
)

// DatabaseStorage 表示数据库存储后端的接口
type DatabaseStorage interface {
	StorageBackend
	// InitDatabase 初始化数据库和表
	InitDatabase() error
}

// NewDatabaseStorage 创建一个新的数据库存储后端
func NewDatabaseStorage(dbType string, dsn string) (DatabaseStorage, error) {
	switch DatabaseType(dbType) {
	case MySQL:
		return NewMySQLStorage(dsn)
	case PostgreSQL:
		return NewPostgreSQLStorage(dsn)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", dbType)
	}
}
