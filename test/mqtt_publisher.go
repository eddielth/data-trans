package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// 温度数据结构
type TemperatureData struct {
	Temp       float64 `json:"temp"`
	Unit       string  `json:"unit"`
	DeviceName string  `json:"device_name"`
	Timestamp  int64   `json:"timestamp"`
}

// 湿度数据结构
type HumidityData struct {
	Humidity   float64 `json:"humidity"`
	DeviceName string  `json:"device_name"`
	Timestamp  int64   `json:"timestamp"`
}

// 设备配置
type DeviceConfig struct {
	ID       string
	Type     string
	Interval time.Duration
}

func main() {
	// 命令行参数
	broker := flag.String("broker", "tcp://localhost:1883", "MQTT broker地址")
	username := flag.String("username", "user", "MQTT用户名")
	password := flag.String("password", "password", "MQTT密码")
	mode := flag.String("mode", "continuous", "运行模式: single, batch, continuous")
	flag.Parse()

	// 创建MQTT客户端选项
	opts := paho.NewClientOptions()
	opts.AddBroker(*broker)
	clientID := fmt.Sprintf("go-mqtt-test-%d", time.Now().Unix())
	opts.SetClientID(clientID)
	opts.SetUsername(*username)
	opts.SetPassword(*password)
	opts.SetAutoReconnect(true)
	opts.SetConnectionLostHandler(func(_ paho.Client, err error) {
		fmt.Printf("连接丢失: %v\n", err)
	})

	// 创建客户端
	client := paho.NewClient(opts)

	// 连接到MQTT服务器
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("连接MQTT服务器失败: %v\n", token.Error())
		os.Exit(1)
	}

	fmt.Printf("已连接到MQTT服务器: %s\n", *broker)

	// 根据模式执行不同的测试
	switch *mode {
	case "single":
		publishSingleTemperature(client)
	case "batch":
		publishBatchHumidity(client)
	case "continuous":
		publishContinuousData(client)
	default:
		fmt.Println("未知的运行模式，请使用 single, batch 或 continuous")
		os.Exit(1)
	}
}

// 发布单个温度数据
func publishSingleTemperature(client paho.Client) {
	deviceID := "go-temp-sensor-001"
	topic := fmt.Sprintf("devices/temperature/%s", deviceID)

	// 生成随机温度
	temp := 25.0 + (rand.Float64()*10 - 5)
	data := TemperatureData{
		Temp:       float64(int(temp*10)) / 10, // 保留一位小数
		Unit:       "C",
		DeviceName: deviceID,
		Timestamp:  time.Now().Unix(),
	}

	// 转换为JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("JSON编码失败: %v\n", err)
		return
	}

	// 发布消息
	token := client.Publish(topic, 0, false, jsonData)
	token.Wait()

	if token.Error() != nil {
		fmt.Printf("发布消息失败: %v\n", token.Error())
	} else {
		fmt.Printf("已发布温度数据: %s\n", string(jsonData))
	}

	// 断开连接
	client.Disconnect(250)
}

// 发布批量湿度数据
func publishBatchHumidity(client paho.Client) {
	// 创建10个湿度传感器
	for i := 1; i <= 10; i++ {
		deviceID := fmt.Sprintf("go-hum-sensor-%03d", i)
		topic := fmt.Sprintf("devices/humidity/%s", deviceID)

		// 生成随机湿度
		humidity := 40.0 + rand.Float64()*40
		data := HumidityData{
			Humidity:   float64(int(humidity*10)) / 10, // 保留一位小数
			DeviceName: deviceID,
			Timestamp:  time.Now().Unix(),
		}

		// 转换为JSON
		jsonData, err := json.Marshal(data)
		if err != nil {
			fmt.Printf("JSON编码失败: %v\n", err)
			continue
		}

		// 发布消息
		token := client.Publish(topic, 0, false, jsonData)
		token.Wait()

		if token.Error() != nil {
			fmt.Printf("发布消息失败: %v\n", token.Error())
		} else {
			fmt.Printf("已发布湿度数据 [%s]: %s\n", deviceID, string(jsonData))
		}

		// 短暂延迟，避免消息拥堵
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("批量发布完成")
	// 断开连接
	client.Disconnect(250)
}

// 发布持续数据
func publishContinuousData(client paho.Client) {
	// 设备配置
	devices := []DeviceConfig{
		{ID: "go-temp-sensor-001", Type: "temperature", Interval: 5 * time.Second},
		{ID: "go-temp-sensor-002", Type: "temperature", Interval: 8 * time.Second},
		{ID: "go-hum-sensor-001", Type: "humidity", Interval: 6 * time.Second},
		{ID: "go-hum-sensor-002", Type: "humidity", Interval: 10 * time.Second},
	}

	// 为每个设备创建一个goroutine
	for _, device := range devices {
		go func(dev DeviceConfig) {
			for {
				publishDeviceData(client, dev)
				time.Sleep(dev.Interval)
			}
		}(device)
		fmt.Printf("设备 %s 将每 %v 上报一次数据\n", device.ID, device.Interval)
	}

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("正在断开连接...")
	client.Disconnect(250)
}

// 发布设备数据
func publishDeviceData(client paho.Client, device DeviceConfig) {
	topic := fmt.Sprintf("devices/%s/%s", device.Type, device.ID)
	var jsonData []byte
	var err error

	if device.Type == "temperature" {
		// 生成随机温度
		temp := 25.0 + (rand.Float64()*10 - 5)
		data := TemperatureData{
			Temp:       float64(int(temp*10)) / 10, // 保留一位小数
			Unit:       "C",
			DeviceName: device.ID,
			Timestamp:  time.Now().Unix(),
		}
		jsonData, err = json.Marshal(data)
	} else if device.Type == "humidity" {
		// 生成随机湿度
		humidity := 40.0 + rand.Float64()*40
		data := HumidityData{
			Humidity:   float64(int(humidity*10)) / 10, // 保留一位小数
			DeviceName: device.ID,
			Timestamp:  time.Now().Unix(),
		}
		jsonData, err = json.Marshal(data)
	}

	if err != nil {
		fmt.Printf("JSON编码失败: %v\n", err)
		return
	}

	// 发布消息
	token := client.Publish(topic, 0, false, jsonData)
	token.Wait()

	if token.Error() != nil {
		fmt.Printf("发布消息失败: %v\n", token.Error())
	} else {
		timestamp := time.Now().Format("15:04:05")
		fmt.Printf("[%s] 已发布设备 %s 数据: %s\n", timestamp, device.ID, string(jsonData))
	}
}
