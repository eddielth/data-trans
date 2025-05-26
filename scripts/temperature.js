/**
 * 温度传感器数据转换脚本
 * 将不同格式的温度传感器数据转换为标准格式
 */

// 转换函数，接收原始数据字符串，返回标准化的数据对象
function transform(data) {
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
  
  // 处理第一种可能的数据格式 (例如: {"temp": 25.5, "unit": "C", "device_name": "temp001"})
  if (parsed.temp !== undefined) {
    return {
      device_name: parsed.device_name || "unknown",
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
  
  // 处理第二种可能的数据格式 (例如: {"temperature": {"value": 25.5, "scale": "celsius"}, "sensor": {"id": "temp001"}})
  if (parsed.temperature && typeof parsed.temperature === "object") {
    return {
      device_name: parsed.sensor && parsed.sensor.id ? parsed.sensor.id : "unknown",
      device_type: "temperature",
      timestamp: parsed.timestamp || Date.now(),
      attributes: [{
        name: "temperature",
        type: "float",
        value: parsed.temperature.value,
        unit: convertTemperatureUnit(parsed.temperature.scale),
        quality: 100,
        metadata: {}
      }],
      metadata: {
        original_data: parsed
      }
    };
  }
  
  // 处理第三种可能的数据格式 (例如: {"readings": [{"type": "temperature", "value": 25.5}], "id": "temp001"})
  if (parsed.readings && Array.isArray(parsed.readings)) {
    var tempReading = parsed.readings.find(function(r) {
      return r.type === "temperature" || r.name === "temperature";
    });

    if (tempReading) {
      return {
        device_name: parsed.id || "unknown",
        device_type: "temperature",
        timestamp: parsed.timestamp || Date.now(),
        attributes: [{
          name: "temperature",
          type: "float",
          value: tempReading.value,
          unit: tempReading.unit || "C",
          quality: tempReading.quality || 100,
          metadata: {}
        }],
        metadata: {
          original_data: parsed
        }
      };
    }
  }
  
  // 无法识别的格式
  return {
    error: "未知的温度传感器数据格式",
    raw: parsed
  };
}

// 辅助函数：转换温度单位标识为标准格式
function convertTemperatureUnit(unit) {
  if (!unit) return "C";
  
  unit = unit.toLowerCase();
  
  if (unit === "c" || unit === "celsius" || unit === "摄氏") {
    return "C";
  } else if (unit === "f" || unit === "fahrenheit" || unit === "华氏") {
    return "F";
  } else if (unit === "k" || unit === "kelvin" || unit === "开尔文") {
    return "K";
  }
  
  return unit;
}

// 辅助函数：解析非JSON格式的数据
function parseNonJsonFormat(data) {
  // 尝试解析简单的键值对格式 (例如: "temp=25.5,unit=C,id=temp001")
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
    
    // 检查是否找到温度数据
    if (result.temp !== undefined || result.temperature !== undefined) {
      return {
        device_name: result.id || result.device_name || "unknown",
        timestamp: Date.now(),
        temperature: result.temp || result.temperature,
        unit: result.unit || "C",
        raw: result
      };
    }
  }
  
  // 尝试解析简单的数字格式 (假设只是一个温度值)
  var numValue = parseFloat(data);
  if (!isNaN(numValue)) {
    return {
      device_name: "unknown",
      timestamp: Date.now(),
      temperature: numValue,
      unit: "C",  // 假设默认单位是摄氏度
      raw: data
    };
  }
  
  // 无法解析的格式
  return {
    error: "无法解析的数据格式",
    raw: data
  };
}