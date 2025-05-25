package logger

import (
	"fmt"
	"log"
	"strings"
)

// 全局日志实例
var defaultLogger *Logger

// 初始化默认日志实例
func init() {
	// 使用默认配置初始化
	logger, err := New(DefaultConfig())
	if err != nil {
		// 如果初始化失败，使用标准日志
		log.Printf("初始化默认日志记录器失败: %v，将使用标准日志", err)
		return
	}

	defaultLogger = logger
}

// InitFromConfig 从配置初始化日志记录器
func InitFromConfig(level, filePath string, maxSize, maxBackups int, console bool) error {
	if defaultLogger != nil {
		// 关闭现有的日志记录器
		defaultLogger.Close()
	}

	// 解析日志级别
	logLevel, err := ParseLogLevel(level)
	if err != nil {
		return err
	}

	// 创建新的日志记录器
	logger, err := New(LoggerConfig{
		Level:      logLevel,
		FilePath:   filePath,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		Console:    console,
	})

	if err != nil {
		return err
	}

	defaultLogger = logger
	return nil
}

// ParseLogLevel 解析日志级别字符串
func ParseLogLevel(level string) (LogLevel, error) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG, nil
	case "INFO":
		return INFO, nil
	case "WARN", "WARNING":
		return WARN, nil
	case "ERROR":
		return ERROR, nil
	default:
		return INFO, fmt.Errorf("未知的日志级别: %s，使用默认级别 INFO", level)
	}
}

// Debug 记录调试级别的日志
func Debug(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debug(format, args...)
	} else {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// Info 记录信息级别的日志
func Info(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Info(format, args...)
	} else {
		log.Printf("[INFO] "+format, args...)
	}
}

// Warn 记录警告级别的日志
func Warn(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warn(format, args...)
	} else {
		log.Printf("[WARN] "+format, args...)
	}
}

// Error 记录错误级别的日志
func Error(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Error(format, args...)
	} else {
		log.Printf("[ERROR] "+format, args...)
	}
}

// Close 关闭日志记录器
func Close() error {
	if defaultLogger != nil {
		return defaultLogger.Close()
	}
	return nil
}
