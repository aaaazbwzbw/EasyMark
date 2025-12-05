package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	jpeg "image/jpeg"
	"io"
	"io/fs"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/UserExistsError/conpty"
	"github.com/gorilla/websocket"

	"golang.org/x/image/draw"
)

// generateUUID 生成简单的 UUID
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]))
}

// ==================== 鎻掍欢绯荤粺 ====================

// PluginManifest 插件清单
type PluginManifest struct {
	ID           string                 `json:"id"`
	Name         interface{}            `json:"name"` // 可以是 string 或 i18n 对象
	Version      string                 `json:"version"`
	Type         interface{}            `json:"type"`                  // 可以是 string 或 []string
	Description  interface{}            `json:"description,omitempty"` // 可以是 string 或 i18n 对象
	Author       string                 `json:"author,omitempty"`
	Entry        string                 `json:"entry,omitempty"`        // 入口可执行文件
	Capabilities map[string]interface{} `json:"capabilities,omitempty"` // 插件能力声明
	ParamsSchema map[string]interface{} `json:"paramsSchema,omitempty"`
	Inference    map[string]interface{} `json:"inference,omitempty"` // 推理配置
	Training     map[string]interface{} `json:"training,omitempty"`  // 训练配置
	Python       map[string]interface{} `json:"python,omitempty"`    // Python 依赖
	Models       []interface{}          `json:"models,omitempty"`    // 模型列表
}

// pluginHasType 检查插件是否包含指定类型
func pluginHasType(p PluginManifest, targetType string) bool {
	switch t := p.Type.(type) {
	case string:
		return t == targetType
	case []interface{}:
		for _, v := range t {
			if s, ok := v.(string); ok && s == targetType {
				return true
			}
		}
	}
	return false
}

// getPluginNameString 获取插件名称的字符串值
func getPluginNameString(name interface{}) string {
	switch n := name.(type) {
	case string:
		return n
	case map[string]interface{}:
		// 优先返回 zh-CN，然后 en-US，最后任意一个
		if v, ok := n["zh-CN"]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		if v, ok := n["en-US"]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		for _, v := range n {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

// InstalledPlugin 宸插畨瑁呮彃浠朵俊鎭?
type InstalledPlugin struct {
	PluginManifest
	InstalledAt string `json:"installedAt"`
	Path        string `json:"path"`           // 鎻掍欢瀹夎璺緞
	Logo        string `json:"logo,omitempty"` // 鎻掍欢 logo URL
	Size        int64  `json:"size,omitempty"` // 鎻掍欢鐩綍鎬诲ぇ灏忥紙瀛楄妭锛?
}

// pluginSupportsImport 鍒ゆ柇鎻掍欢鏄惁鏀寔鏁版嵁闆嗗鍏?
func pluginSupportsImport(p InstalledPlugin) bool {
	// 鍙 capabilities.importFormats 闈炵┖锛屽氨瑙嗕负鏀寔瀵煎叆
	if p.Capabilities == nil {
		return false
	}
	if raw, ok := p.Capabilities["importFormats"]; ok {
		if arr, ok := raw.([]interface{}); ok {
			return len(arr) > 0
		}
	}
	return false
}

// pluginSupportsExport 鍒ゆ柇鎻掍欢鏄惁鏀寔鏁版嵁闆嗗鍑?
func pluginSupportsExport(p InstalledPlugin) bool {
	if p.Capabilities == nil {
		return false
	}
	if raw, ok := p.Capabilities["exportFormats"]; ok {
		if arr, ok := raw.([]interface{}); ok {
			return len(arr) > 0
		}
	}
	return false
}

// 获取用户插件目录
func getPluginsDir() (string, error) {
	cfg, err := loadPathsConfig()
	if err != nil {
		return "", err
	}
	pluginsDir := filepath.Join(cfg.DataPath, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return "", err
	}
	return pluginsDir, nil
}

// 根据插件 ID 获取插件目录
func getPluginDirByID(pluginID string) (string, error) {
	// 加载所有插件，找到对应 ID 的插件目录
	plugins, err := loadInstalledPlugins()
	if err != nil {
		return "", err
	}
	for _, p := range plugins {
		if p.ID == pluginID {
			return p.Path, nil
		}
	}
	return "", fmt.Errorf("plugin not found: %s", pluginID)
}

// 获取内置插件目录列表（仅打包后使用）
func getBuiltinPluginDirs() []string {
	var dirs []string

	// 仅打包后从程序目录加载内置插件
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		// 打包后的内置插件目录
		dirs = append(dirs, filepath.Join(exeDir, "resources", "infer-plugins"))
	}

	return dirs
}

// 从指定目录加载插件
func loadPluginsFromDir(pluginsDir string, isBuiltin bool) []InstalledPlugin {
	var plugins []InstalledPlugin

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return plugins
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(pluginsDir, entry.Name(), "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		var manifest PluginManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}

		// 读取安装时间
		info, _ := entry.Info()
		installedAt := ""
		if info != nil {
			installedAt = info.ModTime().Format(time.RFC3339)
		}

		// 插件目录路径
		pluginPath := filepath.Join(pluginsDir, entry.Name())

		// 检查 logo 文件
		logoURL := ""
		logoExts := []string{".png", ".jpg", ".jpeg", ".svg", ".ico"}
		for _, ext := range logoExts {
			logoPath := filepath.Join(pluginPath, "logo"+ext)
			if _, err := os.Stat(logoPath); err == nil {
				logoURL = fmt.Sprintf("http://localhost:18080/api/plugins/%s/logo", manifest.ID)
				break
			}
		}

		// 计算插件目录大小
		var size int64
		_ = filepath.WalkDir(pluginPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			size += info.Size()
			return nil
		})

		plugins = append(plugins, InstalledPlugin{
			PluginManifest: manifest,
			InstalledAt:    installedAt,
			Path:           pluginPath,
			Logo:           logoURL,
			Size:           size,
		})
	}

	return plugins
}

// 加载已安装插件列表（包括内置和用户安装的）
func loadInstalledPlugins() ([]InstalledPlugin, error) {
	var allPlugins []InstalledPlugin
	seenIds := make(map[string]bool)

	// 1. 先加载内置插件
	for _, dir := range getBuiltinPluginDirs() {
		plugins := loadPluginsFromDir(dir, true)
		for _, p := range plugins {
			if !seenIds[p.ID] {
				seenIds[p.ID] = true
				allPlugins = append(allPlugins, p)
			}
		}
	}

	// 2. 加载用户安装的插件
	pluginsDir, err := getPluginsDir()
	if err == nil {
		plugins := loadPluginsFromDir(pluginsDir, false)
		for _, p := range plugins {
			if !seenIds[p.ID] {
				seenIds[p.ID] = true
				allPlugins = append(allPlugins, p)
			}
		}
	}

	return allPlugins, nil
}

// handlePlugins 鑾峰彇宸插畨瑁呮彃浠跺垪琛?
func handlePlugins(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodGet {
		plugins, err := loadInstalledPlugins()
		if err != nil {
			log.Printf("load plugins error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"load_failed"}`))
			return
		}

		// 支持按 type 过滤插件（如 ?type=inference）
		typeFilter := r.URL.Query().Get("type")
		if typeFilter != "" {
			var filtered []InstalledPlugin
			for _, p := range plugins {
				if pluginHasType(p.PluginManifest, typeFilter) {
					filtered = append(filtered, p)
				}
			}
			plugins = filtered
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"plugins": plugins})
		return
	}

	if r.Method == http.MethodDelete {
		pluginID := r.URL.Query().Get("id")
		if pluginID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"id_required"}`))
			return
		}
		pluginsDir, err := getPluginsDir()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"get_plugins_dir_failed"}`))
			return
		}
		// 瀹夊叏妫€鏌ワ細纭繚ID涓嶅寘鍚矾寰勫垎闅旂
		if strings.ContainsAny(pluginID, "/\\") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_id"}`))
			return
		}
		pluginPath := filepath.Join(pluginsDir, pluginID)
		if err := os.RemoveAll(pluginPath); err != nil {
			log.Printf("delete plugin error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"delete_failed"}`))
			return
		}
		// 同时删除插件的虚拟环境
		dataPath := getDataPath()
		venvPath := filepath.Join(dataPath, "plugins_python_venv", pluginID)
		if _, err := os.Stat(venvPath); err == nil {
			if err := os.RemoveAll(venvPath); err != nil {
				log.Printf("delete plugin venv error: %v", err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

// handlePluginReadme 鑾峰彇鎻掍欢 README
func handlePluginReadme(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 浠庤矾寰勬彁鍙栨彃浠禝D: /api/plugins/{id}/readme
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	pluginID := parts[2]

	// 瀹夊叏妫€鏌?
	if strings.ContainsAny(pluginID, "/\\") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	pluginsDir, err := getPluginsDir()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 灏濊瘯澶氱 README 鏂囦欢鍚?
	readmeNames := []string{"README.md", "readme.md", "README.MD", "Readme.md", "README", "readme"}
	var readmeContent []byte
	for _, name := range readmeNames {
		readmePath := filepath.Join(pluginsDir, pluginID, name)
		if data, err := os.ReadFile(readmePath); err == nil {
			readmeContent = data
			break
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if readmeContent != nil {
		_, _ = w.Write(readmeContent)
	}
}

// handlePluginLogo 鑾峰彇鎻掍欢 Logo
func handlePluginLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 浠庤矾寰勬彁鍙栨彃浠禝D: /api/plugins/{id}/logo
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	pluginID := parts[2]

	// 瀹夊叏妫€鏌?
	if strings.ContainsAny(pluginID, "/\\") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	pluginsDir, err := getPluginsDir()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 鏌ユ壘 logo 鏂囦欢
	logoExts := []string{".png", ".jpg", ".jpeg", ".svg", ".ico"}
	contentTypes := map[string]string{
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
	}
	for _, ext := range logoExts {
		logoPath := filepath.Join(pluginsDir, pluginID, "logo"+ext)
		if data, err := os.ReadFile(logoPath); err == nil {
			w.Header().Set("Content-Type", contentTypes[ext])
			w.Header().Set("Cache-Control", "max-age=3600")
			_, _ = w.Write(data)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

// handlePluginUI 插件 UI 静态文件服务
// 路径格式: /api/plugins/{pluginId}/ui/{filepath}
func handlePluginUI(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 从路径提取插件 ID 和文件路径
	// /api/plugins/{pluginId}/ui/{filepath...}
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/api/plugins/"), "/")
	if len(parts) < 3 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	pluginID := parts[0]
	// 重组文件路径 (parts[1] 是 "ui", parts[2:] 是实际文件路径)
	filePath := strings.Join(parts[2:], "/")

	// 获取插件目录
	pluginsDir, err := getPluginsDir()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 构建完整文件路径
	fullPath := filepath.Join(pluginsDir, pluginID, "ui", filePath)

	// 安全检查：确保路径在插件目录内
	pluginsAbsDir, _ := filepath.Abs(pluginsDir)
	fullAbsPath, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(fullAbsPath, pluginsAbsDir) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 设置 Content-Type
	ext := strings.ToLower(filepath.Ext(filePath))
	contentTypes := map[string]string{
		".html":  "text/html; charset=utf-8",
		".css":   "text/css; charset=utf-8",
		".js":    "application/javascript; charset=utf-8",
		".json":  "application/json; charset=utf-8",
		".png":   "image/png",
		".jpg":   "image/jpeg",
		".jpeg":  "image/jpeg",
		".svg":   "image/svg+xml",
		".ico":   "image/x-icon",
		".woff":  "font/woff",
		".woff2": "font/woff2",
		".ttf":   "font/ttf",
	}
	if ct, ok := contentTypes[ext]; ok {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// 读取并返回文件
	data, err := os.ReadFile(fullPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

// handlePluginFiles 列出插件目录中的文件
// 路径格式: /api/plugins/{pluginId}/files
func handlePluginFiles(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 从路径提取插件 ID
	path := r.URL.Path
	pluginID := strings.TrimSuffix(strings.TrimPrefix(path, "/api/plugins/"), "/files")
	if pluginID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 根据插件 ID 获取插件目录
	pluginDir, err := getPluginDirByID(pluginID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"plugin_not_found"}`))
		return
	}

	// 读取目录中的文件
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type FileInfo struct {
		Name  string `json:"name"`
		Size  int64  `json:"size"`
		IsDir bool   `json:"isDir"`
	}

	files := []FileInfo{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name:  entry.Name(),
			Size:  info.Size(),
			IsDir: entry.IsDir(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pluginId":  pluginID,
		"pluginDir": pluginDir,
		"files":     files,
	})
}

// handlePluginDownloadModel 下载模型到插件目录
// POST /api/plugins/{pluginId}/download-model
func handlePluginDownloadModel(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 从路径提取插件 ID
	path := r.URL.Path
	pluginID := strings.TrimSuffix(strings.TrimPrefix(path, "/api/plugins/"), "/download-model")
	if pluginID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 解析请求
	var req struct {
		URL      string `json:"url"`
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" || req.Filename == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"url_and_filename_required"}`))
		return
	}

	// 根据插件 ID 获取插件目录
	pluginDir, err := getPluginDirByID(pluginID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"plugin_not_found"}`))
		return
	}

	targetPath := filepath.Join(pluginDir, req.Filename)

	// 启动后台下载
	go downloadModelToPlugin(pluginID, req.URL, targetPath, req.Filename)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message":    "download_started",
		"targetPath": targetPath,
	})
}

// downloadModelToPlugin 后台下载模型并通过 WebSocket 推送进度
func downloadModelToPlugin(pluginID, url, targetPath, filename string) {
	// 发送开始下载通知
	broadcastGlobalWS(GlobalWSMessage{
		Type:     "model_download_start",
		PluginID: pluginID,
		Data:     map[string]interface{}{"filename": filename},
	})

	// 创建 HTTP 客户端
	client := &http.Client{}
	resp, err := client.Get(url)
	if err != nil {
		broadcastGlobalWS(GlobalWSMessage{
			Type:     "model_download_error",
			PluginID: pluginID,
			Message:  err.Error(),
			Data:     map[string]interface{}{"filename": filename},
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		broadcastGlobalWS(GlobalWSMessage{
			Type:     "model_download_error",
			PluginID: pluginID,
			Message:  fmt.Sprintf("HTTP %d", resp.StatusCode),
			Data:     map[string]interface{}{"filename": filename},
		})
		return
	}

	totalSize := resp.ContentLength

	// 创建临时文件
	tmpPath := targetPath + ".downloading"
	outFile, err := os.Create(tmpPath)
	if err != nil {
		broadcastGlobalWS(GlobalWSMessage{
			Type:     "model_download_error",
			PluginID: pluginID,
			Message:  err.Error(),
			Data:     map[string]interface{}{"filename": filename},
		})
		return
	}

	// 边读边写，并推送进度
	buf := make([]byte, 32*1024)
	var downloaded int64
	lastProgress := -1

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := outFile.Write(buf[:n])
			if writeErr != nil {
				outFile.Close()
				os.Remove(tmpPath)
				broadcastGlobalWS(GlobalWSMessage{
					Type:     "model_download_error",
					PluginID: pluginID,
					Message:  writeErr.Error(),
					Data:     map[string]interface{}{"filename": filename},
				})
				return
			}
			downloaded += int64(n)

			// 每 1% 推送一次进度
			if totalSize > 0 {
				progress := int(downloaded * 100 / totalSize)
				if progress != lastProgress {
					lastProgress = progress
					broadcastGlobalWS(GlobalWSMessage{
						Type:     "model_download_progress",
						PluginID: pluginID,
						Data: map[string]interface{}{
							"filename":   filename,
							"progress":   progress,
							"downloaded": downloaded,
							"total":      totalSize,
						},
					})
				}
			}
		}
		if readErr != nil {
			if readErr.Error() == "EOF" {
				break
			}
			outFile.Close()
			os.Remove(tmpPath)
			broadcastGlobalWS(GlobalWSMessage{
				Type:     "model_download_error",
				PluginID: pluginID,
				Message:  readErr.Error(),
				Data:     map[string]interface{}{"filename": filename},
			})
			return
		}
	}

	outFile.Close()

	// 重命名为最终文件
	if err := os.Rename(tmpPath, targetPath); err != nil {
		os.Remove(tmpPath)
		broadcastGlobalWS(GlobalWSMessage{
			Type:     "model_download_error",
			PluginID: pluginID,
			Message:  err.Error(),
			Data:     map[string]interface{}{"filename": filename},
		})
		return
	}

	// 下载完成
	broadcastGlobalWS(GlobalWSMessage{
		Type:     "model_download_complete",
		PluginID: pluginID,
		Success:  true,
		Data: map[string]interface{}{
			"filename":   filename,
			"targetPath": targetPath,
			"size":       downloaded,
		},
	})
}

// handlePluginInstall 安装插件（从 zip 文件）
func handlePluginInstall(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 瑙ｆ瀽璇锋眰
	var req struct {
		FilePath string `json:"filePath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FilePath == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"file_path_required"}`))
		return
	}

	pluginsDir, err := getPluginsDir()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"get_plugins_dir_failed"}`))
		return
	}

	// 鍒涘缓涓存椂瑙ｅ帇鐩綍
	tempDir, err := os.MkdirTemp("", "plugin-install-*")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"create_temp_dir_failed"}`))
		return
	}
	defer os.RemoveAll(tempDir)

	// 鏍规嵁鏂囦欢鎵╁睍鍚嶈В鍘?
	ext := strings.ToLower(filepath.Ext(req.FilePath))
	if ext == ".zip" {
		if err := unzip(req.FilePath, tempDir); err != nil {
			log.Printf("unzip error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"unzip_failed"}`))
			return
		}
	} else if ext == ".rar" {
		// 浣跨敤绯荤粺鐨?unrar 鎴?7z 瑙ｅ帇
		if err := unrar(req.FilePath, tempDir); err != nil {
			log.Printf("unrar error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"unrar_failed","message":"` + err.Error() + `"}`))
			return
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"unsupported_format"}`))
		return
	}

	// 鏌ユ壘 manifest.json锛堝彲鑳藉湪鏍圭洰褰曟垨瀛愮洰褰曚腑锛?
	var manifestPath string
	var pluginRoot string
	err = filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "manifest.json" {
			manifestPath = path
			pluginRoot = filepath.Dir(path)
			return filepath.SkipAll
		}
		return nil
	})
	if manifestPath == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"manifest_not_found"}`))
		return
	}

	// 璇诲彇骞堕獙璇?manifest
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"read_manifest_failed"}`))
		return
	}
	var manifest PluginManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_manifest"}`))
		return
	}
	if manifest.ID == "" || manifest.Name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"manifest_missing_fields"}`))
		return
	}

	// 瀹夊叏妫€鏌D
	if strings.ContainsAny(manifest.ID, "/\\") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_plugin_id"}`))
		return
	}

	// 移动到插件目录
	destDir := filepath.Join(pluginsDir, manifest.ID)

	// 如果已存在，保留模型文件后再删除其他文件
	var modelFiles []string
	if _, err := os.Stat(destDir); err == nil {
		// 扫描现有的模型文件
		modelExts := map[string]bool{".pt": true, ".pth": true, ".onnx": true, ".safetensors": true, ".bin": true}
		filepath.WalkDir(destDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if modelExts[ext] {
				relPath, _ := filepath.Rel(destDir, path)
				modelFiles = append(modelFiles, relPath)
			}
			return nil
		})

		if len(modelFiles) > 0 {
			log.Printf("[Plugin Install] Found %d model files to preserve: %v", len(modelFiles), modelFiles)
			// 创建临时目录保存模型文件
			modelTempDir, err := os.MkdirTemp("", "plugin-models-*")
			if err == nil {
				defer os.RemoveAll(modelTempDir)
				// 移动模型文件到临时目录
				for _, relPath := range modelFiles {
					srcPath := filepath.Join(destDir, relPath)
					dstPath := filepath.Join(modelTempDir, relPath)
					os.MkdirAll(filepath.Dir(dstPath), 0755)
					os.Rename(srcPath, dstPath)
				}
				// 删除旧插件目录
				os.RemoveAll(destDir)
				// 安装新插件
				if err := os.Rename(pluginRoot, destDir); err != nil {
					if err := copyDir(pluginRoot, destDir); err != nil {
						log.Printf("copy plugin error: %v", err)
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = w.Write([]byte(`{"error":"install_failed"}`))
						return
					}
				}
				// 恢复模型文件
				for _, relPath := range modelFiles {
					srcPath := filepath.Join(modelTempDir, relPath)
					dstPath := filepath.Join(destDir, relPath)
					os.MkdirAll(filepath.Dir(dstPath), 0755)
					os.Rename(srcPath, dstPath)
				}
				log.Printf("[Plugin Install] Restored %d model files", len(modelFiles))

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"success":         true,
					"plugin":          manifest,
					"modelsPreserved": len(modelFiles),
				})
				return
			}
		}
		// 无模型文件或临时目录创建失败，直接删除
		os.RemoveAll(destDir)
	}

	if err := os.Rename(pluginRoot, destDir); err != nil {
		// rename 可能跨磁盘失败，改用复制
		if err := copyDir(pluginRoot, destDir); err != nil {
			log.Printf("copy plugin error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"install_failed"}`))
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"plugin":  manifest,
	})
}

// unzip 瑙ｅ帇 zip 鏂囦欢
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		// 瀹夊叏妫€鏌ワ細闃叉 zip slip
		if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// unrar 浣跨敤绯荤粺鍛戒护瑙ｅ帇 rar 鏂囦欢
func unrar(src, dest string) error {
	// 灏濊瘯浣跨敤 unrar
	cmd := exec.Command("unrar", "x", "-y", src, dest+string(os.PathSeparator))
	if err := cmd.Run(); err == nil {
		return nil
	}
	// 灏濊瘯浣跨敤 7z
	cmd = exec.Command("7z", "x", src, "-o"+dest, "-y")
	if err := cmd.Run(); err == nil {
		return nil
	}
	return fmt.Errorf("unrar/7z not available or extraction failed")
}

// copyDir 澶嶅埗鐩綍
func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		destPath := filepath.Join(dest, relPath)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()
		_, err = io.Copy(destFile, srcFile)
		return err
	})
}

// handleImportDatasetPlugins 鑾峰彇鍙敤浜庡鍏ユ暟鎹泦鐨勬彃浠跺垪琛?
func handleImportDatasetPlugins(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	plugins, err := loadInstalledPlugins()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"load_failed"}`))
		return
	}

	// 杩囨护鍑哄鍏ョ被鍨嬬殑鎻掍欢
	var importPlugins []InstalledPlugin
	for _, p := range plugins {
		if pluginSupportsImport(p) {
			importPlugins = append(importPlugins, p)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"plugins": importPlugins})
}

// handleDatasetDetect 鎺㈡祴鏁版嵁闆嗘牸寮?
func handleDatasetDetect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RootPath string `json:"rootPath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RootPath == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"root_path_required"}`))
		return
	}

	plugins, err := loadInstalledPlugins()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"load_plugins_failed"}`))
		return
	}

	type DetectResult struct {
		PluginID  string  `json:"pluginId"`
		Supported bool    `json:"supported"`
		Score     float64 `json:"score"`
		Reason    string  `json:"reason"`
	}

	var results []DetectResult
	for _, plugin := range plugins {
		if !pluginSupportsImport(plugin) {
			continue
		}
		// Execute plugin detect command
		result := executePluginDetect(plugin, req.RootPath)
		if result.Supported {
			results = append(results, DetectResult{
				PluginID:  plugin.ID,
				Supported: result.Supported,
				Score:     result.Score,
				Reason:    result.Reason,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
}

// executePluginDetect 鎵ц鎻掍欢鐨勬帰娴嬪懡浠?
func executePluginDetect(plugin InstalledPlugin, rootPath string) struct {
	Supported bool
	Score     float64
	Reason    string
} {
	result := struct {
		Supported bool
		Score     float64
		Reason    string
	}{Supported: false}

	if plugin.Entry == "" {
		log.Printf("[Plugin %s] No entry point defined", plugin.ID)
		return result
	}

	entryPath := filepath.Join(plugin.Path, plugin.Entry)
	if runtime.GOOS == "windows" && !strings.HasSuffix(entryPath, ".exe") {
		entryPath += ".exe"
	}

	// Check if entry exists
	if _, err := os.Stat(entryPath); os.IsNotExist(err) {
		log.Printf("[Plugin %s] Entry not found: %s", plugin.ID, entryPath)
		return result
	}

	// Prepare input
	input := map[string]interface{}{
		"rootPath": rootPath,
	}
	inputJSON, _ := json.Marshal(input)
	log.Printf("[Plugin %s] Detect input: %s", plugin.ID, string(inputJSON))

	// Execute plugin with stderr capture
	cmd := exec.Command(entryPath, "detect")
	cmd.Stdin = strings.NewReader(string(inputJSON))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if stderr.Len() > 0 {
		log.Printf("[Plugin %s] Detect stderr:\n%s", plugin.ID, stderr.String())
	}
	if err != nil {
		log.Printf("[Plugin %s] Detect execution error: %v", plugin.ID, err)
		return result
	}

	output := stdout.Bytes()
	log.Printf("[Plugin %s] Detect output: %s", plugin.ID, string(output))

	// Parse output
	var resp struct {
		Supported bool    `json:"supported"`
		Score     float64 `json:"score"`
		Reason    string  `json:"reason"`
	}
	if err := json.Unmarshal(output, &resp); err != nil {
		log.Printf("[Plugin %s] Detect parse error: %v, raw: %s", plugin.ID, err, string(output))
		return result
	}

	result.Supported = resp.Supported
	result.Score = resp.Score
	result.Reason = resp.Reason
	log.Printf("[Plugin %s] Detect result: supported=%v, score=%.2f, reason=%s", plugin.ID, result.Supported, result.Score, result.Reason)
	return result
}

// handleDatasetImport 鎵ц鏁版嵁闆嗗鍏ワ紙寮傛浠诲姟妯″紡锛?
func handleDatasetImport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProjectID  string                 `json:"projectId"`
		PluginID   string                 `json:"pluginId"`
		RootPath   string                 `json:"rootPath"`
		ImportMode string                 `json:"importMode"` // copy, link, external
		Params     map[string]interface{} `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_request"}`))
		return
	}

	if req.ProjectID == "" || req.PluginID == "" || req.RootPath == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"missing_required_fields"}`))
		return
	}

	// Validate import mode
	importMode := strings.TrimSpace(strings.ToLower(req.ImportMode))
	if importMode == "" {
		importMode = "link"
	}
	if importMode != "copy" && importMode != "link" && importMode != "external" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_import_mode"}`))
		return
	}

	log.Printf("[DatasetImport] Request: projectId=%s, pluginId=%s, rootPath=%s, importMode=%s",
		req.ProjectID, req.PluginID, req.RootPath, importMode)

	// Find plugin
	plugins, _ := loadInstalledPlugins()
	var targetPlugin *InstalledPlugin
	for _, p := range plugins {
		if p.ID == req.PluginID {
			targetPlugin = &p
			break
		}
	}
	if targetPlugin == nil {
		log.Printf("[DatasetImport] Plugin not found: %s", req.PluginID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"plugin_not_found"}`))
		return
	}

	// Load config
	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"config_unavailable"}`))
		return
	}

	// Create task ID
	taskID, err := generateProjectID()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"task_id_failed"}`))
		return
	}

	// Check if another I/O task is running
	if !tryAcquireIOLock(taskID) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"io_task_busy","currentTaskId":"` + getCurrentIOTask() + `"}`))
		return
	}

	// Create import task
	importTasksMu.Lock()
	importTasks[taskID] = &ImportTaskStatus{
		ID:        taskID,
		ProjectID: req.ProjectID,
		TaskType:  taskTypeImportDataset,
		Phase:     importPhaseScanning,
		Progress:  0,
		Imported:  0,
		Total:     0,
	}
	importTasksMu.Unlock()

	// Start async import
	go runDatasetImportTask(cfg.DataPath, req.ProjectID, taskID, importMode, req.RootPath, *targetPlugin, req.Params)

	// Return task ID immediately
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"taskId": taskID,
	})
}

// runDatasetImportTask 寮傛鎵ц鏁版嵁闆嗗鍏ヤ换鍔?
func runDatasetImportTask(dataPath, projectID, taskID, importMode, rootPath string, plugin InstalledPlugin, params map[string]interface{}) {
	defer releaseIOLock(taskID)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[DatasetImport] Task %s panic: %v", taskID, r)
			importTasksMu.Lock()
			if task, ok := importTasks[taskID]; ok {
				task.Phase = importPhaseFailed
				task.Error = "panic_in_import_task"
			}
			importTasksMu.Unlock()
		}
	}()

	log.Printf("[DatasetImport] Task %s started", taskID)

	// Phase 1: Execute plugin to get dataset info
	importTasksMu.Lock()
	if task, ok := importTasks[taskID]; ok {
		task.Phase = importPhaseScanning
	}
	importTasksMu.Unlock()

	result, err := executePluginImport(plugin, rootPath, params)
	if err != nil {
		log.Printf("[DatasetImport] Task %s plugin execution failed: %v", taskID, err)
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseFailed
			task.Error = "plugin_execution_failed"
		}
		importTasksMu.Unlock()
		return
	}
	log.Printf("[DatasetImport] Task %s plugin returned: %d images, %d categories, %d annotations",
		taskID, len(result.Images), len(result.Categories), len(result.Annotations))

	// Setup directories
	projectRoot := filepath.Join(dataPath, "project_item", projectID)
	imagesDir := filepath.Join(projectRoot, "images")
	originalsDir := filepath.Join(imagesDir, "originals")
	thumbsDir := filepath.Join(imagesDir, "thumbs")
	if err := os.MkdirAll(originalsDir, 0755); err != nil {
		log.Printf("[DatasetImport] Task %s mkdir originals failed: %v", taskID, err)
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseFailed
			task.Error = "storage_unavailable"
		}
		importTasksMu.Unlock()
		return
	}
	if err := os.MkdirAll(thumbsDir, 0755); err != nil {
		log.Printf("[DatasetImport] Task %s mkdir thumbs failed: %v", taskID, err)
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseFailed
			task.Error = "storage_unavailable"
		}
		importTasksMu.Unlock()
		return
	}

	// Phase 2: Copy/link images
	importTasksMu.Lock()
	if task, ok := importTasks[taskID]; ok {
		task.Phase = importPhaseCopying
		task.Total = len(result.Images)
		task.Imported = 0
	}
	importTasksMu.Unlock()

	type importedImage struct {
		Key         string
		Filename    string
		OriginalRel string
		ThumbRel    string
	}

	// Check existing files
	existingNames := make(map[string]struct{})
	existingNamesMu := sync.Mutex{}
	entries, _ := os.ReadDir(originalsDir)
	for _, entry := range entries {
		if !entry.IsDir() {
			existingNames[entry.Name()] = struct{}{}
		}
	}

	// 并发处理图片导入
	type importJob struct {
		Index    int
		Key      string
		SrcPath  string
		BaseName string
	}

	totalImages := len(result.Images)
	jobsCh := make(chan importJob, totalImages)
	resultsCh := make(chan importedImage, totalImages)
	var wg sync.WaitGroup
	var processedCount int64

	// worker 数量 = CPU 核心数（至少 2 个）
	workerCount := runtime.NumCPU()
	if workerCount < 2 {
		workerCount = 2
	}
	log.Printf("[DatasetImport] Task %s starting %d workers for %d images", taskID, workerCount, totalImages)

	// 启动 worker
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobsCh {
				var originalRel, thumbRel string

				switch importMode {
				case "external":
					originalRel = filepath.ToSlash(job.SrcPath)
					thumbRel = originalRel
				case "link", "copy":
					originalPath := filepath.Join(originalsDir, job.BaseName)
					originalRel = filepath.ToSlash(filepath.Join("images", "originals", job.BaseName))
					thumbRel = originalRel

					// Check if already exists (with lock)
					existingNamesMu.Lock()
					_, exists := existingNames[job.BaseName]
					if !exists {
						existingNames[job.BaseName] = struct{}{}
					}
					existingNamesMu.Unlock()

					if !exists {
						// Get source file info
						srcInfo, err := os.Stat(job.SrcPath)
						if err != nil {
							log.Printf("[DatasetImport] Task %s source file not found: %s", taskID, job.SrcPath)
							resultsCh <- importedImage{}
							atomic.AddInt64(&processedCount, 1)
							continue
						}

						// Copy or link
						if importMode == "link" {
							if err := os.Link(job.SrcPath, originalPath); err != nil {
								if copyErr := copyFile(job.SrcPath, originalPath); copyErr != nil {
									log.Printf("[DatasetImport] Task %s copy failed: %s -> %s: %v", taskID, job.SrcPath, originalPath, copyErr)
									resultsCh <- importedImage{}
									atomic.AddInt64(&processedCount, 1)
									continue
								}
							}
						} else {
							if err := copyFile(job.SrcPath, originalPath); err != nil {
								log.Printf("[DatasetImport] Task %s copy failed: %s -> %s: %v", taskID, job.SrcPath, originalPath, err)
								resultsCh <- importedImage{}
								atomic.AddInt64(&processedCount, 1)
								continue
							}
						}

						// Generate thumbnail if file > 3MB
						if srcInfo.Size() > thumbSizeThresholdBytes {
							ext := strings.ToLower(filepath.Ext(job.BaseName))
							thumbBase := strings.TrimSuffix(job.BaseName, ext) + ".jpg"
							thumbPath := filepath.Join(thumbsDir, thumbBase)
							srcFile, err := os.Open(originalPath)
							if err == nil {
								srcImg, _, decErr := image.Decode(srcFile)
								_ = srcFile.Close()
								if decErr == nil {
									b := srcImg.Bounds()
									w, h := b.Dx(), b.Dy()
									newW, newH := w, h
									if w >= h && w > thumbMaxDimension {
										newW = thumbMaxDimension
										newH = int(float64(h) * float64(newW) / float64(w))
									} else if h > w && h > thumbMaxDimension {
										newH = thumbMaxDimension
										newW = int(float64(w) * float64(newH) / float64(h))
									}
									thumbImg := image.NewRGBA(image.Rect(0, 0, newW, newH))
									draw.ApproxBiLinear.Scale(thumbImg, thumbImg.Bounds(), srcImg, b, draw.Over, nil)
									if out, err := os.Create(thumbPath); err == nil {
										_ = jpeg.Encode(out, thumbImg, &jpeg.Options{Quality: 75})
										_ = out.Close()
										thumbRel = filepath.ToSlash(filepath.Join("images", "thumbs", thumbBase))
									}
								}
							}
						}
					}
				}

				resultsCh <- importedImage{
					Key:         job.Key,
					Filename:    job.BaseName,
					OriginalRel: originalRel,
					ThumbRel:    thumbRel,
				}

				// Update progress
				processed := atomic.AddInt64(&processedCount, 1)
				importTasksMu.Lock()
				if task, ok := importTasks[taskID]; ok {
					task.Imported = int(processed)
					task.Total = totalImages
					if totalImages > 0 {
						task.Progress = int(float64(processed) * 100 / float64(totalImages))
					}
				}
				importTasksMu.Unlock()
			}
		}()
	}

	// 发送任务
	for i, img := range result.Images {
		srcPath := filepath.Join(rootPath, img.RelativePath)
		baseName := filepath.Base(img.RelativePath)
		jobsCh <- importJob{
			Index:    i,
			Key:      img.Key,
			SrcPath:  srcPath,
			BaseName: baseName,
		}
	}
	close(jobsCh)

	// 等待所有 worker 完成
	wg.Wait()
	close(resultsCh)

	// 收集结果
	importedImages := make([]importedImage, 0, totalImages)
	for img := range resultsCh {
		if img.Key != "" {
			importedImages = append(importedImages, img)
		}
	}

	log.Printf("[DatasetImport] Task %s copied %d images", taskID, len(importedImages))

	// Phase 3: Write to database (indexing phase)
	importTasksMu.Lock()
	if task, ok := importTasks[taskID]; ok {
		task.Phase = importPhaseIndexing
		task.Total = len(importedImages) + len(result.Annotations)
		task.Imported = 0
	}
	importTasksMu.Unlock()

	dbPath := filepath.Join(projectRoot, "db", "project.db")
	db, err := openProjectDB(dbPath)
	if err != nil {
		log.Printf("[DatasetImport] Task %s open db failed: %v", taskID, err)
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseFailed
			task.Error = "db_unavailable"
		}
		importTasksMu.Unlock()
		return
	}
	defer db.Close()

	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		log.Printf("[DatasetImport] Task %s set WAL failed: %v", taskID, err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		log.Printf("[DatasetImport] Task %s set busy_timeout failed: %v", taskID, err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("[DatasetImport] Task %s begin tx failed: %v", taskID, err)
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseFailed
			task.Error = "tx_begin_failed"
		}
		importTasksMu.Unlock()
		return
	}
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	now := time.Now().Format(time.RFC3339)
	imageKeyToID := make(map[string]int64)
	categoryKeyToID := make(map[string]int64)

	// Import categories (or find existing ones)
	// First pass: create all categories and build key->ID map
	// Debug: log all categories from plugin
	for i, cat := range result.Categories {
		metaStr := "{}"
		if cat.Meta != nil {
			if b, err := json.Marshal(cat.Meta); err == nil {
				metaStr = string(b)
			}
		}
		log.Printf("[DatasetImport] Task %s category[%d]: key=%s, name=%s, type=%s, meta=%s", taskID, i, cat.Key, cat.Name, cat.Type, metaStr)
	}
	for _, cat := range result.Categories {
		var existingID int64
		err := tx.QueryRow(`SELECT id FROM categories WHERE name = ?;`, cat.Name).Scan(&existingID)
		if err == nil {
			categoryKeyToID[cat.Key] = existingID
			continue
		}
		mateJSON := "{}"
		if cat.Meta != nil {
			if b, err := json.Marshal(cat.Meta); err == nil {
				mateJSON = string(b)
			}
		}
		res, err := tx.Exec(`INSERT INTO categories (name, type, color, sort_order, mate) VALUES (?, ?, ?, ?, ?);`,
			cat.Name, cat.Type, cat.Color, cat.SortOrder, mateJSON)
		if err != nil {
			log.Printf("[DatasetImport] Task %s insert category error: %v", taskID, err)
			continue
		}
		id, _ := res.LastInsertId()
		categoryKeyToID[cat.Key] = id
	}

	// Second pass: update bbox categories to bind keypoint categories by ID
	boundCount := 0
	for _, cat := range result.Categories {
		if cat.Type != "bbox" {
			continue
		}
		if cat.Meta == nil {
			log.Printf("[DatasetImport] Task %s bbox category %s has nil Meta", taskID, cat.Key)
			continue
		}
		kpKey, hasKpKey := cat.Meta["keypointCategoryKey"].(string)
		if !hasKpKey || kpKey == "" {
			log.Printf("[DatasetImport] Task %s bbox category %s has no keypointCategoryKey in Meta: %v", taskID, cat.Key, cat.Meta)
			continue
		}
		kpCatID, found := categoryKeyToID[kpKey]
		if !found {
			log.Printf("[DatasetImport] Task %s keypoint category key %s not found in categoryKeyToID", taskID, kpKey)
			continue
		}
		bboxCatID, hasBbox := categoryKeyToID[cat.Key]
		if !hasBbox {
			log.Printf("[DatasetImport] Task %s bbox category key %s not found in categoryKeyToID", taskID, cat.Key)
			continue
		}
		// Update the bbox category's mate to include keypointCategoryId
		newMate := map[string]interface{}{"keypointCategoryId": kpCatID}
		mateJSON, _ := json.Marshal(newMate)
		_, err := tx.Exec(`UPDATE categories SET mate = ? WHERE id = ?;`, string(mateJSON), bboxCatID)
		if err != nil {
			log.Printf("[DatasetImport] Task %s update bbox category mate error: %v", taskID, err)
		} else {
			log.Printf("[DatasetImport] Task %s bound bbox category %d to keypoint category %d", taskID, bboxCatID, kpCatID)
			boundCount++
		}
	}
	if boundCount > 0 {
		log.Printf("[DatasetImport] Task %s bound %d bbox categories to keypoint categories", taskID, boundCount)
	}
	log.Printf("[DatasetImport] Task %s imported %d categories", taskID, len(categoryKeyToID))

	// Import images to database
	processed := 0
	for _, img := range importedImages {
		var existingID int64
		err := tx.QueryRow(`SELECT id FROM image_index WHERE filename = ?;`, img.Filename).Scan(&existingID)
		if err == nil {
			imageKeyToID[img.Key] = existingID
		} else {
			res, err := tx.Exec(`INSERT INTO image_index (filename, original_rel_path, thumb_rel_path, deleted_in_project, annotation_status, created_at) VALUES (?, ?, ?, 0, 'none', ?);`,
				img.Filename, img.OriginalRel, img.ThumbRel, now)
			if err != nil {
				log.Printf("[DatasetImport] Task %s insert image error: %v", taskID, err)
				continue
			}
			id, _ := res.LastInsertId()
			imageKeyToID[img.Key] = id
			// 创建引用计数记录
			_, _ = tx.Exec(`INSERT INTO image_ref_count (image_id, ref_count) VALUES (?, 1);`, id)
		}
		processed++
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Imported = processed
			if task.Total > 0 {
				task.Progress = int(float64(task.Imported) * 100 / float64(task.Total))
			}
		}
		importTasksMu.Unlock()
	}
	log.Printf("[DatasetImport] Task %s indexed %d images", taskID, len(imageKeyToID))

	// Import annotations
	importedAnnotations := 0
	for _, ann := range result.Annotations {
		imageID, ok := imageKeyToID[ann.ImageKey]
		if !ok {
			continue
		}
		categoryID, ok := categoryKeyToID[ann.CategoryKey]
		if !ok {
			continue
		}

		// Process annotation data - convert keypointCategoryKey to keypointCategoryId
		annData := ann.Data
		if kpKey, hasKpKey := annData["keypointCategoryKey"].(string); hasKpKey {
			if kpCatID, found := categoryKeyToID[kpKey]; found {
				annData["keypointCategoryId"] = kpCatID
			}
			delete(annData, "keypointCategoryKey") // Remove the key, keep only the ID
		}

		dataJSON, _ := json.Marshal(annData)
		_, err := tx.Exec(`INSERT INTO annotations (image_id, category_id, type, data, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?);`,
			imageID, categoryID, ann.Type, string(dataJSON), now, now)
		if err != nil {
			continue
		}
		importedAnnotations++
		processed++
		if processed%500 == 0 {
			importTasksMu.Lock()
			if task, ok := importTasks[taskID]; ok {
				task.Imported = processed
				if task.Total > 0 {
					task.Progress = int(float64(task.Imported) * 100 / float64(task.Total))
				}
			}
			importTasksMu.Unlock()
		}
	}
	log.Printf("[DatasetImport] Task %s imported %d annotations", taskID, importedAnnotations)

	// Update annotation status - use batch update for better performance
	log.Printf("[DatasetImport] Task %s updating annotation status for %d images...", taskID, len(imageKeyToID))
	_, err = tx.Exec(`UPDATE image_index SET annotation_status = 'annotated' WHERE id IN (SELECT DISTINCT image_id FROM annotations);`)
	if err != nil {
		log.Printf("[DatasetImport] Task %s update annotation status failed: %v", taskID, err)
	}

	log.Printf("[DatasetImport] Task %s committing transaction...", taskID)
	if err := tx.Commit(); err != nil {
		log.Printf("[DatasetImport] Task %s commit failed: %v", taskID, err)
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseFailed
			task.Error = "tx_commit_failed"
		}
		importTasksMu.Unlock()
		return
	}
	tx = nil // Prevent defer from rolling back

	// Mark as completed
	importTasksMu.Lock()
	if task, ok := importTasks[taskID]; ok {
		task.Phase = importPhaseCompleted
		task.Progress = 100
		task.Total = len(imageKeyToID)
		task.Imported = len(imageKeyToID)
	}
	importTasksMu.Unlock()

	log.Printf("[DatasetImport] Task %s SUCCESS - images=%d, categories=%d, annotations=%d",
		taskID, len(imageKeyToID), len(categoryKeyToID), importedAnnotations)
}

// PluginImportResult 鎻掍欢瀵煎叆缁撴灉
type PluginImportResult struct {
	Images []struct {
		Key          string                 `json:"key"`
		RelativePath string                 `json:"relativePath"`
		Meta         map[string]interface{} `json:"meta"`
	} `json:"images"`
	Categories []struct {
		Key       string                 `json:"key"`
		Name      string                 `json:"name"`
		Type      string                 `json:"type"`
		Color     string                 `json:"color"`
		SortOrder int                    `json:"sortOrder"`
		Meta      map[string]interface{} `json:"meta"`
	} `json:"categories"`
	Annotations []struct {
		ImageKey    string                 `json:"imageKey"`
		CategoryKey string                 `json:"categoryKey"`
		Type        string                 `json:"type"`
		Data        map[string]interface{} `json:"data"`
	} `json:"annotations"`
	Stats struct {
		ImageCount         int `json:"imageCount"`
		AnnotationCount    int `json:"annotationCount"`
		SkippedImages      int `json:"skippedImages"`
		SkippedAnnotations int `json:"skippedAnnotations"`
	} `json:"stats"`
	Errors []struct {
		Code    string                 `json:"code"`
		Message string                 `json:"message"`
		Details map[string]interface{} `json:"details"`
	} `json:"errors"`
}

// executePluginImport 鎵ц鎻掍欢鐨勫鍏ュ懡浠?
func executePluginImport(plugin InstalledPlugin, rootPath string, params map[string]interface{}) (*PluginImportResult, error) {
	if plugin.Entry == "" {
		return nil, fmt.Errorf("plugin has no entry point")
	}

	entryPath := filepath.Join(plugin.Path, plugin.Entry)
	if runtime.GOOS == "windows" && !strings.HasSuffix(entryPath, ".exe") {
		entryPath += ".exe"
	}

	log.Printf("[Plugin %s] Import entry path: %s", plugin.ID, entryPath)

	if _, err := os.Stat(entryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin entry not found: %s", entryPath)
	}

	// Prepare input
	input := map[string]interface{}{
		"rootPath": rootPath,
		"formatId": plugin.ID,
		"params":   params,
	}
	inputJSON, _ := json.Marshal(input)
	log.Printf("[Plugin %s] Import input: %s", plugin.ID, string(inputJSON))

	// Execute plugin with stderr capture
	cmd := exec.Command(entryPath, "import")
	cmd.Stdin = strings.NewReader(string(inputJSON))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if stderr.Len() > 0 {
		log.Printf("[Plugin %s] Import stderr:\n%s", plugin.ID, stderr.String())
	}
	if err != nil {
		log.Printf("[Plugin %s] Import execution error: %v", plugin.ID, err)
		return nil, fmt.Errorf("plugin execution failed: %v, stderr: %s", err, stderr.String())
	}

	output := stdout.Bytes()
	log.Printf("[Plugin %s] Import output length: %d bytes", plugin.ID, len(output))
	if len(output) < 2000 {
		log.Printf("[Plugin %s] Import output: %s", plugin.ID, string(output))
	} else {
		log.Printf("[Plugin %s] Import output (truncated): %s...", plugin.ID, string(output[:2000]))
	}

	// Parse output
	var result PluginImportResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse plugin output: %v, raw (first 500): %s", err, string(output[:min(500, len(output))]))
	}

	log.Printf("[Plugin %s] Import result: images=%d, categories=%d, annotations=%d, errors=%d",
		plugin.ID, len(result.Images), len(result.Categories), len(result.Annotations), len(result.Errors))

	// Check for errors
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			log.Printf("[Plugin %s] Import error: %s - %s", plugin.ID, e.Code, e.Message)
		}
		return nil, fmt.Errorf("plugin error: %s - %s", result.Errors[0].Code, result.Errors[0].Message)
	}

	return &result, nil
}

// ==================== Dataset Version Management ====================

// DatasetVersion 鏁版嵁闆嗙増鏈俊鎭?
type DatasetVersion struct {
	Version           int    `json:"version"`
	CreatedAt         string `json:"createdAt"`
	Note              string `json:"note"`
	ImageCount        int    `json:"imageCount"`
	LabeledImageCount int    `json:"labeledImageCount"` // 宸叉爣娉ㄧ殑鍥剧墖鏁?
	CategoryCount     int    `json:"categoryCount"`
	AnnotationCount   int    `json:"annotationCount"`
}

// DatasetVersionMeta 鐗堟湰鍏冩暟鎹紙瀛樺偍鍦?version_meta.json锛?
type DatasetVersionMeta struct {
	Version           int    `json:"version"`
	CreatedAt         string `json:"createdAt"`
	Note              string `json:"note"`
	ImageCount        int    `json:"imageCount"`
	LabeledImageCount int    `json:"labeledImageCount"` // 宸叉爣娉ㄧ殑鍥剧墖鏁?
	CategoryCount     int    `json:"categoryCount"`
	AnnotationCount   int    `json:"annotationCount"`
}

// handleDatasetVersions 鑾峰彇椤圭洰鐨勬墍鏈夌増鏈垪琛?
func handleDatasetVersions(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	projectID := r.URL.Query().Get("projectId")
	if projectID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing_project_id"})
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	versionsDir := filepath.Join(cfg.DataPath, "project_item", projectID, "db", "versions")

	// Check if versions directory exists
	if _, err := os.Stat(versionsDir); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"versions": []DatasetVersion{}})
		return
	}

	// Read all version directories
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"versions": []DatasetVersion{}})
		return
	}

	versions := []DatasetVersion{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Parse version number from directory name (v1, v2, etc.)
		name := entry.Name()
		if len(name) < 2 || name[0] != 'v' {
			continue
		}
		versionNum, err := strconv.Atoi(name[1:])
		if err != nil {
			continue
		}

		// Read version metadata
		metaPath := filepath.Join(versionsDir, name, "version_meta.json")
		metaData, err := os.ReadFile(metaPath)
		if err != nil {
			// If no meta file, create basic info from directory
			info, _ := entry.Info()
			versions = append(versions, DatasetVersion{
				Version:   versionNum,
				CreatedAt: info.ModTime().Format("2006-01-02 15:04:05"),
			})
			continue
		}

		var meta DatasetVersionMeta
		if err := json.Unmarshal(metaData, &meta); err != nil {
			continue
		}
		versions = append(versions, DatasetVersion{
			Version:         meta.Version,
			CreatedAt:       meta.CreatedAt,
			Note:            meta.Note,
			ImageCount:      meta.ImageCount,
			CategoryCount:   meta.CategoryCount,
			AnnotationCount: meta.AnnotationCount,
		})
	}

	// Sort by version number descending
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"versions": versions})
}

// handleCreateDatasetVersion 鍒涘缓鏂扮増鏈紙蹇収锛?
func handleCreateDatasetVersion(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProjectID string `json:"projectId"`
		Note      string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if req.ProjectID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing_project_id"})
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	projectRoot := filepath.Join(cfg.DataPath, "project_item", req.ProjectID)
	dbPath := filepath.Join(projectRoot, "db", "project.db")
	versionsDir := filepath.Join(projectRoot, "db", "versions")

	// Check if source database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "project_not_found"})
		return
	}

	// 鈽?閲嶈锛氬湪澶嶅埗鍓嶆墽琛?checkpoint锛岀‘淇?WAL 涓殑鏁版嵁鍐欏叆涓绘暟鎹簱鏂囦欢
	srcDb, err := openProjectDB(dbPath)
	if err == nil {
		srcDb.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
		srcDb.Close()
	}

	// Ensure versions directory exists
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "create_dir_failed"})
		return
	}

	// Find next version number
	nextVersion := 1
	entries, _ := os.ReadDir(versionsDir)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) >= 2 && name[0] == 'v' {
			if v, err := strconv.Atoi(name[1:]); err == nil && v >= nextVersion {
				nextVersion = v + 1
			}
		}
	}

	// Create version directory
	versionDir := filepath.Join(versionsDir, fmt.Sprintf("v%d", nextVersion))
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "create_version_dir_failed"})
		return
	}

	// Copy database file
	destDbPath := filepath.Join(versionDir, "project.db")
	srcFile, err := os.Open(dbPath)
	if err != nil {
		os.RemoveAll(versionDir)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "open_db_failed"})
		return
	}
	defer srcFile.Close()

	destFile, err := os.Create(destDbPath)
	if err != nil {
		os.RemoveAll(versionDir)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "create_snapshot_failed"})
		return
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		os.RemoveAll(versionDir)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "copy_db_failed"})
		return
	}

	// Get statistics from the copied database
	db, err := openProjectDB(destDbPath)
	if err != nil {
		os.RemoveAll(versionDir)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "open_snapshot_db_failed"})
		return
	}
	defer db.Close()

	var imageCount, labeledImageCount, categoryCount, annotationCount int
	db.QueryRow("SELECT COUNT(*) FROM image_index").Scan(&imageCount)
	db.QueryRow("SELECT COUNT(DISTINCT image_key) FROM annotations").Scan(&labeledImageCount)
	db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&categoryCount)
	db.QueryRow("SELECT COUNT(*) FROM annotations").Scan(&annotationCount)

	// Create metadata file
	now := time.Now().Format("2006-01-02 15:04:05")
	meta := DatasetVersionMeta{
		Version:           nextVersion,
		CreatedAt:         now,
		Note:              req.Note,
		ImageCount:        imageCount,
		LabeledImageCount: labeledImageCount,
		CategoryCount:     categoryCount,
		AnnotationCount:   annotationCount,
	}
	metaData, _ := json.MarshalIndent(meta, "", "  ")
	metaPath := filepath.Join(versionDir, "version_meta.json")
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		log.Printf("Warning: failed to write version metadata: %v", err)
	}

	log.Printf("[DatasetVersion] Created version v%d for project %s: %d images, %d categories, %d annotations",
		nextVersion, req.ProjectID, imageCount, categoryCount, annotationCount)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"version":           nextVersion,
		"createdAt":         now,
		"imageCount":        imageCount,
		"labeledImageCount": labeledImageCount,
		"categoryCount":     categoryCount,
		"annotationCount":   annotationCount,
	})
}

// handleDeleteDatasetVersion 鍒犻櫎鐗堟湰
func handleDeleteDatasetVersion(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProjectID string `json:"projectId"`
		Version   int    `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if req.ProjectID == "" || req.Version <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_params"})
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	projectRoot := filepath.Join(cfg.DataPath, "project_item", req.ProjectID)
	versionDir := filepath.Join(projectRoot, "db", "versions", fmt.Sprintf("v%d", req.Version))

	// Check if version exists
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "version_not_found"})
		return
	}

	// Delete version directory
	if err := os.RemoveAll(versionDir); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "delete_failed"})
		return
	}

	log.Printf("[DatasetVersion] Deleted version v%d for project %s", req.Version, req.ProjectID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// handleUpdateDatasetVersion 鏇存柊鐗堟湰澶囨敞
func handleUpdateDatasetVersion(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProjectID string `json:"projectId"`
		Version   int    `json:"version"`
		Note      string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if req.ProjectID == "" || req.Version <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_params"})
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	metaPath := filepath.Join(cfg.DataPath, "project_item", req.ProjectID, "db", "versions", fmt.Sprintf("v%d", req.Version), "version_meta.json")

	// Read existing metadata
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "version_not_found"})
		return
	}

	var meta DatasetVersionMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "parse_meta_failed"})
		return
	}

	// Update note
	meta.Note = req.Note
	updatedData, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(metaPath, updatedData, 0644); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "write_meta_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// handleRollbackDatasetVersion 浠庣増鏈洖婧埌褰撳墠椤圭洰
func handleRollbackDatasetVersion(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProjectID string `json:"projectId"`
		Version   int    `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if req.ProjectID == "" || req.Version <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_params"})
		return
	}

	// Check I/O lock
	if !tryAcquireIOLock("rollback_" + req.ProjectID) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "io_task_busy"})
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		releaseIOLock("rollback_" + req.ProjectID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	projectRoot := filepath.Join(cfg.DataPath, "project_item", req.ProjectID)
	currentDbPath := filepath.Join(projectRoot, "db", "project.db")
	versionDbPath := filepath.Join(projectRoot, "db", "versions", fmt.Sprintf("v%d", req.Version), "project.db")

	// Check if version exists
	if _, err := os.Stat(versionDbPath); os.IsNotExist(err) {
		releaseIOLock("rollback_" + req.ProjectID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "version_not_found"})
		return
	}

	// 打开版本数据库和当前数据库
	versionDb, err := openProjectDB(versionDbPath)
	if err != nil {
		releaseIOLock("rollback_" + req.ProjectID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "open_version_db_failed"})
		return
	}
	defer versionDb.Close()

	currentDb, err := openProjectDB(currentDbPath)
	if err != nil {
		releaseIOLock("rollback_" + req.ProjectID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "open_current_db_failed"})
		return
	}
	defer currentDb.Close()

	// 开始事务
	tx, err := currentDb.Begin()
	if err != nil {
		releaseIOLock("rollback_" + req.ProjectID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "tx_begin_failed"})
		return
	}

	// 清空当前数据库的索引表、类别表、标注表（不清空引用次数表）
	tx.Exec(`DELETE FROM annotations`)
	tx.Exec(`DELETE FROM categories`)
	tx.Exec(`DELETE FROM image_index`)

	// 从版本数据库复制 image_index 表
	rows, err := versionDb.Query(`SELECT id, filename, original_rel_path, thumb_rel_path, deleted_in_project, annotation_status, created_at FROM image_index`)
	if err == nil {
		for rows.Next() {
			var id int64
			var filename, originalRel, thumbRel, annotationStatus, createdAt string
			var deletedInProject int
			if rows.Scan(&id, &filename, &originalRel, &thumbRel, &deletedInProject, &annotationStatus, &createdAt) == nil {
				tx.Exec(`INSERT INTO image_index (id, filename, original_rel_path, thumb_rel_path, deleted_in_project, annotation_status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					id, filename, originalRel, thumbRel, deletedInProject, annotationStatus, createdAt)
			}
		}
		rows.Close()
	}

	// 从版本数据库复制 categories 表
	rows, err = versionDb.Query(`SELECT id, name, type, color, sort_order, mate FROM categories`)
	if err == nil {
		for rows.Next() {
			var id int64
			var sortOrder int
			var name, catType, color, mate string
			if rows.Scan(&id, &name, &catType, &color, &sortOrder, &mate) == nil {
				tx.Exec(`INSERT INTO categories (id, name, type, color, sort_order, mate) VALUES (?, ?, ?, ?, ?, ?)`,
					id, name, catType, color, sortOrder, mate)
			}
		}
		rows.Close()
	}

	// 从版本数据库复制 annotations 表
	rows, err = versionDb.Query(`SELECT id, image_id, category_id, type, data, created_at, updated_at FROM annotations`)
	if err == nil {
		for rows.Next() {
			var id, imageID, categoryID int64
			var annType, data, createdAt, updatedAt string
			if rows.Scan(&id, &imageID, &categoryID, &annType, &data, &createdAt, &updatedAt) == nil {
				tx.Exec(`INSERT INTO annotations (id, image_id, category_id, type, data, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					id, imageID, categoryID, annType, data, createdAt, updatedAt)
			}
		}
		rows.Close()
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		releaseIOLock("rollback_" + req.ProjectID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "rollback_commit_failed"})
		return
	}

	releaseIOLock("rollback_" + req.ProjectID)

	log.Printf("[DatasetVersion] Rolled back project %s to version v%d (no backup created)",
		req.ProjectID, req.Version)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// ==================== 鏁版嵁闆嗗鍑?====================

// handleExportPlugins 鑾峰彇鏀寔瀵煎嚭鐨勬彃浠跺垪琛?
func handleExportPlugins(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	plugins, err := loadInstalledPlugins()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "load_plugins_failed"})
		return
	}

	// 杩囨护鍑烘敮鎸佸鍑虹殑鎻掍欢
	type ExportPluginInfo struct {
		ID      string   `json:"id"`
		Name    string   `json:"name"`
		Formats []string `json:"formats"`
	}
	var exportPlugins []ExportPluginInfo

	for _, p := range plugins {
		if !pluginSupportsExport(p) {
			continue
		}
		formats := []string{}
		if caps, ok := p.Capabilities["exportFormats"]; ok {
			if arr, ok := caps.([]interface{}); ok {
				for _, f := range arr {
					if s, ok := f.(string); ok {
						formats = append(formats, s)
					}
				}
			}
		}
		if len(formats) > 0 {
			exportPlugins = append(exportPlugins, ExportPluginInfo{
				ID:      p.ID,
				Name:    getPluginNameString(p.Name),
				Formats: formats,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"plugins": exportPlugins})
}

// handleSettingsPaths 杩斿洖閰嶇疆璺緞淇℃伅
func handleSettingsPaths(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"dataPath": cfg.DataPath})
}

// ==================== 导出任务管理 ====================

type ExportTask struct {
	ID         string `json:"id"`
	Phase      string `json:"phase"`     // preparing, exporting, copying, completed, failed
	Progress   int    `json:"progress"`  // 0-100
	Total      int    `json:"total"`     // 总数（图片数）
	Processed  int    `json:"processed"` // 已处理数
	OutputPath string `json:"outputPath"`
	Error      string `json:"error,omitempty"`
}

var (
	exportTasks   = make(map[string]*ExportTask)
	exportTasksMu sync.Mutex
)

// handleExportStatus 查询导出任务状态
func handleExportStatus(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	taskID := r.URL.Query().Get("taskId")
	if taskID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing_task_id"})
		return
	}

	exportTasksMu.Lock()
	task, ok := exportTasks[taskID]
	exportTasksMu.Unlock()

	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "task_not_found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// ExportDatasetRequest 瀵煎嚭璇锋眰
type ExportDatasetRequest struct {
	Categories []struct {
		ProjectID    string `json:"projectId"`
		Version      int    `json:"version"`
		CategoryID   int    `json:"categoryId"`
		CategoryName string `json:"categoryName"`
	} `json:"categories"`
	OutputPath string `json:"outputPath"`
	Format     string `json:"format"`
	PluginID   string `json:"pluginId"`
	Split      struct {
		Train int `json:"train"`
		Val   int `json:"val"`
		Test  int `json:"test"`
	} `json:"split"`
}

// handleDatasetExport 处理数据集导出（异步）
func handleDatasetExport(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Export] Received request: method=%s", r.Method)
	withCORS(w)
	if r.Method == http.MethodOptions {
		log.Printf("[Export] OPTIONS preflight")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		log.Printf("[Export] Invalid method: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req ExportDatasetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Export] ERROR: Failed to decode request: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
		return
	}
	log.Printf("[Export] Request decoded: categories=%d, format=%s, plugin=%s", len(req.Categories), req.Format, req.PluginID)

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	// 查找插件
	plugins, _ := loadInstalledPlugins()
	var pluginPath string
	var pluginEntry string
	for _, p := range plugins {
		if p.ID == req.PluginID {
			pluginPath = p.Path
			pluginEntry = p.Entry
			break
		}
	}
	if pluginPath == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "plugin_not_found"})
		return
	}

	// 创建导出任务
	taskID := generateUUID()
	task := &ExportTask{
		ID:         taskID,
		Phase:      "preparing",
		Progress:   0,
		OutputPath: req.OutputPath,
	}
	exportTasksMu.Lock()
	exportTasks[taskID] = task
	exportTasksMu.Unlock()

	// 立即返回任务 ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"taskId": taskID})

	// 异步执行导出
	go runExportTask(taskID, req, cfg, pluginPath, pluginEntry)
}

// runExportTask 执行导出任务
func runExportTask(taskID string, req ExportDatasetRequest, cfg *PathsConfig, pluginPath, pluginEntry string) {
	updateExportTask := func(phase string, progress int, total int, processed int, err string) {
		exportTasksMu.Lock()
		if task, ok := exportTasks[taskID]; ok {
			task.Phase = phase
			task.Progress = progress
			task.Total = total
			task.Processed = processed
			if err != "" {
				task.Error = err
			}
		}
		exportTasksMu.Unlock()
	}

	log.Printf("[Export] Task %s starting: format=%s, plugin=%s, output=%s", taskID, req.Format, pluginPath, req.OutputPath)
	log.Printf("[Export] Task %s categories to export: %d", taskID, len(req.Categories))

	// 鏀堕泦鎵€鏈夊浘鐗囧拰鏍囨敞
	type ExportImage struct {
		Key          string `json:"key"`
		RelativePath string `json:"relativePath"`
		AbsolutePath string `json:"absolutePath"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		Split        string `json:"split"`
	}
	type ExportCategory struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Type  string `json:"type"`
		Color string `json:"color"`
		Mate  string `json:"mate,omitempty"` // 鍏抽敭鐐归厤缃瓑鍏冩暟鎹?
	}
	type ExportAnnotation struct {
		ID         int                    `json:"id,omitempty"` // 鏍囨敞ID锛岀敤浜庡叧閿偣寮曠敤鐭╁舰妗?
		ImageKey   string                 `json:"imageKey"`
		CategoryID int                    `json:"categoryId"`
		Type       string                 `json:"type"`
		Data       map[string]interface{} `json:"data"`
	}

	var images []ExportImage
	var categories []ExportCategory
	var annotations []ExportAnnotation
	imageSet := make(map[string]bool) // 鍘婚噸

	// 閬嶅巻閫変腑鐨勭被鍒紝鍔犺浇鏁版嵁
	for _, cat := range req.Categories {
		log.Printf("[Export] Processing category: projectId=%s, version=%d, categoryId=%d, name=%s",
			cat.ProjectID, cat.Version, cat.CategoryID, cat.CategoryName)

		// 鍔犺浇鐗堟湰鏁版嵁搴?
		versionDbPath := filepath.Join(cfg.DataPath, "project_item", cat.ProjectID, "db", "versions",
			fmt.Sprintf("v%d", cat.Version), "project.db")

		log.Printf("[Export] Loading database: %s", versionDbPath)

		if _, err := os.Stat(versionDbPath); os.IsNotExist(err) {
			log.Printf("[Export] WARNING: Database not found: %s", versionDbPath)
			continue
		}

		db, err := openProjectDB(versionDbPath)
		if err != nil {
			log.Printf("[Export] ERROR: Failed to open database: %v", err)
			continue
		}

		// 鑾峰彇绫诲埆淇℃伅
		var catType, catColor string
		var catMate sql.NullString
		err = db.QueryRow("SELECT type, color, mate FROM categories WHERE id = ?", cat.CategoryID).Scan(&catType, &catColor, &catMate)
		if err != nil {
			log.Printf("[Export] WARNING: Category not found in db: %v", err)
			db.Close()
			continue
		}

		categories = append(categories, ExportCategory{
			ID:    cat.CategoryID,
			Name:  cat.CategoryName,
			Type:  catType,
			Color: catColor,
			Mate:  catMate.String,
		})
		log.Printf("[Export] Found category: id=%d, type=%s, mate=%s", cat.CategoryID, catType, catMate.String)

		// 鑾峰彇璇ョ被鍒殑鎵€鏈夋爣娉?
		rows, err := db.Query(`
			SELECT a.id, a.image_id, a.type, a.data, i.original_rel_path
			FROM annotations a
			JOIN image_index i ON a.image_id = i.id
			WHERE a.category_id = ?
		`, cat.CategoryID)
		if err != nil {
			log.Printf("[Export] ERROR: Failed to query annotations: %v", err)
			db.Close()
			continue
		}

		annCount := 0
		for rows.Next() {
			var annID, imageID int
			var annType, dataJSON, relativePath string
			if err := rows.Scan(&annID, &imageID, &annType, &dataJSON, &relativePath); err != nil {
				log.Printf("[Export] WARNING: Failed to scan row: %v", err)
				continue
			}

			imageKey := fmt.Sprintf("%s-v%d-%d", cat.ProjectID, cat.Version, imageID)

			// 娣诲姞鍥剧墖锛堝幓閲嶏級
			if !imageSet[imageKey] {
				imageSet[imageKey] = true
				absPath := filepath.Join(cfg.DataPath, "project_item", cat.ProjectID, relativePath)

				// 璇诲彇鍥剧墖灏哄
				imgWidth, imgHeight := 0, 0
				if f, err := os.Open(absPath); err == nil {
					if imgCfg, _, err := image.DecodeConfig(f); err == nil {
						imgWidth, imgHeight = imgCfg.Width, imgCfg.Height
					}
					f.Close()
				}

				images = append(images, ExportImage{
					Key:          imageKey,
					RelativePath: relativePath,
					AbsolutePath: absPath,
					Width:        imgWidth,
					Height:       imgHeight,
				})
			}

			// 瑙ｆ瀽鏍囨敞鏁版嵁
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(dataJSON), &data); err == nil {
				annotations = append(annotations, ExportAnnotation{
					ID:         annID,
					ImageKey:   imageKey,
					CategoryID: cat.CategoryID,
					Type:       annType, // 浣跨敤瀹為檯鏍囨敞绫诲瀷
					Data:       data,
				})
				annCount++
			}
		}
		rows.Close()
		db.Close()

		log.Printf("[Export] Loaded %d annotations for category %s", annCount, cat.CategoryName)
	}

	log.Printf("[Export] Total: %d images, %d categories, %d annotations", len(images), len(categories), len(annotations))

	if len(images) == 0 {
		log.Printf("[Export] Task %s ERROR: No images to export", taskID)
		updateExportTask("failed", 0, 0, 0, "no_images_to_export")
		return
	}

	// 更新任务状态：准备中
	updateExportTask("preparing", 10, len(images), 0, "")

	// 鎸?split 姣斾緥鍒嗛厤鍥剧墖
	totalImages := len(images)
	trainCount := totalImages * req.Split.Train / 100
	valCount := totalImages * req.Split.Val / 100
	testCount := totalImages - trainCount - valCount

	log.Printf("[Export] Split: train=%d, val=%d, test=%d", trainCount, valCount, testCount)

	for i := range images {
		if i < trainCount {
			images[i].Split = "train"
		} else if i < trainCount+valCount {
			images[i].Split = "val"
		} else {
			images[i].Split = "test"
		}
	}

	// 鍑嗗鎻掍欢杈撳叆
	pluginInput := map[string]interface{}{
		"format":      req.Format,
		"outputDir":   req.OutputPath,
		"images":      images,
		"categories":  categories,
		"annotations": annotations,
		"split": map[string]int{
			"train": req.Split.Train,
			"val":   req.Split.Val,
			"test":  req.Split.Test,
		},
	}

	inputJSON, _ := json.Marshal(pluginInput)
	log.Printf("[Export] Plugin input prepared, size=%d bytes", len(inputJSON))

	// 璋冪敤鎻掍欢
	entryPath := filepath.Join(pluginPath, pluginEntry)
	if runtime.GOOS == "windows" {
		entryPath += ".exe"
	}

	log.Printf("[Export] Calling plugin: %s export", entryPath)

	cmd := exec.Command(entryPath, "export")
	cmd.Stdin = strings.NewReader(string(inputJSON))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 更新任务状态：调用插件中
	updateExportTask("exporting", 20, len(images), 0, "")

	if err := cmd.Run(); err != nil {
		log.Printf("[Export] Task %s ERROR: Plugin execution failed: %v", taskID, err)
		log.Printf("[Export] Task %s Plugin stderr: %s", taskID, stderr.String())
		updateExportTask("failed", 20, len(images), 0, "plugin_execution_failed")
		return
	}

	log.Printf("[Export] Task %s Plugin stdout length: %d", taskID, len(stdout.String()))

	// 解析插件响应
	var pluginResp struct {
		Success   bool `json:"success"`
		Structure struct {
			Directories []string `json:"directories"`
			Files       []struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			} `json:"files"`
			CopyImages []struct {
				From string `json:"from"`
				To   string `json:"to"`
			} `json:"copyImages"`
		} `json:"structure"`
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &pluginResp); err != nil {
		log.Printf("[Export] Task %s ERROR: Failed to parse plugin response: %v", taskID, err)
		updateExportTask("failed", 20, len(images), 0, "invalid_plugin_response")
		return
	}

	if !pluginResp.Success {
		errMsg := "unknown error"
		if len(pluginResp.Errors) > 0 {
			errMsg = pluginResp.Errors[0].Message
		}
		log.Printf("[Export] Task %s ERROR: Plugin returned error: %s", taskID, errMsg)
		updateExportTask("failed", 20, len(images), 0, errMsg)
		return
	}

	// 创建目录
	for _, dir := range pluginResp.Structure.Directories {
		dirPath := filepath.Join(req.OutputPath, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			log.Printf("[Export] Task %s WARNING: Failed to create directory: %v", taskID, err)
		}
	}

	// 写入文件
	for _, file := range pluginResp.Structure.Files {
		filePath := filepath.Join(req.OutputPath, file.Path)
		if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
			log.Printf("[Export] Task %s WARNING: Failed to write file: %v", taskID, err)
		}
	}

	// 更新任务状态：复制图片中
	totalCopyImages := len(pluginResp.Structure.CopyImages)
	updateExportTask("copying", 30, totalCopyImages, 0, "")

	// 复制图片并更新进度
	copiedCount := 0
	for _, cp := range pluginResp.Structure.CopyImages {
		destPath := filepath.Join(req.OutputPath, cp.To)
		if err := copyFile(cp.From, destPath); err != nil {
			log.Printf("[Export] Task %s WARNING: Failed to copy image %s -> %s: %v", taskID, cp.From, destPath, err)
		} else {
			copiedCount++
		}
		// 更新进度：30% + 70% * (copiedCount / totalCopyImages)
		if totalCopyImages > 0 {
			progress := 30 + (70 * copiedCount / totalCopyImages)
			updateExportTask("copying", progress, totalCopyImages, copiedCount, "")
		}
	}

	log.Printf("[Export] Task %s Completed: %d directories, %d files, %d images copied",
		taskID, len(pluginResp.Structure.Directories), len(pluginResp.Structure.Files), copiedCount)

	// 完成
	updateExportTask("completed", 100, totalCopyImages, copiedCount, "")
}

// handleOpenFolder 鎵撳紑鏂囦欢澶?
func handleOpenFolder(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_path"})
		return
	}

	// 妫€鏌ヨ矾寰勬槸鍚﹀瓨鍦?
	if _, err := os.Stat(req.Path); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "path_not_found"})
		return
	}

	// Windows: 浣跨敤 explorer 鎵撳紑
	cmd := exec.Command("explorer", req.Path)
	if err := cmd.Start(); err != nil {
		log.Printf("[Shell] Failed to open folder: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "open_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}

// ============ 绯荤粺璧勬簮鐩戞帶 ============

// getSystemStats 鑾峰彇绯荤粺璧勬簮鐘舵€?
func getSystemStats() map[string]interface{} {
	// 鑾峰彇CPU鏍稿績鏁?
	cpuCores := runtime.NumCPU()

	// 鑾峰彇CPU浣跨敤鐜?
	cpuUsage := 0.0
	cmd := exec.Command("wmic", "cpu", "get", "loadpercentage")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && line != "LoadPercentage" {
				if val, err := strconv.ParseFloat(line, 64); err == nil {
					cpuUsage = val
					break
				}
			}
		}
	}

	// 鑾峰彇鍐呭瓨浣跨敤鎯呭喌
	memUsage := 0.0
	memUsed := 0.0
	memTotal := 0.0
	memCmd := exec.Command("wmic", "OS", "get", "FreePhysicalMemory,TotalVisibleMemorySize", "/Value")
	memOutput, err := memCmd.Output()
	if err == nil {
		lines := strings.Split(string(memOutput), "\n")
		var freeKB, totalKB float64
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "FreePhysicalMemory=") {
				if val, err := strconv.ParseFloat(strings.TrimPrefix(line, "FreePhysicalMemory="), 64); err == nil {
					freeKB = val
				}
			} else if strings.HasPrefix(line, "TotalVisibleMemorySize=") {
				if val, err := strconv.ParseFloat(strings.TrimPrefix(line, "TotalVisibleMemorySize="), 64); err == nil {
					totalKB = val
				}
			}
		}
		if totalKB > 0 {
			memTotal = totalKB / 1024 / 1024
			memUsed = (totalKB - freeKB) / 1024 / 1024
			memUsage = (totalKB - freeKB) / totalKB * 100
		}
	}

	// 妫€鏌PU
	gpuAvailable := false
	gpuUsage := 0.0
	vramUsed := 0.0
	vramTotal := 0.0
	gpuCmd := exec.Command("nvidia-smi", "--query-gpu=utilization.gpu,memory.used,memory.total", "--format=csv,noheader,nounits")
	gpuOutput, err := gpuCmd.Output()
	if err == nil {
		gpuAvailable = true
		parts := strings.Split(strings.TrimSpace(string(gpuOutput)), ",")
		if len(parts) >= 3 {
			if val, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err == nil {
				gpuUsage = val
			}
			if val, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
				vramUsed = val / 1024
			}
			if val, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64); err == nil {
				vramTotal = val / 1024
			}
		}
	}

	return map[string]interface{}{
		"cpuUsage":     cpuUsage,
		"cpuCores":     cpuCores,
		"memUsage":     math.Round(memUsage*10) / 10,
		"memUsed":      math.Round(memUsed*10) / 10,
		"memTotal":     math.Round(memTotal*10) / 10,
		"gpuAvailable": gpuAvailable,
		"gpuUsage":     gpuUsage,
		"vramUsed":     math.Round(vramUsed*100) / 100,
		"vramTotal":    math.Round(vramTotal*100) / 100,
	}
}

// handleSystemStats 鑾峰彇绯荤粺璧勬簮鐘舵€?
func handleSystemStats(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(getSystemStats())
}

// ============ Python 鐜绠＄悊 ============

// getPythonVenvPath 鑾峰彇铏氭嫙鐜璺緞
func getPythonVenvPath() string {
	dataPath := getDataPath()
	return filepath.Join(dataPath, "EasyMark_python_venv")
}

// isPythonVenvDeployed 妫€鏌ヨ櫄鎷熺幆澧冩槸鍚﹀凡閮ㄧ讲
func isPythonVenvDeployed() bool {
	venvPath := getPythonVenvPath()
	pythonExe := filepath.Join(venvPath, "Scripts", "python.exe")
	_, err := os.Stat(pythonExe)
	return err == nil
}

// getSystemPythonVersion 鑾峰彇绯荤粺Python鐗堟湰
func getSystemPythonVersion() string {
	cmd := exec.Command("python", "--version")
	output, err := cmd.Output()
	if err != nil {
		// 灏濊瘯 python3
		cmd = exec.Command("python3", "--version")
		output, err = cmd.Output()
		if err != nil {
			return ""
		}
	}
	// Python 3.10.11
	parts := strings.Split(strings.TrimSpace(string(output)), " ")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// handlePythonStatus 鑾峰彇Python鐜鐘舵€?
func handlePythonStatus(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	deployed := isPythonVenvDeployed()
	pythonVersion := getSystemPythonVersion()
	venvPath := ""
	if deployed {
		venvPath = getPythonVenvPath()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"deployed":      deployed,
		"pythonVersion": pythonVersion,
		"venvPath":      venvPath,
	})
}

// handlePythonDeploy 閮ㄧ讲Python铏氭嫙鐜
func handlePythonDeploy(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	venvPath := getPythonVenvPath()

	// 妫€鏌ユ槸鍚﹀凡閮ㄧ讲
	if isPythonVenvDeployed() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"venvPath": venvPath,
			"message":  "already_deployed",
		})
		return
	}

	// 鍒涘缓铏氭嫙鐜
	log.Printf("[Python] Creating virtual environment at: %s", venvPath)
	cmd := exec.Command("python", "-m", "venv", venvPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[Python] Failed to create venv: %v, output: %s", err, string(output))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "venv_creation_failed",
			"details": string(output),
		})
		return
	}

	log.Printf("[Python] Virtual environment created successfully")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"venvPath": venvPath,
		"output":   string(output),
	})
}

// handlePythonInstall 瀹夎Python鍖?(浣跨敤ConPTY瀹炵幇鐪熸鐨勭粓绔晥鏋?
func handlePythonInstall(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Package  string `json:"package"`
		Mirror   string `json:"mirror"`
		IndexUrl string `json:"indexUrl"` // 涓撶敤婧愶紙濡?PyTorch CUDA wheel锛?
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Package == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_package"})
		return
	}

	if !isPythonVenvDeployed() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "venv_not_deployed"})
		return
	}

	// 璁剧疆SSE鍝嶅簲澶?
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	venvPath := getPythonVenvPath()
	pipExe := filepath.Join(venvPath, "Scripts", "pip.exe")

	// 鏋勫缓瀹屾暣鍛戒护琛?
	args := []string{pipExe, "install"}
	if req.IndexUrl != "" {
		// 浣跨敤涓撶敤婧愶紙濡?PyTorch CUDA wheel锛?
		args = append(args, "--index-url", req.IndexUrl)
	} else if req.Mirror != "" {
		// 浣跨敤闀滃儚婧?
		args = append(args, "-i", req.Mirror)
	}
	packages := strings.Fields(req.Package)
	args = append(args, packages...)
	cmdLine := strings.Join(args, " ")

	log.Printf("[Python] Installing with ConPTY: %s", cmdLine)

	// 浣跨敤 ConPTY 鍒涘缓浼粓绔?(80鍒?x 25琛?
	cpty, err := conpty.Start(cmdLine, conpty.ConPtyDimensions(120, 30))
	if err != nil {
		log.Printf("[Python] ConPTY failed, falling back to normal exec: %v", err)
		// 鍥為€€鍒版櫘閫氭墽琛?
		handlePythonInstallFallback(w, flusher, pipExe, req.Package, req.Mirror, req.IndexUrl)
		return
	}
	defer cpty.Close()

	// 鍙戦€佽緭鍑虹殑杈呭姪鍑芥暟
	sendOutput := func(data []byte) {
		if len(data) == 0 {
			return
		}
		msg := map[string]interface{}{
			"type":    "output",
			"message": string(data),
		}
		jsonData, _ := json.Marshal(msg)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}

	// 浣跨敤 goroutine 璇诲彇杈撳嚭锛屼富绾跨▼绛夊緟杩涚▼缁撴潫
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := cpty.Read(buf)
			if n > 0 {
				sendOutput(buf[:n])
			}
			if err != nil {
				break
			}
		}
		close(done)
	}()

	// 绛夊緟杩涚▼缁撴潫锛堝甫瓒呮椂锛?
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	exitCode, err := cpty.Wait(ctx)

	// 绛夊緟璇诲彇 goroutine 缁撴潫锛堟渶澶氱瓑 2 绉掞級
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}

	if err != nil || exitCode != 0 {
		log.Printf("[Python] Install failed with exit code: %d, err: %v", exitCode, err)
		fmt.Fprintf(w, "data: {\"type\":\"done\",\"success\":false,\"error\":\"install_failed\"}\n\n")
	} else {
		log.Printf("[Python] Package installed successfully: %s", req.Package)
		fmt.Fprintf(w, "data: {\"type\":\"done\",\"success\":true}\n\n")
	}
	flusher.Flush()
}

// handlePythonInstallFallback 鍥為€€鏂规锛氫笉浣跨敤 ConPTY
func handlePythonInstallFallback(w http.ResponseWriter, flusher http.Flusher, pipExe, pkg, mirror, indexUrl string) {
	args := []string{"install"}
	if indexUrl != "" {
		args = append(args, "--index-url", indexUrl)
	} else if mirror != "" {
		args = append(args, "-i", mirror)
	}
	packages := strings.Fields(pkg)
	args = append(args, packages...)

	cmd := exec.Command(pipExe, args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(w, "data: {\"type\":\"error\",\"message\":\"%s\"}\n\n", err.Error())
		flusher.Flush()
		return
	}

	sendOutput := func(data []byte) {
		if len(data) == 0 {
			return
		}
		msg := map[string]interface{}{"type": "output", "message": string(data)}
		jsonData, _ := json.Marshal(msg)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				sendOutput(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	buf := make([]byte, 4096)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			sendOutput(buf[:n])
		}
		if err != nil {
			break
		}
	}

	err := cmd.Wait()
	if err != nil {
		fmt.Fprintf(w, "data: {\"type\":\"done\",\"success\":false,\"error\":\"install_failed\"}\n\n")
	} else {
		fmt.Fprintf(w, "data: {\"type\":\"done\",\"success\":true}\n\n")
	}
	flusher.Flush()
}

// handlePythonPackages 鑾峰彇宸插畨瑁呯殑Python鍖呭垪琛?
func handlePythonPackages(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !isPythonVenvDeployed() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"packages": []interface{}{},
		})
		return
	}

	venvPath := getPythonVenvPath()
	pipExe := filepath.Join(venvPath, "Scripts", "pip.exe")

	cmd := exec.Command(pipExe, "list", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("[Python] Failed to list packages: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"packages": []interface{}{},
		})
		return
	}

	var packages []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(output, &packages); err != nil {
		log.Printf("[Python] Failed to parse package list: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"packages": []interface{}{},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"packages": packages,
	})
}

// handlePythonUndeploy 鍗歌浇Python铏氭嫙鐜
func handlePythonUndeploy(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !isPythonVenvDeployed() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "not_deployed",
		})
		return
	}

	venvPath := getPythonVenvPath()
	log.Printf("[Python] Removing virtual environment at: %s", venvPath)

	// 鍒犻櫎铏氭嫙鐜鐩綍
	if err := os.RemoveAll(venvPath); err != nil {
		log.Printf("[Python] Failed to remove venv: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "remove_failed",
			"details": err.Error(),
		})
		return
	}

	log.Printf("[Python] Virtual environment removed successfully")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handlePythonUninstall 鍗歌浇Python鍖?
func handlePythonUninstall(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Package string `json:"package"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Package == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_package"})
		return
	}

	if !isPythonVenvDeployed() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "venv_not_deployed"})
		return
	}

	venvPath := getPythonVenvPath()
	pipExe := filepath.Join(venvPath, "Scripts", "pip.exe")

	log.Printf("[Python] Uninstalling package: %s", req.Package)
	cmd := exec.Command(pipExe, "uninstall", "-y", req.Package)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[Python] Failed to uninstall package: %v, output: %s", err, string(output))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "uninstall_failed",
			"details": string(output),
		})
		return
	}

	log.Printf("[Python] Package uninstalled successfully: %s", req.Package)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"output":  string(output),
	})
}

// ============ 璁粌闆嗙鐞?============

// Trainset 璁粌闆嗙粨鏋?
type Trainset struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Categories []TrainsetCategoryItem `json:"categories"`
	CreatedAt  string                 `json:"createdAt"`
	UpdatedAt  string                 `json:"updatedAt"`
}

type TrainsetCategoryItem struct {
	ProjectID       string `json:"projectId"`
	Version         int    `json:"version"`
	CategoryID      int    `json:"categoryId"`
	CategoryName    string `json:"categoryName"`
	CategoryType    string `json:"categoryType"`
	ImageCount      int    `json:"imageCount"`
	AnnotationCount int    `json:"annotationCount"`
}

// getTrainsetsDir 鑾峰彇璁粌闆嗗瓨鍌ㄧ洰褰?
func getTrainsetsDir() string {
	dataPath := getDataPath()
	return filepath.Join(dataPath, "train_data")
}

// handleTrainsets 鑾峰彇鎵€鏈夎缁冮泦
func handleTrainsets(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	dir := getTrainsetsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[Trainset] Failed to create directory: %v", err)
	}

	var trainsets []Trainset
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("[Trainset] Failed to read directory: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"trainsets": []Trainset{}})
		return
	}

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			continue
		}
		var ts Trainset
		if err := json.Unmarshal(data, &ts); err != nil {
			continue
		}
		trainsets = append(trainsets, ts)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"trainsets": trainsets})
}

// handleTrainsetSave 淇濆瓨璁粌闆?
func handleTrainsetSave(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID         string                 `json:"id"`
		Name       string                 `json:"name"`
		Categories []TrainsetCategoryItem `json:"categories"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || len(req.Categories) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
		return
	}

	dir := getTrainsetsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "create_dir_failed"})
		return
	}

	now := time.Now().Format(time.RFC3339)
	ts := Trainset{
		ID:         req.ID,
		Name:       req.Name,
		Categories: req.Categories,
		UpdatedAt:  now,
	}

	// 鏂板缓鎴栨洿鏂?
	if ts.ID == "" {
		ts.ID = fmt.Sprintf("ts_%d", time.Now().UnixNano())
		ts.CreatedAt = now
	} else {
		// 璇诲彇宸叉湁鐨勫垱寤烘椂闂?
		existingPath := filepath.Join(dir, ts.ID+".json")
		if data, err := os.ReadFile(existingPath); err == nil {
			var existing Trainset
			if json.Unmarshal(data, &existing) == nil {
				ts.CreatedAt = existing.CreatedAt
			}
		}
		if ts.CreatedAt == "" {
			ts.CreatedAt = now
		}
	}

	data, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "marshal_failed"})
		return
	}

	filePath := filepath.Join(dir, ts.ID+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "write_failed"})
		return
	}

	log.Printf("[Trainset] Saved trainset: %s (%s)", ts.Name, ts.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "trainset": ts})
}

// handleTrainsetDelete 鍒犻櫎璁粌闆?
func handleTrainsetDelete(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
		return
	}

	dir := getTrainsetsDir()
	filePath := filepath.Join(dir, req.ID+".json")

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			// 鏂囦欢涓嶅瓨鍦ㄤ篃绠楁垚鍔?
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "delete_failed"})
		return
	}

	log.Printf("[Trainset] Deleted trainset: %s", req.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// ============ 璁粌鎻掍欢涓庢ā鍨嬬鐞?============

// 鏍囧噯 YOLO 妯″瀷涓嬭浇閾炬帴
var standardModelURLs = map[string]string{
	// YOLOv8 Detection
	"yolov8n": "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8n.pt",
	"yolov8s": "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8s.pt",
	"yolov8m": "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8m.pt",
	"yolov8l": "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8l.pt",
	"yolov8x": "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8x.pt",
	// YOLOv8 Pose
	"yolov8n-pose": "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8n-pose.pt",
	"yolov8s-pose": "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8s-pose.pt",
	"yolov8m-pose": "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8m-pose.pt",
	"yolov8l-pose": "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8l-pose.pt",
	"yolov8x-pose": "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8x-pose.pt",
	// YOLOv8 Seg
	"yolov8n-seg": "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8n-seg.pt",
	"yolov8s-seg": "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8s-seg.pt",
	"yolov8m-seg": "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8m-seg.pt",
	"yolov8l-seg": "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8l-seg.pt",
	"yolov8x-seg": "https://github.com/ultralytics/assets/releases/download/v8.2.0/yolov8x-seg.pt",
	// YOLOv11 Detection (支持 yolov11 和 yolo11 两种格式)
	"yolo11n":  "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11n.pt",
	"yolo11s":  "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11s.pt",
	"yolo11m":  "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11m.pt",
	"yolo11l":  "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11l.pt",
	"yolo11x":  "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11x.pt",
	"yolov11n": "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11n.pt",
	"yolov11s": "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11s.pt",
	"yolov11m": "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11m.pt",
	"yolov11l": "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11l.pt",
	"yolov11x": "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11x.pt",
	// YOLOv11 Pose
	"yolo11n-pose":  "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11n-pose.pt",
	"yolo11s-pose":  "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11s-pose.pt",
	"yolo11m-pose":  "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11m-pose.pt",
	"yolo11l-pose":  "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11l-pose.pt",
	"yolo11x-pose":  "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11x-pose.pt",
	"yolov11n-pose": "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11n-pose.pt",
	"yolov11s-pose": "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11s-pose.pt",
	"yolov11m-pose": "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11m-pose.pt",
	"yolov11l-pose": "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11l-pose.pt",
	"yolov11x-pose": "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11x-pose.pt",
	// YOLOv11 Seg
	"yolo11n-seg":  "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11n-seg.pt",
	"yolo11s-seg":  "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11s-seg.pt",
	"yolo11m-seg":  "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11m-seg.pt",
	"yolo11l-seg":  "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11l-seg.pt",
	"yolo11x-seg":  "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11x-seg.pt",
	"yolov11n-seg": "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11n-seg.pt",
	"yolov11s-seg": "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11s-seg.pt",
	"yolov11m-seg": "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11m-seg.pt",
	"yolov11l-seg": "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11l-seg.pt",
	"yolov11x-seg": "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11x-seg.pt",
	// YOLOv5 Detection
	"yolov5n": "https://github.com/ultralytics/yolov5/releases/download/v7.0/yolov5n.pt",
	"yolov5s": "https://github.com/ultralytics/yolov5/releases/download/v7.0/yolov5s.pt",
	"yolov5m": "https://github.com/ultralytics/yolov5/releases/download/v7.0/yolov5m.pt",
	"yolov5l": "https://github.com/ultralytics/yolov5/releases/download/v7.0/yolov5l.pt",
	"yolov5x": "https://github.com/ultralytics/yolov5/releases/download/v7.0/yolov5x.pt",
}

// getModelsDir 鑾峰彇妯″瀷瀛樺偍鐩綍
func getModelsDir() string {
	dataPath := getDataPath()
	return filepath.Join(dataPath, "models")
}

// TrainingPluginManifest 训练插件清单
type TrainingPluginManifest struct {
	ID           string                 `json:"id"`
	Name         interface{}            `json:"name"` // 支持字符串或国际化对象
	Version      string                 `json:"version"`
	Type         interface{}            `json:"type"`        // 支持字符串或数组
	Description  interface{}            `json:"description"` // 支持字符串或国际化对象
	Author       string                 `json:"author"`
	Entry        string                 `json:"entry"`
	Capabilities map[string]interface{} `json:"capabilities"`
	Models       []PluginModel          `json:"models"`
	Python       *PluginPython          `json:"python"`
	ParamsSchema map[string]interface{} `json:"paramsSchema"`
	Training     map[string]interface{} `json:"training"`   // 训练配置
	PluginPath   string                 `json:"pluginPath"` // 运行时添加
}

type PluginModel struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	TrainTypes  []string `json:"trainTypes"`
	Variants    []string `json:"variants"`
	IsStandard  bool     `json:"isStandard"`
	DownloadURL string   `json:"downloadUrl,omitempty"`
}

type PluginPython struct {
	MinVersion   string         `json:"minVersion"`
	Requirements []string       `json:"requirements"`
	Pytorch      *PytorchConfig `json:"pytorch,omitempty"`
}

type PytorchConfig struct {
	Packages []string            `json:"packages"`
	Gpu      *PytorchIndexConfig `json:"gpu,omitempty"`
	Cpu      *PytorchIndexConfig `json:"cpu,omitempty"`
}

type PytorchIndexConfig struct {
	MinCuda  string `json:"minCuda,omitempty"`
	IndexUrl string `json:"indexUrl"`
}

// handleTrainingPlugins 鑾峰彇璁粌鎻掍欢鍒楄〃
func handleTrainingPlugins(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var plugins []TrainingPluginManifest

	// 只从已安装插件目录读取（DataPath/plugins），不使用开发目录
	installed, err := loadInstalledPlugins()
	if err != nil {
		log.Printf("[Training] loadInstalledPlugins error: %v", err)
	} else {
		for _, p := range installed {
			// 使用 pluginHasType 支持数组类型（如 ["training", "inference"]）
			if !pluginHasType(p.PluginManifest, "training") {
				continue
			}
			manifestPath := filepath.Join(p.Path, "manifest.json")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				log.Printf("[Training] Failed to read manifest from installed plugin %s: %v", p.ID, err)
				continue
			}
			var manifest TrainingPluginManifest
			if err := json.Unmarshal(data, &manifest); err != nil {
				log.Printf("[Training] Invalid training manifest in installed plugin %s: %v", p.ID, err)
				continue
			}
			manifest.PluginPath = p.Path
			plugins = append(plugins, manifest)
			log.Printf("[Training] Found installed training plugin: %s (%s)", manifest.Name, manifest.ID)
		}
	}

	log.Printf("[Training] Total training plugins found: %d", len(plugins))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"plugins": plugins})
}

// handleCheckPluginDeps 妫€鏌ユ彃浠朵緷璧?
func handleCheckPluginDeps(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Requirements []string `json:"requirements"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
		return
	}

	if !isPythonVenvDeployed() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deployed":   false,
			"missing":    req.Requirements,
			"installed":  []string{},
			"mismatched": []string{},
		})
		return
	}

	// 鑾峰彇宸插畨瑁呯殑鍖?
	venvPath := getPythonVenvPath()
	pipExe := filepath.Join(venvPath, "Scripts", "pip.exe")
	cmd := exec.Command(pipExe, "list", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deployed": true,
			"error":    "pip_list_failed",
		})
		return
	}

	var installedPkgs []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	json.Unmarshal(output, &installedPkgs)

	// 鍒涘缓宸插畨瑁呭寘鐨?map
	installedMap := make(map[string]string)
	for _, pkg := range installedPkgs {
		installedMap[strings.ToLower(pkg.Name)] = pkg.Version
	}

	var missing, installed, mismatched []string
	for _, reqStr := range req.Requirements {
		// 绠€鍗曡В鏋?requirement (濡?"torch>=2.0.0")
		reqStr = strings.TrimSpace(reqStr)
		pkgName := reqStr
		var requiredVersion string
		var op string

		for _, sep := range []string{">=", "<=", "==", ">", "<", "!="} {
			if idx := strings.Index(reqStr, sep); idx > 0 {
				pkgName = strings.TrimSpace(reqStr[:idx])
				requiredVersion = strings.TrimSpace(reqStr[idx+len(sep):])
				op = sep
				break
			}
		}

		pkgNameLower := strings.ToLower(strings.ReplaceAll(pkgName, "-", "_"))
		pkgNameLower2 := strings.ToLower(strings.ReplaceAll(pkgName, "_", "-"))

		installedVer, ok1 := installedMap[pkgNameLower]
		if !ok1 {
			installedVer, ok1 = installedMap[pkgNameLower2]
		}

		if !ok1 {
			missing = append(missing, reqStr)
		} else if requiredVersion != "" && op == ">=" {
			// 绠€鍗曠増鏈瘮杈?
			if compareVersions(installedVer, requiredVersion) < 0 {
				mismatched = append(mismatched, fmt.Sprintf("%s (installed: %s, required: %s)", pkgName, installedVer, requiredVersion))
			} else {
				installed = append(installed, fmt.Sprintf("%s==%s", pkgName, installedVer))
			}
		} else {
			installed = append(installed, fmt.Sprintf("%s==%s", pkgName, installedVer))
		}
	}

	// 妫€鏌?torch 鏄惁鏀寔 CUDA锛堝鏋滃凡瀹夎 torch锛?
	torchCudaAvailable := false
	torchVersion, hasTorch := installedMap["torch"]
	log.Printf("[CheckDeps] torch installed: %v, version: %s", hasTorch, torchVersion)
	if hasTorch {
		// 杩愯 Python 妫€鏌?CUDA 鍙敤鎬?
		pythonExe := filepath.Join(venvPath, "Scripts", "python.exe")
		checkCmd := exec.Command(pythonExe, "-c", "import torch; print('CUDA' if torch.cuda.is_available() else 'CPU')")
		if output, err := checkCmd.Output(); err == nil {
			result := strings.TrimSpace(string(output))
			torchCudaAvailable = (result == "CUDA")
			log.Printf("[CheckDeps] torch CUDA check: %s", result)
		} else {
			log.Printf("[CheckDeps] torch CUDA check failed: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"deployed":           true,
		"missing":            missing,
		"installed":          installed,
		"mismatched":         mismatched,
		"torchInstalled":     hasTorch,
		"torchCudaAvailable": torchCudaAvailable,
	})
}

// compareVersions 绠€鍗曠増鏈瘮杈?(杩斿洖 -1, 0, 1)
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}
	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			// 鍘绘帀闈炴暟瀛楀悗缂€
			numStr := strings.TrimFunc(parts1[i], func(r rune) bool { return r < '0' || r > '9' })
			n1, _ = strconv.Atoi(numStr)
		}
		if i < len(parts2) {
			numStr := strings.TrimFunc(parts2[i], func(r rune) bool { return r < '0' || r > '9' })
			n2, _ = strconv.Atoi(numStr)
		}
		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}
	return 0
}

// 妯″瀷涓嬭浇鐘舵€?
var modelDownloadStatus = struct {
	sync.RWMutex
	downloads map[string]*ModelDownloadProgress
}{downloads: make(map[string]*ModelDownloadProgress)}

type ModelDownloadProgress struct {
	ModelID    string  `json:"modelId"`
	Status     string  `json:"status"` // downloading, completed, failed
	Progress   float64 `json:"progress"`
	Downloaded int64   `json:"downloaded"`
	Total      int64   `json:"total"`
	Error      string  `json:"error,omitempty"`
}

// handleDownloadModel 涓嬭浇妯″瀷
func handleDownloadModel(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ModelID     string `json:"modelId"`     // 濡?"yolov8n" 鎴?"yolov8n-pose"
		DownloadURL string `json:"downloadUrl"` // 鑷畾涔変笅杞介摼鎺ワ紙闈炴爣鍑嗘ā鍨嬶級
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ModelID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
		return
	}

	// 鑾峰彇涓嬭浇閾炬帴
	downloadURL := req.DownloadURL
	if downloadURL == "" {
		// 灏濊瘯浠庢爣鍑嗘ā鍨嬪垪琛ㄨ幏鍙?
		if url, ok := standardModelURLs[req.ModelID]; ok {
			downloadURL = url
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "unknown_model"})
			return
		}
	}

	// 妫€鏌ユ槸鍚﹀凡鍦ㄤ笅杞?
	modelDownloadStatus.RLock()
	if progress, exists := modelDownloadStatus.downloads[req.ModelID]; exists && progress.Status == "downloading" {
		modelDownloadStatus.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "already_downloading"})
		return
	}
	modelDownloadStatus.RUnlock()

	// 鍒涘缓涓嬭浇鐩綍
	modelsDir := getModelsDir()
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "create_dir_failed"})
		return
	}

	// 鐩爣鏂囦欢璺緞
	fileName := req.ModelID + ".pt"
	filePath := filepath.Join(modelsDir, fileName)

	// 鍒濆鍖栦笅杞界姸鎬?
	modelDownloadStatus.Lock()
	modelDownloadStatus.downloads[req.ModelID] = &ModelDownloadProgress{
		ModelID:  req.ModelID,
		Status:   "downloading",
		Progress: 0,
	}
	modelDownloadStatus.Unlock()

	// 鍚庡彴涓嬭浇
	go downloadModelFile(req.ModelID, downloadURL, filePath)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "download_started"})
}

// downloadModelFile 鍚庡彴涓嬭浇妯″瀷鏂囦欢
func downloadModelFile(modelID, url, filePath string) {
	log.Printf("[Model] Starting download: %s from %s", modelID, url)

	updateProgress := func(status string, progress float64, downloaded, total int64, errMsg string) {
		modelDownloadStatus.Lock()
		if p, exists := modelDownloadStatus.downloads[modelID]; exists {
			p.Status = status
			p.Progress = progress
			p.Downloaded = downloaded
			p.Total = total
			p.Error = errMsg
		}
		modelDownloadStatus.Unlock()
	}

	// 鍒涘缓 HTTP 璇锋眰
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[Model] Download failed: %v", err)
		updateProgress("failed", 0, 0, 0, err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Model] Download failed: HTTP %d", resp.StatusCode)
		updateProgress("failed", 0, 0, 0, fmt.Sprintf("HTTP %d", resp.StatusCode))
		return
	}

	total := resp.ContentLength
	log.Printf("[Model] Download started: %s, Content-Length: %d, Status: %d", modelID, total, resp.StatusCode)
	updateProgress("downloading", 0, 0, total, "")

	// 鍒涘缓涓存椂鏂囦欢
	tempFile := filePath + ".tmp"
	out, err := os.Create(tempFile)
	if err != nil {
		log.Printf("[Model] Create file failed: %v", err)
		updateProgress("failed", 0, 0, total, err.Error())
		return
	}

	var downloaded int64
	buf := make([]byte, 32*1024)
	lastReport := time.Now()

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				out.Close()
				os.Remove(tempFile)
				log.Printf("[Model] Write failed: %v", writeErr)
				updateProgress("failed", 0, downloaded, total, writeErr.Error())
				return
			}
			downloaded += int64(n)

			// 每 500ms 更新一次进度
			if time.Since(lastReport) > 500*time.Millisecond {
				var progress float64
				if total > 0 {
					progress = float64(downloaded) / float64(total)
				} else {
					// Content-Length 未知时，根据已下载大小估算（假设模型约 10MB）
					progress = float64(downloaded) / float64(10*1024*1024)
					if progress > 0.99 {
						progress = 0.99 // 未知大小时最多显示 99%
					}
				}
				updateProgress("downloading", progress, downloaded, total, "")
				lastReport = time.Now()
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			out.Close()
			os.Remove(tempFile)
			log.Printf("[Model] Read failed: %v", err)
			updateProgress("failed", 0, downloaded, total, err.Error())
			return
		}
	}

	out.Close()

	// 閲嶅懡鍚嶄复鏃舵枃浠?
	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile)
		log.Printf("[Model] Rename failed: %v", err)
		updateProgress("failed", 1, downloaded, total, err.Error())
		return
	}

	log.Printf("[Model] Download completed: %s (%d bytes)", modelID, downloaded)
	updateProgress("completed", 1, downloaded, total, "")
}

// handleModelStatus 鑾峰彇妯″瀷涓嬭浇鐘舵€?
func handleModelStatus(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	modelID := r.URL.Query().Get("modelId")

	// 濡傛灉鎸囧畾浜?modelId锛岃繑鍥炲崟涓姸鎬?
	if modelID != "" {
		// 妫€鏌ユ枃浠舵槸鍚﹀瓨鍦?
		modelsDir := getModelsDir()
		filePath := filepath.Join(modelsDir, modelID+".pt")
		fileExists := false
		if _, err := os.Stat(filePath); err == nil {
			fileExists = true
		}

		// 妫€鏌ヤ笅杞界姸鎬?
		modelDownloadStatus.RLock()
		progress, downloading := modelDownloadStatus.downloads[modelID]
		modelDownloadStatus.RUnlock()

		if downloading {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(progress)
			return
		}

		status := "not_downloaded"
		if fileExists {
			status = "completed"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"modelId":  modelID,
			"status":   status,
			"progress": 0,
			"exists":   fileExists,
		})
		return
	}

	// 杩斿洖鎵€鏈変笅杞界姸鎬?
	modelDownloadStatus.RLock()
	downloads := make(map[string]*ModelDownloadProgress)
	for k, v := range modelDownloadStatus.downloads {
		downloads[k] = v
	}
	modelDownloadStatus.RUnlock()

	// 妫€鏌ュ凡瀛樺湪鐨勬ā鍨嬫枃浠?
	modelsDir := getModelsDir()
	existingModels := []string{}
	if files, err := os.ReadDir(modelsDir); err == nil {
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".pt") {
				existingModels = append(existingModels, strings.TrimSuffix(f.Name(), ".pt"))
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"downloads":      downloads,
		"existingModels": existingModels,
		"modelsDir":      modelsDir,
	})
}

// handleCancelDownload 鍙栨秷妯″瀷涓嬭浇
func handleCancelDownload(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ModelID string `json:"modelId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ModelID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
		return
	}

	modelDownloadStatus.Lock()
	if progress, exists := modelDownloadStatus.downloads[req.ModelID]; exists {
		progress.Status = "cancelled"
		// 鍒犻櫎涓存椂鏂囦欢
		modelsDir := getModelsDir()
		tempFile := filepath.Join(modelsDir, req.ModelID+".pt.tmp")
		os.Remove(tempFile)
	}
	delete(modelDownloadStatus.downloads, req.ModelID)
	modelDownloadStatus.Unlock()

	log.Printf("[Model] Download cancelled: %s", req.ModelID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// prepareDirectoryDataset 准备目录数据集（检查并自动分割 train/val）
func prepareDirectoryDataset(datasetPath, taskID string, trainRatio float64) (string, error) {
	log.Printf("[PrepareDirectoryDataset] Checking dataset: %s", datasetPath)

	// 读取 data.yaml
	dataYamlPath := filepath.Join(datasetPath, "data.yaml")
	dataYamlBytes, err := os.ReadFile(dataYamlPath)
	if err != nil {
		// data.yaml 不存在，可能是 VOC/COCO 格式，尝试自动检测并转换
		log.Printf("[PrepareDirectoryDataset] No data.yaml found, trying to detect and convert dataset format")
		return convertNonYoloDataset(datasetPath, taskID, trainRatio)
	}
	dataYamlContent := string(dataYamlBytes)

	// 简单检查是否有 val 字段（使用正则匹配行首的 val:）
	hasVal := regexp.MustCompile(`(?m)^val\s*:`).MatchString(dataYamlContent)
	hasTrain := regexp.MustCompile(`(?m)^train\s*:`).MatchString(dataYamlContent)

	if hasVal && hasTrain {
		// 数据集已有 train/val 分割，但需要验证 val 目录是否真的存在
		valPathMatch := regexp.MustCompile(`(?m)^val\s*:\s*(.+)$`).FindStringSubmatch(dataYamlContent)
		if len(valPathMatch) >= 2 {
			valRelPath := strings.TrimSpace(valPathMatch[1])
			// 去掉可能的引号
			valRelPath = strings.Trim(valRelPath, `"'`)
			// 解析相对路径
			var valFullPath string
			if filepath.IsAbs(valRelPath) {
				valFullPath = valRelPath
			} else {
				valFullPath = filepath.Join(datasetPath, valRelPath)
			}
			// 检查 val 目录是否存在且有图片
			if _, err := os.Stat(valFullPath); os.IsNotExist(err) {
				log.Printf("[PrepareDirectoryDataset] val path declared in data.yaml does not exist: %s", valFullPath)
				hasVal = false // 标记为没有有效的 val，后续会自动分割
			} else if !hasImagesInDir(valFullPath) {
				log.Printf("[PrepareDirectoryDataset] val path exists but has no images: %s", valFullPath)
				hasVal = false
			}
		}
	}

	if hasVal && hasTrain {
		// 数据集已有 train/val 分割，检查路径是否需要修正
		log.Printf("[PrepareDirectoryDataset] Dataset already has train/val split")
		return ensureAbsolutePathsInDataYaml(datasetPath, dataYamlContent, taskID)
	}

	if !hasTrain {
		return "", fmt.Errorf("data.yaml missing 'train' field")
	}

	log.Printf("[PrepareDirectoryDataset] Dataset missing 'val', auto-splitting with ratio %.2f", trainRatio)

	// 鎻愬彇 train 璺緞
	trainPathMatch := regexp.MustCompile(`(?m)^train\s*:\s*(.+)$`).FindStringSubmatch(dataYamlContent)
	if len(trainPathMatch) < 2 {
		return "", fmt.Errorf("cannot parse train path from data.yaml")
	}
	trainPathStr := strings.TrimSpace(trainPathMatch[1])

	// 鎻愬彇 names銆乶c 鍜?kpt_shape
	namesMatch := regexp.MustCompile(`(?m)^names\s*:\s*(\[.+\]|[\s\S]*?)(?:\n[a-z]|\z)`).FindStringSubmatch(dataYamlContent)
	ncMatch := regexp.MustCompile(`(?m)^nc\s*:\s*(\d+)`).FindStringSubmatch(dataYamlContent)
	kptShapeMatch := regexp.MustCompile(`(?m)^kpt_shape\s*:\s*(\[.+\])`).FindStringSubmatch(dataYamlContent)

	// 闇€瑕佽嚜鍔ㄥ垎鍓诧細鍒涘缓涓存椂鐩綍
	cfg, err := loadPathsConfig()
	if err != nil {
		return "", fmt.Errorf("config error: %v", err)
	}
	outputDir := filepath.Join(cfg.DataPath, "training_datasets", taskID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir failed: %v", err)
	}

	// 鍒涘缓 train/val 瀛愮洰褰?
	trainImgDir := filepath.Join(outputDir, "train", "images")
	trainLblDir := filepath.Join(outputDir, "train", "labels")
	valImgDir := filepath.Join(outputDir, "val", "images")
	valLblDir := filepath.Join(outputDir, "val", "labels")
	os.MkdirAll(trainImgDir, 0755)
	os.MkdirAll(trainLblDir, 0755)
	os.MkdirAll(valImgDir, 0755)
	os.MkdirAll(valLblDir, 0755)

	// 鑾峰彇鍘熷鍥剧墖鐩綍
	srcImgDir := filepath.Join(datasetPath, trainPathStr)
	if filepath.IsAbs(trainPathStr) {
		srcImgDir = trainPathStr
	}

	// 瀵瑰簲鐨?labels 鐩綍
	srcLblDir := strings.Replace(srcImgDir, "images", "labels", 1)

	// 鍒楀嚭鎵€鏈夊浘鐗?
	imgFiles, err := os.ReadDir(srcImgDir)
	if err != nil {
		return "", fmt.Errorf("cannot read images dir: %v", err)
	}

	// 杩囨护鍥剧墖鏂囦欢
	var images []string
	for _, f := range imgFiles {
		if f.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Name()))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".bmp" {
			images = append(images, f.Name())
		}
	}

	if len(images) == 0 {
		return "", fmt.Errorf("no images found in dataset")
	}

	// 鎸夋瘮渚嬪垎鍓?
	trainCount := int(float64(len(images)) * trainRatio)
	if trainCount == 0 {
		trainCount = 1
	}
	if trainCount >= len(images) {
		trainCount = len(images) - 1
	}

	log.Printf("[PrepareDirectoryDataset] Splitting %d images: %d train, %d val",
		len(images), trainCount, len(images)-trainCount)

	// 澶嶅埗鏂囦欢
	for i, imgName := range images {
		var dstImgDir, dstLblDir string
		if i < trainCount {
			dstImgDir = trainImgDir
			dstLblDir = trainLblDir
		} else {
			dstImgDir = valImgDir
			dstLblDir = valLblDir
		}

		// 澶嶅埗鍥剧墖
		srcImg := filepath.Join(srcImgDir, imgName)
		dstImg := filepath.Join(dstImgDir, imgName)
		if err := copyFile(srcImg, dstImg); err != nil {
			log.Printf("[PrepareDirectoryDataset] Failed to copy image %s: %v", imgName, err)
			continue
		}

		// 澶嶅埗鏍囩
		lblName := strings.TrimSuffix(imgName, filepath.Ext(imgName)) + ".txt"
		srcLbl := filepath.Join(srcLblDir, lblName)
		dstLbl := filepath.Join(dstLblDir, lblName)
		if _, err := os.Stat(srcLbl); err == nil {
			copyFile(srcLbl, dstLbl)
		}
	}

	// 鐢熸垚鏂扮殑 data.yaml锛堢畝鍗曟枃鏈牸寮忥級
	var newDataYaml strings.Builder
	newDataYaml.WriteString(fmt.Sprintf("path: %s\n", outputDir))
	newDataYaml.WriteString("train: train/images\n")
	newDataYaml.WriteString("val: val/images\n")
	if ncMatch != nil && len(ncMatch) > 1 {
		newDataYaml.WriteString(fmt.Sprintf("nc: %s\n", ncMatch[1]))
	}
	if namesMatch != nil && len(namesMatch) > 1 {
		newDataYaml.WriteString(fmt.Sprintf("names: %s\n", strings.TrimSpace(namesMatch[1])))
	}
	if kptShapeMatch != nil && len(kptShapeMatch) > 1 {
		newDataYaml.WriteString(fmt.Sprintf("kpt_shape: %s\n", kptShapeMatch[1]))
	}

	newDataYamlPath := filepath.Join(outputDir, "data.yaml")
	if err := os.WriteFile(newDataYamlPath, []byte(newDataYaml.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write data.yaml: %v", err)
	}

	log.Printf("[PrepareDirectoryDataset] Created split dataset at: %s", outputDir)
	return outputDir, nil
}

// ensureAbsolutePathsInDataYaml 确保 data.yaml 中的路径是绝对路径
func ensureAbsolutePathsInDataYaml(datasetPath, dataYamlContent, taskID string) (string, error) {
	// 检查是否已有 path 字段且为绝对路径
	pathMatch := regexp.MustCompile(`(?m)^path\s*:\s*(.+)$`).FindStringSubmatch(dataYamlContent)
	if pathMatch != nil && len(pathMatch) > 1 {
		existingPath := strings.TrimSpace(pathMatch[1])
		// 如果 path 已经是绝对路径且指向数据集目录，直接使用
		if filepath.IsAbs(existingPath) && existingPath == datasetPath {
			log.Printf("[PrepareDirectoryDataset] data.yaml already has correct absolute path")
			return datasetPath, nil
		}
	}

	// 需要修正路径：创建一个临时 data.yaml
	log.Printf("[PrepareDirectoryDataset] Fixing relative paths in data.yaml")

	cfg, err := loadPathsConfig()
	if err != nil {
		return "", fmt.Errorf("config error: %v", err)
	}
	outputDir := filepath.Join(cfg.DataPath, "training_datasets", taskID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir failed: %v", err)
	}

	// 修改 data.yaml：添加或替换 path 字段为数据集的绝对路径
	var newContent string
	if pathMatch != nil {
		// 替换现有 path
		newContent = regexp.MustCompile(`(?m)^path\s*:.*$`).ReplaceAllString(dataYamlContent, fmt.Sprintf("path: %s", datasetPath))
	} else {
		// 添加 path 字段到开头
		newContent = fmt.Sprintf("path: %s\n%s", datasetPath, dataYamlContent)
	}

	// 写入新的 data.yaml
	newDataYamlPath := filepath.Join(outputDir, "data.yaml")
	if err := os.WriteFile(newDataYamlPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write fixed data.yaml: %v", err)
	}

	log.Printf("[PrepareDirectoryDataset] Created fixed data.yaml at: %s", outputDir)
	return outputDir, nil
}

// convertNonYoloDataset 转换非 YOLO 格式数据集或为 YOLO 格式生成 data.yaml
func convertNonYoloDataset(datasetPath, taskID string, trainRatio float64) (string, error) {
	log.Printf("[ConvertDataset] Attempting to detect and prepare dataset: %s", datasetPath)

	// 1. 首先检查是否是 YOLO 格式（有 labels 目录或 train/val 子目录结构）
	yoloType := detectYoloFormat(datasetPath)
	if yoloType != "" {
		log.Printf("[ConvertDataset] Detected YOLO format: %s", yoloType)
		return generateYoloDataYaml(datasetPath, taskID, yoloType, trainRatio)
	}

	// 2. 检测其他格式
	var detectedFormat string

	// 检查是否是 VOC 格式（有 Annotations 目录或 xml 文件）
	annotationsDir := filepath.Join(datasetPath, "Annotations")
	if _, err := os.Stat(annotationsDir); err == nil {
		detectedFormat = "VOC"
	} else {
		// 检查根目录是否有 xml 文件
		files, _ := os.ReadDir(datasetPath)
		for _, f := range files {
			if strings.HasSuffix(strings.ToLower(f.Name()), ".xml") {
				detectedFormat = "VOC"
				break
			}
		}
	}

	// 检查是否是 COCO 格式（有 annotations 目录下的 json 文件）
	if detectedFormat == "" {
		cocoAnnotationsDir := filepath.Join(datasetPath, "annotations")
		if files, err := os.ReadDir(cocoAnnotationsDir); err == nil {
			for _, f := range files {
				if strings.HasSuffix(strings.ToLower(f.Name()), ".json") {
					detectedFormat = "COCO"
					break
				}
			}
		}
	}

	if detectedFormat == "" {
		return "", fmt.Errorf("dataset format not supported: no data.yaml, no labels/*.txt (YOLO), no XML files (VOC), no annotations/*.json (COCO)")
	}

	// 目前训练功能只支持 YOLO 格式数据集
	return "", fmt.Errorf("detected %s format dataset, but training currently only supports YOLO format. Please import this dataset into a project first, then export as YOLO format", detectedFormat)
}

// detectYoloFormat 检测 YOLO 格式类型
// 返回: "flat" (images+labels), "split" (train/val子目录), "" (非YOLO格式)
func detectYoloFormat(datasetPath string) string {
	// 检查是否有 train/val 子目录结构
	trainDir := filepath.Join(datasetPath, "train")
	if _, err := os.Stat(trainDir); err == nil {
		// 检查 train 目录下是否有 images 或直接有图片
		trainImagesDir := filepath.Join(trainDir, "images")
		trainLabelsDir := filepath.Join(trainDir, "labels")
		if _, err := os.Stat(trainImagesDir); err == nil {
			if _, err := os.Stat(trainLabelsDir); err == nil {
				return "split"
			}
		}
		// 检查 train 目录下是否直接有图片和标注
		if hasYoloLabels(trainDir) {
			return "split_flat"
		}
	}

	// 检查扁平结构: images/ + labels/
	imagesDir := filepath.Join(datasetPath, "images")
	labelsDir := filepath.Join(datasetPath, "labels")
	if _, err := os.Stat(imagesDir); err == nil {
		if _, err := os.Stat(labelsDir); err == nil {
			if hasYoloLabels(labelsDir) {
				return "flat"
			}
		}
	}

	return ""
}

// hasYoloLabels 检查目录下是否有 .txt 标注文件
func hasYoloLabels(dir string) bool {
	files, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".txt") && f.Name() != "classes.txt" {
			return true
		}
	}
	return false
}

// hasImagesInDir 检查目录下是否有图片文件
func hasImagesInDir(dir string) bool {
	imageExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".bmp": true, ".webp": true}
	files, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Name()))
		if imageExts[ext] {
			return true
		}
	}
	return false
}

// generateYoloDataYaml 为 YOLO 格式数据集生成 data.yaml
func generateYoloDataYaml(datasetPath, taskID, yoloType string, trainRatio float64) (string, error) {
	log.Printf("[GenerateDataYaml] Generating data.yaml for %s format at %s", yoloType, datasetPath)

	// 读取类别信息
	classes := readYoloClasses(datasetPath, yoloType)
	if len(classes) == 0 {
		// 如果没有 classes.txt，尝试从标注文件中推断类别数量
		classes = inferClassesFromLabels(datasetPath, yoloType)
	}
	log.Printf("[GenerateDataYaml] Found %d classes", len(classes))

	// 创建临时输出目录
	cfg, err := loadPathsConfig()
	if err != nil {
		return "", fmt.Errorf("config error: %v", err)
	}
	outputDir := filepath.Join(cfg.DataPath, "training_datasets", taskID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir failed: %v", err)
	}

	var dataYamlContent string

	switch yoloType {
	case "split", "split_flat":
		// 已有 train/val 分割，检查实际目录结构
		trainImagesPath := filepath.Join(datasetPath, "train", "images")
		valImagesPath := filepath.Join(datasetPath, "val", "images")
		testImagesPath := filepath.Join(datasetPath, "test", "images")

		// 确定实际的子路径格式
		var trainSubPath, valSubPath, testSubPath string

		// 检查 train 目录结构
		if _, err := os.Stat(trainImagesPath); err == nil {
			trainSubPath = "train/images"
		} else if _, err := os.Stat(filepath.Join(datasetPath, "train")); err == nil {
			trainSubPath = "train" // 图片直接在 train/ 下
		} else {
			return splitAndGenerateDataYaml(datasetPath, outputDir, classes, trainRatio, yoloType)
		}

		// 检查 val 目录结构
		if _, err := os.Stat(valImagesPath); err == nil {
			valSubPath = "val/images"
		} else if _, err := os.Stat(filepath.Join(datasetPath, "val")); err == nil {
			// val 目录存在但没有 images 子目录，检查是否有图片
			if hasImagesInDir(filepath.Join(datasetPath, "val")) {
				valSubPath = "val"
			} else {
				// val 目录为空或无图片，需要从 train 分割
				log.Printf("[GenerateDataYaml] val directory exists but has no images, splitting from train")
				return splitAndGenerateDataYaml(datasetPath, outputDir, classes, trainRatio, yoloType)
			}
		} else {
			// 没有 val 目录，需要从 train 分割
			return splitAndGenerateDataYaml(datasetPath, outputDir, classes, trainRatio, yoloType)
		}

		// 检查 test 目录
		if _, err := os.Stat(testImagesPath); err == nil {
			testSubPath = "test/images"
		} else if hasImagesInDir(filepath.Join(datasetPath, "test")) {
			testSubPath = "test"
		}

		// 构建 data.yaml 内容
		dataYamlContent = fmt.Sprintf("path: %s\ntrain: %s\nval: %s\n", datasetPath, trainSubPath, valSubPath)
		if testSubPath != "" {
			dataYamlContent += fmt.Sprintf("test: %s\n", testSubPath)
		}
		dataYamlContent += fmt.Sprintf("\nnc: %d\nnames:\n", len(classes))
		for i, name := range classes {
			dataYamlContent += fmt.Sprintf("  %d: %s\n", i, name)
		}

	case "flat":
		// 扁平结构，需要分割 train/val
		return splitAndGenerateDataYaml(datasetPath, outputDir, classes, trainRatio, yoloType)
	}

	// 写入 data.yaml
	dataYamlPath := filepath.Join(outputDir, "data.yaml")
	if err := os.WriteFile(dataYamlPath, []byte(dataYamlContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write data.yaml: %v", err)
	}

	log.Printf("[GenerateDataYaml] Created data.yaml at: %s", outputDir)
	return outputDir, nil
}

// readYoloClasses 读取 YOLO 数据集的类别信息
func readYoloClasses(datasetPath, yoloType string) []string {
	// 尝试多个可能的位置
	possiblePaths := []string{
		filepath.Join(datasetPath, "classes.txt"),
		filepath.Join(datasetPath, "data", "classes.txt"),
	}
	if yoloType == "split" || yoloType == "split_flat" {
		possiblePaths = append(possiblePaths, filepath.Join(datasetPath, "train", "classes.txt"))
	}

	for _, p := range possiblePaths {
		if data, err := os.ReadFile(p); err == nil {
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			var classes []string
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					classes = append(classes, line)
				}
			}
			if len(classes) > 0 {
				return classes
			}
		}
	}
	return nil
}

// inferClassesFromLabels 从标注文件推断类别数量
func inferClassesFromLabels(datasetPath, yoloType string) []string {
	var labelsDir string
	switch yoloType {
	case "flat":
		labelsDir = filepath.Join(datasetPath, "labels")
	case "split", "split_flat":
		labelsDir = filepath.Join(datasetPath, "train", "labels")
		if _, err := os.Stat(labelsDir); os.IsNotExist(err) {
			labelsDir = filepath.Join(datasetPath, "train")
		}
	}

	maxClassID := -1
	files, _ := os.ReadDir(labelsDir)
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".txt") || f.Name() == "classes.txt" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(labelsDir, f.Name()))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				if classID, err := strconv.Atoi(parts[0]); err == nil && classID > maxClassID {
					maxClassID = classID
				}
			}
		}
	}

	if maxClassID < 0 {
		return []string{"object"} // 默认一个类别
	}

	// 生成默认类别名
	classes := make([]string, maxClassID+1)
	for i := range classes {
		classes[i] = fmt.Sprintf("class_%d", i)
	}
	return classes
}

// splitAndGenerateDataYaml 分割数据集并生成 data.yaml
func splitAndGenerateDataYaml(srcPath, outputDir string, classes []string, trainRatio float64, yoloType string) (string, error) {
	log.Printf("[SplitDataset] Splitting dataset with ratio %.2f", trainRatio)

	// 确定源目录
	var srcImagesDir, srcLabelsDir string
	if yoloType == "flat" {
		srcImagesDir = filepath.Join(srcPath, "images")
		srcLabelsDir = filepath.Join(srcPath, "labels")
	} else {
		srcImagesDir = filepath.Join(srcPath, "train", "images")
		srcLabelsDir = filepath.Join(srcPath, "train", "labels")
		if _, err := os.Stat(srcImagesDir); os.IsNotExist(err) {
			srcImagesDir = filepath.Join(srcPath, "train")
			srcLabelsDir = filepath.Join(srcPath, "train")
		}
	}

	// 收集图片文件
	imageExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".bmp": true, ".webp": true}
	files, _ := os.ReadDir(srcImagesDir)
	var images []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Name()))
		if imageExts[ext] {
			images = append(images, f.Name())
		}
	}

	if len(images) == 0 {
		return "", fmt.Errorf("no images found in %s", srcImagesDir)
	}

	// 随机打乱
	for i := len(images) - 1; i > 0; i-- {
		j := i - (i % (i + 1)) // 简单洗牌
		images[i], images[j] = images[j], images[i]
	}

	// 分割
	trainCount := int(float64(len(images)) * trainRatio)
	if trainCount == 0 {
		trainCount = 1
	}
	if trainCount >= len(images) {
		trainCount = len(images) - 1
	}

	// 创建目录
	trainImgDir := filepath.Join(outputDir, "train", "images")
	trainLblDir := filepath.Join(outputDir, "train", "labels")
	valImgDir := filepath.Join(outputDir, "val", "images")
	valLblDir := filepath.Join(outputDir, "val", "labels")
	os.MkdirAll(trainImgDir, 0755)
	os.MkdirAll(trainLblDir, 0755)
	os.MkdirAll(valImgDir, 0755)
	os.MkdirAll(valLblDir, 0755)

	log.Printf("[SplitDataset] Splitting %d images: %d train, %d val", len(images), trainCount, len(images)-trainCount)

	// 复制文件
	for i, imgName := range images {
		var dstImgDir, dstLblDir string
		if i < trainCount {
			dstImgDir, dstLblDir = trainImgDir, trainLblDir
		} else {
			dstImgDir, dstLblDir = valImgDir, valLblDir
		}

		// 复制图片
		srcImg := filepath.Join(srcImagesDir, imgName)
		dstImg := filepath.Join(dstImgDir, imgName)
		if err := copyFile(srcImg, dstImg); err != nil {
			log.Printf("[SplitDataset] Failed to copy image %s: %v", imgName, err)
			continue
		}

		// 复制标注
		baseName := strings.TrimSuffix(imgName, filepath.Ext(imgName))
		srcLbl := filepath.Join(srcLabelsDir, baseName+".txt")
		dstLbl := filepath.Join(dstLblDir, baseName+".txt")
		copyFile(srcLbl, dstLbl) // 标注可能不存在，忽略错误
	}

	// 生成 data.yaml
	dataYamlContent := fmt.Sprintf("path: %s\ntrain: train/images\nval: val/images\n\nnc: %d\nnames:\n", outputDir, len(classes))
	for i, name := range classes {
		dataYamlContent += fmt.Sprintf("  %d: %s\n", i, name)
	}

	dataYamlPath := filepath.Join(outputDir, "data.yaml")
	if err := os.WriteFile(dataYamlPath, []byte(dataYamlContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write data.yaml: %v", err)
	}

	log.Printf("[SplitDataset] Created split dataset at: %s", outputDir)
	return outputDir, nil
}

// prepareTrainingDataset 准备训练数据集（复用现有导出逻辑，导出 YOLO 格式到临时目录）
func prepareTrainingDataset(trainsetID, taskID string, trainRatio float64) (string, error) {
	log.Printf("[PrepareDataset] Starting: trainsetId=%s, taskId=%s, trainRatio=%.2f", trainsetID, taskID, trainRatio)

	// 1. 璇诲彇璁粌闆嗛厤缃?
	trainsetPath := filepath.Join(getTrainsetsDir(), trainsetID+".json")
	data, err := os.ReadFile(trainsetPath)
	if err != nil {
		return "", fmt.Errorf("trainset not found: %v", err)
	}
	var trainset Trainset
	if err := json.Unmarshal(data, &trainset); err != nil {
		return "", fmt.Errorf("invalid trainset: %v", err)
	}
	log.Printf("[PrepareDataset] Loaded trainset: %s with %d categories", trainset.Name, len(trainset.Categories))

	// 2. 鏌ユ壘 dataset.common 鎻掍欢
	plugins, _ := loadInstalledPlugins()
	var pluginPath, entryName string
	for _, p := range plugins {
		if p.ID == "dataset.common" {
			pluginPath = p.Path
			entryName = p.Entry
			break
		}
	}
	if pluginPath == "" {
		return "", fmt.Errorf("dataset.common plugin not found")
	}
	log.Printf("[PrepareDataset] Found plugin: %s", pluginPath)

	// 3. 鍒涘缓杈撳嚭鐩綍
	cfg, err := loadPathsConfig()
	if err != nil {
		return "", fmt.Errorf("config error: %v", err)
	}
	outputDir := filepath.Join(cfg.DataPath, "training_datasets", taskID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir failed: %v", err)
	}

	// 4. 鏀堕泦鍥剧墖鍜屾爣娉紙澶嶇敤 handleDatasetExport 鐨勯€昏緫锛?
	type ExportImage struct {
		Key          string `json:"key"`
		RelativePath string `json:"relativePath"`
		AbsolutePath string `json:"absolutePath"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		Split        string `json:"split"`
	}
	type ExportCategory struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Type  string `json:"type"`
		Color string `json:"color"`
		Mate  string `json:"mate,omitempty"` // 鍏抽敭鐐归厤缃瓑鍏冩暟鎹?
	}
	type ExportAnnotation struct {
		ID         int                    `json:"id,omitempty"` // 鏍囨敞ID锛岀敤浜庡叧閿偣寮曠敤鐭╁舰妗?
		ImageKey   string                 `json:"imageKey"`
		CategoryID int                    `json:"categoryId"`
		Type       string                 `json:"type"`
		Data       map[string]interface{} `json:"data"`
	}

	var images []ExportImage
	var categories []ExportCategory
	var annotations []ExportAnnotation
	imageSet := make(map[string]bool)

	// 涓虹被鍒噸鏂板垎閰嶈繛缁殑ID锛圷OLO闇€瑕佷粠0寮€濮嬬殑杩炵画绱㈠紩锛?
	catIDMap := make(map[int]int) // 鍘熷categoryId -> 鏂扮殑杩炵画绱㈠紩

	for _, cat := range trainset.Categories {
		log.Printf("[PrepareDataset] Processing: %s (projectId=%s, v%d, catId=%d)",
			cat.CategoryName, cat.ProjectID, cat.Version, cat.CategoryID)

		// 鍔犺浇鐗堟湰鏁版嵁搴?
		versionDbPath := filepath.Join(cfg.DataPath, "project_item", cat.ProjectID, "db", "versions",
			fmt.Sprintf("v%d", cat.Version), "project.db")

		if _, err := os.Stat(versionDbPath); os.IsNotExist(err) {
			log.Printf("[PrepareDataset] DB not found: %s", versionDbPath)
			continue
		}

		db, err := openProjectDB(versionDbPath)
		if err != nil {
			log.Printf("[PrepareDataset] Failed to open db: %v", err)
			continue
		}

		// 鍒嗛厤鏂扮殑杩炵画绱㈠紩锛屽苟鑾峰彇 mate 淇℃伅
		if _, exists := catIDMap[cat.CategoryID]; !exists {
			newID := len(categories)
			catIDMap[cat.CategoryID] = newID

			// 浠庢暟鎹簱鑾峰彇 mate 淇℃伅
			var catMate sql.NullString
			db.QueryRow("SELECT mate FROM categories WHERE id = ?", cat.CategoryID).Scan(&catMate)

			categories = append(categories, ExportCategory{
				ID:    newID,
				Name:  cat.CategoryName,
				Type:  cat.CategoryType,
				Color: "#FF6B6B",
				Mate:  catMate.String,
			})
			log.Printf("[PrepareDataset] Category mate: %s", catMate.String)
		}

		// 鑾峰彇璇ョ被鍒殑鎵€鏈夋爣娉?
		rows, err := db.Query(`
			SELECT a.id, a.image_id, a.type, a.data, i.original_rel_path
			FROM annotations a
			JOIN image_index i ON a.image_id = i.id
			WHERE a.category_id = ?
		`, cat.CategoryID)
		if err != nil {
			log.Printf("[PrepareDataset] Query failed: %v", err)
			db.Close()
			continue
		}

		annCount := 0
		for rows.Next() {
			var annID, imageID int
			var annoType, dataJSON, relativePath string
			if err := rows.Scan(&annID, &imageID, &annoType, &dataJSON, &relativePath); err != nil {
				continue
			}

			imageKey := fmt.Sprintf("%s-v%d-%d", cat.ProjectID, cat.Version, imageID)

			// 娣诲姞鍥剧墖锛堝幓閲嶏級
			if !imageSet[imageKey] {
				imageSet[imageKey] = true
				absPath := filepath.Join(cfg.DataPath, "project_item", cat.ProjectID, relativePath)

				// 璇诲彇鍥剧墖灏哄
				imgWidth, imgHeight := 0, 0
				if f, err := os.Open(absPath); err == nil {
					if imgCfg, _, err := image.DecodeConfig(f); err == nil {
						imgWidth, imgHeight = imgCfg.Width, imgCfg.Height
					}
					f.Close()
				}

				images = append(images, ExportImage{
					Key:          imageKey,
					RelativePath: relativePath,
					AbsolutePath: absPath,
					Width:        imgWidth,
					Height:       imgHeight,
				})
			}

			// 瑙ｆ瀽鏍囨敞鏁版嵁
			var rawData map[string]interface{}
			if err := json.Unmarshal([]byte(dataJSON), &rawData); err != nil {
				continue
			}

			// 鏍规嵁鏍囨敞绫诲瀷澶勭悊
			var exportType string
			var exportData map[string]interface{}

			switch annoType {
			case "keypoint", "polygon":
				// 淇濈暀鍘熷绫诲瀷鍜屾暟鎹?
				exportType = annoType
				exportData = rawData
			default:
				// bbox 鍙婂叾浠栫被鍨嬶細杞崲涓?bbox 鏍煎紡
				exportType = "bbox"
				exportData = convertToBbox(annoType, rawData)
				if exportData == nil {
					continue
				}
			}

			annotations = append(annotations, ExportAnnotation{
				ID:         annID,
				ImageKey:   imageKey,
				CategoryID: catIDMap[cat.CategoryID],
				Type:       exportType,
				Data:       exportData,
			})
			annCount++
		}
		rows.Close()
		db.Close()
		log.Printf("[PrepareDataset] Category %s: %d annotations", cat.CategoryName, annCount)
	}

	log.Printf("[PrepareDataset] Total: %d images, %d categories, %d annotations",
		len(images), len(categories), len(annotations))

	if len(images) == 0 {
		return "", fmt.Errorf("no images found in trainset")
	}

	// 5. 鎸夋瘮渚嬪垎閰?train/val
	trainPercent := int(trainRatio * 100)
	valPercent := 100 - trainPercent
	trainCount := len(images) * trainPercent / 100

	for i := range images {
		if i < trainCount {
			images[i].Split = "train"
		} else {
			images[i].Split = "val"
		}
	}

	// 6. 璋冪敤鎻掍欢瀵煎嚭
	pluginInput := map[string]interface{}{
		"format":      "yolo",
		"outputDir":   outputDir,
		"images":      images,
		"categories":  categories,
		"annotations": annotations,
		"split": map[string]int{
			"train": trainPercent,
			"val":   valPercent,
			"test":  0,
		},
	}

	// 璋冭瘯锛氭鏌ユ爣娉ㄦ暟鎹槸鍚﹀寘鍚?keypoints
	for i, ann := range annotations {
		if i < 3 { // 鍙墦鍗板墠3涓?
			dataJSON, _ := json.Marshal(ann.Data)
			log.Printf("[PrepareDataset] Ann %d: Type=%s, Data=%s", i, ann.Type, string(dataJSON))
		}
	}

	inputJSON, _ := json.Marshal(pluginInput)
	log.Printf("[PrepareDataset] Categories: %+v", categories)

	entryPath := filepath.Join(pluginPath, entryName)
	if runtime.GOOS == "windows" && !strings.HasSuffix(entryPath, ".exe") {
		entryPath += ".exe"
	}

	log.Printf("[PrepareDataset] Calling plugin: %s export", entryPath)

	cmd := exec.Command(entryPath, "export")
	cmd.Stdin = strings.NewReader(string(inputJSON))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("[PrepareDataset] Plugin failed: %v, stderr: %s", err, stderr.String())
		return "", fmt.Errorf("plugin execution failed: %v", err)
	}

	// 7. 瑙ｆ瀽鎻掍欢鍝嶅簲骞舵墽琛屾枃浠舵搷浣?
	var pluginResp struct {
		Success   bool `json:"success"`
		Structure struct {
			Directories []string `json:"directories"`
			Files       []struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			} `json:"files"`
			CopyImages []struct {
				From string `json:"from"`
				To   string `json:"to"`
			} `json:"copyImages"`
		} `json:"structure"`
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &pluginResp); err != nil {
		log.Printf("[PrepareDataset] Failed to parse plugin response: %v", err)
		return "", fmt.Errorf("invalid plugin response: %v", err)
	}

	if !pluginResp.Success {
		errMsg := "unknown error"
		if len(pluginResp.Errors) > 0 {
			errMsg = pluginResp.Errors[0].Message
		}
		return "", fmt.Errorf("plugin error: %s", errMsg)
	}

	// 鍒涘缓鐩綍
	for _, dir := range pluginResp.Structure.Directories {
		os.MkdirAll(filepath.Join(outputDir, dir), 0755)
	}

	// 鍐欏叆鏂囦欢锛堝寘鎷?data.yaml锛?
	for _, file := range pluginResp.Structure.Files {
		filePath := filepath.Join(outputDir, file.Path)
		os.WriteFile(filePath, []byte(file.Content), 0644)
	}

	// 澶嶅埗鍥剧墖
	for _, cp := range pluginResp.Structure.CopyImages {
		destPath := filepath.Join(outputDir, cp.To)
		copyFile(cp.From, destPath)
	}

	log.Printf("[PrepareDataset] Dataset exported to: %s (train: %d, val: %d)",
		outputDir, trainCount, len(images)-trainCount)

	return outputDir, nil
}

// convertToBbox 灏嗕笉鍚岀被鍨嬬殑鏍囨敞杞崲涓?bbox 鏍煎紡
func convertToBbox(annoType string, data map[string]interface{}) map[string]interface{} {
	switch annoType {
	case "bbox":
		// 宸茬粡鏄?bbox 鏍煎紡锛歿x, y, width, height}
		return data

	case "polygon":
		// polygon 鏍煎紡锛歿points: [[x1,y1], [x2,y2], ...]}
		// 杞崲涓哄鎺ョ煩褰?
		points, ok := data["points"].([]interface{})
		if !ok || len(points) == 0 {
			return nil
		}

		minX, minY := math.MaxFloat64, math.MaxFloat64
		maxX, maxY := -math.MaxFloat64, -math.MaxFloat64

		for _, p := range points {
			pt, ok := p.([]interface{})
			if !ok || len(pt) < 2 {
				continue
			}
			x, _ := toFloat(pt[0])
			y, _ := toFloat(pt[1])
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}

		if minX == math.MaxFloat64 {
			return nil
		}

		return map[string]interface{}{
			"x":      minX,
			"y":      minY,
			"width":  maxX - minX,
			"height": maxY - minY,
		}

	case "keypoint":
		// keypoint 鏍煎紡锛歿points: [[x1,y1,v1], [x2,y2,v2], ...]}
		// 杞崲涓哄鎺ョ煩褰?
		points, ok := data["points"].([]interface{})
		if !ok || len(points) == 0 {
			return nil
		}

		minX, minY := math.MaxFloat64, math.MaxFloat64
		maxX, maxY := -math.MaxFloat64, -math.MaxFloat64

		for _, p := range points {
			pt, ok := p.([]interface{})
			if !ok || len(pt) < 2 {
				continue
			}
			x, _ := toFloat(pt[0])
			y, _ := toFloat(pt[1])
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}

		if minX == math.MaxFloat64 {
			return nil
		}

		return map[string]interface{}{
			"x":      minX,
			"y":      minY,
			"width":  maxX - minX,
			"height": maxY - minY,
		}

	default:
		return nil
	}
}

// toFloat 灏?interface{} 杞崲涓?float64
func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

// getProjectPath 鑾峰彇椤圭洰璺緞
func getProjectPath(projectID string) string {
	dataPath := getDataPath()
	// 椤圭洰瀛樺偍鍦?project_item/{projectId} 鐩綍涓?
	projectDir := filepath.Join(dataPath, "project_item", projectID)
	if _, err := os.Stat(projectDir); err == nil {
		return projectDir
	}
	return ""
}

// ============ 璁粌浠诲姟绠＄悊 ============

// TrainingTask 璁粌浠诲姟
type TrainingTask struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	PluginID    string                 `json:"pluginId"`
	PluginPath  string                 `json:"pluginPath"`
	Entry       string                 `json:"entry"`
	Status      string                 `json:"status"` // pending, running, completed, error
	Params      map[string]interface{} `json:"params"`
	Progress    map[string]interface{} `json:"progress,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Logs        []string               `json:"logs,omitempty"`    // 鍐呭瓨涓殑鏈€鏂版棩蹇楋紙闄愬埗鏁伴噺锛?
	LogFile     string                 `json:"logFile,omitempty"` // 鏃ュ織鏂囦欢璺緞
	LogIndex    int                    `json:"-"`                 // 鍐呴儴浣跨敤
	Process     *exec.Cmd              `json:"-"`
	SocketPort  int                    `json:"-"`
	DatasetPath string                 `json:"-"`                    // 涓存椂鏁版嵁闆嗙洰褰曪紙璁粌瀹屾垚鍚庡垹闄わ級
	OutputPath  string                 `json:"outputPath,omitempty"` // 妯″瀷杈撳嚭鐩綍
	StartedAt   time.Time              `json:"startedAt"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
}

var trainingTasks = struct {
	sync.RWMutex
	tasks map[string]*TrainingTask
	queue []string // 绛夊緟鎵ц鐨勪换鍔￠槦鍒楋紙鎸夊姞鍏ラ『搴忥級
}{tasks: make(map[string]*TrainingTask)}

// hasRunningTask 妫€鏌ユ槸鍚︽湁姝ｅ湪杩愯鐨勪换鍔?
func hasRunningTask() bool {
	for _, t := range trainingTasks.tasks {
		if t.Status == "running" {
			return true
		}
	}
	return false
}

// getNextPendingTaskID 鑾峰彇闃熷垪涓笅涓€涓緟鎵ц浠诲姟鐨処D
func getNextPendingTaskID() string {
	if len(trainingTasks.queue) == 0 {
		return ""
	}
	taskID := trainingTasks.queue[0]
	trainingTasks.queue = trainingTasks.queue[1:]
	return taskID
}

// startNextPendingTask 鍚姩闃熷垪涓殑涓嬩竴涓緟鎵ц浠诲姟
func startNextPendingTask() {
	trainingTasks.Lock()
	// 妫€鏌ユ槸鍚﹁繕鏈夋鍦ㄨ繍琛岀殑浠诲姟
	if hasRunningTask() {
		trainingTasks.Unlock()
		return
	}
	// 鑾峰彇闃熷垪涓殑涓嬩竴涓换鍔?
	nextTaskID := getNextPendingTaskID()
	if nextTaskID == "" {
		trainingTasks.Unlock()
		return
	}
	// 鑾峰彇浠诲姟淇℃伅
	task, ok := trainingTasks.tasks[nextTaskID]
	if !ok || task.Status != "pending" {
		trainingTasks.Unlock()
		return
	}
	// 澶嶅埗浠诲姟淇℃伅
	taskID := task.ID
	taskName := task.Name
	pluginID := task.PluginID
	pluginPath := task.PluginPath
	entryName := task.Entry
	params := task.Params
	trainingTasks.Unlock()

	log.Printf("[Training] Starting queued task: %s", taskID)
	if err := startTrainingTask(taskID, taskName, pluginID, pluginPath, entryName, params); err != nil {
		log.Printf("[Training] Failed to start queued task %s: %v", taskID, err)
		// 鏍囪浠诲姟涓洪敊璇姸鎬?
		trainingTasks.Lock()
		if t, ok := trainingTasks.tasks[taskID]; ok {
			t.Status = "error"
			t.Error = "Failed to start: " + err.Error()
		}
		trainingTasks.Unlock()
		broadcastTrainingUpdate(taskID, "error")
	}
}

// ============ 全局 WebSocket ============
var globalWSUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// globalWSClient 封装全局 WebSocket 连接
type globalWSClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *globalWSClient) WriteJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}

var globalWSClients = struct {
	sync.RWMutex
	clients map[*globalWSClient]bool
}{clients: make(map[*globalWSClient]bool)}

// GlobalWSMessage 全局 WebSocket 消息
type GlobalWSMessage struct {
	Type     string      `json:"type"`
	TaskID   string      `json:"taskId,omitempty"`
	PluginID string      `json:"pluginId,omitempty"`
	Message  string      `json:"message,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	Success  bool        `json:"success,omitempty"`
}

// 正在进行的 pip 安装任务
type pipTask struct {
	PluginID string
	Cmd      *exec.Cmd
}

var pipInstallingTasks = struct {
	sync.RWMutex
	tasks map[string]*pipTask // taskId -> task info
}{tasks: make(map[string]*pipTask)}

// handleGlobalWS 处理全局 WebSocket 连接
func handleGlobalWS(w http.ResponseWriter, r *http.Request) {
	conn, err := globalWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[GlobalWS] Upgrade failed: %v", err)
		return
	}

	client := &globalWSClient{conn: conn}
	globalWSClients.Lock()
	globalWSClients.clients[client] = true
	globalWSClients.Unlock()

	log.Printf("[GlobalWS] Client connected, total: %d", len(globalWSClients.clients))

	// 发送当前正在进行的安装任务
	pipInstallingTasks.RLock()
	installingPlugins := make([]string, 0)
	for _, task := range pipInstallingTasks.tasks {
		installingPlugins = append(installingPlugins, task.PluginID)
	}
	pipInstallingTasks.RUnlock()

	if len(installingPlugins) > 0 {
		client.WriteJSON(GlobalWSMessage{
			Type: "pip_installing",
			Data: installingPlugins,
		})
	}

	defer func() {
		globalWSClients.Lock()
		delete(globalWSClients.clients, client)
		globalWSClients.Unlock()
		conn.Close()
		log.Printf("[GlobalWS] Client disconnected, total: %d", len(globalWSClients.clients))
	}()

	// 保持连接，处理客户端消息
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// broadcastGlobalWS 广播消息到所有全局 WebSocket 客户端
func broadcastGlobalWS(msg GlobalWSMessage) {
	globalWSClients.RLock()
	defer globalWSClients.RUnlock()

	for client := range globalWSClients.clients {
		if err := client.WriteJSON(msg); err != nil {
			log.Printf("[GlobalWS] Write error: %v", err)
		}
	}
}

// ============ 训练 WebSocket ============
var trainingWSUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// wsClient 封装训练 WebSocket 连接
type wsClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *wsClient) WriteJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}

var trainingWSClients = struct {
	sync.RWMutex
	clients map[*wsClient]bool
}{clients: make(map[*wsClient]bool)}

// handleTrainingWS 澶勭悊璁粌 WebSocket 杩炴帴
func handleTrainingWS(w http.ResponseWriter, r *http.Request) {
	conn, err := trainingWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[TrainingWS] Upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// 鍖呰杩炴帴锛屾坊鍔犲啓鍏ラ攣
	client := &wsClient{conn: conn}

	// 娉ㄥ唽瀹㈡埛绔?
	trainingWSClients.Lock()
	trainingWSClients.clients[client] = true
	trainingWSClients.Unlock()

	log.Printf("[TrainingWS] Client connected, total: %d", len(trainingWSClients.clients))

	// 绔嬪嵆鍙戦€佸綋鍓嶆墍鏈変换鍔＄姸鎬?
	trainingTasks.RLock()
	tasks := make([]*TrainingTask, 0, len(trainingTasks.tasks))
	for _, t := range trainingTasks.tasks {
		tasks = append(tasks, t)
	}
	trainingTasks.RUnlock()

	client.WriteJSON(map[string]interface{}{
		"type":  "init",
		"tasks": tasks,
	})

	// 鍚姩璧勬簮鐩戞帶骞挎挱锛堟瘡2绉掞級
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				stats := getSystemStats()
				client.WriteJSON(map[string]interface{}{
					"type":  "resources",
					"stats": stats,
				})
			}
		}
	}()

	// 淇濇寔杩炴帴锛岀洃鍚叧闂?
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
	close(done)

	// 娉ㄩ攢瀹㈡埛绔?
	trainingWSClients.Lock()
	delete(trainingWSClients.clients, client)
	trainingWSClients.Unlock()
	log.Printf("[TrainingWS] Client disconnected, total: %d", len(trainingWSClients.clients))
}

// broadcastTrainingUpdate 骞挎挱璁粌澧為噺鏇存柊
func broadcastTrainingUpdate(taskID string, updateType string) {
	trainingWSClients.RLock()
	defer trainingWSClients.RUnlock()

	if len(trainingWSClients.clients) == 0 {
		return
	}

	trainingTasks.RLock()
	task, ok := trainingTasks.tasks[taskID]
	if !ok {
		trainingTasks.RUnlock()
		return
	}

	// 鏋勫缓澧為噺娑堟伅锛屼笉鍙戦€佸畬鏁存棩蹇?
	msg := map[string]interface{}{
		"type":   updateType,
		"taskId": taskID,
	}

	switch updateType {
	case "progress":
		// 鍙彂閫佽繘搴﹀拰鎸囨爣
		msg["progress"] = task.Progress
		msg["status"] = task.Status
	case "log":
		// 鍙彂閫佹渶鏂颁竴鏉℃棩蹇?
		if len(task.Logs) > 0 {
			msg["log"] = task.Logs[len(task.Logs)-1]
			msg["logIndex"] = len(task.Logs) - 1
		}
	case "epoch_end":
		// 鍙戦€侀獙璇佹寚鏍?
		msg["progress"] = task.Progress
	case "done", "error":
		// 鐘舵€佸彉鏇达紝鍙戦€佺姸鎬佸拰缁撴灉
		msg["status"] = task.Status
		msg["error"] = task.Error
		msg["result"] = task.Result
		msg["completedAt"] = task.CompletedAt
		msg["outputPath"] = task.OutputPath
	}
	trainingTasks.RUnlock()

	for client := range trainingWSClients.clients {
		if err := client.WriteJSON(msg); err != nil {
			log.Printf("[TrainingWS] Write failed: %v", err)
		}
	}
}

// handleStartTraining 鍚姩璁粌浠诲姟
func handleStartTraining(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TaskID     string                 `json:"taskId"`
		TaskName   string                 `json:"taskName"`
		PluginID   string                 `json:"pluginId"`
		PluginPath string                 `json:"pluginPath"`
		Entry      string                 `json:"entry"`
		Params     map[string]interface{} `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Training] Request decode error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request", "message": "decode_failed"})
		return
	}
	if req.TaskID == "" || req.PluginPath == "" || req.Entry == "" {
		log.Printf("[Training] Invalid request: taskId=%s, pluginPath=%s, entry=%s", req.TaskID, req.PluginPath, req.Entry)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request", "message": "missing_fields"})
		return
	}

	// 检查插件虚拟环境是否存在（优先使用插件虚拟环境，不存在则检查全局虚拟环境）
	dataPath := getDataPath()
	pluginVenvPath := filepath.Join(dataPath, "plugins_python_venv", req.PluginID)
	pluginPython := filepath.Join(pluginVenvPath, "Scripts", "python.exe")
	if _, err := os.Stat(pluginPython); os.IsNotExist(err) {
		// 插件虚拟环境不存在，检查全局虚拟环境
		if !isPythonVenvDeployed() {
			log.Printf("[Training] No Python environment found for plugin %s", req.PluginID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "python_not_deployed"})
			return
		}
	}

	// 鏍规嵁 pluginId 浼樺厛浣跨敤宸插畨瑁呯殑 training 鎻掍欢璺緞鍜屽叆鍙?
	pluginPath := req.PluginPath
	entryName := req.Entry
	if req.PluginID != "" {
		installed, err := loadInstalledPlugins()
		if err != nil {
			log.Printf("[Training] loadInstalledPlugins error in handleStartTraining: %v", err)
		} else {
			for _, p := range installed {
				if p.ID == req.PluginID && p.Type == "training" {
					pluginPath = p.Path
					entryName = p.Entry
					log.Printf("[Training] Using installed training plugin: %s at %s (entry=%s)", p.ID, pluginPath, entryName)
					break
				}
			}
		}
	}

	// 妫€鏌ユ槸鍚︽湁姝ｅ湪杩愯鐨勪换鍔★紝濡傛灉鏈夊垯鍔犲叆闃熷垪
	trainingTasks.Lock()
	if hasRunningTask() {
		// 鍒涘缓 pending 鐘舵€佺殑浠诲姟
		outputDir := filepath.Join(getDataPath(), "training_outputs", req.TaskID)
		os.MkdirAll(outputDir, 0755)
		task := &TrainingTask{
			ID:         req.TaskID,
			Name:       req.TaskName,
			PluginID:   req.PluginID,
			PluginPath: pluginPath,
			Entry:      entryName,
			Status:     "pending",
			Params:     req.Params,
			OutputPath: outputDir,
			StartedAt:  time.Now(),
		}
		trainingTasks.tasks[req.TaskID] = task
		trainingTasks.queue = append(trainingTasks.queue, req.TaskID)
		queuePos := len(trainingTasks.queue)
		trainingTasks.Unlock()

		log.Printf("[Training] Task queued: %s (position: %d)", req.TaskID, queuePos)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"taskId":   req.TaskID,
			"status":   "pending",
			"position": queuePos,
		})
		broadcastTrainingUpdate(req.TaskID, "status")
		return
	}
	trainingTasks.Unlock()

	// 娌℃湁姝ｅ湪杩愯鐨勪换鍔★紝鐩存帴鍚姩
	if err := startTrainingTask(req.TaskID, req.TaskName, req.PluginID, pluginPath, entryName, req.Params); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "start_failed", "message": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "taskId": req.TaskID, "status": "running"})
}

// startTrainingTask 瀹為檯鍚姩璁粌浠诲姟鐨勯€昏緫
func startTrainingTask(taskID, taskName, pluginID, pluginPath, entryName string, params map[string]interface{}) error {
	// 1. 鍒涘缓 Socket 鏈嶅姟鍣紙浣跨敤绔彛 0 璁╃郴缁熻嚜鍔ㄥ垎閰嶅彲鐢ㄧ鍙ｏ級
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Printf("[Training] Failed to create socket server: %v", err)
		return fmt.Errorf("socket_failed: %v", err)
	}
	// 鑾峰彇绯荤粺鍒嗛厤鐨勫疄闄呯鍙?
	socketPort := listener.Addr().(*net.TCPAddr).Port

	// 2. 构建启动命令（使用插件对应的虚拟环境）
	entryPath := filepath.Join(pluginPath, entryName)
	var cmd *exec.Cmd

	// 获取插件虚拟环境路径
	dataPath := getDataPath()
	pluginVenvPath := filepath.Join(dataPath, "plugins_python_venv", pluginID)
	pluginPython := filepath.Join(pluginVenvPath, "Scripts", "python.exe")

	// 检查插件虚拟环境是否存在，不存在则回退到全局虚拟环境
	if _, err := os.Stat(pluginPython); os.IsNotExist(err) {
		pluginVenvPath = getPythonVenvPath()
		pluginPython = filepath.Join(pluginVenvPath, "Scripts", "python.exe")
		log.Printf("[Training] Plugin venv not found, using global venv: %s", pluginVenvPath)
	} else {
		log.Printf("[Training] Using plugin venv: %s", pluginVenvPath)
	}

	if strings.HasSuffix(entryName, ".exe") {
		cmd = exec.Command(entryPath, "--socket-port", fmt.Sprintf("%d", socketPort), "--task-id", taskID)
		sitePackages := filepath.Join(pluginVenvPath, "Lib", "site-packages")
		cmd.Env = append(os.Environ(), "PYTHONPATH="+sitePackages)
	} else {
		cmd = exec.Command(pluginPython, entryPath, "--socket-port", fmt.Sprintf("%d", socketPort), "--task-id", taskID)
	}
	cmd.Dir = pluginPath

	// 3. 鍚姩杩涚▼
	if err := cmd.Start(); err != nil {
		listener.Close()
		log.Printf("[Training] Failed to start: %v", err)
		return fmt.Errorf("start_failed: %v", err)
	}

	// 璁板綍浠诲姟
	outputDir := filepath.Join(getDataPath(), "training_outputs", taskID)
	os.MkdirAll(outputDir, 0755)
	logFilePath := filepath.Join(outputDir, "training.log")

	// 鍐欏叆浠诲姟淇℃伅鏂囦欢锛堢敤浜庡巻鍙茶褰曟寔涔呭寲锛?
	taskInfoPath := filepath.Join(outputDir, "task_info.json")
	taskInfo := map[string]interface{}{
		"id":        taskID,
		"name":      taskName,
		"pluginId":  pluginID,
		"status":    "running",
		"params":    params,
		"startedAt": time.Now().Format(time.RFC3339),
	}
	if infoBytes, err := json.MarshalIndent(taskInfo, "", "  "); err == nil {
		os.WriteFile(taskInfoPath, infoBytes, 0644)
	}

	task := &TrainingTask{
		ID:         taskID,
		Name:       taskName,
		PluginID:   pluginID,
		PluginPath: pluginPath,
		Entry:      entryName,
		Status:     "running",
		Params:     params,
		LogFile:    logFilePath,
		Process:    cmd,
		SocketPort: socketPort,
		OutputPath: outputDir,
		StartedAt:  time.Now(),
	}
	trainingTasks.Lock()
	trainingTasks.tasks[taskID] = task
	trainingTasks.Unlock()

	log.Printf("[Training] Task started: %s (port: %d, pid: %d)", taskID, socketPort, cmd.Process.Pid)

	// 4. 鍚庡彴澶勭悊 Socket 閫氫俊
	go func() {
		defer listener.Close()

		// 绛夊緟鎻掍欢杩炴帴锛堣秴鏃?10 绉掞級
		listener.(*net.TCPListener).SetDeadline(time.Now().Add(10 * time.Second))
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("[Training] Plugin connection timeout: %v", err)
			trainingTasks.Lock()
			if t, ok := trainingTasks.tasks[taskID]; ok {
				t.Status = "error"
				t.Error = "Plugin connection timeout"
			}
			trainingTasks.Unlock()
			cmd.Process.Kill()
			return
		}
		defer conn.Close()
		log.Printf("[Training] Plugin connected: %s", taskID)

		// 鍑嗗鏁版嵁闆嗚矾寰?
		dataPath := getDataPath()
		var datasetPath string

		// 濡傛灉鏄粠璁粌闆嗚缁冿紝闇€瑕佸噯澶囨暟鎹?
		if sourceType, _ := params["sourceType"].(string); sourceType == "trainset" {
			trainsetID, _ := params["trainsetId"].(string)
			trainRatio, _ := params["trainRatio"].(float64)
			if trainRatio == 0 {
				trainRatio = 0.8
			}

			preparedPath, err := prepareTrainingDataset(trainsetID, taskID, trainRatio)
			if err != nil {
				log.Printf("[Training] Failed to prepare dataset: %v", err)
				trainingTasks.Lock()
				if t, ok := trainingTasks.tasks[taskID]; ok {
					t.Status = "error"
					t.Error = "Failed to prepare dataset: " + err.Error()
				}
				trainingTasks.Unlock()

				// 鍙戦€侀敊璇秷鎭粰鎻掍欢
				errMsg := map[string]interface{}{
					"type":    "STOP_TRAIN",
					"payload": map[string]string{"reason": "dataset_preparation_failed"},
				}
				msgBytes, _ := json.Marshal(errMsg)
				conn.Write(append(msgBytes, '\n'))
				return
			}
			datasetPath = preparedPath
			// 瀛樺偍涓存椂鏁版嵁闆嗚矾寰勶紝渚夸簬璁粌瀹屾垚鍚庢竻鐞?
			trainingTasks.Lock()
			if t, ok := trainingTasks.tasks[taskID]; ok {
				t.DatasetPath = preparedPath
			}
			trainingTasks.Unlock()
		} else {
			// 浠庣洰褰曡缁冿細妫€鏌ュ苟鑷姩鍒嗗壊鏁版嵁闆?
			rawDatasetPath, _ := params["datasetPath"].(string)
			trainRatio, _ := params["trainRatio"].(float64)
			if trainRatio == 0 {
				trainRatio = 0.8
			}

			preparedPath, err := prepareDirectoryDataset(rawDatasetPath, taskID, trainRatio)
			if err != nil {
				log.Printf("[Training] Failed to prepare directory dataset: %v", err)
				errMsg := "Failed to prepare dataset: " + err.Error()
				trainingTasks.Lock()
				if t, ok := trainingTasks.tasks[taskID]; ok {
					t.Status = "error"
					t.Error = errMsg
					// 将错误信息添加到日志中，以便前端显示
					t.Logs = append(t.Logs, "[ERROR] "+errMsg)
				}
				trainingTasks.Unlock()
				broadcastTrainingUpdate(taskID, "log")
				broadcastTrainingUpdate(taskID, "error")

				// 发送错误消息给插件
				stopMsg := map[string]interface{}{
					"type":    "STOP_TRAIN",
					"payload": map[string]string{"reason": "dataset_preparation_failed"},
				}
				msgBytes, _ := json.Marshal(stopMsg)
				conn.Write(append(msgBytes, '\n'))
				return
			}
			datasetPath = preparedPath
			// 如果生成了新的临时目录，记录用于清理
			if preparedPath != rawDatasetPath {
				trainingTasks.Lock()
				if t, ok := trainingTasks.tasks[taskID]; ok {
					t.DatasetPath = preparedPath
				}
				trainingTasks.Unlock()
			}
		}

		// 鍑嗗璁粌閰嶇疆
		trainConfig := map[string]interface{}{
			"datasetPath": datasetPath,
			"outputPath":  filepath.Join(dataPath, "training_outputs", taskID),
			"modelPath":   filepath.Join(getModelsDir(), fmt.Sprintf("%v", params["modelPath"])),
			"params":      params,
		}

		// 鍙戦€?START_TRAIN 娑堟伅
		startMsg := map[string]interface{}{
			"type":    "START_TRAIN",
			"payload": trainConfig,
		}
		msgBytes, _ := json.Marshal(startMsg)
		conn.Write(append(msgBytes, '\n'))
		log.Printf("[Training] Sent START_TRAIN to plugin")

		// 澶勭悊鎻掍欢娑堟伅
		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			var msg map[string]interface{}
			if json.Unmarshal([]byte(line), &msg) != nil {
				continue
			}

			msgType, _ := msg["type"].(string)
			payload, _ := msg["payload"].(map[string]interface{})

			switch msgType {
			case "LOG":
				level, _ := payload["level"].(string)
				text, _ := payload["text"].(string)
				log.Printf("[Training][%s] %s: %s", taskID, level, text)
			case "OUTPUT":
				// 璁粌鑴氭湰鐨?stdout/stderr 杈撳嚭
				text, _ := payload["text"].(string)
				log.Printf("[Training][%s] > %s", taskID, text)
				// 淇濆瓨鍒颁换鍔℃棩蹇椾腑骞跺啓鍏ユ枃浠?
				trainingTasks.Lock()
				if t, ok := trainingTasks.tasks[taskID]; ok {
					t.Logs = append(t.Logs, text)
					// 闄愬埗鍐呭瓨涓棩蹇楁暟閲?
					if len(t.Logs) > 200 {
						t.Logs = t.Logs[len(t.Logs)-100:]
					}
					// 杩藉姞鍐欏叆鏃ュ織鏂囦欢
					if t.LogFile != "" {
						if f, err := os.OpenFile(t.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
							f.WriteString(time.Now().Format("15:04:05") + " " + text + "\n")
							f.Close()
						}
					}
				}
				trainingTasks.Unlock()
				broadcastTrainingUpdate(taskID, "log")
			case "PROGRESS":
				trainingTasks.Lock()
				if t, ok := trainingTasks.tasks[taskID]; ok {
					// 鎻掍欢宸茬粡璁＄畻濂?progress锛岀洿鎺ヤ娇鐢?
					// 淇濈暀涔嬪墠鐨勯獙璇佹寚鏍?
					if t.Progress != nil {
						if mAP50, ok := t.Progress["mAP50"]; ok {
							payload["mAP50"] = mAP50
						}
						if mAP5095, ok := t.Progress["mAP5095"]; ok {
							payload["mAP5095"] = mAP5095
						}
						if valLoss, ok := t.Progress["valLoss"]; ok {
							payload["valLoss"] = valLoss
						}
					}
					t.Progress = payload
				}
				trainingTasks.Unlock()
				broadcastTrainingUpdate(taskID, "progress")
			case "EPOCH_END":
				trainingTasks.Lock()
				if t, ok := trainingTasks.tasks[taskID]; ok {
					// 鍚堝苟楠岃瘉鎸囨爣鍒?Progress
					if t.Progress == nil {
						t.Progress = make(map[string]interface{})
					}
					// 鎸囨爣鍦?payload.metrics 涓紙鎻掍欢鍙戦€佺殑缁撴瀯锛?
					if metrics, ok := payload["metrics"].(map[string]interface{}); ok {
						if mAP50, ok := metrics["mAP50"]; ok {
							t.Progress["mAP50"] = mAP50
						}
						if mAP5095, ok := metrics["mAP5095"]; ok {
							t.Progress["mAP5095"] = mAP5095
						}
						if valLoss, ok := metrics["valLoss"]; ok {
							t.Progress["valLoss"] = valLoss
						}
					}
					if epoch, ok := payload["epoch"]; ok {
						t.Progress["epoch"] = epoch
					}
				}
				trainingTasks.Unlock()
				broadcastTrainingUpdate(taskID, "epoch_end")
			case "DONE":
				trainingTasks.Lock()
				var datasetPathToDelete string
				var taskOutputPath string
				var taskName string
				if t, ok := trainingTasks.tasks[taskID]; ok {
					t.Status = "completed"
					t.Result = payload
					now := time.Now()
					t.CompletedAt = &now
					datasetPathToDelete = t.DatasetPath
					taskOutputPath = t.OutputPath
					taskName = t.Name
				}
				trainingTasks.Unlock()
				broadcastTrainingUpdate(taskID, "done")
				log.Printf("[Training] Task completed: %s", taskID)

				// 鏇存柊 task_info.json 涓哄凡瀹屾垚鐘舵€?
				if taskOutputPath != "" {
					taskInfoPath := filepath.Join(taskOutputPath, "task_info.json")
					taskInfo := map[string]interface{}{
						"id":          taskID,
						"name":        taskName,
						"status":      "completed",
						"completedAt": time.Now().Format(time.RFC3339),
						"result":      payload,
					}
					if infoBytes, err := json.MarshalIndent(taskInfo, "", "  "); err == nil {
						os.WriteFile(taskInfoPath, infoBytes, 0644)
					}
				}

				// 璁粌鎴愬姛瀹屾垚鍚庯紝鍒犻櫎涓存椂鏁版嵁闆嗙洰褰?
				if datasetPathToDelete != "" {
					if err := os.RemoveAll(datasetPathToDelete); err != nil {
						log.Printf("[Training] Failed to cleanup temp dataset: %v", err)
					} else {
						log.Printf("[Training] Cleaned up temp dataset: %s", datasetPathToDelete)
					}
				}
			case "ERROR":
				errMsg, _ := payload["message"].(string)
				trainingTasks.Lock()
				if t, ok := trainingTasks.tasks[taskID]; ok {
					t.Status = "error"
					t.Error = errMsg
				}
				trainingTasks.Unlock()
				broadcastTrainingUpdate(taskID, "error")
				log.Printf("[Training] Task error: %s - %s", taskID, errMsg)
			}
		}

		// 绛夊緟杩涚▼缁撴潫
		cmd.Wait()
		trainingTasks.Lock()
		if t, ok := trainingTasks.tasks[taskID]; ok && t.Status == "running" {
			t.Status = "completed"
			now := time.Now()
			t.CompletedAt = &now
		}
		trainingTasks.Unlock()
		log.Printf("[Training] Task finished: %s", taskID)

		// 浠诲姟瀹屾垚鍚庯紝妫€鏌ュ苟鍚姩闃熷垪涓殑涓嬩竴涓换鍔?
		startNextPendingTask()
	}()

	return nil
}

// handleStopTraining 鍋滄璁粌浠诲姟
func handleStopTraining(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TaskID string `json:"taskId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TaskID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
		return
	}

	trainingTasks.Lock()
	task, ok := trainingTasks.tasks[req.TaskID]
	trainingTasks.Unlock()

	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "task_not_found"})
		return
	}

	if task.Process != nil && task.Process.Process != nil {
		task.Process.Process.Kill()
		task.Status = "failed"
		log.Printf("[Training] Task stopped: %s", req.TaskID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// handleTrainingTasks 鑾峰彇璁粌浠诲姟鍒楄〃
func handleTrainingTasks(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	trainingTasks.RLock()
	tasks := make([]*TrainingTask, 0, len(trainingTasks.tasks))
	for _, t := range trainingTasks.tasks {
		tasks = append(tasks, t)
	}
	trainingTasks.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tasks": tasks})
}

// handleTrainingHistory 鑾峰彇璁粌鍘嗗彶璁板綍锛堟壂鎻忚緭鍑虹洰褰曚腑鐨?task_info.json锛?
func handleTrainingHistory(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	historyDir := filepath.Join(getDataPath(), "training_outputs")
	var history []map[string]interface{}

	// 鎵弿鐩綍
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		// 鐩綍涓嶅瓨鍦ㄦ椂杩斿洖绌哄垪琛?
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"history": []map[string]interface{}{}})
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		taskInfoPath := filepath.Join(historyDir, entry.Name(), "task_info.json")
		data, err := os.ReadFile(taskInfoPath)
		if err != nil {
			continue
		}
		var info map[string]interface{}
		if json.Unmarshal(data, &info) != nil {
			continue
		}
		// 鍙繑鍥炲凡瀹屾垚鐨勪换鍔?
		if status, ok := info["status"].(string); ok && status == "completed" {
			info["outputPath"] = filepath.Join(historyDir, entry.Name())
			history = append(history, info)
		}
	}

	// 鎸夊畬鎴愭椂闂村€掑簭鎺掑簭
	sort.Slice(history, func(i, j int) bool {
		ti, _ := history[i]["completedAt"].(string)
		tj, _ := history[j]["completedAt"].(string)
		return ti > tj
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"history": history})
}

// handleDeleteTrainingTask 鍒犻櫎璁粌浠诲姟锛堝寘鎷ā鍨嬭緭鍑虹洰褰曪級
func handleDeleteTrainingTask(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TaskID       string `json:"taskId"`
		DeleteOutput bool   `json:"deleteOutput"` // 鏄惁鍒犻櫎杈撳嚭鐩綍
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
		return
	}

	var outputPath, datasetPath string

	trainingTasks.Lock()
	task, ok := trainingTasks.tasks[req.TaskID]
	if ok {
		// 濡傛灉浠诲姟杩樺湪杩愯锛屽厛鍋滄瀹?
		if task.Status == "running" && task.Process != nil && task.Process.Process != nil {
			task.Process.Process.Kill()
		}
		outputPath = task.OutputPath
		datasetPath = task.DatasetPath
		delete(trainingTasks.tasks, req.TaskID)
	}
	trainingTasks.Unlock()

	// 濡傛灉鍐呭瓨涓病鏈夋壘鍒颁换鍔★紝灏濊瘯浠庣鐩樻煡鎵撅紙鐢ㄤ簬鍚庣閲嶅惎鍚庡垹闄ゅ巻鍙茶褰曪級
	if !ok {
		historyDir := filepath.Join(getDataPath(), "training_outputs")
		possiblePath := filepath.Join(historyDir, req.TaskID)
		taskInfoPath := filepath.Join(possiblePath, "task_info.json")
		if _, err := os.Stat(taskInfoPath); err == nil {
			// 鎵惧埌浜嗗搴旂殑杈撳嚭鐩綍
			outputPath = possiblePath
			ok = true
		}
	}

	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "task_not_found"})
		return
	}

	// 鍒犻櫎杈撳嚭鐩綍锛堟ā鍨嬬洰褰曪級
	if req.DeleteOutput && outputPath != "" {
		if err := os.RemoveAll(outputPath); err != nil {
			log.Printf("[Training] Failed to delete output dir: %v", err)
		} else {
			log.Printf("[Training] Deleted output dir: %s", outputPath)
		}
	}

	// 鍚屾椂娓呯悊涓存椂鏁版嵁闆嗙洰褰曪紙濡傛灉杩樺瓨鍦級
	if datasetPath != "" {
		os.RemoveAll(datasetPath)
	}

	log.Printf("[Training] Task deleted: %s", req.TaskID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// ==================== 妯″瀷鎺ㄧ悊杈呭姪 ====================

// handleTrainingOutputs 鑾峰彇宸茶缁冩ā鍨嬪垪琛?
func handleTrainingOutputs(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"outputs": []interface{}{}})
		return
	}

	outputsDir := filepath.Join(cfg.DataPath, "training_outputs")
	entries, err := os.ReadDir(outputsDir)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"outputs": []interface{}{}})
		return
	}

	var outputs []map[string]interface{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		taskDir := filepath.Join(outputsDir, entry.Name(), "train", "weights")
		bestModel := filepath.Join(taskDir, "best.pt")
		if _, err := os.Stat(bestModel); err == nil {
			info, _ := entry.Info()
			outputs = append(outputs, map[string]interface{}{
				"taskId":    entry.Name(),
				"name":      entry.Name(),
				"modelPath": bestModel,
				"createdAt": info.ModTime().Format(time.RFC3339),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"outputs": outputs})
}

// handleInferenceEnsureDir 纭繚鎺ㄧ悊妯″瀷鐩綍瀛樺湪
func handleInferenceEnsureDir(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	inferenceDir := filepath.Join(cfg.DataPath, "input_model")
	if err := os.MkdirAll(inferenceDir, 0755); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "create_dir_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "path": inferenceDir})
}

// handleInferenceModels 鑾峰彇瀵煎叆鐨勬ā鍨嬪垪琛?
func handleInferenceModels(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"models": []interface{}{}})
		return
	}

	inferenceDir := filepath.Join(cfg.DataPath, "input_model")
	entries, err := os.ReadDir(inferenceDir)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"models": []interface{}{}})
		return
	}

	var models []map[string]interface{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".pt" && ext != ".onnx" {
			continue
		}
		info, _ := entry.Info()
		models = append(models, map[string]interface{}{
			"name":      name,
			"path":      filepath.Join(inferenceDir, name),
			"createdAt": info.ModTime().Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"models": models})
}

// handleInferenceImportModel 瀵煎叆妯″瀷鏂囦欢
func handleInferenceImportModel(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	inferenceDir := filepath.Join(cfg.DataPath, "input_model")
	os.MkdirAll(inferenceDir, 0755)

	// 鑾峰彇鏂囦欢鍚?
	filename := filepath.Base(req.Path)
	destPath := filepath.Join(inferenceDir, filename)

	// 妫€鏌ユ槸鍚﹀凡瀛樺湪
	if _, err := os.Stat(destPath); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "file_exists"})
		return
	}

	// 澶嶅埗鏂囦欢
	if err := copyFile(req.Path, destPath); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "copy_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "name": filename})
}

// handleInferenceUploadModel 鐩存帴涓婁紶妯″瀷鏂囦欢鍐呭鍒版帹鐞嗙洰褰?
func handleInferenceUploadModel(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 瑙ｆ瀽 multipart 琛ㄥ崟锛岄檺鍒舵渶澶?1GB锛堟ā鍨嬩竴鑸笉浼氳繖涔堝ぇ锛?
	err := r.ParseMultipartForm(1 << 30)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_multipart"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "file_missing"})
		return
	}
	defer file.Close()

	name := header.Filename
	ext := strings.ToLower(filepath.Ext(name))
	if ext != ".pt" && ext != ".onnx" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "unsupported_extension"})
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	inferenceDir := filepath.Join(cfg.DataPath, "input_model")
	if err := os.MkdirAll(inferenceDir, 0755); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "mkdir_failed"})
		return
	}

	destPath := filepath.Join(inferenceDir, name)
	if _, err := os.Stat(destPath); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "file_exists"})
		return
	}

	out, err := os.Create(destPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "create_failed"})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "write_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "name": name})
}

// ============ 鎺ㄧ悊鏈嶅姟绠＄悊 ============

// 鎺ㄧ悊鏈嶅姟杩涚▼
var inferenceServer struct {
	sync.Mutex
	cmd     *exec.Cmd
	running bool
}

// handleInferenceStart 鍚姩鎺ㄧ悊鏈嶅姟
func handleInferenceStart(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	inferenceServer.Lock()
	defer inferenceServer.Unlock()

	// 妫€鏌ユ槸鍚﹀凡鍦ㄨ繍琛?
	if inferenceServer.running && inferenceServer.cmd != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "status": "already_running"})
		return
	}

	// 鑾峰彇 Python 铏氭嫙鐜璺緞
	venvPath := getPythonVenvPath()
	pythonExe := filepath.Join(venvPath, "Scripts", "python.exe")
	if _, err := os.Stat(pythonExe); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "python_not_deployed"})
		return
	}

	// 鎺ㄧ悊鏈嶅姟鑴氭湰璺緞锛堜笌璁粌鑴氭湰鍚岀洰褰曪級
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	cwd, _ := os.Getwd()

	// 灏濊瘯澶氫釜鍙兘鐨勮矾寰?
	candidateDirs := []string{
		filepath.Join(exeDir, "host-plugins", "train_python"),
		filepath.Join(filepath.Dir(exeDir), "host-plugins", "train_python"),
		filepath.Join(cwd, "host-plugins", "train_python"),
		filepath.Join(filepath.Dir(cwd), "host-plugins", "train_python"),
	}

	var pluginDir, scriptPath string
	for _, dir := range candidateDirs {
		sp := filepath.Join(dir, "inference_server.py")
		if _, err := os.Stat(sp); err == nil {
			pluginDir = dir
			scriptPath = sp
			break
		}
	}

	if scriptPath == "" {
		log.Printf("[Inference] Script not found in any of: %v", candidateDirs)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "script_not_found", "candidates": fmt.Sprintf("%v", candidateDirs)})
		return
	}

	// 璇诲彇鏁版嵁鐩綍閰嶇疆锛屼紶閫掔粰 Python 浣滀负鐜鍙橀噺锛岄伩鍏嶇‖缂栫爜璺緞
	pathsCfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "paths_config_unavailable"})
		return
	}

	// 鍚姩鎺ㄧ悊鏈嶅姟
	cmd := exec.Command(pythonExe, scriptPath, "--port", "18081")
	cmd.Dir = pluginDir
	// 灏嗘暟鎹牴鐩綍閫氳繃鐜鍙橀噺浼犵粰 Python锛屼緵鍏惰嚜琛屾嫾鎺ュ浘鐗囪矾寰?
	cmd.Env = append(os.Environ(), "EASYMARK_DATA_PATH="+pathsCfg.DataPath)
	// 灏?Python 杈撳嚭閲嶅畾鍚戝埌 Go 缁堢
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("[Inference] Starting server: %s %s --port 18081", pythonExe, scriptPath)
	log.Printf("[Inference] Working directory: %s", pluginDir)

	if err := cmd.Start(); err != nil {
		log.Printf("[Inference] Failed to start server: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "start_failed", "details": err.Error()})
		return
	}

	inferenceServer.cmd = cmd
	inferenceServer.running = true

	log.Printf("[Inference] Server started (pid: %d)", cmd.Process.Pid)

	// 鍚庡彴鐩戞帶杩涚▼鐘舵€?
	go func() {
		cmd.Wait()
		inferenceServer.Lock()
		inferenceServer.running = false
		inferenceServer.cmd = nil
		inferenceServer.Unlock()
		log.Printf("[Inference] Server stopped")
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "status": "started", "pid": cmd.Process.Pid})
}

// handleInferenceStop 鍋滄鎺ㄧ悊鏈嶅姟
func handleInferenceStop(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	inferenceServer.Lock()
	defer inferenceServer.Unlock()

	if !inferenceServer.running || inferenceServer.cmd == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "status": "not_running"})
		return
	}

	// 缁堟杩涚▼
	if err := inferenceServer.cmd.Process.Kill(); err != nil {
		log.Printf("[Inference] Failed to kill server: %v", err)
	}

	inferenceServer.running = false
	inferenceServer.cmd = nil

	log.Printf("[Inference] Server stopped by request")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "status": "stopped"})
}

// handleInferenceStatus 鑾峰彇鎺ㄧ悊鏈嶅姟鐘舵€?
func handleInferenceStatus(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	inferenceServer.Lock()
	running := inferenceServer.running
	var pid int
	if inferenceServer.cmd != nil && inferenceServer.cmd.Process != nil {
		pid = inferenceServer.cmd.Process.Pid
	}
	inferenceServer.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"running": running, "pid": pid})
}

// handleInferenceOpenFolder 鎵撳紑鎺ㄧ悊妯″瀷鐩綍
func handleInferenceOpenFolder(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	inferenceDir := filepath.Join(cfg.DataPath, "input_model")
	os.MkdirAll(inferenceDir, 0755)

	// 浣跨敤绯荤粺鍛戒护鎵撳紑鐩綍
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", inferenceDir)
	case "darwin":
		cmd = exec.Command("open", inferenceDir)
	default:
		cmd = exec.Command("xdg-open", inferenceDir)
	}
	cmd.Start()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// ==================== Python 环境管理（新接口） ====================

// PythonEnvSettings Python 环境设置
type PythonEnvSettings struct {
	PipSource   string `json:"pipSource"`
	HttpProxy   string `json:"httpProxy"`
	Socks5Proxy string `json:"socks5Proxy"`
}

var pythonEnvSettings = PythonEnvSettings{}
var pythonEnvSettingsMu sync.Mutex

// handlePythonVersion 获取系统 Python 版本
func handlePythonVersion(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 尝试获取 Python 版本
	cmd := exec.Command("python", "--version")
	output, err := cmd.CombinedOutput()

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		// 尝试 python3
		cmd = exec.Command("python3", "--version")
		output, err = cmd.CombinedOutput()
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"version": nil})
			return
		}
	}

	// 解析版本号，如 "Python 3.11.0"
	version := strings.TrimSpace(string(output))
	version = strings.TrimPrefix(version, "Python ")
	json.NewEncoder(w).Encode(map[string]interface{}{"version": version})
}

// handlePythonEnvStatus 获取插件虚拟环境状态和依赖列表
func handlePythonEnvStatus(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	pluginId := r.URL.Query().Get("pluginId")
	if pluginId == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "pluginId required"})
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	// 检查虚拟环境是否存在
	venvPath := filepath.Join(cfg.DataPath, "plugins_python_venv", pluginId)
	venvPython := filepath.Join(venvPath, "Scripts", "python.exe")
	if runtime.GOOS != "windows" {
		venvPython = filepath.Join(venvPath, "bin", "python")
	}
	hasVenv := false
	if _, err := os.Stat(venvPython); err == nil {
		hasVenv = true
	}

	// 加载插件清单获取依赖列表（根据插件 ID 查找正确的目录）
	pluginDir, err := getPluginDirByID(pluginId)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"hasVenv":      hasVenv,
			"dependencies": []interface{}{},
		})
		return
	}
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"hasVenv":      hasVenv,
			"dependencies": []interface{}{},
		})
		return
	}

	var manifest PluginManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"hasVenv":      hasVenv,
			"dependencies": []interface{}{},
		})
		return
	}

	// 获取 Python 依赖（支持 dependencies 或 requirements）
	var requiredDeps []string
	var pytorchDeps []string
	if manifest.Python != nil {
		// 先尝试 dependencies
		if deps, ok := manifest.Python["dependencies"].([]interface{}); ok {
			for _, d := range deps {
				if s, ok := d.(string); ok {
					requiredDeps = append(requiredDeps, s)
				}
			}
		}
		// 再尝试 requirements
		if len(requiredDeps) == 0 {
			if deps, ok := manifest.Python["requirements"].([]interface{}); ok {
				for _, d := range deps {
					if s, ok := d.(string); ok {
						requiredDeps = append(requiredDeps, s)
					}
				}
			}
		}
		// 获取 pytorch 依赖（新格式：packages 在根级别）
		if pytorch, ok := manifest.Python["pytorch"].(map[string]interface{}); ok {
			if pkgs, ok := pytorch["packages"].([]interface{}); ok {
				for _, p := range pkgs {
					if s, ok := p.(string); ok {
						pytorchDeps = append(pytorchDeps, s)
					}
				}
			}
		}
	}

	// 检查已安装的包
	dependencies := make([]map[string]interface{}, 0)
	installedPkgs := make(map[string]string)

	if hasVenv {
		// 列出已安装的包
		cmd := exec.Command(venvPython, "-m", "pip", "list", "--format=json")
		output, err := cmd.Output()
		if err == nil {
			var pkgs []struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			}
			if json.Unmarshal(output, &pkgs) == nil {
				for _, p := range pkgs {
					installedPkgs[strings.ToLower(p.Name)] = p.Version
				}
			}
		}
	}

	// 合并所有依赖
	allDeps := append(requiredDeps, pytorchDeps...)

	for _, dep := range allDeps {
		// 解析包名（去除版本约束）
		pkgName := strings.Split(dep, ">=")[0]
		pkgName = strings.Split(pkgName, "==")[0]
		pkgName = strings.Split(pkgName, "<")[0]
		pkgName = strings.TrimSpace(pkgName)

		installed := false
		version := ""
		if v, ok := installedPkgs[strings.ToLower(pkgName)]; ok {
			installed = true
			version = v
		}
		dependencies = append(dependencies, map[string]interface{}{
			"name":      dep,
			"installed": installed,
			"version":   version,
		})
	}

	// 检测 PyTorch 安装状态
	pytorchInstalled := false
	pytorchType := ""
	if torchVersion, ok := installedPkgs["torch"]; ok {
		pytorchInstalled = true
		// 检测是 GPU 还是 CPU 版本
		if strings.Contains(torchVersion, "cu") {
			pytorchType = "gpu"
		} else {
			pytorchType = "cpu"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"hasVenv":          hasVenv,
		"dependencies":     dependencies,
		"pytorchInstalled": pytorchInstalled,
		"pytorchType":      pytorchType,
	})
}

// handlePythonPluginsDepsSummary 检查是否存在依赖未安装的 Python 插件
// 用于在应用启动时给 Python 环境入口打橙色角标提示用户需要前往配置
func handlePythonPluginsDepsSummary(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	pluginsRoot := filepath.Join(cfg.DataPath, "plugins")
	entries, err := os.ReadDir(pluginsRoot)
	if err != nil {
		// 读取失败时按“无问题”处理，避免阻塞应用启动
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"hasProblem": false})
		return
	}

	hasProblem := false
	log.Printf("[PluginsDepsSummary] Checking plugins in: %s", pluginsRoot)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginID := entry.Name()
		log.Printf("[PluginsDepsSummary] Checking plugin: %s", pluginID)

		// 读取 manifest，获取实际的插件 ID 和 python 段
		manifestPath := filepath.Join(pluginsRoot, pluginID, "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		var manifest struct {
			ID     string                 `json:"id"`
			Python map[string]interface{} `json:"python"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil || manifest.Python == nil {
			continue
		}

		// 使用 manifest 中的实际 ID（虚拟环境是按此 ID 创建的）
		actualPluginID := manifest.ID
		if actualPluginID == "" {
			actualPluginID = pluginID // 回退到目录名
		}

		// 检查插件虚拟环境是否存在
		venvPath := filepath.Join(cfg.DataPath, "plugins_python_venv", actualPluginID)
		venvPython := filepath.Join(venvPath, "Scripts", "python.exe")
		if runtime.GOOS != "windows" {
			venvPython = filepath.Join(venvPath, "bin", "python")
		}
		if _, err := os.Stat(venvPython); err != nil {
			log.Printf("[PluginsDepsSummary] Plugin %s: venv not found at %s", pluginID, venvPython)
			// 没有虚拟环境也视为"有问题"
			hasProblem = true
			break
		}

		// 收集依赖：普通 requirements + pytorch packages
		var requiredDeps []string
		var pytorchDeps []string
		if manifest.Python != nil {
			if deps, ok := manifest.Python["dependencies"].([]interface{}); ok {
				for _, d := range deps {
					if s, ok := d.(string); ok {
						requiredDeps = append(requiredDeps, s)
					}
				}
			}
			if len(requiredDeps) == 0 {
				if deps, ok := manifest.Python["requirements"].([]interface{}); ok {
					for _, d := range deps {
						if s, ok := d.(string); ok {
							requiredDeps = append(requiredDeps, s)
						}
					}
				}
			}
			if pytorch, ok := manifest.Python["pytorch"].(map[string]interface{}); ok {
				if gpu, ok := pytorch["gpu"].(map[string]interface{}); ok {
					if pkgs, ok := gpu["packages"].([]interface{}); ok {
						for _, p := range pkgs {
							if s, ok := p.(string); ok {
								pytorchDeps = append(pytorchDeps, s)
							}
						}
					}
				}
				if len(pytorchDeps) == 0 {
					if cpu, ok := pytorch["cpu"].(map[string]interface{}); ok {
						if pkgs, ok := cpu["packages"].([]interface{}); ok {
							for _, p := range pkgs {
								if s, ok := p.(string); ok {
									pytorchDeps = append(pytorchDeps, s)
								}
							}
						}
					}
				}
			}
		}

		allDeps := append(requiredDeps, pytorchDeps...)
		if len(allDeps) == 0 {
			continue
		}

		// 列出已安装的包
		installedPkgs := make(map[string]string)
		cmd := exec.Command(venvPython, "-m", "pip", "list", "--format=json")
		output, err := cmd.Output()
		if err != nil {
			// pip 列表失败时保守认为有问题
			hasProblem = true
			break
		}
		var pkgs []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		if err := json.Unmarshal(output, &pkgs); err != nil {
			// 解析失败也按有问题处理
			hasProblem = true
			break
		}
		for _, p := range pkgs {
			// pip 中 - 和 _ 是等价的，统一转换为 -
			normalizedName := strings.ToLower(strings.ReplaceAll(p.Name, "_", "-"))
			installedPkgs[normalizedName] = p.Version
		}

		// 只要有一个声明的依赖未安装，就认为存在问题
		missing := false
		var missingPkg string
		for _, dep := range allDeps {
			pkgName := strings.Split(dep, ">=")[0]
			pkgName = strings.Split(pkgName, "==")[0]
			pkgName = strings.Split(pkgName, "<")[0]
			pkgName = strings.Split(pkgName, "[")[0] // 处理 extras，如 package[extra]
			pkgName = strings.TrimSpace(pkgName)
			if pkgName == "" {
				continue
			}
			// 统一转换为 - 进行匹配
			normalizedPkgName := strings.ToLower(strings.ReplaceAll(pkgName, "_", "-"))
			if _, ok := installedPkgs[normalizedPkgName]; !ok {
				missing = true
				missingPkg = dep
				log.Printf("[PluginsDepsSummary] Plugin %s missing dep: %s (normalized: %s)", pluginID, dep, normalizedPkgName)
				break
			}
		}
		if missing {
			log.Printf("[PluginsDepsSummary] Plugin %s has problem, missing: %s", pluginID, missingPkg)
			hasProblem = true
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"hasProblem": hasProblem})
}

// handlePythonCreateVenv 为插件创建虚拟环境
func handlePythonCreateVenv(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PluginId string `json:"pluginId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PluginId == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "pluginId required"})
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	venvPath := filepath.Join(cfg.DataPath, "plugins_python_venv", req.PluginId)
	os.MkdirAll(filepath.Dir(venvPath), 0755)

	// 流式输出
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	flusher, _ := w.(http.Flusher)

	writeOutput := func(msg string) {
		data, _ := json.Marshal(map[string]string{"type": "output", "message": msg})
		w.Write(data)
		w.Write([]byte("\n"))
		if flusher != nil {
			flusher.Flush()
		}
	}

	// 创建虚拟环境
	writeOutput(fmt.Sprintf("Creating virtual environment at %s...\n", venvPath))
	cmd := exec.Command("python", "-m", "venv", venvPath)
	cmd.Stdout = &streamWriter{write: writeOutput}
	cmd.Stderr = &streamWriter{write: writeOutput}

	if err := cmd.Run(); err != nil {
		writeOutput(fmt.Sprintf("Error: %v\n", err))
		data, _ := json.Marshal(map[string]interface{}{"type": "done", "success": false})
		w.Write(data)
		w.Write([]byte("\n"))
		return
	}

	writeOutput("Virtual environment created successfully.\n")
	data, _ := json.Marshal(map[string]interface{}{"type": "done", "success": true})
	w.Write(data)
	w.Write([]byte("\n"))
}

// handlePythonDeleteVenv 删除插件虚拟环境
func handlePythonDeleteVenv(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PluginId string `json:"pluginId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PluginId == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "pluginId required"})
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	venvPath := filepath.Join(cfg.DataPath, "plugins_python_venv", req.PluginId)
	if err := os.RemoveAll(venvPath); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// handlePythonInstallDeps 安装插件依赖
func handlePythonInstallDeps(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PluginId string   `json:"pluginId"`
		Packages []string `json:"packages"`
		IndexUrl string   `json:"indexUrl"`
		Proxy    string   `json:"proxy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PluginId == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "pluginId required"})
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "config_error"})
		return
	}

	venvPath := filepath.Join(cfg.DataPath, "plugins_python_venv", req.PluginId)
	venvPython := filepath.Join(venvPath, "Scripts", "python.exe")
	if runtime.GOOS != "windows" {
		venvPython = filepath.Join(venvPath, "bin", "python")
	}

	// 检查虚拟环境是否存在
	if _, err := os.Stat(venvPython); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "venv_not_exists"})
		return
	}

	// 生成任务 ID
	taskId := generateUUID()

	// 立即返回任务 ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"taskId": taskId})

	// 在后台执行安装
	go func() {
		pluginId := req.PluginId
		packages := req.Packages

		// 记录正在安装的任务（暂时不保存 cmd，后续会更新）
		pipInstallingTasks.Lock()
		pipInstallingTasks.tasks[taskId] = &pipTask{PluginID: pluginId, Cmd: nil}
		pipInstallingTasks.Unlock()

		sendOutput := func(message string) {
			broadcastGlobalWS(GlobalWSMessage{
				Type:     "pip_output",
				TaskID:   taskId,
				PluginID: pluginId,
				Message:  message,
			})
		}

		sendDone := func(success bool) {
			// 移除已完成的任务
			pipInstallingTasks.Lock()
			delete(pipInstallingTasks.tasks, taskId)
			pipInstallingTasks.Unlock()

			broadcastGlobalWS(GlobalWSMessage{
				Type:     "pip_done",
				TaskID:   taskId,
				PluginID: pluginId,
				Success:  success,
			})
		}

		// 构建 pip 命令
		args := []string{"-m", "pip", "install", "--progress-bar=on"}

		// 使用设置中的下载源或请求中的
		pythonEnvSettingsMu.Lock()
		indexUrl := req.IndexUrl
		if indexUrl == "" {
			indexUrl = pythonEnvSettings.PipSource
		}
		proxy := req.Proxy
		if proxy == "" {
			proxy = pythonEnvSettings.HttpProxy
		}
		pythonEnvSettingsMu.Unlock()

		if indexUrl != "" {
			args = append(args, "-i", indexUrl, "--trusted-host", extractHost(indexUrl))
		}
		if proxy != "" {
			args = append(args, "--proxy", proxy)
		}
		args = append(args, packages...)

		// 构建完整命令行
		cmdArgs := []string{venvPython}
		cmdArgs = append(cmdArgs, args...)
		cmdLine := strings.Join(cmdArgs, " ")

		sendOutput(fmt.Sprintf("$ pip install %s\n", strings.Join(packages, " ")))

		// 使用 ConPTY 创建伪终端
		cpty, err := conpty.Start(cmdLine, conpty.ConPtyDimensions(120, 30))
		if err != nil {
			log.Printf("[Python] ConPTY failed: %v, falling back to normal exec", err)
			// 回退到普通执行
			cmd := exec.Command(venvPython, args...)
			cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
			stdout, _ := cmd.StdoutPipe()
			stderr, _ := cmd.StderrPipe()
			if err := cmd.Start(); err != nil {
				sendOutput(fmt.Sprintf("Error: %v\n", err))
				sendDone(false)
				return
			}
			go func() {
				buf := make([]byte, 1024)
				for {
					n, err := stdout.Read(buf)
					if n > 0 {
						sendOutput(string(buf[:n]))
					}
					if err != nil {
						break
					}
				}
			}()
			go func() {
				buf := make([]byte, 1024)
				for {
					n, err := stderr.Read(buf)
					if n > 0 {
						sendOutput(string(buf[:n]))
					}
					if err != nil {
						break
					}
				}
			}()
			success := cmd.Wait() == nil
			sendOutput("\n[Done]\n")
			sendDone(success)
			return
		}
		defer cpty.Close()

		// 使用 goroutine 读取输出
		done := make(chan struct{})
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := cpty.Read(buf)
				if n > 0 {
					sendOutput(string(buf[:n]))
				}
				if err != nil {
					break
				}
			}
			close(done)
		}()

		// 等待进程结束
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		exitCode, _ := cpty.Wait(ctx)

		// 等待读取完成
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}

		sendOutput("\n[Done]\n")
		sendDone(exitCode == 0)
	}()
}

// handlePythonInstallingTasks 查询正在进行的安装任务
func handlePythonInstallingTasks(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	pipInstallingTasks.RLock()
	installingPlugins := make([]string, 0)
	for _, task := range pipInstallingTasks.tasks {
		installingPlugins = append(installingPlugins, task.PluginID)
	}
	pipInstallingTasks.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"installingPlugins": installingPlugins,
	})
}

// handlePythonStopInstall 停止正在进行的依赖安装
func handlePythonStopInstall(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PluginId string `json:"pluginId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PluginId == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "pluginId required"})
		return
	}

	// 查找并终止该插件的安装任务
	pipInstallingTasks.Lock()
	var taskIdToRemove string
	for taskId, task := range pipInstallingTasks.tasks {
		if task.PluginID == req.PluginId {
			taskIdToRemove = taskId
			if task.Cmd != nil && task.Cmd.Process != nil {
				task.Cmd.Process.Kill()
			}
			break
		}
	}
	if taskIdToRemove != "" {
		delete(pipInstallingTasks.tasks, taskIdToRemove)
	}
	pipInstallingTasks.Unlock()

	// 广播安装已取消
	if taskIdToRemove != "" {
		broadcastGlobalWS(GlobalWSMessage{
			Type:     "pip_done",
			TaskID:   taskIdToRemove,
			PluginID: req.PluginId,
			Success:  false,
			Message:  "cancelled",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": taskIdToRemove != ""})
}

// handlePythonUninstallDep 卸载插件虚拟环境中的单个依赖
func handlePythonUninstallDep(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PluginId string `json:"pluginId"`
		Package  string `json:"package"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PluginId == "" || req.Package == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
		return
	}

	cfg, _ := loadPathsConfig()
	venvPath := filepath.Join(cfg.DataPath, "plugins_python_venv", req.PluginId)
	venvPython := filepath.Join(venvPath, "Scripts", "python.exe")
	if runtime.GOOS != "windows" {
		venvPython = filepath.Join(venvPath, "bin", "python")
	}

	if _, err := os.Stat(venvPython); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "venv_not_exists"})
		return
	}

	// 执行 pip uninstall
	cmd := exec.Command(venvPython, "-m", "pip", "uninstall", "-y", req.Package)
	output, err := cmd.CombinedOutput()

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("[Python] Failed to uninstall %s: %v, output: %s", req.Package, err, string(output))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "uninstall_failed",
			"output":  string(output),
		})
		return
	}

	log.Printf("[Python] Uninstalled %s from plugin %s", req.Package, req.PluginId)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"output":  string(output),
	})
}

// handlePythonSettings 获取/保存 Python 环境设置
func handlePythonSettings(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	cfg, _ := loadPathsConfig()
	settingsPath := filepath.Join(cfg.DataPath, "python_env_settings.json")

	if r.Method == http.MethodGet {
		// 读取设置
		pythonEnvSettingsMu.Lock()
		defer pythonEnvSettingsMu.Unlock()

		// 尝试从文件加载
		if data, err := os.ReadFile(settingsPath); err == nil {
			json.Unmarshal(data, &pythonEnvSettings)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pythonEnvSettings)
		return
	}

	if r.Method == http.MethodPost {
		var settings PythonEnvSettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
			return
		}

		pythonEnvSettingsMu.Lock()
		pythonEnvSettings = settings
		pythonEnvSettingsMu.Unlock()

		// 保存到文件
		data, _ := json.MarshalIndent(settings, "", "  ")
		os.WriteFile(settingsPath, data, 0644)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

// extractHost 从 URL 中提取主机名
func extractHost(url string) string {
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		hostParts := strings.Split(parts[0], ":")
		return hostParts[0]
	}
	return ""
}

// streamWriter 用于流式输出
type streamWriter struct {
	write func(string)
}

func (sw *streamWriter) Write(p []byte) (n int, err error) {
	sw.write(string(p))
	return len(p), nil
}

// handleSystemGpu 检测系统 GPU 信息
func handleSystemGpu(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	result := map[string]interface{}{
		"hasNvidia":   false,
		"cudaVersion": "",
		"gpuName":     "",
	}

	// 尝试运行 nvidia-smi 检测 NVIDIA GPU
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,driver_version", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) > 0 {
			parts := strings.Split(lines[0], ", ")
			if len(parts) >= 1 {
				result["hasNvidia"] = true
				result["gpuName"] = strings.TrimSpace(parts[0])
			}
		}
	}

	// 检测 CUDA 版本
	cmd = exec.Command("nvidia-smi", "--query-gpu=driver_version", "--format=csv,noheader")
	output, err = cmd.Output()
	if err == nil {
		// 从 nvidia-smi 输出中获取 CUDA 版本
		cmd2 := exec.Command("nvidia-smi")
		output2, err2 := cmd2.Output()
		if err2 == nil {
			// 解析输出找 CUDA Version
			lines := strings.Split(string(output2), "\n")
			for _, line := range lines {
				if strings.Contains(line, "CUDA Version") {
					// 格式类似: | NVIDIA-SMI 535.154.05   Driver Version: 535.154.05   CUDA Version: 12.2     |
					parts := strings.Split(line, "CUDA Version:")
					if len(parts) >= 2 {
						cudaVer := strings.TrimSpace(strings.Split(parts[1], "|")[0])
						result["cudaVersion"] = cudaVer
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handlePythonInstallPytorch 安装 PyTorch
func handlePythonInstallPytorch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PluginID string   `json:"pluginId"`
		Version  string   `json:"version"`            // "gpu" 或 "cpu"
		Packages []string `json:"packages,omitempty"` // 可选，自定义安装的包列表
		IndexUrl string   `json:"indexUrl,omitempty"` // 可选，自定义 PyTorch 下载源
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// 获取插件路径
	pluginsDir, _ := getPluginsDir()
	manifestPath := ""

	// 查找插件 manifest
	filepath.Walk(pluginsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.Name() == "manifest.json" {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			var m struct {
				ID string `json:"id"`
			}
			if json.Unmarshal(data, &m) == nil && m.ID == req.PluginID {
				manifestPath = path
				return filepath.SkipAll
			}
		}
		return nil
	})

	if manifestPath == "" {
		http.Error(w, "Plugin not found", http.StatusNotFound)
		return
	}

	// 读取 manifest
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		http.Error(w, "Cannot read manifest", http.StatusInternalServerError)
		return
	}

	var manifest struct {
		Python map[string]interface{} `json:"python"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		http.Error(w, "Invalid manifest", http.StatusInternalServerError)
		return
	}

	// 获取 pytorch 配置
	pytorch, ok := manifest.Python["pytorch"].(map[string]interface{})
	if !ok {
		http.Error(w, "No pytorch config", http.StatusBadRequest)
		return
	}

	versionConfig, ok := pytorch[req.Version].(map[string]interface{})
	if !ok {
		http.Error(w, "Version not supported", http.StatusBadRequest)
		return
	}

	// 构建需要安装的包列表：优先使用请求中的自定义 packages，其次使用 manifest 中的默认 packages
	packages := []string{}
	if len(req.Packages) > 0 {
		packages = append(packages, req.Packages...)
	} else if pkgs, ok := versionConfig["packages"].([]interface{}); ok {
		for _, p := range pkgs {
			if s, ok := p.(string); ok {
				packages = append(packages, s)
			}
		}
	}

	// 优先使用请求中的 indexUrl，其次使用 manifest 中的配置
	indexUrl := req.IndexUrl
	if indexUrl == "" {
		if url, ok := versionConfig["indexUrl"].(string); ok {
			indexUrl = url
		}
	}

	// 获取虚拟环境路径
	dataPath := getDataPath()
	venvDir := filepath.Join(dataPath, "plugins_python_venv", req.PluginID)
	venvPython := filepath.Join(venvDir, "Scripts", "python.exe")

	if _, err := os.Stat(venvPython); os.IsNotExist(err) {
		http.Error(w, "Venv not exists", http.StatusBadRequest)
		return
	}

	// 生成任务 ID
	taskId := generateUUID()

	// 立即返回任务 ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"taskId": taskId})

	// 在后台执行安装
	go func() {
		pluginId := req.PluginID

		// 记录正在安装的任务
		pipInstallingTasks.Lock()
		pipInstallingTasks.tasks[taskId] = &pipTask{PluginID: pluginId, Cmd: nil}
		pipInstallingTasks.Unlock()

		sendOutput := func(message string) {
			broadcastGlobalWS(GlobalWSMessage{
				Type:     "pip_output",
				TaskID:   taskId,
				PluginID: pluginId,
				Message:  message,
			})
		}

		sendDone := func(success bool) {
			// 移除已完成的任务
			pipInstallingTasks.Lock()
			delete(pipInstallingTasks.tasks, taskId)
			pipInstallingTasks.Unlock()

			broadcastGlobalWS(GlobalWSMessage{
				Type:     "pip_done",
				TaskID:   taskId,
				PluginID: pluginId,
				Success:  success,
			})
		}

		// 读取代理设置
		settingsPath := filepath.Join(dataPath, "python_env_settings.json")
		var settings struct {
			PypiMirror  string `json:"pypiMirror"`
			HttpProxy   string `json:"httpProxy"`
			Socks5Proxy string `json:"socks5Proxy"`
		}
		if data, err := os.ReadFile(settingsPath); err == nil {
			json.Unmarshal(data, &settings)
		}

		// 构建 pip 命令
		args := []string{"-m", "pip", "install", "--progress-bar=on"}
		args = append(args, packages...)
		if indexUrl != "" {
			args = append(args, "--index-url", indexUrl)
		}
		if settings.PypiMirror != "" {
			args = append(args, "--extra-index-url", settings.PypiMirror)
			args = append(args, "--trusted-host", extractHost(settings.PypiMirror))
		}

		// 构建完整命令行
		cmdArgs := []string{venvPython}
		cmdArgs = append(cmdArgs, args...)
		cmdLine := strings.Join(cmdArgs, " ")

		// 设置代理环境变量
		if settings.HttpProxy != "" {
			os.Setenv("HTTP_PROXY", settings.HttpProxy)
			os.Setenv("HTTPS_PROXY", settings.HttpProxy)
		}
		if settings.Socks5Proxy != "" {
			os.Setenv("ALL_PROXY", settings.Socks5Proxy)
		}

		sendOutput(fmt.Sprintf("$ %s\n", cmdLine))

		// 使用 ConPTY 创建伪终端
		cpty, err := conpty.Start(cmdLine, conpty.ConPtyDimensions(120, 30))
		if err != nil {
			log.Printf("[Python] ConPTY failed: %v, falling back to normal exec", err)
			// 回退到普通执行
			cmd := exec.Command(venvPython, args...)
			cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
			stdout, _ := cmd.StdoutPipe()
			stderr, _ := cmd.StderrPipe()
			if err := cmd.Start(); err != nil {
				sendOutput(fmt.Sprintf("Error: %v\n", err))
				sendDone(false)
				return
			}
			go func() {
				buf := make([]byte, 1024)
				for {
					n, err := stdout.Read(buf)
					if n > 0 {
						sendOutput(string(buf[:n]))
					}
					if err != nil {
						break
					}
				}
			}()
			go func() {
				buf := make([]byte, 1024)
				for {
					n, err := stderr.Read(buf)
					if n > 0 {
						sendOutput(string(buf[:n]))
					}
					if err != nil {
						break
					}
				}
			}()
			success := cmd.Wait() == nil
			sendOutput("\n[Done]\n")
			sendDone(success)
			return
		}
		defer cpty.Close()

		// 使用 goroutine 读取输出
		done := make(chan struct{})
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := cpty.Read(buf)
				if n > 0 {
					sendOutput(string(buf[:n]))
				}
				if err != nil {
					break
				}
			}
			close(done)
		}()

		// 等待进程结束
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		exitCode, _ := cpty.Wait(ctx)

		// 等待读取完成
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}

		sendOutput("\n[Done]\n")
		sendDone(exitCode == 0)
	}()
}

// handlePythonRunCommand 在插件虚拟环境中运行自定义命令
func handlePythonRunCommand(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PluginID string `json:"pluginId"`
		Command  string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	dataPath := getDataPath()
	venvDir := filepath.Join(dataPath, "plugins_python_venv", req.PluginID)
	venvPython := filepath.Join(venvDir, "Scripts", "python.exe")

	if _, err := os.Stat(venvPython); os.IsNotExist(err) {
		http.Error(w, "Venv not exists", http.StatusBadRequest)
		return
	}

	taskId := generateUUID()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"taskId": taskId})

	go func() {
		pluginId := req.PluginID

		sendOutput := func(message string) {
			broadcastGlobalWS(GlobalWSMessage{
				Type:     "pip_output",
				TaskID:   taskId,
				PluginID: pluginId,
				Message:  message,
			})
		}

		sendDone := func(success bool) {
			broadcastGlobalWS(GlobalWSMessage{
				Type:     "pip_done",
				TaskID:   taskId,
				PluginID: pluginId,
				Success:  success,
			})
		}

		activateScript := filepath.Join(venvDir, "Scripts", "activate.bat")
		fullCmd := fmt.Sprintf("cmd /C \"%s && %s\"", activateScript, req.Command)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		cpty, err := conpty.Start(fullCmd, conpty.ConPtyDimensions(120, 30))
		if err != nil {
			sendOutput(fmt.Sprintf("Error: %v\n", err))
			sendDone(false)
			return
		}
		defer cpty.Close()

		done := make(chan struct{})
		go func() {
			defer close(done)
			buf := make([]byte, 1024)
			for {
				n, err := cpty.Read(buf)
				if n > 0 {
					sendOutput(string(buf[:n]))
				}
				if err != nil {
					break
				}
			}
		}()

		exitCode, _ := cpty.Wait(ctx)

		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}

		sendOutput("\n[Done]\n")
		sendDone(exitCode == 0)
	}()
}
