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

// Client represents an MQTT client
type Client struct {
	client  mqtt.Client
	config  config.MQTTConfig
	handler MessageHandler
}

// MessageHandler is the callback function type for handling MQTT messages
type MessageHandler func(topic string, payload []byte)

// Manager MQTT Manager
type Manager struct {
	client             *Client
	transformerManager *transformer.Manager
	storageManager     *storage.Manager
}

// NewManager creates a new MQTT manager
func NewManager(cfg *config.Config, transformerManager *transformer.Manager, storageManager *storage.Manager) (*Manager, error) {
	// Create message handler function
	messageHandler := createMessageHandler(transformerManager, storageManager)

	// Initialize MQTT client
	mqttClient, err := newClient(cfg.MQTT, messageHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MQTT client: %v", err)
	}

	return &Manager{
		client:             mqttClient,
		transformerManager: transformerManager,
		storageManager:     storageManager,
	}, nil
}

// Start starts the MQTT service
func (m *Manager) Start() error {
	// Connect to MQTT broker
	if err := m.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %v", err)
	}

	// Subscribe to configured topics
	for _, topic := range m.client.config.Topics {
		if err := m.client.Subscribe(topic); err != nil {
			logger.Warn("failed to subscribe to topic %s: %v", topic, err)
		}
	}

	return nil
}

// Stop stops the MQTT service
func (m *Manager) Stop() {
	m.client.Disconnect()
}

// createMessageHandler creates an MQTT message handler function
func createMessageHandler(transformerManager *transformer.Manager, storageManager *storage.Manager) MessageHandler {
	return func(topic string, payload []byte) {
		// Determine device type based on topic
		deviceType := GetDeviceTypeFromTopic(topic)
		if deviceType == "" {
			logger.Warn("unable to determine device type from topic %s", topic)
			return
		}

		logger.Debug("received data from device type %s: %s", deviceType, string(payload))

		// Process data using corresponding transformer
		result, err := transformerManager.Transform(deviceType, payload)
		if err != nil {
			logger.Error("failed to transform data [%s]: %v", deviceType, err)
			return
		}

		// Process transformed data
		logger.Info("device type: %s, transformed data: %v", deviceType, result)

		// Store data
		if err := storageManager.Store(deviceType, result); err != nil {
			logger.Error("failed to store data: %v", err)
		}
	}
}

// newClient creates a new MQTT client
func newClient(config config.MQTTConfig, handler MessageHandler) (*Client, error) {
	if config.Broker == "" {
		return nil, fmt.Errorf("MQTT broker address cannot be empty")
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
		logger.Error("MQTT connection lost: %v", err)
	})

	opts.SetReconnectingHandler(func(_ mqtt.Client, _ *mqtt.ClientOptions) {
		logger.Info("trying to reconnect to MQTT broker...")
	})

	client := mqtt.NewClient(opts)

	return &Client{
		client:  client,
		config:  config,
		handler: handler,
	}, nil
}

// Connect connects to the MQTT broker
func (c *Client) Connect() error {
	token := c.client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("connection to MQTT broker timed out")
	}

	if err := token.Error(); err != nil {
		return err
	}

	logger.Info("successfully connected to MQTT broker: %s", c.config.Broker)
	return nil
}

// Subscribe subscribes to the specified topic
func (c *Client) Subscribe(topic string) error {
	token := c.client.Subscribe(topic, 0, func(_ mqtt.Client, msg mqtt.Message) {
		logger.Debug("received message from topic %s", msg.Topic())
		c.handler(msg.Topic(), msg.Payload())
	})

	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("subscription to topic %s timed out", topic)
	}

	if err := token.Error(); err != nil {
		return err
	}

	logger.Info("successfully subscribed to topic: %s", topic)
	return nil
}

// Disconnect disconnects from the MQTT broker
func (c *Client) Disconnect() {
	c.client.Disconnect(250)
	logger.Info("disconnected from MQTT broker")
}

// GetDeviceTypeFromTopic extracts the device type from the topic
// The topic format is assumed to be: devices/{device_type}/{device_name}
func GetDeviceTypeFromTopic(topic string) string {
	// Use regex to match topic format
	re := regexp.MustCompile(`devices/([^/]+)/.*`)
	matches := re.FindStringSubmatch(topic)

	if len(matches) > 1 {
		return matches[1]
	}

	// Try simple split method
	parts := strings.Split(topic, "/")
	if len(parts) >= 2 && parts[0] == "devices" {
		return parts[1]
	}

	return ""
}
