package config

import (
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config 表示应用程序的配置// 更新Config结构体
type Config struct {
	MQTT         MQTTConfig             `mapstructure:"mqtt"`
	Transformers map[string]Transformer `mapstructure:"transformers"`
	Storage      StorageConfig          `mapstructure:"storage"`
	Logger       LoggerConfig           `mapstructure:"logger"`
}

// MQTTConfig 表示MQTT连接的配置
type MQTTConfig struct {
	Broker   string   `mapstructure:"broker"`
	ClientID string   `mapstructure:"client_id"`
	Username string   `mapstructure:"username"`
	Password string   `mapstructure:"password"`
	Topics   []string `mapstructure:"topics"`
}

// Transformer 表示数据转换器的配置
type Transformer struct {
	ScriptPath string `mapstructure:"script_path"`
	ScriptCode string `mapstructure:"script_code"`
}

// LoggerConfig 表示日志配置
type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	Console    bool   `mapstructure:"console"`
}

// ConfigChangeCallback 是配置文件变更时的回调函数类型
type ConfigChangeCallback func(cfg *Config) error

// LoadConfig 从指定路径加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// WatchConfig 监听配置文件变化并调用回调函数
func WatchConfig(configPath string, callback ConfigChangeCallback) error {
	// 获取配置文件的绝对路径
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return err
	}

	// 设置Viper监听配置文件变化
	viper.SetConfigFile(absPath)
	viper.WatchConfig()

	// 防抖动处理，避免短时间内多次触发
	var lastChangeTime time.Time
	var debounceInterval = 2 * time.Second

	viper.OnConfigChange(func(e fsnotify.Event) {
		// 检查是否是写入操作
		if e.Op&fsnotify.Write == fsnotify.Write {
			// 防抖动处理
			now := time.Now()
			if now.Sub(lastChangeTime) < debounceInterval {
				return
			}
			lastChangeTime = now

			log.Printf("检测到配置文件变更: %s", e.Name)

			// 重新加载配置
			var newConfig Config
			err := viper.Unmarshal(&newConfig)
			if err != nil {
				log.Printf("解析更新后的配置失败: %v", err)
				return
			}

			// 调用回调函数处理新配置
			if err := callback(&newConfig); err != nil {
				log.Printf("应用新配置失败: %v", err)
				return
			}

			log.Println("配置已成功更新并应用")
		}
	})

	return nil
}

// StorageConfig 表示存储配置
type StorageConfig struct {
	File     FileStorageConfig     `mapstructure:"file"`
	Database DatabaseStorageConfig `mapstructure:"database"`
}

// FileStorageConfig 表示文件存储配置
type FileStorageConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

// DatabaseStorageConfig 表示数据库存储配置
type DatabaseStorageConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Type    string `mapstructure:"type"`
	DSN     string `mapstructure:"dsn"`
}
