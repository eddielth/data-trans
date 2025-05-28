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

// MySQLStorage 表示MySQL数据库存储后端
type MySQLStorage struct {
	db       *sql.DB
	dsn      string
	database string
}

// NewMySQLStorage 创建一个新的MySQL存储后端
func NewMySQLStorage(dsn string) (*MySQLStorage, error) {
	// 解析DSN获取数据库名
	database, serverDSN, err := parseMySQLDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("解析MySQL DSN失败: %v", err)
	}

	// 先连接到MySQL服务器（不指定数据库）
	serverDB, err := sql.Open("mysql", serverDSN)
	if err != nil {
		return nil, fmt.Errorf("连接MySQL服务器失败: %v", err)
	}
	defer serverDB.Close()

	// 创建数据库（如果不存在）
	_, err = serverDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", database))
	if err != nil {
		return nil, fmt.Errorf("创建数据库失败: %v", err)
	}

	logger.Info("确保MySQL数据库 %s 存在", database)

	// 连接到指定的数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接MySQL数据库失败: %v", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("MySQL数据库连接测试失败: %v", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Minute * 5)

	storage := &MySQLStorage{
		db:       db,
		dsn:      dsn,
		database: database,
	}

	// 初始化数据库和表
	if err := storage.InitDatabase(); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化MySQL数据库失败: %v", err)
	}

	logger.Info("MySQL数据库存储初始化成功")
	return storage, nil
}

// parseMySQLDSN 解析MySQL DSN字符串，提取数据库名和不包含数据库的DSN
func parseMySQLDSN(dsn string) (database string, serverDSN string, err error) {
	// 查找DSN中的数据库名
	parts := strings.Split(dsn, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("DSN格式无效，无法提取数据库名")
	}

	// 最后一部分可能包含参数
	dbParts := strings.Split(parts[len(parts)-1], "?")
	database = dbParts[0]

	// 创建不包含数据库名的DSN（用于连接到服务器）
	serverDSN = strings.Join(parts[:len(parts)-1], "/") + "/"
	if len(dbParts) > 1 {
		serverDSN += "?" + dbParts[1]
	}

	return database, serverDSN, nil
}

// InitDatabase 初始化数据库和表
func (ms *MySQLStorage) InitDatabase() error {
	// 创建设备数据表
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

	// 创建设备属性表
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

	// 执行创建表SQL
	_, err := ms.db.Exec(deviceTableSQL)
	if err != nil {
		return fmt.Errorf("创建设备数据表失败: %v", err)
	}

	_, err = ms.db.Exec(attributeTableSQL)
	if err != nil {
		return fmt.Errorf("创建设备属性表失败: %v", err)
	}

	logger.Info("MySQL数据库表初始化成功")
	return nil
}

// Store 将数据存储到MySQL数据库
func (ms *MySQLStorage) Store(deviceType string, data transformer.DeviceData) error {
	// 开始事务
	tx, err := ms.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}

	// 确保事务最终会提交或回滚
	defer func() {
		if err != nil {
			tx.Rollback()
			logger.Error("MySQL事务回滚: %v", err)
		}
	}()

	// 将元数据转换为JSON
	metadataJSON, err := json.Marshal(data.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %v", err)
	}

	// 插入设备数据
	deviceSQL := `INSERT INTO device_data (device_name, device_type, timestamp, metadata) VALUES (?, ?, ?, ?)`
	result, err := tx.Exec(deviceSQL, data.DeviceName, data.DeviceType, data.Timestamp, metadataJSON)
	if err != nil {
		return fmt.Errorf("插入设备数据失败: %v", err)
	}

	// 获取插入的ID
	deviceDataID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取插入ID失败: %v", err)
	}

	// 批量插入属性
	if len(data.Attributes) > 0 {
		// 构建批量插入SQL
		valueStrings := make([]string, 0, len(data.Attributes))
		valueArgs := make([]interface{}, 0, len(data.Attributes)*7)

		for _, attr := range data.Attributes {
			// 将属性值转换为字符串
			valueStr := fmt.Sprintf("%v", attr.Value)

			// 将属性元数据转换为JSON
			attrMetadataJSON, err := json.Marshal(attr.Metadata)
			if err != nil {
				return fmt.Errorf("序列化属性元数据失败: %v", err)
			}

			valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?)")
			valueArgs = append(valueArgs, deviceDataID, attr.Name, attr.Type, valueStr, attr.Unit, attr.Quality, attrMetadataJSON)
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

	logger.Debug("已将 %s 类型的数据存储到MySQL数据库", deviceType)
	return nil
}

// Close 关闭数据库连接
func (ms *MySQLStorage) Close() error {
	if ms.db != nil {
		err := ms.db.Close()
		if err != nil {
			return fmt.Errorf("关闭MySQL数据库连接失败: %v", err)
		}
		logger.Info("MySQL数据库连接已关闭")
	}
	return nil
}
