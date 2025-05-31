# Data-Trans

A Go-based IoT device data transformation service for receiving, transforming, validating, and storing data from various devices.

## Features

- Supports receiving device data via MQTT protocol
- Flexible data transformation using JavaScript scripts
- Supports multiple device types (temperature sensors, humidity sensors, gateway devices, etc.)
- Hot reload configuration files, update transformation rules without restarting the service
- High concurrency processing capability, suitable for large-scale device data processing
- Supports multiple storage backends (file, MySQL, PostgreSQL)
- Comprehensive logging system, supports file and console output
- Data validation function to ensure data quality

## System Architecture

The system mainly consists of the following parts:

1. **MQTT Client**: Responsible for connecting to the MQTT server, subscribing to device topics, receiving device data
2. **Transformer Manager**: Manages data transformers for different device types
3. **JavaScript Engine**: Executes data transformation scripts
4. **Storage System**: Supports multiple storage backends, including file storage and database storage
5. **Configuration Management**: Loads and monitors configuration file changes, supports hot reload
6. **Logging System**: Records system running status and error information
7. **Validator**: Validates the validity and legality of data

## Installation

### Prerequisites

- Go 1.23 or higher
- MQTT server (such as Mosquitto, EMQ X, etc.)
- Optional: MySQL or PostgreSQL database

### Install from Source

```bash
# Clone the repository
git clone https://github.com/eddielth/data-trans.git
cd data-trans

# Install dependencies
go mod download

# Compile
go build -o data-trans

# Run
./data-trans
```

## Configuration

Service uses YAML formatted configuration file `config.yaml`, configuration example:

```yaml
# MQTT configuration
mqtt:
  broker: "tcp://localhost:1883"
  client_id: "data-trans-client"
  username: "user"
  password: "password"
  topics:
    - "devices/temperature/+"
    - "devices/humidity/+"

# Logging configuration
logger:
  level: "DEBUG"       # Log level: DEBUG, INFO, WARN, ERROR
  file_path: "./logs/app.log"  # Log file path
  max_size: 10        # Single log file maximum size (MB)
  max_backups: 5      # Maximum number of log files to retain
  console: true       # Whether to log to the console

# Storage configuration
storage:
  # File storage
  file:
    enabled: true
    path: "./data"
  # Database storage
  database:
    enabled: true
    # Database type: mysql or postgresql
    type: "mysql"
    # MySQL connection string example: user:password@tcp(localhost:3306)/data_trans
    # PostgreSQL connection string example: postgres://user:password@localhost:5432/data_trans?sslmode=disable
    dsn: "user:password@tcp(localhost:3306)/data_trans"

# Transformer configuration
transformers:
  # Temperature sensor transformer
  temperature:
    script_path: "./scripts/temperature.js"
  
  # Humidity sensor transformer
  humidity:
    script_path: "./scripts/humidity.js"
```

### Configuration Options

#### MQTT Configuration

- `broker`: MQTT server address
- `client_id`: Client ID
- `username`: Username (optional)
- `password`: Password (optional)
- `topics`: List of topics to subscribe

#### Logging Configuration

- `level`: Log level (DEBUG, INFO, WARN, ERROR)
- `file_path`: Log file path
- `max_size`: Single log file maximum size (MB)
- `max_backups`: Maximum number of log files to retain
- `console`: Whether to log to the console

#### Storage Configuration

- `file`: File storage configuration
  - `enabled`: Whether to enable file storage
  - `path`: File storage path
- `database`: Database storage configuration
  - `enabled`: Whether to enable database storage
  - `type`: Database type (mysql or postgresql)
  - `dsn`: Database connection string

#### Transformer Configuration

Each device type can configure a transformer, with two ways to provide transformation scripts:

1. `script_path`: External JavaScript file path
2. `script_code`: Inline JavaScript code

## Data Transformation Scripts

Transformation scripts must provide a function named `transform`, which receives the original data string and returns the transformed data object.

```javascript
function transform(data) {
  // Parse data
  var parsed = parseJSON(data);
  if (!parsed) return { error: "Invalid data format" };
  
  // Transform data
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

### Available Helper Functions

- `log(message)`: Output log
- `parseJSON(jsonString)`: Parse JSON string
- `formatDate(timestamp, format)`: Format date and time
- `convertTemperature(value, fromUnit, toUnit)`: Temperature unit conversion
- `validateRange(value, min, max)`: Validate if a value is within the specified range

## Device Data Structure

The system uses a unified device data structure to represent different types of device data:

```go
type DeviceData struct {
  DeviceName string                 `json:"device_name"` // Device name
  DeviceType string                 `json:"device_type"` // Device type (obtained from Topic)
  Timestamp  int64                  `json:"timestamp"`   // Data timestamp
  Attributes []DeviceAttribute      `json:"attributes"`  // Device attribute list
  Metadata   map[string]interface{} `json:"metadata"`    // Additional metadata
}

type DeviceAttribute struct {
  Name     string      `json:"name"`     // Attribute name
  Type     string      `json:"type"`     // Attribute type
  Value    interface{} `json:"value"`    // Attribute value
  Unit     string      `json:"unit"`     // Unit (optional)
  Quality  int         `json:"quality"`  // Data quality (0-100)
  Metadata interface{} `json:"metadata"` // Attribute-related metadata
}
```

## Topic Format

The service defaults to using topics in the following format:

```
devices/{device_type}/{device_name}
```

For example:
- `devices/temperature/temp001`
- `devices/humidity/hum001`

## Development

### Project Structure

```
.
├── config/             # Configuration-related code
│   └── config.go
├── logger/             # Logging system
│   ├── instance.go
│   └── logger.go
├── mqtt/               # MQTT client
│   └── client.go
├── scripts/            # Transformation scripts
│   ├── humidity.js
│   └── temperature.js
├── storage/            # Storage system
│   ├── database.go
│   ├── file.go
│   ├── mysql.go
│   ├── postgresql.go
│   └── storage.go
├── transformer/        # Transformer
│   ├── device_data.go
│   └── manager.go
├── validator/          # Data validation
│   └── validator.go
├── config.yaml         # Configuration file
├── go.mod
├── go.sum
├── main.go
└── README.md
```

### Adding New Device Types

1. Create corresponding transformation scripts in the `scripts/` directory
2. Add new transformer configuration in the configuration file

### Adding New Storage Backends

1. Create new storage backend implementation in the `storage/` directory
2. Implement the `StorageBackend` interface
3. Add new storage backend type in `storage/database.go`
4. Add new storage backend configuration in the configuration file

## Contributing

Issues and pull requests are welcome!