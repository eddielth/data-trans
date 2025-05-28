package storage

import (
	"sync"

	"github.com/eddielth/data-trans/logger"
	"github.com/eddielth/data-trans/transformer"
)

// StorageBackend 表示存储后端接口
type StorageBackend interface {
	// Store 存储数据
	Store(deviceType string, data transformer.DeviceData) error
	// Close 关闭存储连接
	Close() error
}

// Manager 管理多个存储后端
type Manager struct {
	backends []StorageBackend
	mutex    sync.RWMutex
}

// NewManager 创建一个新的存储管理器
func NewManager(backends []StorageBackend) *Manager {
	return &Manager{
		backends: backends,
	}
}

// Store 将数据存储到所有后端
func (m *Manager) Store(deviceType string, data transformer.DeviceData) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, backend := range m.backends {
		if err := backend.Store(deviceType, data); err != nil {
			// 记录错误但继续尝试其他后端
			logger.Error("存储数据到后端失败: %v", err)
		}
	}

	return nil
}

// Close 关闭所有存储后端连接
func (m *Manager) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, backend := range m.backends {
		if err := backend.Close(); err != nil {
			logger.Error("关闭存储后端连接失败: %v", err)
		}
	}
}

// AddBackend 添加新的存储后端
func (m *Manager) AddBackend(backend StorageBackend) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.backends = append(m.backends, backend)
}

// RemoveBackendByType 根据后端类型删除存储后端
func (m *Manager) RemoveBackendByType(backendType string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var newBackends []StorageBackend
	for _, backend := range m.backends {
		// 检查后端类型
		switch backend.(type) {
		case *MySQLStorage:
			if backendType != "mysql" {
				newBackends = append(newBackends, backend)
			} else {
				// 关闭要移除的后端连接
				if err := backend.Close(); err != nil {
					logger.Error("关闭MySQL存储后端连接失败: %v", err)
				}
				logger.Info("已移除MySQL存储后端")
			}
		case *PostgreSQLStorage:
			if backendType != "postgresql" {
				newBackends = append(newBackends, backend)
			} else {
				// 关闭要移除的后端连接
				if err := backend.Close(); err != nil {
					logger.Error("关闭PostgreSQL存储后端连接失败: %v", err)
				}
				logger.Info("已移除PostgreSQL存储后端")
			}
		case *FileStorage:
			if backendType != "file" {
				newBackends = append(newBackends, backend)
			} else {
				logger.Info("已移除文件存储后端")
			}
		default:
			// 保留未知类型的后端
			newBackends = append(newBackends, backend)
		}
	}

	m.backends = newBackends
}
