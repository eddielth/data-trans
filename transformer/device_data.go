package transformer

import (
	"fmt"
	"reflect"
)

// TemperatureData 表示温度传感器数据
type TemperatureData struct {
	DeviceID    string      `json:"device_id"`
	Timestamp   int64       `json:"timestamp"`
	Temperature float64     `json:"temperature"`
	Unit        string      `json:"unit"`
	Battery     interface{} `json:"battery"`
	Raw         interface{} `json:"raw"`
}

// HumidityData 表示湿度传感器数据
type HumidityData struct {
	DeviceID  string      `json:"device_id"`
	Timestamp int64       `json:"timestamp"`
	Humidity  float64     `json:"humidity"`
	Battery   interface{} `json:"battery"`
	Raw       interface{} `json:"raw"`
}

// GatewayData 表示网关设备数据
type GatewayData struct {
	GatewayID       string       `json:"gateway_id"`
	Timestamp       int64        `json:"timestamp"`
	FirmwareVersion string       `json:"firmware_version"`
	Devices         []DeviceInfo `json:"devices"`
	Raw             interface{}  `json:"raw"`
}

// DeviceInfo 表示连接到网关的设备信息
type DeviceInfo struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// ToMap 将interface{}转换为map[string]interface{}
// 这个函数用于将JavaScript返回的结果转换为Go中的map类型
func ToMap(data interface{}) (map[string]interface{}, error) {
	// 检查是否已经是map[string]interface{}
	if m, ok := data.(map[string]interface{}); ok {
		return m, nil
	}

	// 检查是否是map但键类型不是string
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Map {
		m := make(map[string]interface{})
		for _, key := range v.MapKeys() {
			// 尝试将键转换为字符串
			strKey := fmt.Sprintf("%v", key.Interface())
			m[strKey] = v.MapIndex(key).Interface()
		}
		return m, nil
	}

	// 如果不是map类型，返回错误
	return nil, fmt.Errorf("无法将类型 %T 转换为 map[string]interface{}", data)
}

// ParseDeviceData 根据设备类型解析转换后的数据
func ParseDeviceData(deviceType string, data interface{}) (interface{}, error) {
	// 首先将数据转换为map
	dataMap, err := ToMap(data)
	if err != nil {
		return nil, fmt.Errorf("转换数据失败: %v", err)
	}

	// 检查是否有错误信息
	if errMsg, ok := dataMap["error"]; ok {
		return nil, fmt.Errorf("数据转换错误: %v", errMsg)
	}
	return dataMap, nil
}

// parseTemperatureData 解析温度传感器数据
func parseTemperatureData(data map[string]interface{}) (*TemperatureData, error) {
	result := &TemperatureData{
		Raw: data,
	}

	// 提取设备ID
	if id, ok := data["device_id"].(string); ok {
		result.DeviceID = id
	} else {
		result.DeviceID = "unknown"
	}

	// 提取时间戳
	switch ts := data["timestamp"].(type) {
	case float64:
		result.Timestamp = int64(ts)
	case int64:
		result.Timestamp = ts
	case int:
		result.Timestamp = int64(ts)
	default:
		// 如果没有时间戳或格式不正确，使用0
		result.Timestamp = 0
	}

	// 提取温度值
	switch temp := data["temperature"].(type) {
	case float64:
		result.Temperature = temp
	case int64:
		result.Temperature = float64(temp)
	case int:
		result.Temperature = float64(temp)
	default:
		return nil, fmt.Errorf("无效的温度值类型: %T", data["temperature"])
	}

	// 提取单位
	if unit, ok := data["unit"].(string); ok {
		result.Unit = unit
	} else {
		result.Unit = "C" // 默认摄氏度
	}

	// 提取电池信息（可以是任何类型）
	result.Battery = data["battery"]

	return result, nil
}

// parseHumidityData 解析湿度传感器数据
func parseHumidityData(data map[string]interface{}) (*HumidityData, error) {
	result := &HumidityData{
		Raw: data,
	}

	// 提取设备ID
	if id, ok := data["device_id"].(string); ok {
		result.DeviceID = id
	} else {
		result.DeviceID = "unknown"
	}

	// 提取时间戳
	switch ts := data["timestamp"].(type) {
	case float64:
		result.Timestamp = int64(ts)
	case int64:
		result.Timestamp = ts
	case int:
		result.Timestamp = int64(ts)
	default:
		// 如果没有时间戳或格式不正确，使用0
		result.Timestamp = 0
	}

	// 提取湿度值
	switch humidity := data["humidity"].(type) {
	case float64:
		result.Humidity = humidity
	case int64:
		result.Humidity = float64(humidity)
	case int:
		result.Humidity = float64(humidity)
	default:
		return nil, fmt.Errorf("无效的湿度值类型: %T", data["humidity"])
	}

	// 提取电池信息（可以是任何类型）
	result.Battery = data["battery"]

	return result, nil
}

// parseGatewayData 解析网关设备数据
func parseGatewayData(data map[string]interface{}) (*GatewayData, error) {
	result := &GatewayData{
		Raw: data,
	}

	// 提取网关ID
	if id, ok := data["gateway_id"].(string); ok {
		result.GatewayID = id
	} else {
		result.GatewayID = "unknown"
	}

	// 提取时间戳
	switch ts := data["timestamp"].(type) {
	case float64:
		result.Timestamp = int64(ts)
	case int64:
		result.Timestamp = ts
	case int:
		result.Timestamp = int64(ts)
	default:
		// 如果没有时间戳或格式不正确，使用0
		result.Timestamp = 0
	}

	// 提取固件版本
	if version, ok := data["firmware_version"].(string); ok {
		result.FirmwareVersion = version
	} else {
		result.FirmwareVersion = "unknown"
	}

	// 提取设备列表
	result.Devices = []DeviceInfo{}
	if devices, ok := data["devices"].([]interface{}); ok {
		for _, dev := range devices {
			if devMap, err := ToMap(dev); err == nil {
				devInfo := DeviceInfo{}

				// 提取设备ID
				if id, ok := devMap["id"].(string); ok {
					devInfo.ID = id
				} else {
					devInfo.ID = "unknown"
				}

				// 提取设备类型
				if typ, ok := devMap["type"].(string); ok {
					devInfo.Type = typ
				} else {
					devInfo.Type = "unknown"
				}

				// 提取设备状态
				if status, ok := devMap["status"].(string); ok {
					devInfo.Status = status
				} else {
					devInfo.Status = "unknown"
				}

				result.Devices = append(result.Devices, devInfo)
			}
		}
	}

	return result, nil
}
