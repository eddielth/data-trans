package logger

import (
	"fmt"
	"log"
	"strings"
)

// Global logger instance
var defaultLogger *Logger

// Initialize default logger instance
func init() {
	// Initialize with default configuration
	logger, err := New(DefaultConfig())
	if err != nil {
		// If initialization fails, use standard log
		log.Printf("Failed to initialize default logger: %v, using standard log", err)
		return
	}

	defaultLogger = logger
}

// InitFromConfig initializes the logger from configuration
func InitFromConfig(level, filePath string, maxSize, maxBackups int, console bool) error {
	if defaultLogger != nil {
		// Close existing logger
		defaultLogger.Close()
	}

	// Parse log level
	logLevel, err := ParseLogLevel(level)
	if err != nil {
		return err
	}

	// Create new logger
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

// ParseLogLevel parses log level string
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
		return INFO, fmt.Errorf("unknown log level: %s, using default level INFO", level)
	}
}

// Debug logs debug level messages
func Debug(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debug(format, args...)
	} else {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// Info logs info level messages
func Info(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Info(format, args...)
	} else {
		log.Printf("[INFO] "+format, args...)
	}
}

// Warn logs warning level messages
func Warn(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warn(format, args...)
	} else {
		log.Printf("[WARN] "+format, args...)
	}
}

// Error logs error level messages
func Error(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Error(format, args...)
	} else {
		log.Printf("[ERROR] "+format, args...)
	}
}

// Close closes the logger
func Close() error {
	if defaultLogger != nil {
		return defaultLogger.Close()
	}
	return nil
}