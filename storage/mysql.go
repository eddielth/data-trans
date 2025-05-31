package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/eddielth/data-trans/logger"
	"github.com/eddielth/data-trans/transformer"
	_ "github.com/go-sql-driver/mysql"
)

// MySQLStorage represents a MySQL database storage backend
type MySQLStorage struct {
	db       *sql.DB
	dsn      string
	database string
}

// NewMySQLStorage creates a new MySQL storage backend
func NewMySQLStorage(dsn string) (*MySQLStorage, error) {
	// Parse DSN to get database name
	database, serverDSN, err := parseMySQLDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MySQL DSN: %v", err)
	}

	// First connect to MySQL server (without specifying database)
	serverDB, err := sql.Open("mysql", serverDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL server: %v", err)
	}
	defer serverDB.Close()

	// Create database if not exists
	_, err = serverDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", database))
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %v", err)
	}

	logger.Info("Ensured MySQL database %s exists", database)

	// Connect to specified database
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("MySQL database connection test failed: %v", err)
	}

	// Set connection pool parameters
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Minute * 5)

	storage := &MySQLStorage{
		db:       db,
		dsn:      dsn,
		database: database,
	}

	// Initialize database and tables
	if err := storage.InitDatabase(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize MySQL database: %v", err)
	}

	logger.Info("MySQL database storage initialized successfully")
	return storage, nil
}

// parseMySQLDSN parses MySQL DSN string, extracts database name and DSN without database
func parseMySQLDSN(dsn string) (database string, serverDSN string, err error) {
	// Find database name in DSN
	parts := strings.Split(dsn, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid DSN format, unable to extract database name")
	}

	// Last part may contain parameters
	dbParts := strings.Split(parts[len(parts)-1], "?")
	database = dbParts[0]

	// Create DSN without database name (for connecting to server)
	serverDSN = strings.Join(parts[:len(parts)-1], "/") + "/"
	if len(dbParts) > 1 {
		serverDSN += "?" + dbParts[1]
	}

	return database, serverDSN, nil
}

// InitDatabase initializes database and tables
func (ms *MySQLStorage) InitDatabase() error {
	// Create device data table
	deviceTableSQL := `
	CREATE TABLE IF NOT EXISTS device_data (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		device_name VARCHAR(255) NOT NULL,
		device_type VARCHAR(255) NOT NULL,
		timestamp BIGINT NOT NULL,
		metadata JSON,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_device_type (device_type),
		INDEX idx_device_name (device_name),
		INDEX idx_timestamp (timestamp)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`

	// Create device attributes table
	attributeTableSQL := `
	CREATE TABLE IF NOT EXISTS device_attributes (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		device_data_id BIGINT NOT NULL,
		name VARCHAR(255) NOT NULL,
		type VARCHAR(50) NOT NULL,
		value TEXT NOT NULL,
		unit VARCHAR(50),
		quality INT,
		metadata JSON,
		FOREIGN KEY (device_data_id) REFERENCES device_data(id) ON DELETE CASCADE,
		INDEX idx_device_data_id (device_data_id),
		INDEX idx_name (name)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`

	// Execute table creation SQL
	_, err := ms.db.Exec(deviceTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create device data table: %v", err)
	}

	_, err = ms.db.Exec(attributeTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create device attributes table: %v", err)
	}

	logger.Info("MySQL database tables initialized successfully")
	return nil
}

// Store stores data into MySQL database
func (ms *MySQLStorage) Store(deviceType string, data transformer.DeviceData) error {
	// Start transaction
	tx, err := ms.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}

	// Ensure transaction will be committed or rolled back
	defer func() {
		if err != nil {
			tx.Rollback()
			logger.Error("MySQL transaction rolled back: %v", err)
		}
	}()

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(data.Metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %v", err)
	}

	// Insert device data
	deviceSQL := `INSERT INTO device_data (device_name, device_type, timestamp, metadata) VALUES (?, ?, ?, ?)`
	result, err := tx.Exec(deviceSQL, data.DeviceName, data.DeviceType, data.Timestamp, metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to insert device data: %v", err)
	}

	// Get inserted ID
	deviceDataID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get insert ID: %v", err)
	}

	// Batch insert attributes
	if len(data.Attributes) > 0 {
		// Build batch insert SQL
		valueStrings := make([]string, 0, len(data.Attributes))
		valueArgs := make([]interface{}, 0, len(data.Attributes)*7)

		for _, attr := range data.Attributes {
			// Convert attribute value to string
			valueStr := fmt.Sprintf("%v", attr.Value)

			// Convert attribute metadata to JSON
			attrMetadataJSON, err := json.Marshal(attr.Metadata)
			if err != nil {
				return fmt.Errorf("failed to serialize attribute metadata: %v", err)
			}

			valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?)")
			valueArgs = append(valueArgs, deviceDataID, attr.Name, attr.Type, valueStr, attr.Unit, attr.Quality, attrMetadataJSON)
		}

		attrSQL := fmt.Sprintf("INSERT INTO device_attributes (device_data_id, name, type, value, unit, quality, metadata) VALUES %s",
			strings.Join(valueStrings, ","))

		_, err = tx.Exec(attrSQL, valueArgs...)
		if err != nil {
			return fmt.Errorf("failed to insert device attributes: %v", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	logger.Debug("Stored %s type data in MySQL database", deviceType)
	return nil
}

// Close closes the database connection
func (ms *MySQLStorage) Close() error {
	if ms.db != nil {
		err := ms.db.Close()
		if err != nil {
			return fmt.Errorf("failed to close MySQL database connection: %v", err)
		}
		logger.Info("MySQL database connection closed")
	}
	return nil
}