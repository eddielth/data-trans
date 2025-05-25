package storage

import (
	"sync"

	"github.com/eddielth/data-trans/logger"
)

// StorageBackend 表示存储后端接口
type StorageBackend interface {
	// Store 存储数据
	Store(deviceType string, data interface{}) error
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
func (m *Manager) Store(deviceType string, data interface{}) error {
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
