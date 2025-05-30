/**
 * 一次性上报温度数据测试脚本
 * 用于测试向MQTT服务器发送单个温度数据
 */

const mqtt = require('mqtt');
const client = mqtt.connect('tcp://localhost:1883', {
  clientId: 'test-temp-single-' + Math.random().toString(16).substr(2, 8),
  username: 'user',
  password: 'password'
});

// 设备ID
const deviceId = 'temp-sensor-001';

// 温度数据
const temperatureData = {
  temp: 25.5 + (Math.random() * 5).toFixed(1),
  unit: 'C',
  device_name: deviceId,
  timestamp: Date.now()
};

// 连接成功回调
client.on('connect', function () {
  console.log('已连接到MQTT服务器');
  
  // 发布温度数据到指定主题
  const topic = `devices/temperature/${deviceId}`;
  const message = JSON.stringify(temperatureData);
  
  client.publish(topic, message, { qos: 0 }, function(err) {
    if (err) {
      console.error('发布消息失败:', err);
    } else {
      console.log('已发布温度数据:', message);
      console.log('主题:', topic);
    }
    
    // 断开连接
    client.end();
  });
});

// 错误处理
client.on('error', function(err) {
  console.error('MQTT连接错误:', err);
  client.end();
});