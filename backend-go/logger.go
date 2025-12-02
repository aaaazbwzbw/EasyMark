package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const maxLogSize = 128 * 1024 // 128KB

// RingWriter 环形写入器，超出大小后删除最早的内容
type RingWriter struct {
	mu       sync.Mutex
	file     *os.File
	filePath string
	size     int64
	maxSize  int64
}

// NewRingWriter 创建环形写入器
func NewRingWriter(filePath string, maxSize int64) (*RingWriter, error) {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// 清空或创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	return &RingWriter{
		file:     file,
		filePath: filePath,
		size:     0,
		maxSize:  maxSize,
	}, nil
}

func (w *RingWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 如果写入后会超出大小限制，截断文件
	if w.size+int64(len(p)) > w.maxSize {
		// 读取当前文件内容
		w.file.Seek(0, 0)
		content, _ := io.ReadAll(w.file)

		// 计算需要删除的字节数
		excess := w.size + int64(len(p)) - w.maxSize
		if excess > int64(len(content)) {
			excess = int64(len(content))
		}

		// 找到第一个换行符之后的位置（保持行完整性）
		start := int(excess)
		for i := start; i < len(content); i++ {
			if content[i] == '\n' {
				start = i + 1
				break
			}
		}

		// 重写文件
		w.file.Truncate(0)
		w.file.Seek(0, 0)
		if start < len(content) {
			w.file.Write(content[start:])
			w.size = int64(len(content) - start)
		} else {
			w.size = 0
		}
	}

	// 写入新内容
	n, err = w.file.Write(p)
	w.size += int64(n)
	w.file.Sync()
	return n, err
}

func (w *RingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

var logWriter *RingWriter

// initLogger 初始化日志系统
func initLogger() {
	dataPath := getDataPath()
	logPath := filepath.Join(dataPath, "log.log")

	var err error
	logWriter, err = NewRingWriter(logPath, maxLogSize)
	if err != nil {
		log.Printf("[Logger] Failed to create log file: %v", err)
		return
	}

	// 同时输出到控制台和文件
	multiWriter := io.MultiWriter(os.Stdout, logWriter)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	log.Printf("[Logger] Log file initialized: %s (max size: %d KB)", logPath, maxLogSize/1024)
}
