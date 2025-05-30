/**
 * 批量上报湿度数据测试脚本
 * 用于测试向MQTT服务器批量发送多个湿度传感器的数据
 */

const mqtt = require('mqtt');
const client = mqtt.connect('tcp://localhost:1883', {
  clientId: 'test-hum-batch-' + Math.random().toString(16).substr(2, 8),
  username: 'user',
  password: 'password'
});

// 生成随机湿度值
function getRandomHumidity() {
  return (40 + Math.random() * 40).toFixed(1);
}

// 创建多个湿度传感器数据
function createHumiditySensors(count) {
  const sensors = [];
  
  for (let i = 1; i <= count; i++) {
    const deviceId = `hum-sensor-${i.toString().padStart(3, '0')}`;
    
    // 随机选择不同的数据格式
    const formatType = Math.floor(Math.random() * 3);
    let data;
    
    switch (formatType) {
      case 0:
        // 格式1: {"humidity": 65, "device_name": "hum001"}
        data = {
          humidity: parseFloat(getRandomHumidity()),
          device_name: deviceId,
          timestamp: Date.now()
        };
        break;
      
      case 1:
        // 格式2: {"data": {"hum": 65}, "id": "hum001"}
        data = {
          data: {
            hum: parseFloat(getRandomHumidity())
          },
          id: deviceId,
          timestamp: Date.now()
        };
        break;
      
      case 2:
        // 格式3: {"readings": [{"type": "humidity", "value": 65}], "id": "hum001"}
        data = {
          readings: [
            {
              type: 'humidity',
              value: parseFloat(getRandomHumidity()),
              unit: '%RH',
              quality: 95
            }
          ],
          id: deviceId,
          timestamp: Date.now()
        };
        break;
    }
    
    sensors.push({
      deviceId: deviceId,
      data: data
    });
  }
  
  return sensors;
}

// 连接成功回调
client.on('connect', function () {
  console.log('已连接到MQTT服务器');
  
  // 创建10个湿度传感器数据
  const sensors = createHumiditySensors(10);
  let publishedCount = 0;
  
  // 发布所有传感器数据
  sensors.forEach(sensor => {
    const topic = `devices/humidity/${sensor.deviceId}`;
    const message = JSON.stringify(sensor.data);
    
    client.publish(topic, message, { qos: 0 }, function(err) {
      publishedCount++;
      
      if (err) {
        console.error(`发布传感器 ${sensor.deviceId} 数据失败:`, err);
      } else {
        console.log(`已发布传感器 ${sensor.deviceId} 数据:`, message);
      }
      
      // 所有数据发布完成后断开连接
      if (publishedCount === sensors.length) {
        console.log(`已完成所有 ${sensors.length} 个湿度传感器数据的发布`);
        client.end();
      }
    });
  });
});

// 错误处理
client.on('error', function(err) {
  console.error('MQTT连接错误:', err);
  client.end();
});