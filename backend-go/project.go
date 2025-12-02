package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ProjectMeta 项目元数据
type ProjectMeta struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

// loadProjects 加载项目列表
func loadProjects(dataPath string) ([]ProjectMeta, error) {
	projectsPath := filepath.Join(dataPath, "projects.json")
	data, err := os.ReadFile(projectsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []ProjectMeta{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return []ProjectMeta{}, nil
	}
	var projects []ProjectMeta
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

// saveProjects 保存项目列表
func saveProjects(dataPath string, projects []ProjectMeta) error {
	projectsPath := filepath.Join(dataPath, "projects.json")
	encoded, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(projectsPath, encoded, 0o644)
}

// initProjectStorage 初始化项目存储
func initProjectStorage(dataPath string, project ProjectMeta) (int, error) {
	projectRoot := filepath.Join(dataPath, "project_item")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		return 0, err
	}
	projectDir := filepath.Join(projectRoot, project.ID)
	imagesDir := filepath.Join(projectDir, "images")
	originalsDir := filepath.Join(imagesDir, "originals")
	thumbsDir := filepath.Join(imagesDir, "thumbs")
	dbDir := filepath.Join(projectDir, "db")

	if err := os.MkdirAll(originalsDir, 0o755); err != nil {
		return 0, err
	}
	if err := os.MkdirAll(thumbsDir, 0o755); err != nil {
		return 0, err
	}
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return 0, err
	}

	dbPath := filepath.Join(dbDir, "project.db")
	db, err := openProjectDB(dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		return 0, err
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return 0, err
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		return 0, err
	}

	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS image_index (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	filename TEXT NOT NULL,
	original_rel_path TEXT NOT NULL,
	thumb_rel_path TEXT NOT NULL,
	deleted_in_project INTEGER NOT NULL DEFAULT 0,
	annotation_status TEXT NOT NULL DEFAULT 'none',
	created_at TEXT NOT NULL
);
`)
	if err != nil {
		return 0, err
	}

	// 图片引用计数表
	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS image_ref_count (
	image_id INTEGER PRIMARY KEY,
	ref_count INTEGER NOT NULL DEFAULT 1,
	FOREIGN KEY(image_id) REFERENCES image_index(id) ON DELETE CASCADE
);
`)
	if err != nil {
		return 0, err
	}

	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS annotations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	image_id INTEGER NOT NULL,
	category_id INTEGER NOT NULL,
	type TEXT NOT NULL,
	data TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY(image_id) REFERENCES image_index(id) ON DELETE CASCADE
);
`)
	if err != nil {
		return 0, err
	}

	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS categories (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	color TEXT NOT NULL,
	sort_order INTEGER NOT NULL DEFAULT 0,
	mate TEXT NOT NULL DEFAULT ''
);
`)
	if err != nil {
		return 0, err
	}

	_, err = db.Exec(`
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_name ON categories(name);
`)
	if err != nil {
		return 0, err
	}

	// 初始图片数量为 0
	return 0, nil
}

// isDriveRoot 判断是否是驱动器根目录
func isDriveRoot(p string) bool {
	clean := filepath.Clean(p)
	vol := filepath.VolumeName(clean)
	if vol != "" {
		rest := strings.TrimPrefix(clean, vol)
		rest = strings.Trim(rest, "\\/")
		return rest == ""
	}
	return clean == string(os.PathSeparator)
}

// isImagePath 判断是否是图片路径
func isImagePath(p string) bool {
	ext := strings.ToLower(filepath.Ext(p))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".bmp", ".gif", ".webp":
		return true
	default:
		return false
	}
}

// collectImageFilesFromDirectory 从目录收集图片文件
func collectImageFilesFromDirectory(root string, maxDepth int) ([]string, error) {
	var result []string
	var visited int
	var matched int
	lastLog := time.Now()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		visited++
		// 周期性打印扫描进度，避免大目录时长时间没有任何反馈
		if time.Since(lastLog) >= 2*time.Second {
			log.Printf("collect images: scanning... visited=%d matched=%d last=%s", visited, matched, path)
			lastLog = time.Now()
		}
		if d.IsDir() {
			// 计算相对于 root 的深度
			if maxDepth >= 0 {
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return nil
				}
				if rel == "." {
					return nil
				}
				depth := strings.Count(rel, string(os.PathSeparator))
				if depth > maxDepth {
					return fs.SkipDir
				}
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if !isImagePath(path) {
			return nil
		}
		result = append(result, path)
		matched++
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	outFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, srcFile); err != nil {
		return err
	}
	return nil
}

// nextImageSequence 获取下一个图片序号
func nextImageSequence(projectID, originalsDir string) (int, error) {
	entries, err := os.ReadDir(originalsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	maxSeq := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		prefix := projectID + "_"
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rest := strings.TrimPrefix(name, prefix)
		ext := filepath.Ext(rest)
		if ext != "" {
			rest = strings.TrimSuffix(rest, ext)
		}
		if rest == "" {
			continue
		}
		seq, err := strconv.Atoi(rest)
		if err != nil {
			continue
		}
		if seq > maxSeq {
			maxSeq = seq
		}
	}
	return maxSeq, nil
}

// handleHealthz 健康检查
func handleHealthz(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// handleProjects 项目列表和创建
func handleProjects(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"config_unavailable"}`))
		return
	}

	if r.Method == http.MethodGet {
		projects, err := loadProjects(cfg.DataPath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"projects_unavailable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(projects)
		return
	}

	// DELETE 删除项目
	if r.Method == http.MethodDelete {
		projectID := r.URL.Query().Get("id")
		if projectID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"id_required"}`))
			return
		}

		projects, err := loadProjects(cfg.DataPath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"projects_unavailable"}`))
			return
		}

		// 查找项目
		var foundIndex = -1
		for i, p := range projects {
			if p.ID == projectID {
				foundIndex = i
				break
			}
		}
		if foundIndex == -1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"project_not_found"}`))
			return
		}

		// 删除项目目录
		projectDir := filepath.Join(cfg.DataPath, "project_item", projectID)
		if err := os.RemoveAll(projectDir); err != nil {
			log.Printf("[Project] Failed to remove project directory: %v", err)
		}

		// 从列表中移除
		projects = append(projects[:foundIndex], projects[foundIndex+1:]...)
		if err := saveProjects(cfg.DataPath, projects); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"projects_persist_failed"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
		return
	}

	// PUT 重命名项目
	if r.Method == http.MethodPut {
		type renameRequest struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		var body renameRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_json"}`))
			return
		}

		if body.ID == "" || strings.TrimSpace(body.Name) == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"id_and_name_required"}`))
			return
		}

		newName := strings.TrimSpace(body.Name)
		if strings.ContainsAny(newName, "<>:\"/\\|?*") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"name_invalid"}`))
			return
		}

		projects, err := loadProjects(cfg.DataPath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"projects_unavailable"}`))
			return
		}

		// 检查是否存在同名项目
		for _, p := range projects {
			if p.ID != body.ID && strings.EqualFold(p.Name, newName) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":"project_exists"}`))
				return
			}
		}

		// 查找并更新项目
		var found bool
		for i, p := range projects {
			if p.ID == body.ID {
				projects[i].Name = newName
				found = true
				break
			}
		}
		if !found {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"project_not_found"}`))
			return
		}

		if err := saveProjects(cfg.DataPath, projects); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"projects_persist_failed"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type createProjectRequest struct {
		Name string `json:"name"`
	}

	var body createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_json"}`))
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"name_required"}`))
		return
	}
	if strings.ContainsAny(name, "<>:\"/\\|?*") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"name_invalid"}`))
		return
	}

	projects, err := loadProjects(cfg.DataPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"projects_unavailable"}`))
		return
	}

	for _, p := range projects {
		if strings.EqualFold(p.Name, name) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"project_exists"}`))
			return
		}
	}

	id, err := generateProjectID()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"id_generation_failed"}`))
		return
	}

	projectMeta := ProjectMeta{
		ID:        id,
		Name:      name,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	imageCount, err := initProjectStorage(cfg.DataPath, projectMeta)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"create_failed"}`))
		return
	}

	projects = append(projects, projectMeta)
	if err := saveProjects(cfg.DataPath, projects); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"projects_persist_failed"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	type createProjectResponse struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		ImageCount int    `json:"imageCount"`
	}
	_ = json.NewEncoder(w).Encode(createProjectResponse{
		ID:         projectMeta.ID,
		Name:       projectMeta.Name,
		ImageCount: imageCount,
	})
}
