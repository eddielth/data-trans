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

func main() {
	// 配置文件路径
	configPath := "config.yaml"

	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Error("加载配置失败: %v", err)
		os.Exit(1)
	}

	// 初始化日志系统
	if err = logger.InitFromConfig(
		cfg.Logger.Level,
		cfg.Logger.FilePath,
		cfg.Logger.MaxSize,
		cfg.Logger.MaxBackups,
		cfg.Logger.Console,
	); err != nil {
		logger.Error("初始化日志系统失败: %v", err)
		// 继续使用默认日志配置
	}

	logger.Info("数据转换服务正在启动...")
	defer logger.Close()

	// 初始化转换器管理器
	transformerManager, err := transformer.NewManager(cfg.Transformers)
	if err != nil {
		logger.Error("初始化转换器管理器失败: %v", err)
		os.Exit(1)
	}

	// 初始化存储管理器
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

	// 添加数据库存储后端（如果已实现）
	if cfg.Storage.Database.Enabled {
		// 初始化数据库存储
		// ...
	}

	storageManager := storage.NewManager(storageBackends)
	defer storageManager.Close()

	// 初始化MQTT客户端
	mqttClient, err := mqtt.NewClient(cfg.MQTT, func(topic string, payload []byte) {
		// 根据主题确定设备类型
		deviceType := mqtt.GetDeviceTypeFromTopic(topic)
		if deviceType == "" {
			logger.Warn("无法从主题 %s 确定设备类型", topic)
			return
		}

		logger.Debug("收到来自设备类型 %s 的数据: %s", deviceType, string(payload))

		// 使用对应的转换器处理数据
		result, err := transformerManager.Transform(deviceType, payload)
		if err != nil {
			logger.Error("转换数据失败 [%s]: %v", deviceType, err)
			return
		}

		// 处理转换后的数据
		logger.Info("设备类型: %s, 转换后数据: %v", deviceType, result)

		// 存储数据
		if err := storageManager.Store(deviceType, result); err != nil {
			logger.Error("存储数据失败: %v", err)
		}
	})

	if err != nil {
		logger.Error("初始化MQTT客户端失败: %v", err)
		os.Exit(1)
	}

	// 连接MQTT服务器
	if err = mqttClient.Connect(); err != nil {
		logger.Error("连接MQTT服务器失败: %v", err)
		os.Exit(1)
	}

	// 订阅配置的主题
	for _, topic := range cfg.MQTT.Topics {
		if err = mqttClient.Subscribe(topic); err != nil {
			logger.Warn("订阅主题 %s 失败: %v", topic, err)
		}
	}

	// 监听配置文件变化
	err = config.WatchConfig(configPath, func(newCfg *config.Config) error {
		logger.Info("正在应用新的配置...")

		// 检查并更新日志配置
		if err = logger.InitFromConfig(
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
			if err = transformerManager.ReloadTransformer(deviceType, transformerCfg); err != nil {
				logger.Warn("重新加载转换器 %s 失败: %v", deviceType, err)
				// 继续处理其他转换器，不中断整个过程
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

	logger.Info("数据转换服务已启动，等待设备数据...")

	// 等待中断信号退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 断开连接
	mqttClient.Disconnect()
	logger.Info("服务已安全停止")
}
