// Conversion function, receives raw data string, returns standardized data object
function transform(data) {
  // Attempt to parse JSON data
  var parsed;
  try {
    parsed = parseJSON(data);
  } catch (e) {
    // If not JSON format, attempt other format parsing
    return parseNonJsonFormat(data);
  }
  
  if (!parsed) {
    return { error: "Invalid data format" };
  }
  
  // Process first possible data format (e.g.: {"temp": 25.5, "unit": "C", "device_name": "temp001"})
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
  
  // Process second possible data format (e.g.: {"temperature": {"value": 25.5, "scale": "celsius"}, "sensor": {"id": "temp001"}})
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
  
  // Process third possible data format (e.g.: {"readings": [{"type": "temperature", "value": 25.5}], "id": "temp001"})
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
  
  // Unrecognized format
  return {
    error: "Unknown temperature sensor data format",
    raw: parsed
  };
}

// Helper function: Convert temperature unit identifier to standard format
function convertTemperatureUnit(unit) {
  if (!unit) return "C";
  
  unit = unit.toLowerCase();
  
  if (unit === "c" || unit === "celsius" || unit === "celsius") {
    return "C";
  } else if (unit === "f" || unit === "fahrenheit" || unit === "fahrenheit") {
    return "F";
  } else if (unit === "k" || unit === "kelvin" || unit === "kelvin") {
    return "K";
  }
  
  return unit;
}

// Helper function: Parse non-JSON format data
function parseNonJsonFormat(data) {
  // Attempt to parse simple key-value format (e.g.: "temp=25.5,unit=C,id=temp001")
  if (data.indexOf("=") !== -1) {
    var result = {};
    var pairs = data.split(",");
    
    for (var i = 0; i < pairs.length; i++) {
      var parts = pairs[i].split("=");
      if (parts.length === 2) {
        var key = parts[0].trim();
        var value = parts[1].trim();
        
        // Attempt to convert numeric values to numbers
        if (!isNaN(value)) {
          value = parseFloat(value);
        }
        
        result[key] = value;
      }
    }
    
    // Check if temperature data was found
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
  
  // Attempt to parse simple numeric format (assume only a temperature value)
  var numValue = parseFloat(data);
  if (!isNaN(numValue)) {
    return {
      device_name: "unknown",
      timestamp: Date.now(),
      temperature: numValue,
      unit: "C",  // Default unit assumed as Celsius
      raw: data
    };
  }
  
  // Unparseable format
  return {
    error: "Unparseable data format",
    raw: data
  };
}