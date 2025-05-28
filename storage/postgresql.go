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

// PostgreSQLStorage 表示PostgreSQL数据库存储后端
type PostgreSQLStorage struct {
	db       *sql.DB
	dsn      string
	database string
}

// NewPostgreSQLStorage 创建一个新的PostgreSQL存储后端
func NewPostgreSQLStorage(dsn string) (*PostgreSQLStorage, error) {
	// 解析DSN获取数据库名和服务器DSN
	database, serverDSN, err := parsePostgreSQLDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("解析PostgreSQL DSN失败: %v", err)
	}

	// 先连接到PostgreSQL服务器（不指定数据库）
	serverDB, err := sql.Open("postgres", serverDSN)
	if err != nil {
		return nil, fmt.Errorf("连接PostgreSQL服务器失败: %v", err)
	}
	defer serverDB.Close()

	// 检查数据库是否存在
	var exists bool
	err = serverDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", database).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("检查数据库是否存在失败: %v", err)
	}

	// 如果数据库不存在，则创建
	if !exists {
		// 创建数据库需要使用单独的连接，因为它不能在事务中执行
		_, err = serverDB.Exec(fmt.Sprintf("CREATE DATABASE %s", database))
		if err != nil {
			return nil, fmt.Errorf("创建数据库失败: %v", err)
		}
		logger.Info("已创建PostgreSQL数据库: %s", database)
	} else {
		logger.Info("PostgreSQL数据库已存在: %s", database)
	}

	// 连接到指定的数据库
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接PostgreSQL数据库失败: %v", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("PostgreSQL数据库连接测试失败: %v", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Minute * 5)

	storage := &PostgreSQLStorage{
		db:       db,
		dsn:      dsn,
		database: database,
	}

	// 初始化数据库和表
	if err := storage.InitDatabase(); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化PostgreSQL数据库失败: %v", err)
	}

	logger.Info("PostgreSQL数据库存储初始化成功")
	return storage, nil
}

// parsePostgreSQLDSN 解析PostgreSQL DSN字符串，提取数据库名和不包含数据库的DSN
func parsePostgreSQLDSN(dsn string) (database string, serverDSN string, err error) {
	// 检查是否是URL格式的DSN
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		// 解析URL格式的DSN
		// 格式: postgres://username:password@host:port/database?param=value
		parts := strings.Split(dsn, "/")
		if len(parts) < 4 {
			return "", "", fmt.Errorf("DSN格式无效，无法提取数据库名")
		}

		// 最后一部分可能包含参数
		dbParts := strings.Split(parts[len(parts)-1], "?")
		database = dbParts[0]

		// 创建不包含数据库名的DSN（用于连接到服务器）
		serverDSN = strings.Join(parts[:len(parts)-1], "/") + "/postgres"
		if len(dbParts) > 1 {
			serverDSN += "?" + dbParts[1]
		}
	} else {
		// 解析键值对格式的DSN
		// 格式: host=localhost port=5432 user=postgres password=secret dbname=mydb
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
			return "", "", fmt.Errorf("DSN格式无效，无法提取数据库名")
		}

		database = dbname
		serverDSN = strings.Join(serverKVPairs, " ") + " dbname=postgres"
	}

	return database, serverDSN, nil
}

// InitDatabase 初始化数据库和表
func (ps *PostgreSQLStorage) InitDatabase() error {
	// 创建设备数据表
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

	// 创建设备属性表
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

	// 执行创建表SQL
	_, err := ps.db.Exec(deviceTableSQL)
	if err != nil {
		return fmt.Errorf("创建设备数据表失败: %v", err)
	}

	_, err = ps.db.Exec(attributeTableSQL)
	if err != nil {
		return fmt.Errorf("创建设备属性表失败: %v", err)
	}

	logger.Info("PostgreSQL数据库表初始化成功")
	return nil
}

// Store 将数据存储到PostgreSQL数据库
func (ps *PostgreSQLStorage) Store(deviceType string, data transformer.DeviceData) error {
	// 开始事务
	tx, err := ps.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}

	// 确保事务最终会提交或回滚
	defer func() {
		if err != nil {
			tx.Rollback()
			logger.Error("PostgreSQL事务回滚: %v", err)
		}
	}()

	// 将元数据转换为JSON
	metadataJSON, err := json.Marshal(data.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %v", err)
	}

	// 插入设备数据
	deviceSQL := `INSERT INTO device_data (device_name, device_type, timestamp, metadata) VALUES ($1, $2, $3, $4) RETURNING id`
	var deviceDataID int64
	err = tx.QueryRow(deviceSQL, data.DeviceName, data.DeviceType, data.Timestamp, metadataJSON).Scan(&deviceDataID)
	if err != nil {
		return fmt.Errorf("插入设备数据失败: %v", err)
	}

	// 批量插入属性
	if len(data.Attributes) > 0 {
		// 构建批量插入SQL
		valueStrings := make([]string, 0, len(data.Attributes))
		valueArgs := make([]interface{}, 0, len(data.Attributes)*7)
		paramCounter := 1

		for _, attr := range data.Attributes {
			// 将属性值转换为字符串
			valueStr := fmt.Sprintf("%v", attr.Value)

			// 将属性元数据转换为JSON
			attrMetadataJSON, err := json.Marshal(attr.Metadata)
			if err != nil {
				return fmt.Errorf("序列化属性元数据失败: %v", err)
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
			return fmt.Errorf("插入设备属性失败: %v", err)
		}
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	logger.Debug("已将 %s 类型的数据存储到PostgreSQL数据库", deviceType)
	return nil
}

// Close 关闭数据库连接
func (ps *PostgreSQLStorage) Close() error {
	if ps.db != nil {
		err := ps.db.Close()
		if err != nil {
			return fmt.Errorf("关闭PostgreSQL数据库连接失败: %v", err)
		}
		logger.Info("PostgreSQL数据库连接已关闭")
	}
	return nil
}
