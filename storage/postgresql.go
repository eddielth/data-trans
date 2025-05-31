package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/eddielth/data-trans/logger"
	"github.com/eddielth/data-trans/transformer"
	_ "github.com/lib/pq"
)

// PostgreSQLStorage represents a PostgreSQL database storage backend
type PostgreSQLStorage struct {
	db       *sql.DB
	dsn      string
	database string
}

// NewPostgreSQLStorage creates a new PostgreSQL storage backend
func NewPostgreSQLStorage(dsn string) (*PostgreSQLStorage, error) {
	// Parse DSN to get database name and server DSN
	database, serverDSN, err := parsePostgreSQLDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PostgreSQL DSN: %v", err)
	}

	// First connect to PostgreSQL server (without specifying database)
	serverDB, err := sql.Open("postgres", serverDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL server: %v", err)
	}
	defer serverDB.Close()

	// Check if database exists
	var exists bool
	err = serverDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", database).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check database existence: %v", err)
	}

	// Create database if not exists
	if !exists {
		// Creating database requires separate connection as it can't be in transaction
		_, err = serverDB.Exec(fmt.Sprintf("CREATE DATABASE %s", database))
		if err != nil {
			return nil, fmt.Errorf("failed to create database: %v", err)
		}
		logger.Info("Created PostgreSQL database: %s", database)
	} else {
		logger.Info("PostgreSQL database already exists: %s", database)
	}

	// Connect to specified database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("PostgreSQL database connection test failed: %v", err)
	}

	// Set connection pool parameters
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Minute * 5)

	storage := &PostgreSQLStorage{
		db:       db,
		dsn:      dsn,
		database: database,
	}

	// Initialize database and tables
	if err := storage.InitDatabase(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize PostgreSQL database: %v", err)
	}

	logger.Info("PostgreSQL database storage initialized successfully")
	return storage, nil
}

// parsePostgreSQLDSN parses PostgreSQL DSN string, extracts database name and DSN without database
func parsePostgreSQLDSN(dsn string) (database string, serverDSN string, err error) {
	// Check if it's URL format DSN
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		// Parse URL format DSN
		// Format: postgres://username:password@host:port/database?param=value
		parts := strings.Split(dsn, "/")
		if len(parts) < 4 {
			return "", "", fmt.Errorf("invalid DSN format, unable to extract database name")
		}

		// Last part may contain parameters
		dbParts := strings.Split(parts[len(parts)-1], "?")
		database = dbParts[0]

		// Create DSN without database name (for connecting to server)
		serverDSN = strings.Join(parts[:len(parts)-1], "/") + "/postgres"
		if len(dbParts) > 1 {
			serverDSN += "?" + dbParts[1]
		}
	} else {
		// Parse key-value format DSN
		// Format: host=localhost port=5432 user=postgres password=secret dbname=mydb
		kvPairs := strings.Fields(dsn)
		dbname := ""
		serverKVPairs := make([]string, 0, len(kvPairs))

		for _, kv := range kvPairs {
			if strings.HasPrefix(kv, "dbname=") {
				dbname = strings.TrimPrefix(kv, "dbname=")
			} else {
				serverKVPairs = append(serverKVPairs, kv)
			}
		}

		if dbname == "" {
			return "", "", fmt.Errorf("invalid DSN format, unable to extract database name")
		}

		database = dbname
		serverDSN = strings.Join(serverKVPairs, " ") + " dbname=postgres"
	}

	return database, serverDSN, nil
}

// InitDatabase initializes database and tables
func (ps *PostgreSQLStorage) InitDatabase() error {
	// Create device data table
	deviceTableSQL := `
	CREATE TABLE IF NOT EXISTS device_data (
		id SERIAL PRIMARY KEY,
		device_name VARCHAR(255) NOT NULL,
		device_type VARCHAR(255) NOT NULL,
		timestamp BIGINT NOT NULL,
		metadata JSONB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_device_type ON device_data(device_type);
	CREATE INDEX IF NOT EXISTS idx_device_name ON device_data(device_name);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON device_data(timestamp);
	`

	// Create device attributes table
	attributeTableSQL := `
	CREATE TABLE IF NOT EXISTS device_attributes (
		id SERIAL PRIMARY KEY,
		device_data_id INTEGER NOT NULL,
		name VARCHAR(255) NOT NULL,
		type VARCHAR(50) NOT NULL,
		value TEXT NOT NULL,
		unit VARCHAR(50),
		quality INTEGER,
		metadata JSONB,
		FOREIGN KEY (device_data_id) REFERENCES device_data(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_device_data_id ON device_attributes(device_data_id);
	CREATE INDEX IF NOT EXISTS idx_name ON device_attributes(name);
	`

	// Execute table creation SQL
	_, err := ps.db.Exec(deviceTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create device data table: %v", err)
	}

	_, err = ps.db.Exec(attributeTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create device attributes table: %v", err)
	}

	logger.Info("PostgreSQL database tables initialized successfully")
	return nil
}

// Store stores data into PostgreSQL database
func (ps *PostgreSQLStorage) Store(deviceType string, data transformer.DeviceData) error {
	// Start transaction
	tx, err := ps.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}

	// Ensure transaction will be committed or rolled back
	defer func() {
		if err != nil {
			tx.Rollback()
			logger.Error("PostgreSQL transaction rolled back: %v", err)
		}
	}()

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(data.Metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %v", err)
	}

	// Insert device data
	deviceSQL := `INSERT INTO device_data (device_name, device_type, timestamp, metadata) VALUES ($1, $2, $3, $4) RETURNING id`
	var deviceDataID int64
	err = tx.QueryRow(deviceSQL, data.DeviceName, data.DeviceType, data.Timestamp, metadataJSON).Scan(&deviceDataID)
	if err != nil {
		return fmt.Errorf("failed to insert device data: %v", err)
	}

	// Batch insert attributes
	if len(data.Attributes) > 0 {
		// Build batch insert SQL
		valueStrings := make([]string, 0, len(data.Attributes))
		valueArgs := make([]interface{}, 0, len(data.Attributes)*7)
		paramCounter := 1

		for _, attr := range data.Attributes {
			// Convert attribute value to string
			valueStr := fmt.Sprintf("%v", attr.Value)

			// Convert attribute metadata to JSON
			attrMetadataJSON, err := json.Marshal(attr.Metadata)
			if err != nil {
				return fmt.Errorf("failed to serialize attribute metadata: %v", err)
			}

			placeholders := fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				paramCounter, paramCounter+1, paramCounter+2, paramCounter+3, paramCounter+4, paramCounter+5, paramCounter+6)
			valueStrings = append(valueStrings, placeholders)
			valueArgs = append(valueArgs, deviceDataID, attr.Name, attr.Type, valueStr, attr.Unit, attr.Quality, attrMetadataJSON)
			paramCounter += 7
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

	logger.Debug("Stored %s type data in PostgreSQL database", deviceType)
	return nil
}

// Close closes the database connection
func (ps *PostgreSQLStorage) Close() error {
	if ps.db != nil {
		err := ps.db.Close()
		if err != nil {
			return fmt.Errorf("failed to close PostgreSQL database connection: %v", err)
		}
		logger.Info("PostgreSQL database connection closed")
	}
	return nil
}
