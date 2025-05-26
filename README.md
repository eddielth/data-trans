# Data-Trans

一个基于Go语言的物联网设备数据转换服务，用于接收、转换和处理来自各种设备的数据。

## 功能特点

- 支持通过MQTT协议接收设备数据
- 使用JavaScript脚本进行灵活的数据转换
- 支持多种设备类型（温度传感器、湿度传感器、网关设备等）
- 配置文件热重载，无需重启服务即可更新转换规则
- 高并发处理能力，适合大规模设备数据处理

## 系统架构

系统主要由以下几个部分组成：

1. **MQTT客户端**：负责连接MQTT服务器，订阅设备主题，接收设备数据
2. **转换器管理器**：管理不同设备类型的数据转换器
3. **JavaScript引擎**：执行数据转换脚本
4. **配置管理**：加载和监控配置文件变化

## 安装

### 前置条件

- Go 1.23或更高版本
- MQTT服务器（如Mosquitto、EMQ X等）

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
    timestamp: parsed.timestamp || Date.now(),
    temperature: parsed.temp,
    unit: parsed.unit || "C",
    raw: parsed
  };
}
```

### 可用的辅助函数

- `log(message)`: 输出日志
- `parseJSON(jsonString)`: 解析JSON字符串

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
├── mqtt/              # MQTT客户端
│   └── client.go
├── scripts/           # 转换脚本
│   ├── humidity.js
│   └── temperature.js
├── transformer/       # 转换器
│   ├── device_data.go
│   └── manager.go
├── config.yaml        # 配置文件
├── go.mod
├── go.sum
├── main.go
└── README.md
```

### 添加新的设备类型

1. 在`transformer/device_data.go`中定义新的数据结构
2. 创建对应的转换脚本
3. 在配置文件中添加新的转换器配置

## 贡献

欢迎提交问题和拉取请求！
