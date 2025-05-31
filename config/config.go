package config

import (
	"path/filepath"
	"time"

	"github.com/eddielth/data-trans/logger"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config represents the application's configuration
type Config struct {
	MQTT         MQTTConfig             `mapstructure:"mqtt"`
	Transformers map[string]Transformer `mapstructure:"transformers"`
	Storage      StorageConfig          `mapstructure:"storage"`
	Logger       LoggerConfig           `mapstructure:"logger"`
}

// MQTTConfig represents the configuration for MQTT connection
type MQTTConfig struct {
	Broker   string   `mapstructure:"broker"`
	ClientID string   `mapstructure:"client_id"`
	Username string   `mapstructure:"username"`
	Password string   `mapstructure:"password"`
	Topics   []string `mapstructure:"topics"`
}

// Transformer represents the configuration for data transformers
type Transformer struct {
	ScriptPath string `mapstructure:"script_path"`
	ScriptCode string `mapstructure:"script_code"`
}

// LoggerConfig represents the configuration for logging
type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	Console    bool   `mapstructure:"console"`
}

// ConfigChangeCallback is the callback function type for configuration file changes
type ConfigChangeCallback func(cfg *Config) error

// LoadConfig loads the configuration file from the specified path
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

// WatchConfig monitors configuration file changes and calls the callback function
func WatchConfig(configPath string, callback ConfigChangeCallback) error {
	// Get the absolute path of the configuration file
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return err
	}

	// Set Viper to watch configuration file changes
	viper.SetConfigFile(absPath)
	viper.WatchConfig()

	// Debounce handling to avoid multiple triggers in a short time
	var lastChangeTime time.Time
	var debounceInterval = 2 * time.Second

	viper.OnConfigChange(func(e fsnotify.Event) {
		// Check if it's a write operation
		if e.Op&fsnotify.Write == fsnotify.Write {
			// Debounce handling
			now := time.Now()
			if now.Sub(lastChangeTime) < debounceInterval {
				return
			}
			lastChangeTime = now

			logger.Info("Configuration file change detected: %s", e.Name)

			// Reload configuration
			var newConfig Config
			err := viper.Unmarshal(&newConfig)
			if err != nil {
				logger.Error("Failed to parse updated configuration: %v", err)
				return
			}

			// Call callback function to handle new configuration
			if err := callback(&newConfig); err != nil {
				logger.Error("Failed to apply new configuration: %v", err)
				return
			}

			logger.Info("Configuration has been successfully updated and applied")
		}
	})

	return nil
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	File     FileStorageConfig     `mapstructure:"file"`
	Database DatabaseStorageConfig `mapstructure:"database"`
}

// FileStorageConfig represents file storage configuration
type FileStorageConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

// DatabaseStorageConfig represents database storage configuration
type DatabaseStorageConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Type    string `mapstructure:"type"`
	DSN     string `mapstructure:"dsn"`
}