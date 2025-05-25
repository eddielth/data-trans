package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eddielth/data-trans/logger"
)

// FileStorage 表示文件存储后端
type FileStorage struct {
	basePath string
}

// NewFileStorage 创建一个新的文件存储后端
func NewFileStorage(basePath string) (*FileStorage, error) {
	// 确保目录存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %v", err)
	}

	logger.Info("初始化文件存储，路径: %s", basePath)
	return &FileStorage{
		basePath: basePath,
	}, nil
}

// Store 将数据存储到文件
func (fs *FileStorage) Store(deviceType string, data interface{}) error {
	// 创建设备类型目录
	deviceDir := filepath.Join(fs.basePath, deviceType)
	if err := os.MkdirAll(deviceDir, 0755); err != nil {
		return fmt.Errorf("创建设备目录失败: %v", err)
	}

	// 生成文件名
	timestamp := time.Now().Format("20060102-150405.000")
	filename := filepath.Join(deviceDir, fmt.Sprintf("%s.json", timestamp))

	// 将数据序列化为JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化数据失败: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	logger.Debug("已将 %s 类型的数据存储到文件: %s", deviceType, filename)
	return nil
}

// Close 实现StorageBackend接口
func (fs *FileStorage) Close() error {
	// 文件存储不需要关闭连接
	return nil
}
