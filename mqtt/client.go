package mqtt

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/eddielth/data-trans/config"
	"github.com/eddielth/data-trans/logger"
	"github.com/eddielth/data-trans/storage"
	"github.com/eddielth/data-trans/transformer"
)

// Client 表示MQTT客户端
type Client struct {
	client  mqtt.Client
	config  config.MQTTConfig
	handler MessageHandler
}

// MessageHandler 是处理MQTT消息的回调函数类型
type MessageHandler func(topic string, payload []byte)

// Manager MQTT管理器
type Manager struct {
	client             *Client
	transformerManager *transformer.Manager
	storageManager     *storage.Manager
}

// NewManager 创建一个新的MQTT管理器
func NewManager(cfg *config.Config, transformerManager *transformer.Manager, storageManager *storage.Manager) (*Manager, error) {
	// 创建消息处理函数
	messageHandler := createMessageHandler(transformerManager, storageManager)

	// 初始化MQTT客户端
	mqttClient, err := newClient(cfg.MQTT, messageHandler)
	if err != nil {
		return nil, fmt.Errorf("初始化MQTT客户端失败: %v", err)
	}

	return &Manager{
		client:             mqttClient,
		transformerManager: transformerManager,
		storageManager:     storageManager,
	}, nil
}

// Start 启动MQTT服务
func (m *Manager) Start() error {
	// 连接MQTT服务器
	if err := m.client.Connect(); err != nil {
		return fmt.Errorf("连接MQTT服务器失败: %v", err)
	}

	// 订阅配置的主题
	for _, topic := range m.client.config.Topics {
		if err := m.client.Subscribe(topic); err != nil {
			logger.Warn("订阅主题 %s 失败: %v", topic, err)
		}
	}

	return nil
}

// Stop 停止MQTT服务
func (m *Manager) Stop() {
	m.client.Disconnect()
}

// createMessageHandler 创建MQTT消息处理函数
func createMessageHandler(transformerManager *transformer.Manager, storageManager *storage.Manager) MessageHandler {
	return func(topic string, payload []byte) {
		// 根据主题确定设备类型
		deviceType := GetDeviceTypeFromTopic(topic)
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
	}
}

// newClient 创建一个新的MQTT客户端
func newClient(config config.MQTTConfig, handler MessageHandler) (*Client, error) {
	if config.Broker == "" {
		return nil, fmt.Errorf("MQTT broker地址不能为空")
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Broker)

	if config.ClientID == "" {
		config.ClientID = fmt.Sprintf("data-trans-%d", time.Now().Unix())
	}
	opts.SetClientID(config.ClientID)

	if config.Username != "" {
		opts.SetUsername(config.Username)
		opts.SetPassword(config.Password)
	}

	opts.SetAutoReconnect(true)
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		logger.Error("MQTT连接丢失: %v", err)
	})

	opts.SetReconnectingHandler(func(_ mqtt.Client, _ *mqtt.ClientOptions) {
		logger.Info("正在尝试重新连接MQTT服务器...")
	})

	client := mqtt.NewClient(opts)

	return &Client{
		client:  client,
		config:  config,
		handler: handler,
	}, nil
}

// Connect 连接到MQTT服务器
func (c *Client) Connect() error {
	token := c.client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("连接MQTT服务器超时")
	}

	if err := token.Error(); err != nil {
		return err
	}

	logger.Info("已成功连接到MQTT服务器: %s", c.config.Broker)
	return nil
}

// Subscribe 订阅指定主题
func (c *Client) Subscribe(topic string) error {
	token := c.client.Subscribe(topic, 0, func(_ mqtt.Client, msg mqtt.Message) {
		logger.Debug("收到来自主题 %s 的消息", msg.Topic())
		c.handler(msg.Topic(), msg.Payload())
	})

	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("订阅主题 %s 超时", topic)
	}

	if err := token.Error(); err != nil {
		return err
	}

	logger.Info("已成功订阅主题: %s", topic)
	return nil
}

// Disconnect 断开与MQTT服务器的连接
func (c *Client) Disconnect() {
	c.client.Disconnect(250)
	logger.Info("已断开与MQTT服务器的连接")
}

// GetDeviceTypeFromTopic 从主题中提取设备类型
// 假设主题格式为: devices/{device_type}/{device_id}
func GetDeviceTypeFromTopic(topic string) string {
	// 使用正则表达式匹配主题格式
	re := regexp.MustCompile(`devices/([^/]+)/.*`)
	matches := re.FindStringSubmatch(topic)

	if len(matches) > 1 {
		return matches[1]
	}

	// 尝试简单的分割方法
	parts := strings.Split(topic, "/")
	if len(parts) >= 2 && parts[0] == "devices" {
		return parts[1]
	}

	return ""
}
