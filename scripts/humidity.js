/**
 * 湿度传感器数据转换脚本
 * 将不同格式的湿度传感器数据转换为标准格式
 */

// 转换函数，接收原始数据字符串，返回标准化的数据对象
function transform(data) {
  log("处理湿度传感器数据: " + data);

  console.log("处理湿度传感器数据: " + data);
  
  // 尝试解析JSON数据
  var parsed;
  try {
    parsed = parseJSON(data);
  } catch (e) {
    // 如果不是JSON格式，尝试其他格式解析
    return parseNonJsonFormat(data);
  }
  
  if (!parsed) {
    return { error: "无效的数据格式" };
  }
  
  // 处理第一种可能的数据格式 (例如: {"humidity": 65, "device_id": "hum001"})
  if (parsed.humidity !== undefined) {
    return {
      device_id: parsed.device_id || "unknown",
      timestamp: parsed.timestamp || Date.now(),
      humidity: parsed.humidity,
      battery: parsed.battery || null,
      raw: parsed
    };
  }
  
  // 处理第二种可能的数据格式 (例如: {"data": {"hum": 65}, "id": "hum001"})
  if (parsed.data && typeof parsed.data === "object" && parsed.data.hum !== undefined) {
    return {
      device_id: parsed.id || "unknown",
      timestamp: parsed.timestamp || Date.now(),
      humidity: parsed.data.hum,
      battery: parsed.battery || null,
      raw: parsed
    };
  }
  
  // 处理第三种可能的数据格式 (例如: {"readings": [{"type": "humidity", "value": 65}], "id": "hum001"})
  if (parsed.readings && Array.isArray(parsed.readings)) {
    var humReading = parsed.readings.find(function(r) {
      return r.type === "humidity" || r.name === "humidity" || r.type === "hum";
    });
    
    if (humReading) {
      return {
        device_id: parsed.id || "unknown",
        timestamp: parsed.timestamp || Date.now(),
        humidity: humReading.value,
        battery: parsed.battery || null,
        raw: parsed
      };
    }
  }
  
  // 无法识别的格式
  return {
    error: "未知的湿度传感器数据格式",
    raw: parsed
  };
}

// 辅助函数：解析非JSON格式的数据
function parseNonJsonFormat(data) {
  // 尝试解析简单的键值对格式 (例如: "hum=65,id=hum001")
  if (data.indexOf("=") !== -1) {
    var result = {};
    var pairs = data.split(",");
    
    for (var i = 0; i < pairs.length; i++) {
      var parts = pairs[i].split("=");
      if (parts.length === 2) {
        var key = parts[0].trim();
        var value = parts[1].trim();
        
        // 尝试将数值转换为数字
        if (!isNaN(value)) {
          value = parseFloat(value);
        }
        
        result[key] = value;
      }
    }
    
    // 检查是否找到湿度数据
    if (result.hum !== undefined || result.humidity !== undefined) {
      return {
        device_id: result.id || result.device_id || "unknown",
        timestamp: Date.now(),
        humidity: result.hum || result.humidity,
        battery: result.battery || null,
        raw: result
      };
    }
  }
  
  // 尝试解析简单的数字格式 (假设只是一个湿度值)
  var numValue = parseFloat(data);
  if (!isNaN(numValue)) {
    return {
      device_id: "unknown",
      timestamp: Date.now(),
      humidity: numValue,
      raw: data
    };
  }
  
  // 无法解析的格式
  return {
    error: "无法解析的数据格式",
    raw: data
  };
}