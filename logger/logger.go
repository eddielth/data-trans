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

// LogLevel 表示日志级别
type LogLevel int

const (
	// DEBUG 调试级别
	DEBUG LogLevel = iota
	// INFO 信息级别
	INFO
	// WARN 警告级别
	WARN
	// ERROR 错误级别
	ERROR
)

// 日志级别对应的字符串表示
var levelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
}

// Logger 表示日志记录器
type Logger struct {
	level       LogLevel
	output      io.Writer
	filePath    string
	maxSize     int64 // 单位：字节
	maxBackups  int
	currentSize int64
	mu          sync.Mutex
}

// LoggerConfig 表示日志记录器的配置
type LoggerConfig struct {
	// 日志级别
	Level LogLevel
	// 日志文件路径
	FilePath string
	// 单个日志文件的最大大小（MB）
	MaxSize int
	// 最大保留的日志文件数量
	MaxBackups int
	// 是否同时输出到控制台
	Console bool
}

// DefaultConfig 返回默认的日志配置
func DefaultConfig() LoggerConfig {
	return LoggerConfig{
		Level:      INFO,
		FilePath:   "./logs/app.log",
		MaxSize:    10, // 10MB
		MaxBackups: 5,
		Console:    true,
	}
}

// New 创建一个新的日志记录器
func New(config LoggerConfig) (*Logger, error) {
	// 确保日志目录存在
	logDir := filepath.Dir(config.FilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 打开日志文件
	file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %v", err)
	}

	// 获取当前文件大小
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("获取日志文件信息失败: %v", err)
	}

	// 创建输出写入器
	var output io.Writer = file
	if config.Console {
		// 同时输出到控制台和文件
		output = io.MultiWriter(os.Stdout, file)
	}

	return &Logger{
		level:       config.Level,
		output:      output,
		filePath:    config.FilePath,
		maxSize:     int64(config.MaxSize) * 1024 * 1024, // 转换为字节
		maxBackups:  config.MaxBackups,
		currentSize: info.Size(),
		mu:          sync.Mutex{},
	}, nil
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// log 记录日志的内部方法
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	// 检查日志级别
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 获取调用者信息
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		file = "unknown"
		line = 0
	}

	// 只保留文件名
	file = filepath.Base(file)

	// 格式化日志消息
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelStr := levelNames[level]
	msg := fmt.Sprintf(format, args...)

	colorCode := ""
	resetColor := "\033[0m"

	switch level {
	case DEBUG:
		colorCode = "\033[90m" // 灰色
	case INFO:
		colorCode = "\033[32m" // 绿色
	case WARN:
		colorCode = "\033[33m" // 黄色
	case ERROR:
		colorCode = "\033[31m" // 红色
	}

	logEntry := fmt.Sprintf("%s [%s%s%s] %s:%d: %s\n", timestamp, colorCode, levelStr, resetColor, file, line, msg)

	// 写入日志
	n, err := io.WriteString(l.output, logEntry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "写入日志失败: %v\n", err)
		return
	}

	// 更新当前文件大小
	l.currentSize += int64(n)

	// 检查是否需要轮转日志
	if l.currentSize >= l.maxSize {
		l.rotate()
	}
}

// rotate 轮转日志文件
func (l *Logger) rotate() {
	// 关闭当前日志文件
	if closer, ok := l.output.(io.Closer); ok {
		closer.Close()
	}

	// 生成新的日志文件名（使用时间戳）
	timestamp := time.Now().Format("20060102-150405")
	dir := filepath.Dir(l.filePath)
	base := filepath.Base(l.filePath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	backupPath := filepath.Join(dir, fmt.Sprintf("%s.%s%s", name, timestamp, ext))

	// 重命名当前日志文件
	os.Rename(l.filePath, backupPath)

	// 清理旧的日志文件
	l.cleanOldLogs()

	// 创建新的日志文件
	file, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建新的日志文件失败: %v\n", err)
		return
	}

	// 更新输出
	// 检查之前是否配置了控制台输出
	var consoleConfigured bool
	// 尝试从字段名或结构体特征判断是否有控制台输出
	_, consoleConfigured = l.output.(*os.File)
	if !consoleConfigured && l.output != nil {
		// 如果不是文件类型但输出不为空，假设之前配置了多输出
		consoleConfigured = true
	}

	// 根据之前的配置决定新的输出方式
	if consoleConfigured {
		// 如果之前配置了控制台输出，继续使用多输出
		l.output = io.MultiWriter(os.Stdout, file)
	} else {
		// 否则只输出到文件
		l.output = file
	}

	l.currentSize = 0
}

// cleanOldLogs 清理旧的日志文件
func (l *Logger) cleanOldLogs() {
	dir := filepath.Dir(l.filePath)
	base := filepath.Base(l.filePath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	pattern := filepath.Join(dir, name+".*"+ext)

	// 查找所有匹配的日志文件
	matches, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "查找旧日志文件失败: %v\n", err)
		return
	}

	// 如果日志文件数量超过最大备份数，删除最旧的文件
	if len(matches) > l.maxBackups {
		// 按修改时间排序
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

		// 按时间排序（最旧的在前面）
		for i := 0; i < len(files); i++ {
			for j := i + 1; j < len(files); j++ {
				if files[i].time.After(files[j].time) {
					files[i], files[j] = files[j], files[i]
				}
			}
		}

		// 删除多余的旧文件
		for i := 0; i < len(files)-l.maxBackups; i++ {
			os.Remove(files[i].path)
		}
	}
}

// Debug 记录调试级别的日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info 记录信息级别的日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn 记录警告级别的日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error 记录错误级别的日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Close 关闭日志记录器
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if closer, ok := l.output.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
