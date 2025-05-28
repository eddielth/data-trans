package transformer

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/eddielth/data-trans/config"
	"github.com/eddielth/data-trans/logger"
)

// Manager 管理多个数据转换器
type Manager struct {
	transformers map[string]*Transformer
	mutex        sync.RWMutex
}

// Transformer 表示一个数据转换器
type Transformer struct {
	vm         *goja.Runtime
	transform  goja.Callable
	scriptPath string
}

// NewManager 创建一个新的转换器管理器
func NewManager(configs map[string]config.Transformer) (*Manager, error) {
	manager := &Manager{
		transformers: make(map[string]*Transformer),
	}

	// 为每种设备类型创建转换器
	for deviceType, cfg := range configs {
		var scriptCode string
		var err error

		// 优先使用配置中的脚本代码
		if cfg.ScriptCode != "" {
			scriptCode = cfg.ScriptCode
		} else if cfg.ScriptPath != "" {
			var scriptBytes []byte
			// 从文件加载脚本
			scriptBytes, err = os.ReadFile(cfg.ScriptPath)
			if err != nil {
				return nil, fmt.Errorf("无法加载脚本文件 %s: %v", cfg.ScriptPath, err)
			}
			scriptCode = string(scriptBytes)
		} else {
			return nil, fmt.Errorf("设备类型 %s 没有提供脚本代码或脚本路径", deviceType)
		}

		// 创建转换器
		transformer, err := newTransformer(scriptCode, cfg.ScriptPath)
		if err != nil {
			return nil, fmt.Errorf("为设备类型 %s 创建转换器失败: %v", deviceType, err)
		}

		manager.transformers[deviceType] = transformer
		logger.Info("已为设备类型 %s 加载转换器", deviceType)
	}

	return manager, nil
}

// newTransformer 创建一个新的转换器
func newTransformer(scriptCode, scriptPath string) (*Transformer, error) {
	// 创建JavaScript运行时
	vm := goja.New()

	// 注入辅助函数
	_ = vm.Set("log", func(msg string) {
		logger.Info("[JS] %s", msg)
	})

	_ = vm.Set("parseJSON", func(jsonStr string) interface{} {
		var data interface{}
		err := json.Unmarshal([]byte(jsonStr), &data)
		if err != nil {
			logger.Warn("解析JSON失败: %v", err)
			return nil
		}
		return data
	})

	// 格式化日期时间
	_ = vm.Set("formatDate", func(timestamp int64, format string) string {
		if format == "" {
			format = "2006-01-02 15:04:05"
		}
		return time.Unix(timestamp, 0).Format(format)
	})

	// 单位转换
	_ = vm.Set("convertTemperature", func(value float64, fromUnit string, toUnit string) float64 {
		// 标准化单位
		fromUnit = strings.ToUpper(fromUnit)
		toUnit = strings.ToUpper(toUnit)

		// 转换为摄氏度
		var celsius float64
		switch fromUnit {
		case "C":
			celsius = value
		case "F":
			celsius = (value - 32) * 5 / 9
		case "K":
			celsius = value - 273.15
		default:
			return value // 未知单位，返回原值
		}

		// 从摄氏度转换为目标单位
		switch toUnit {
		case "C":
			return celsius
		case "F":
			return celsius*9/5 + 32
		case "K":
			return celsius + 273.15
		default:
			return celsius // 未知单位，返回摄氏度
		}
	})

	// 数据验证
	_ = vm.Set("validateRange", func(value float64, min float64, max float64) bool {
		return value >= min && value <= max
	})

	// 执行脚本
	_, err := vm.RunString(scriptCode)
	if err != nil {
		return nil, fmt.Errorf("执行脚本失败: %v", err)
	}

	// 获取转换函数
	transformValue := vm.Get("transform")
	if transformValue == nil {
		return nil, fmt.Errorf("脚本中没有定义 'transform' 函数")
	}

	transform, ok := goja.AssertFunction(transformValue)
	if !ok {
		return nil, fmt.Errorf("'transform' 不是一个函数")
	}

	return &Transformer{
		vm:         vm,
		transform:  transform,
		scriptPath: scriptPath,
	}, nil
}

// Transform 使用指定设备类型的转换器转换数据
func (m *Manager) Transform(deviceType string, data []byte) (DeviceData, error) {
	m.mutex.RLock()
	transformer, exists := m.transformers[deviceType]
	m.mutex.RUnlock()

	if !exists {
		return DeviceData{}, fmt.Errorf("没有找到设备类型 %s 的转换器", deviceType)
	}

	// 调用JavaScript转换函数
	result, err := transformer.transform(goja.Undefined(), transformer.vm.ToValue(string(data)))
	if err != nil {
		return DeviceData{}, fmt.Errorf("执行转换失败: %v", err)
	}

	// 将JavaScript值导出为Go值
	jsResult := result.Export()

	// 将结果转换为JSON
	jsonData, err := json.Marshal(jsResult)
	if err != nil {
		return DeviceData{}, fmt.Errorf("序列化JavaScript结果失败: %v", err)
	}

	// 解析为DeviceData结构
	var deviceData DeviceData
	if err := json.Unmarshal(jsonData, &deviceData); err != nil {
		return DeviceData{}, fmt.Errorf("解析为DeviceData结构失败: %v", err)
	}

	// 确保设备类型字段正确设置
	if deviceData.DeviceType == "" {
		deviceData.DeviceType = deviceType
	}

	return deviceData, nil
}

// ReloadTransformer 重新加载指定设备类型的转换器
func (m *Manager) ReloadTransformer(deviceType string, cfg config.Transformer) error {
	var scriptCode string
	var err error

	// 优先使用配置中的脚本代码
	if cfg.ScriptCode != "" {
		scriptCode = cfg.ScriptCode
	} else if cfg.ScriptPath != "" {
		// 从文件加载脚本
		scriptBytes, err := os.ReadFile(cfg.ScriptPath)
		if err != nil {
			return fmt.Errorf("无法加载脚本文件 %s: %v", cfg.ScriptPath, err)
		}
		scriptCode = string(scriptBytes)
	} else {
		return fmt.Errorf("没有提供脚本代码或脚本路径")
	}

	// 创建新的转换器
	transformer, err := newTransformer(scriptCode, cfg.ScriptPath)
	if err != nil {
		return fmt.Errorf("创建转换器失败: %v", err)
	}

	// 更新转换器
	m.mutex.Lock()
	m.transformers[deviceType] = transformer
	m.mutex.Unlock()

	logger.Info("已重新加载设备类型 %s 的转换器", deviceType)
	return nil
}
