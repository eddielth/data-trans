/**
 * 持续上报温度和湿度数据测试脚本
 * 用于测试向MQTT服务器持续发送多个设备的数据
 */

const mqtt = require('mqtt');
const client = mqtt.connect('tcp://localhost:1883', {
  clientId: 'test-continuous-' + Math.random().toString(16).substr(2, 8),
  username: 'user',
  password: 'password'
});

// 设备配置
const devices = [
  { id: 'temp-sensor-001', type: 'temperature', interval: 5000 },  // 5秒上报一次
  { id: 'temp-sensor-002', type: 'temperature', interval: 8000 },  // 8秒上报一次
  { id: 'hum-sensor-001', type: 'humidity', interval: 6000 },     // 6秒上报一次
  { id: 'hum-sensor-002', type: 'humidity', interval: 10000 }     // 10秒上报一次
];

// 生成随机温度数据
function generateTemperatureData(deviceId) {
  // 基础温度值在20-30之间波动
  const baseTemp = 25;
  const variation = (Math.random() * 10 - 5).toFixed(1);
  const temperature = parseFloat(baseTemp) + parseFloat(variation);
  
  return {
    temp: parseFloat(temperature.toFixed(1)),
    unit: 'C',
    device_name: deviceId,
    timestamp: Date.now()
  };
}

// 生成随机湿度数据
function generateHumidityData(deviceId) {
  // 基础湿度值在40-80之间波动
  const baseHumidity = 60;
  const variation = (Math.random() * 40 - 20).toFixed(1);
  const humidity = parseFloat(baseHumidity) + parseFloat(variation);
  
  return {
    humidity: parseFloat(humidity.toFixed(1)),
    device_name: deviceId,
    timestamp: Date.now()
  };
}

// 发布设备数据
function publishDeviceData(device) {
  let data;
  const topic = `devices/${device.type}/${device.id}`;
  
  if (device.type === 'temperature') {
    data = generateTemperatureData(device.id);
  } else if (device.type === 'humidity') {
    data = generateHumidityData(device.id);
  }
  
  const message = JSON.stringify(data);
  
  client.publish(topic, message, { qos: 0 }, function(err) {
    if (err) {
      console.error(`发布设备 ${device.id} 数据失败:`, err);
    } else {
      console.log(`已发布设备 ${device.id} 数据:`, message);
    }
  });
}

// 连接成功回调
client.on('connect', function () {
  console.log('已连接到MQTT服务器');
  console.log('开始持续上报数据...');
  console.log('按 Ctrl+C 停止程序');
  
  // 为每个设备设置定时发布任务
  devices.forEach(device => {
    // 立即发布一次数据
    publishDeviceData(device);
    
    // 然后按照设定的时间间隔持续发布
    setInterval(() => {
      publishDeviceData(device);
    }, device.interval);
    
    console.log(`设备 ${device.id} 将每 ${device.interval/1000} 秒上报一次数据`);
  });
});

// 错误处理
client.on('error', function(err) {
  console.error('MQTT连接错误:', err);
});

// 处理程序退出
process.on('SIGINT', function() {
  console.log('正在断开MQTT连接...');
  client.end();
  process.exit();
});