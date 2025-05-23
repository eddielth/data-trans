package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/eddielth/data-trans/config"
	"github.com/eddielth/data-trans/mqtt"
	"github.com/eddielth/data-trans/transformer"
)

func main() {
	// 配置文件路径
	configPath := "config.yaml"

	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化转换器管理器
	transformerManager, err := transformer.NewManager(cfg.Transformers)
	if err != nil {
		log.Fatalf("初始化转换器管理器失败: %v", err)
	}

	// 初始化MQTT客户端
	mqttClient, err := mqtt.NewClient(cfg.MQTT, func(topic string, payload []byte) {
		// 根据主题确定设备类型
		deviceType := mqtt.GetDeviceTypeFromTopic(topic)
		if deviceType == "" {
			log.Printf("无法从主题 %s 确定设备类型", topic)
			return
		}

		// 使用对应的转换器处理数据
		result, err := transformerManager.Transform(deviceType, payload)
		if err != nil {
			log.Printf("转换数据失败 [%s]: %v", deviceType, err)
			return
		}

		// 处理转换后的数据
		fmt.Printf("设备类型: %s, 转换后数据: %v\n", deviceType, result)
		// 这里可以添加数据存储逻辑
	})

	if err != nil {
		log.Fatalf("初始化MQTT客户端失败: %v", err)
	}

	// 连接MQTT服务器
	if err = mqttClient.Connect(); err != nil {
		log.Fatalf("连接MQTT服务器失败: %v", err)
	}

	// 订阅配置的主题
	for _, topic := range cfg.MQTT.Topics {
		if err = mqttClient.Subscribe(topic); err != nil {
			log.Printf("订阅主题 %s 失败: %v", topic, err)
		}
	}

	// 监听配置文件变化
	err = config.WatchConfig(configPath, func(newCfg *config.Config) error {
		log.Println("正在应用新的配置...")

		// 检查并更新转换器
		for deviceType, transformerCfg := range newCfg.Transformers {
			if err = transformerManager.ReloadTransformer(deviceType, transformerCfg); err != nil {
				log.Printf("重新加载转换器 %s 失败: %v", deviceType, err)
				// 继续处理其他转换器，不中断整个过程
			}
		}

		// 如果MQTT配置发生变化，可以在这里处理
		// 例如重新连接MQTT服务器或重新订阅主题
		// 这里简化处理，仅打印日志
		log.Println("MQTT配置更新将在服务重启后生效")

		return nil
	})

	if err != nil {
		log.Printf("监听配置文件变化失败: %v", err)
		// 不致命，继续运行
	} else {
		log.Println("已启动配置文件监听")
	}

	log.Println("数据转换服务已启动，等待设备数据...")

	// 等待中断信号退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 断开连接
	mqttClient.Disconnect()
	log.Println("服务已安全停止")
}
