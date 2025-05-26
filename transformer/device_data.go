package transformer

// DeviceData 表示统一的设备数据结构
type DeviceData struct {
	DeviceName string                 `json:"device_name"` // 设备名字
	DeviceType string                 `json:"device_type"` // 设备类型（从Topic获取）
	Timestamp  int64                  `json:"timestamp"`   // 数据时间戳
	Attributes []DeviceAttribute      `json:"attributes"`  // 设备属性列表
	Metadata   map[string]interface{} `json:"metadata"`    // 额外元数据
}

// DeviceAttribute 表示设备属性
type DeviceAttribute struct {
	Name     string      `json:"name"`     // 属性名称
	Type     string      `json:"type"`     // 属性类型
	Value    interface{} `json:"value"`    // 属性值
	Unit     string      `json:"unit"`     // 单位（可选）
	Quality  int         `json:"quality"`  // 数据质量（0-100）
	Metadata interface{} `json:"metadata"` // 属性相关元数据
}
