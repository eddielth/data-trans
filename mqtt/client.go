package mqtt

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/eddielth/data-trans/config"
)

// Client 表示MQTT客户端
type Client struct {
	client  mqtt.Client
	config  config.MQTTConfig
	handler MessageHandler
}

// MessageHandler 是处理MQTT消息的回调函数类型
type MessageHandler func(topic string, payload []byte)

// NewClient 创建一个新的MQTT客户端
func NewClient(config config.MQTTConfig, handler MessageHandler) (*Client, error) {
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
		log.Printf("MQTT连接丢失: %v", err)
	})

	opts.SetReconnectingHandler(func(_ mqtt.Client, _ *mqtt.ClientOptions) {
		log.Println("正在尝试重新连接MQTT服务器...")
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

	log.Printf("已成功连接到MQTT服务器: %s", c.config.Broker)
	return nil
}

// Subscribe 订阅指定主题
func (c *Client) Subscribe(topic string) error {
	token := c.client.Subscribe(topic, 0, func(_ mqtt.Client, msg mqtt.Message) {
		log.Printf("收到来自主题 %s 的消息", msg.Topic())
		c.handler(msg.Topic(), msg.Payload())
	})

	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("订阅主题 %s 超时", topic)
	}

	if err := token.Error(); err != nil {
		return err
	}

	log.Printf("已成功订阅主题: %s", topic)
	return nil
}

// Disconnect 断开与MQTT服务器的连接
func (c *Client) Disconnect() {
	c.client.Disconnect(250)
	log.Println("已断开与MQTT服务器的连接")
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
