package storage

import (
	"sync"

	"github.com/eddielth/data-trans/logger"
	"github.com/eddielth/data-trans/transformer"
)

// StorageBackend represents the storage backend interface
type StorageBackend interface {
	// Store stores data
	Store(deviceType string, data transformer.DeviceData) error
	// Close closes the storage connection
	Close() error
}

// Manager manages multiple storage backends
type Manager struct {
	backends []StorageBackend
	mutex    sync.RWMutex
}

// NewManager creates a new storage manager
func NewManager(backends []StorageBackend) *Manager {
	return &Manager{
		backends: backends,
	}
}

// Store stores data to all backends
func (m *Manager) Store(deviceType string, data transformer.DeviceData) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, backend := range m.backends {
		if err := backend.Store(deviceType, data); err != nil {
			// Log error but continue to other backends
			logger.Error("Failed to store data to backend: %v", err)
		}
	}

	return nil
}

// Close closes all storage backend connections
func (m *Manager) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, backend := range m.backends {
		if err := backend.Close(); err != nil {
			logger.Error("Failed to close storage backend connection: %v", err)
		}
	}
}

// AddBackend adds a new storage backend
func (m *Manager) AddBackend(backend StorageBackend) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.backends = append(m.backends, backend)
}

// RemoveBackendByType removes storage backend by type
func (m *Manager) RemoveBackendByType(backendType string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var newBackends []StorageBackend
	for _, backend := range m.backends {
		// Check backend type
		switch backend.(type) {
		case *MySQLStorage:
			if backendType != "mysql" {
				newBackends = append(newBackends, backend)
			} else {
				// Close connection of backend to be removed
				if err := backend.Close(); err != nil {
					logger.Error("Failed to close MySQL storage backend connection: %v", err)
				}
				logger.Info("MySQL storage backend removed")
			}
		case *PostgreSQLStorage:
			if backendType != "postgresql" {
				newBackends = append(newBackends, backend)
			} else {
				// Close connection of backend to be removed
				if err := backend.Close(); err != nil {
					logger.Error("Failed to close PostgreSQL storage backend connection: %v", err)
				}
				logger.Info("PostgreSQL storage backend removed")
			}
		case *FileStorage:
			if backendType != "file" {
				newBackends = append(newBackends, backend)
			} else {
				logger.Info("File storage backend removed")
			}
		default:
			// Keep backends of unknown types
			newBackends = append(newBackends, backend)
		}
	}

	m.backends = newBackends
}
