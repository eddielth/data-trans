# 数据上报测试脚本

本目录包含用于测试数据上报功能的JavaScript和Go脚本，可以模拟不同设备向MQTT服务器发送数据。

## 前置条件

1. 确保已安装Node.js环境（用于JavaScript测试脚本）
2. 确保已安装Go环境（用于Go测试脚本）
3. 确保MQTT服务器已启动并可访问（默认连接到localhost:1883）
4. 确保数据转换服务已启动并正在监听MQTT消息

## JavaScript测试脚本

### 安装依赖

在首次运行JavaScript测试脚本前，需要安装MQTT客户端库：

```bash
cd test
npm install
```

### 运行测试脚本

可以使用npm脚本或直接运行：

```bash
# 使用npm脚本
npm run test:single      # 一次性上报温度数据
npm run test:batch       # 批量上报湿度数据
npm run test:continuous  # 持续上报温度和湿度数据

# 或直接运行
node single_temperature_report.js
node batch_humidity_report.js
node continuous_data_report.js
```

## Go测试程序

### 安装依赖

在首次运行Go测试程序前，需要下载依赖：

```bash
cd test
go mod download
```

### 编译和运行

```bash
# 编译
go build -o mqtt_publisher mqtt_publisher.go

# 运行（三种模式）
./mqtt_publisher -mode=single      # 一次性上报温度数据
./mqtt_publisher -mode=batch       # 批量上报湿度数据
./mqtt_publisher -mode=continuous  # 持续上报温度和湿度数据

# 自定义MQTT服务器
./mqtt_publisher -broker=tcp://192.168.1.100:1883 -username=myuser -password=mypass -mode=continuous
```

按 `Ctrl+C` 停止持续上报。

## 数据格式

这些测试脚本会生成多种不同格式的数据，以测试数据转换服务的兼容性：

### 温度数据格式示例

```json
{"temp": 25.5, "unit": "C", "device_name": "temp-sensor-001"}
```

### 湿度数据格式示例

```json
{"humidity": 65.2, "device_name": "hum-sensor-001"}
```

```json
{"data": {"hum": 58.7}, "id": "hum-sensor-002"}
```

```json
{"readings": [{"type": "humidity", "value": 62.3, "unit": "%RH"}], "id": "hum-sensor-003"}
```

## 测试脚本说明

### JavaScript脚本

1. `single_temperature_report.js` - 模拟单个温度传感器上报一次数据
2. `batch_humidity_report.js` - 模拟多个湿度传感器同时上报数据
3. `continuous_data_report.js` - 模拟多个设备按照不同时间间隔持续上报数据

### Go程序

`mqtt_publisher.go` - 一个多功能的Go程序，支持三种运行模式：
1. 单次上报模式（single）
2. 批量上报模式（batch）
3. 持续上报模式（continuous）