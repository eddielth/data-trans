// Conversion function, receives raw data string, returns standardized data object
function transform(data) {
  log("Processing humidity sensor data: " + data);

  console.log("Processing humidity sensor data: " + data);
  
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
  
  // Process first possible data format (e.g.: {"humidity": 65, "device_name": "hum001"})
  if (parsed.humidity !== undefined) {
    return {
      device_name: parsed.device_name || "unknown",
      device_type: "humidity",
      timestamp: parsed.timestamp || Date.now(),
      attributes: [{
        name: "humidity",
        type: "float",
        value: parsed.humidity,
        unit: "%RH",
        quality: 100,
        metadata: {}
      }],
      metadata: {
        original_data: parsed
      }
    };
  }
  
  // Process second possible data format (e.g.: {"data": {"hum": 65}, "id": "hum001"})
  if (parsed.data && typeof parsed.data === "object" && parsed.data.hum !== undefined) {
    return {
      device_name: parsed.id || "unknown",
      device_type: "humidity",
      timestamp: parsed.timestamp || Date.now(),
      attributes: [{
        name: "humidity",
        type: "float",
        value: parsed.data.hum,
        unit: "%RH",
        quality: 100,
        metadata: {}
      }],
      metadata: {
        original_data: parsed
      }
    };
  }
  
  // Process third possible data format (e.g.: {"readings": [{"type": "humidity", "value": 65}], "id": "hum001"})
  if (parsed.readings && Array.isArray(parsed.readings)) {
    var humReading = parsed.readings.find(function(r) {
      return r.type === "humidity" || r.name === "humidity" || r.type === "hum";
    });

    if (humReading) {
      return {
        device_name: parsed.id || "unknown",
        device_type: "humidity",
        timestamp: parsed.timestamp || Date.now(),
        attributes: [{
          name: "humidity",
          type: "float",
          value: humReading.value,
          unit: humReading.unit || "%RH",
          quality: humReading.quality || 100,
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
    error: "Unknown humidity sensor data format",
    raw: parsed
  };
}

// Helper function: Parse non-JSON format data
function parseNonJsonFormat(data) {
  // Attempt to parse simple key-value format (e.g.: "hum=65,id=hum001")
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
    
    // Check if humidity data was found
    if (result.hum !== undefined || result.humidity !== undefined) {
      return {
        device_name: result.id || result.device_name || "unknown",
        timestamp: Date.now(),
        humidity: result.hum || result.humidity,
        raw: result
      };
    }
  }
  
  // Attempt to parse simple numeric format (assume only a humidity value)
  var numValue = parseFloat(data);
  if (!isNaN(numValue)) {
    return {
      device_name: "unknown",
      timestamp: Date.now(),
      humidity: numValue,
      raw: data
    };
  }
  
  // Unparseable format
  return {
    error: "Unparseable data format",
    raw: data
  };
}