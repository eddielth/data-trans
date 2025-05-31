package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents the log level
type LogLevel int

const (
	// DEBUG level
	DEBUG LogLevel = iota
	// INFO level
	INFO
	// WARN level
	WARN
	// ERROR level
	ERROR
)

// String representation of log levels
var levelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
}

// Logger represents the logger
type Logger struct {
	level       LogLevel
	output      io.Writer
	filePath    string
	maxSize     int64 // Unit: bytes
	maxBackups  int
	currentSize int64
	mu          sync.Mutex
}

// LoggerConfig represents the configuration for the logger
type LoggerConfig struct {
	// Log level
	Level LogLevel
	// Log file path
	FilePath string
	// Maximum log file size
	MaxSize int
	// Maximum number of backups
	MaxBackups int
	// Whether to log to console
	Console bool
}

// DefaultConfig returns default logger configuration
func DefaultConfig() LoggerConfig {
	return LoggerConfig{
		Level:      INFO,
		FilePath:   "./logs/app.log",
		MaxSize:    10, // 10MB
		MaxBackups: 5,
		Console:    true,
	}
}

// New creates a new logger
func New(config LoggerConfig) (*Logger, error) {
	// Ensure log directory exists
	logDir := filepath.Dir(config.FilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// Open log file
	file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to get log file info: %v", err)
	}

	// Create output writer
	var output io.Writer = file
	if config.Console {
		// Output to both console and file
		output = io.MultiWriter(os.Stdout, file)
	}

	return &Logger{
		level:       config.Level,
		output:      output,
		filePath:    config.FilePath,
		maxSize:     int64(config.MaxSize) * 1024 * 1024, // Convert to bytes
		maxBackups:  config.MaxBackups,
		currentSize: info.Size(),
		mu:          sync.Mutex{},
	}, nil
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// log is the internal method for logging
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	// Check log level
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Get caller information
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Keep only the filename
	file = filepath.Base(file)

	// Format log message
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelStr := levelNames[level]
	msg := fmt.Sprintf(format, args...)

	colorCode := ""
	resetColor := "\033[0m"

	switch level {
	case DEBUG:
		colorCode = "\033[90m" // Gray
	case INFO:
		colorCode = "\033[32m" // Green
	case WARN:
		colorCode = "\033[33m" // Yellow
	case ERROR:
		colorCode = "\033[31m" // Red
	}

	logEntry := fmt.Sprintf("%s [%s%s%s] %s:%d: %s\n", timestamp, colorCode, levelStr, resetColor, file, line, msg)

	// Write log
	n, err := io.WriteString(l.output, logEntry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write log: %v\n", err)
		return
	}

	// Update current file size
	l.currentSize += int64(n)

	// Check if log rotation is needed
	if l.currentSize >= l.maxSize {
		l.rotate()
	}
}

// rotate rotates the log file
func (l *Logger) rotate() {
	// Close current log file
	if closer, ok := l.output.(io.Closer); ok {
		closer.Close()
	}

	// Generate new log filename (with timestamp)
	timestamp := time.Now().Format("20060102-150405")
	dir := filepath.Dir(l.filePath)
	base := filepath.Base(l.filePath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	backupPath := filepath.Join(dir, fmt.Sprintf("%s.%s%s", name, timestamp, ext))

	// Rename current log file
	os.Rename(l.filePath, backupPath)

	// Clean up old log files
	l.cleanOldLogs()

	// Create new log file
	file, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create new log file: %v\n", err)
		return
	}

	// Update output
	// Check if console output was previously configured
	var consoleConfigured bool
	// Try to determine if console output was configured from field names or struct characteristics
	_, consoleConfigured = l.output.(*os.File)
	if !consoleConfigured && l.output != nil {
		// If it's not a file type but output is not nil, assume multi-output was previously configured
		consoleConfigured = true
	}

	// Decide new output method based on previous configuration
	if consoleConfigured {
		// If console output was previously configured, continue using multi-output
		l.output = io.MultiWriter(os.Stdout, file)
	} else {
		// Otherwise, output to file only
		l.output = file
	}

	l.currentSize = 0
}

// cleanOldLogs cleans up old log files
func (l *Logger) cleanOldLogs() {
	dir := filepath.Dir(l.filePath)
	base := filepath.Base(l.filePath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	pattern := filepath.Join(dir, name+".*"+ext)

	// Find all matching log files
	matches, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to find old log files: %v\n", err)
		return
	}

	// If the number of log files exceeds the maximum backup count, delete the oldest files
	if len(matches) > l.maxBackups {
		// Sort by modification time
		type fileInfo struct {
			path string
			time time.Time
		}
		files := make([]fileInfo, 0, len(matches))

		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				continue
			}
			files = append(files, fileInfo{match, info.ModTime()})
		}

		// Sort by time (oldest first)
		for i := 0; i < len(files); i++ {
			for j := i + 1; j < len(files); j++ {
				if files[i].time.After(files[j].time) {
					files[i], files[j] = files[j], files[i]
				}
			}
		}

		// Delete excess old files
		for i := 0; i < len(files)-l.maxBackups; i++ {
			os.Remove(files[i].path)
		}
	}
}

// Debug logs debug level messages
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info logs info level messages
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn logs warning level messages
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error logs error level messages
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Close closes the logger
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if closer, ok := l.output.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}