package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	// SQLite 驱动（副作用导入，注册数据库驱动）
	_ "modernc.org/sqlite"
)

// PathsConfig 配置文件结构
type PathsConfig struct {
	DataPath string `json:"dataPath"`
}

// ImportTaskPhase 导入任务阶段
type ImportTaskPhase string

const (
	importPhaseScanning  ImportTaskPhase = "scanning"
	importPhaseCopying   ImportTaskPhase = "copying"
	importPhaseIndexing  ImportTaskPhase = "indexing"
	importPhaseDeleting  ImportTaskPhase = "deleting"
	importPhaseCompleted ImportTaskPhase = "completed"
	importPhaseFailed    ImportTaskPhase = "failed"
)

// ImportTaskType 导入任务类型
type ImportTaskType string

const (
	taskTypeImportImages  ImportTaskType = "import_images"
	taskTypeImportDataset ImportTaskType = "import_dataset"
	taskTypeDeleteImages  ImportTaskType = "delete_images"
)

// ImportTaskStatus 导入任务状态
type ImportTaskStatus struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"projectId"`
	TaskType  ImportTaskType  `json:"taskType"`
	Phase     ImportTaskPhase `json:"phase"`
	Progress  int             `json:"progress"`
	Imported  int             `json:"imported"`
	Total     int             `json:"total"`
	Error     string          `json:"error,omitempty"`
}

// 全局变量
var (
	importTasksMu sync.RWMutex
	importTasks   = make(map[string]*ImportTaskStatus)
	imageFileSem  = make(chan struct{}, 4)

	// Global I/O lock - only one I/O intensive task can run at a time
	ioTaskMu        sync.Mutex
	currentIOTaskID string
)

// 常量
const (
	maxImportDirectoryDepth = -1
	thumbSizeThresholdBytes = 3 * 1024 * 1024 // 3MB
	thumbMaxDimension       = 256
)

// tryAcquireIOLock attempts to acquire the global I/O lock
// Returns true if acquired, false if another task is running
func tryAcquireIOLock(taskID string) bool {
	ioTaskMu.Lock()
	defer ioTaskMu.Unlock()
	if currentIOTaskID != "" {
		return false
	}
	currentIOTaskID = taskID
	return true
}

// releaseIOLock releases the global I/O lock
func releaseIOLock(taskID string) {
	ioTaskMu.Lock()
	defer ioTaskMu.Unlock()
	if currentIOTaskID == taskID {
		currentIOTaskID = ""
	}
}

// getCurrentIOTask returns the current running I/O task ID (empty if none)
func getCurrentIOTask() string {
	ioTaskMu.Lock()
	defer ioTaskMu.Unlock()
	return currentIOTaskID
}

// getConfigPath 获取配置文件路径（支持环境变量指定）
func getConfigPath() string {
	// 优先使用环境变量指定的配置路径
	if configPath := os.Getenv("EASYMARK_CONFIG_PATH"); configPath != "" {
		return configPath + "/paths.json"
	}
	// 默认使用相对路径
	return "config/paths.json"
}

// loadPathsConfig 加载配置文件
func loadPathsConfig() (*PathsConfig, error) {
	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg PathsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.DataPath) == "" {
		return nil, errors.New("empty dataPath in config")
	}
	return &cfg, nil
}

// getDataPath 获取数据目录路径，失败时返回默认路径
func getDataPath() string {
	cfg, err := loadPathsConfig()
	if err != nil {
		log.Printf("[Config] Failed to load paths config: %v, using default", err)
		return "D:/EasyMark/Data"
	}
	return cfg.DataPath
}

// generateProjectID 生成项目ID
func generateProjectID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// 生成类似 UUID 的字符串: 8-4-4-4-12
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
