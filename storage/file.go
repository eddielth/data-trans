package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eddielth/data-trans/logger"
	"github.com/eddielth/data-trans/transformer"
)

// FileStorage
type FileStorage struct {
	basePath string
}

// NewFileStorage
func NewFileStorage(basePath string) (*FileStorage, error) {
	// make dir
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("create dir %s failed: %v", basePath, err)
	}

	logger.Info("init file storage: %s", basePath)
	return &FileStorage{
		basePath: basePath,
	}, nil
}

// Store save data to file
func (fs *FileStorage) Store(deviceType string, data transformer.DeviceData) error {
	deviceDir := filepath.Join(fs.basePath, deviceType)
	if err := os.MkdirAll(deviceDir, 0755); err != nil {
		return fmt.Errorf("create dir %s failed: %v", deviceDir, err)
	}

	timestamp := time.Now().Format("20060102-150405.000")
	filename := filepath.Join(deviceDir, fmt.Sprintf("%s.json", timestamp))

	// marshal data
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize data failed: %v", err)
	}

	// write file
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("write file %s failed: %v", filename, err)
	}

	logger.Debug("has stored data to file: %s", filename)
	return nil
}

// Close implement StorageBackend
func (fs *FileStorage) Close() error {
	return nil
}
