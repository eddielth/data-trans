package storage

import (
	"fmt"
)

// DatabaseType
type DatabaseType string

const (
	// MySQL
	MySQL DatabaseType = "mysql"
	// PostgreSQL
	PostgreSQL DatabaseType = "postgresql"
)

// DatabaseStorage
type DatabaseStorage interface {
	StorageBackend
	// InitDatabase
	InitDatabase() error
}

// NewDatabaseStorage
func NewDatabaseStorage(dbType string, dsn string) (DatabaseStorage, error) {
	switch DatabaseType(dbType) {
	case MySQL:
		return NewMySQLStorage(dsn)
	case PostgreSQL:
		return NewPostgreSQLStorage(dsn)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
