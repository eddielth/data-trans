package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/eddielth/data-trans/config"
	"github.com/eddielth/data-trans/logger"
	"github.com/eddielth/data-trans/mqtt"
	"github.com/eddielth/data-trans/storage"
	"github.com/eddielth/data-trans/transformer"
)

// 初始化配置
func initConfig(configPath string) (*config.Config, error) {
	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Error("加载配置失败: %v", err)
		return nil, err
	}
	return cfg, nil
}

// 初始化日志系统
func initLogger(cfg *config.Config) error {
	err := logger.InitFromConfig(
		cfg.Logger.Level,
		cfg.Logger.FilePath,
		cfg.Logger.MaxSize,
		cfg.Logger.MaxBackups,
		cfg.Logger.Console,
	)
	if err != nil {
		logger.Error("初始化日志系统失败: %v", err)
		// 继续使用默认日志配置
	}
	return nil
}

// 初始化存储系统
func initStorage(cfg *config.Config) (*storage.Manager, error) {
	var storageBackends []storage.StorageBackend

	// 添加文件存储后端
	if cfg.Storage.File.Enabled {
		fileStorage, err := storage.NewFileStorage(cfg.Storage.File.Path)
		if err != nil {
			logger.Warn("初始化文件存储失败: %v", err)
		} else {
			storageBackends = append(storageBackends, fileStorage)
			logger.Info("已启用文件存储")
		}
	}

	// 添加数据库存储后端
	if cfg.Storage.Database.Enabled {
		// 初始化数据库存储
		dbStorage, err := storage.NewDatabaseStorage(cfg.Storage.Database.Type, cfg.Storage.Database.DSN)
		if err != nil {
			logger.Warn("初始化数据库存储失败: %v", err)
		} else {
			storageBackends = append(storageBackends, dbStorage)
			logger.Info("已启用%s数据库存储", cfg.Storage.Database.Type)
		}
	}

	return storage.NewManager(storageBackends), nil
}

// 监听配置文件变化
func watchConfigChanges(configPath string, transformerManager *transformer.Manager, storageManager *storage.Manager) error {
	err := config.WatchConfig(configPath, func(newCfg *config.Config) error {
		logger.Info("正在应用新的配置...")

		// 检查并更新日志配置
		if err := logger.InitFromConfig(
			newCfg.Logger.Level,
			newCfg.Logger.FilePath,
			newCfg.Logger.MaxSize,
			newCfg.Logger.MaxBackups,
			newCfg.Logger.Console,
		); err != nil {
			logger.Warn("重新加载日志配置失败: %v", err)
		} else {
			logger.Info("已重新加载日志配置")
		}

		// 检查并更新转换器
		for deviceType, transformerCfg := range newCfg.Transformers {
			if err := transformerManager.ReloadTransformer(deviceType, transformerCfg); err != nil {
				logger.Warn("重新加载转换器 %s 失败: %v", deviceType, err)
				// 继续处理其他转换器，不中断整个过程
			}
		}

		// 检查并更新数据库存储配置
		if newCfg.Storage.Database.Enabled {
			// 先移除同类型的旧数据库后端
			storageManager.RemoveBackendByType(newCfg.Storage.Database.Type)

			// 初始化新的数据库连接
			dbStorage, err := storage.NewDatabaseStorage(newCfg.Storage.Database.Type, newCfg.Storage.Database.DSN)
			if err != nil {
				logger.Warn("重新加载数据库存储失败: %v", err)
			} else {
				// 添加新的数据库后端
				storageManager.AddBackend(dbStorage)
				logger.Info("已重新加载%s数据库存储", newCfg.Storage.Database.Type)
			}
		}

		// 如果MQTT配置发生变化，可以在这里处理
		// 例如重新连接MQTT服务器或重新订阅主题
		// 这里简化处理，仅打印日志
		logger.Info("MQTT配置更新将在服务重启后生效")

		return nil
	})

	if err != nil {
		logger.Warn("监听配置文件变化失败: %v", err)
		// 不致命，继续运行
	} else {
		logger.Info("已启动配置文件监听")
	}

	return nil
}

// 等待退出信号
func waitForExitSignal() os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	return <-sigChan
}

func main() {
	// 配置文件路径
	configPath := "config.yaml"

	// 初始化配置
	cfg, err := initConfig(configPath)
	if err != nil {
		os.Exit(1)
	}

	// 初始化日志系统
	initLogger(cfg)
	logger.Info("数据转换服务正在启动...")
	defer logger.Close()

	// 初始化转换器管理器
	transformerManager, err := transformer.NewManager(cfg.Transformers)
	if err != nil {
		logger.Error("初始化转换器管理器失败: %v", err)
		os.Exit(1)
	}

	// 初始化存储系统
	storageManager, err := initStorage(cfg)
	if err != nil {
		logger.Error("初始化存储系统失败: %v", err)
		os.Exit(1)
	}
	defer storageManager.Close()

	// 初始化MQTT管理器
	mqttManager, err := mqtt.NewManager(cfg, transformerManager, storageManager)
	if err != nil {
		logger.Error("初始化MQTT管理器失败: %v", err)
		os.Exit(1)
	}

	// 启动MQTT服务
	if err := mqttManager.Start(); err != nil {
		logger.Error("启动MQTT服务失败: %v", err)
		os.Exit(1)
	}

	// 监听配置文件变化
	watchConfigChanges(configPath, transformerManager, storageManager)

	logger.Info("数据转换服务已启动，等待设备数据...")

	// 等待退出信号
	_ = waitForExitSignal()

	// 停止MQTT服务
	mqttManager.Stop()
	logger.Info("服务已安全停止")
}
