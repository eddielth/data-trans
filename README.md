# Data-Trans

一个基于Go语言的物联网设备数据转换服务，用于接收、转换、验证和存储来自各种设备的数据。

## 功能特点

- 支持通过MQTT协议接收设备数据
- 使用JavaScript脚本进行灵活的数据转换
- 支持多种设备类型（温度传感器、湿度传感器、网关设备等）
- 配置文件热重载，无需重启服务即可更新转换规则
- 高并发处理能力，适合大规模设备数据处理
- 多种存储后端支持（文件、MySQL、PostgreSQL）
- 完善的日志系统，支持文件和控制台输出
- 数据验证功能，确保数据质量

## 系统架构

系统主要由以下几个部分组成：

1. **MQTT客户端**：负责连接MQTT服务器，订阅设备主题，接收设备数据
2. **转换器管理器**：管理不同设备类型的数据转换器
3. **JavaScript引擎**：执行数据转换脚本
4. **存储系统**：支持多种存储后端，包括文件存储和数据库存储
5. **配置管理**：加载和监控配置文件变化，支持热重载
6. **日志系统**：记录系统运行状态和错误信息
7. **验证器**：验证数据的有效性和合法性

## 安装

### 前置条件

- Go 1.23或更高版本
- MQTT服务器（如Mosquitto、EMQ X等）
- 可选：MySQL或PostgreSQL数据库

### 从源码安装

```bash
# 克隆仓库
git clone https://github.com/eddielth/data-trans.git
cd data-trans

# 安装依赖
go mod download

# 编译
go build -o data-trans

# 运行
./data-trans
```

## 配置

服务使用YAML格式的配置文件`config.yaml`，配置示例：

```yaml
# MQTT配置
mqtt:
  broker: "tcp://localhost:1883"
  client_id: "data-trans-client"
  username: "user"
  password: "password"
  topics:
    - "devices/temperature/+"
    - "devices/humidity/+"

# 日志配置
logger:
  level: "DEBUG"       # 日志级别: DEBUG, INFO, WARN, ERROR
  file_path: "./logs/app.log"  # 日志文件路径
  max_size: 10        # 单个日志文件最大大小（MB）
  max_backups: 5      # 最大保留的日志文件数量
  console: true       # 是否同时输出到控制台

# 存储配置
storage:
  # 文件存储
  file:
    enabled: true
    path: "./data"
  # 数据库存储
  database:
    enabled: true
    # 数据库类型: mysql 或 postgresql
    type: "mysql"
    # MySQL连接字符串示例: user:password@tcp(localhost:3306)/data_trans
    # PostgreSQL连接字符串示例: postgres://user:password@localhost:5432/data_trans?sslmode=disable
    dsn: "user:password@tcp(localhost:3306)/data_trans"

# 转换器配置
transformers:
  # 温度传感器转换器
  temperature:
    script_path: "./scripts/temperature.js"
  
  # 湿度传感器转换器
  humidity:
    script_path: "./scripts/humidity.js"
```

### 配置项说明

#### MQTT配置

- `broker`: MQTT服务器地址
- `client_id`: 客户端ID
- `username`: 用户名（可选）
- `password`: 密码（可选）
- `topics`: 要订阅的主题列表

#### 日志配置

- `level`: 日志级别（DEBUG, INFO, WARN, ERROR）
- `file_path`: 日志文件路径
- `max_size`: 单个日志文件最大大小（MB）
- `max_backups`: 最大保留的日志文件数量
- `console`: 是否同时输出到控制台

#### 存储配置

- `file`: 文件存储配置
  - `enabled`: 是否启用文件存储
  - `path`: 文件存储路径
- `database`: 数据库存储配置
  - `enabled`: 是否启用数据库存储
  - `type`: 数据库类型（mysql 或 postgresql）
  - `dsn`: 数据库连接字符串

#### 转换器配置

每个设备类型可以配置一个转换器，有两种方式提供转换脚本：

1. `script_path`: 外部JavaScript文件路径
2. `script_code`: 内联JavaScript代码

## 数据转换脚本

转换脚本必须提供一个名为`transform`的函数，该函数接收原始数据字符串，返回转换后的数据对象。

```javascript
function transform(data) {
  // 解析数据
  var parsed = parseJSON(data);
  if (!parsed) return { error: "无效的数据格式" };
  
  // 转换数据
  return {
    device_name: parsed.id || "unknown",
    device_type: "temperature",
    timestamp: parsed.timestamp || Date.now(),
    attributes: [{
      name: "temperature",
      type: "float",
      value: parsed.temp,
      unit: parsed.unit || "C",
      quality: 100,
      metadata: {}
    }],
    metadata: {
      original_data: parsed
    }
  };
}
```

### 可用的辅助函数

- `log(message)`: 输出日志
- `parseJSON(jsonString)`: 解析JSON字符串
- `formatDate(timestamp, format)`: 格式化日期时间
- `convertTemperature(value, fromUnit, toUnit)`: 温度单位转换
- `validateRange(value, min, max)`: 验证数值是否在指定范围内

## 设备数据结构

系统使用统一的设备数据结构来表示不同类型的设备数据：

```go
type DeviceData struct {
  DeviceName string                 `json:"device_name"` // 设备名字
  DeviceType string                 `json:"device_type"` // 设备类型（从Topic获取）
  Timestamp  int64                  `json:"timestamp"`   // 数据时间戳
  Attributes []DeviceAttribute      `json:"attributes"`  // 设备属性列表
  Metadata   map[string]interface{} `json:"metadata"`    // 额外元数据
}

type DeviceAttribute struct {
  Name     string      `json:"name"`     // 属性名称
  Type     string      `json:"type"`     // 属性类型
  Value    interface{} `json:"value"`    // 属性值
  Unit     string      `json:"unit"`     // 单位（可选）
  Quality  int         `json:"quality"`  // 数据质量（0-100）
  Metadata interface{} `json:"metadata"` // 属性相关元数据
}
```

## 主题格式

服务默认使用以下格式的主题：

```
devices/{device_type}/{device_name}
```

例如：
- `devices/temperature/temp001`
- `devices/humidity/hum001`

## 开发

### 项目结构

```
.
├── config/             # 配置相关代码
│   └── config.go
├── logger/             # 日志系统
│   ├── instance.go
│   └── logger.go
├── mqtt/               # MQTT客户端
│   └── client.go
├── scripts/            # 转换脚本
│   ├── humidity.js
│   └── temperature.js
├── storage/            # 存储系统
│   ├── database.go
│   ├── file.go
│   ├── mysql.go
│   ├── postgresql.go
│   └── storage.go
├── transformer/        # 转换器
│   ├── device_data.go
│   └── manager.go
├── validator/          # 数据验证
│   └── validator.go
├── config.yaml         # 配置文件
├── go.mod
├── go.sum
├── main.go
└── README.md
```

### 添加新的设备类型

1. 在`scripts/`目录下创建对应的转换脚本
2. 在配置文件中添加新的转换器配置

### 添加新的存储后端

1. 在`storage/`目录下创建新的存储后端实现
2. 实现`StorageBackend`接口
3. 在`storage/database.go`中添加新的存储后端类型
4. 在配置文件中添加新的存储后端配置

## 贡献

欢迎提交问题和拉取请求！
