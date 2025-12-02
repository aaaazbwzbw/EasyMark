package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/image/draw"
)

// projectImageListItem 项目图片列表项
type projectImageListItem struct {
	ID               int64  `json:"id"`
	Filename         string `json:"filename"`
	HasThumb         bool   `json:"hasThumb"`
	IsExternal       bool   `json:"isExternal"`
	ThumbPath        string `json:"thumbPath"`
	OriginalPath     string `json:"originalPath"`
	AnnotationStatus string `json:"annotationStatus"`
}

// projectImageListResponse 项目图片列表响应
type projectImageListResponse struct {
	Items            []projectImageListItem `json:"items"`
	Total            int64                  `json:"total"`
	AnnotatedCount   int64                  `json:"annotatedCount"`
	UnannotatedCount int64                  `json:"unannotatedCount"`
	NegativeCount    int64                  `json:"negativeCount"`
}

// runImportImagesTask 异步执行导入图片任务
func runImportImagesTask(dataPath, projectID, taskID, importMode string, imagePaths []string) {
	defer releaseIOLock(taskID)
	defer func() {
		if r := recover(); r != nil {
			importTasksMu.Lock()
			if task, ok := importTasks[taskID]; ok {
				task.Phase = importPhaseFailed
				task.Error = "panic_in_import_task"
				task.Progress = 0
			}
			importTasksMu.Unlock()
		}
	}()

	importMode = strings.TrimSpace(strings.ToLower(importMode))
	if importMode != "copy" && importMode != "link" && importMode != "external" {
		importMode = "copy"
	}

	projectRoot := filepath.Join(dataPath, "project_item", projectID)
	imagesDir := filepath.Join(projectRoot, "images")
	originalsDir := filepath.Join(imagesDir, "originals")
	thumbsDir := filepath.Join(imagesDir, "thumbs")
	if err := os.MkdirAll(originalsDir, 0o755); err != nil {
		log.Printf("import task: mkdir originals failed: %v", err)
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseFailed
			task.Error = "storage_unavailable"
		}
		importTasksMu.Unlock()
		return
	}
	if err := os.MkdirAll(thumbsDir, 0o755); err != nil {
		log.Printf("import task: mkdir thumbs failed: %v", err)
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseFailed
			task.Error = "storage_unavailable"
		}
		importTasksMu.Unlock()
		return
	}

	type importJob struct {
		SrcPath string
		Name    string
		Size    int64
	}

	type importedImage struct {
		Filename    string
		OriginalRel string
		ThumbRel    string
	}

	existingNames := make(map[string]struct{})
	entries, err := os.ReadDir(originalsDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if name != "" {
				existingNames[name] = struct{}{}
			}
		}
	}

	jobs := make([]importJob, 0, len(imagePaths))
	for _, srcPath := range imagePaths {
		info, err := os.Stat(srcPath)
		if err != nil || info.IsDir() {
			continue
		}
		baseName := filepath.Base(srcPath)
		baseName = strings.TrimSpace(baseName)
		if baseName == "" {
			continue
		}
		if _, exists := existingNames[baseName]; exists {
			continue
		}
		existingNames[baseName] = struct{}{}
		jobs = append(jobs, importJob{
			SrcPath: srcPath,
			Name:    baseName,
			Size:    info.Size(),
		})
	}

	if len(jobs) == 0 {
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseCompleted
			task.Progress = 100
			task.Total = 0
			task.Imported = 0
		}
		importTasksMu.Unlock()
		return
	}

	importTasksMu.Lock()
	if task, ok := importTasks[taskID]; ok {
		task.Total = len(jobs)
	}
	importTasksMu.Unlock()

	jobsCh := make(chan importJob)
	resultsCh := make(chan importedImage, len(jobs))
	var wg sync.WaitGroup
	var hardLinkSuccess uint64
	var hardLinkFallback uint64
	workerCount := runtime.NumCPU()
	if workerCount < 2 {
		workerCount = 2
	}
	if workerCount > 8 {
		workerCount = 8
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobsCh {
				var physicalPath string
				var originalRel string
				var thumbRel string

				switch importMode {
				case "external":
					physicalPath = job.SrcPath
					originalRel = filepath.ToSlash(job.SrcPath)
					thumbRel = originalRel
				case "link":
					originalPath := filepath.Join(originalsDir, job.Name)
					if err := os.Link(job.SrcPath, originalPath); err != nil {
						atomic.AddUint64(&hardLinkFallback, 1)
						log.Printf("import task: hard link failed for %s -> %s, fallback to copy: %v", job.SrcPath, originalPath, err)
						if err := copyFile(job.SrcPath, originalPath); err != nil {
							log.Printf("import task: link/copy file failed: %v", err)
							resultsCh <- importedImage{}
							continue
						}
					} else {
						atomic.AddUint64(&hardLinkSuccess, 1)
					}
					physicalPath = originalPath
					originalRel = filepath.ToSlash(filepath.Join("images", "originals", job.Name))
					thumbRel = originalRel
				default:
					originalPath := filepath.Join(originalsDir, job.Name)
					if err := copyFile(job.SrcPath, originalPath); err != nil {
						log.Printf("import task: copy file failed: %v", err)
						resultsCh <- importedImage{}
						continue
					}
					physicalPath = originalPath
					originalRel = filepath.ToSlash(filepath.Join("images", "originals", job.Name))
					thumbRel = originalRel
				}

				if job.Size > thumbSizeThresholdBytes {
					ext := strings.ToLower(filepath.Ext(job.Name))
					base := strings.TrimSuffix(job.Name, ext)
					thumbFilename := base + ".jpg"
					thumbPath := filepath.Join(thumbsDir, thumbFilename)
					srcFile, err := os.Open(physicalPath)
					if err == nil {
						img, _, decErr := image.Decode(srcFile)
						_ = srcFile.Close()
						if decErr == nil {
							b := img.Bounds()
							w := b.Dx()
							h := b.Dy()
							newW, newH := w, h
							if w >= h && w > thumbMaxDimension {
								newW = thumbMaxDimension
								newH = int(float64(h) * float64(newW) / float64(w))
							} else if h > w && h > thumbMaxDimension {
								newH = thumbMaxDimension
								newW = int(float64(w) * float64(newH) / float64(h))
							}
							thumbImg := image.NewRGBA(image.Rect(0, 0, newW, newH))
							draw.ApproxBiLinear.Scale(thumbImg, thumbImg.Bounds(), img, b, draw.Over, nil)
							if out, err := os.Create(thumbPath); err == nil {
								_ = jpeg.Encode(out, thumbImg, &jpeg.Options{Quality: 75})
								_ = out.Close()
								thumbRel = filepath.ToSlash(filepath.Join("images", "thumbs", thumbFilename))
							}
						}
					}
				}

				resultsCh <- importedImage{
					Filename:    job.Name,
					OriginalRel: originalRel,
					ThumbRel:    thumbRel,
				}
			}
		}()
	}

	go func() {
		for _, job := range jobs {
			jobsCh <- job
		}
		close(jobsCh)
	}()

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	imported := make([]importedImage, 0, len(jobs))
	processed := 0
	for res := range resultsCh {
		if res.Filename != "" {
			imported = append(imported, res)
		}
		processed++
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseCopying
			task.Imported = processed
			task.Total = len(jobs)
			if task.Total > 0 {
				task.Progress = int(float64(task.Imported) * 100 / float64(task.Total))
			}
		}
		importTasksMu.Unlock()
	}

	if importMode == "link" {
		s := atomic.LoadUint64(&hardLinkSuccess)
		f := atomic.LoadUint64(&hardLinkFallback)
		log.Printf("import task: hard link summary for task %s - success=%d, fallback_to_copy=%d", taskID, s, f)
	}

	dbPath := filepath.Join(projectRoot, "db", "project.db")
	db, err := openProjectDB(dbPath)
	if err != nil {
		log.Printf("import task: open db failed: %v", err)
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
		log.Printf("import task: set WAL failed: %v", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		log.Printf("import task: enable foreign_keys failed: %v", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		log.Printf("import task: set busy_timeout failed: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("import task: begin tx failed: %v", err)
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseFailed
			task.Error = "tx_begin_failed"
		}
		importTasksMu.Unlock()
		return
	}

	for idx, imgInfo := range imported {
		res, err := tx.Exec(
			`INSERT INTO image_index (filename, original_rel_path, thumb_rel_path, deleted_in_project, created_at) VALUES (?, ?, ?, 0, ?);`,
			imgInfo.Filename,
			imgInfo.OriginalRel,
			imgInfo.ThumbRel,
			time.Now().UTC().Format(time.RFC3339),
		)
		if err != nil {
			log.Printf("import task: insert image_index failed: %v", err)
			continue
		}

		// 创建引用计数记录
		if imageID, err := res.LastInsertId(); err == nil {
			_, _ = tx.Exec(`INSERT INTO image_ref_count (image_id, ref_count) VALUES (?, 1);`, imageID)
		}

		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseIndexing
			task.Imported = idx + 1
			task.Total = len(imported)
			if task.Total > 0 {
				task.Progress = int(float64(task.Imported) * 100 / float64(task.Total))
			}
		}
		importTasksMu.Unlock()
	}

	if err := tx.Commit(); err != nil {
		log.Printf("import task: commit tx failed: %v", err)
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseFailed
			task.Error = "tx_commit_failed"
		}
		importTasksMu.Unlock()
		return
	}

	importTasksMu.Lock()
	if task, ok := importTasks[taskID]; ok {
		task.Phase = importPhaseCompleted
		task.Progress = 100
		task.Total = len(imported)
		task.Imported = len(imported)
	}
	importTasksMu.Unlock()
}

// handleImportImages 处理导入图片请求
func handleImportImages(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type importImagesRequest struct {
		ProjectID  string   `json:"projectId"`
		Mode       string   `json:"mode"`
		ImportMode string   `json:"importMode"`
		Paths      []string `json:"paths"`
	}

	var req importImagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_json"}`))
		return
	}
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	req.Mode = strings.TrimSpace(strings.ToLower(req.Mode))
	req.ImportMode = strings.TrimSpace(strings.ToLower(req.ImportMode))
	log.Printf("import images: request project=%s mode=%s importMode=%s paths=%d", req.ProjectID, req.Mode, req.ImportMode, len(req.Paths))
	if req.ProjectID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"project_required"}`))
		return
	}
	if req.Mode != "directory" && req.Mode != "files" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"mode_invalid"}`))
		return
	}
	if req.ImportMode == "" {
		req.ImportMode = "copy"
	}
	if req.ImportMode != "copy" && req.ImportMode != "link" && req.ImportMode != "external" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"import_mode_invalid"}`))
		return
	}
	if len(req.Paths) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"paths_required"}`))
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"config_unavailable"}`))
		return
	}

	taskID, err := generateProjectID()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"task_id_failed"}`))
		return
	}

	if !tryAcquireIOLock(taskID) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"io_task_busy","currentTaskId":"` + getCurrentIOTask() + `"}`))
		return
	}

	log.Printf("import images: creating task %s for project=%s, mode=%s, importMode=%s", taskID, req.ProjectID, req.Mode, req.ImportMode)

	importTasksMu.Lock()
	importTasks[taskID] = &ImportTaskStatus{
		ID:        taskID,
		ProjectID: req.ProjectID,
		TaskType:  taskTypeImportImages,
		Phase:     importPhaseScanning,
		Progress:  0,
		Imported:  0,
		Total:     0,
	}
	importTasksMu.Unlock()

	go func() {
		const scanTimeout = 60 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
		defer cancel()

		var imagePaths []string
		scanStart := time.Now()

		done := make(chan struct{})
		go func() {
			defer close(done)
			if req.Mode == "directory" {
				root := strings.TrimSpace(req.Paths[0])
				info, err := os.Stat(root)
				if err != nil || !info.IsDir() {
					log.Printf("import images: async scan directory invalid: %s", root)
					importTasksMu.Lock()
					if task, ok := importTasks[taskID]; ok {
						task.Phase = importPhaseFailed
						task.Error = "directory_invalid"
					}
					importTasksMu.Unlock()
					return
				}
				log.Printf("import images: async scanning directory %s", root)
				files, err := collectImageFilesFromDirectory(root, maxImportDirectoryDepth)
				if err != nil {
					log.Printf("import images: async scan failed for %s: %v", root, err)
					importTasksMu.Lock()
					if task, ok := importTasks[taskID]; ok {
						task.Phase = importPhaseFailed
						task.Error = "scan_failed"
					}
					importTasksMu.Unlock()
					return
				}
				imagePaths = files
			} else {
				seen := make(map[string]struct{})
				for _, p := range req.Paths {
					clean := strings.TrimSpace(p)
					if clean == "" {
						continue
					}
					if _, ok := seen[clean]; ok {
						continue
					}
					info, err := os.Stat(clean)
					if err != nil || info.IsDir() {
						continue
					}
					if !isImagePath(clean) {
						continue
					}
					seen[clean] = struct{}{}
					imagePaths = append(imagePaths, clean)
				}
			}
		}()

		select {
		case <-ctx.Done():
			log.Printf("import images: async scan timeout for task %s after %s", taskID, time.Since(scanStart))
			importTasksMu.Lock()
			if task, ok := importTasks[taskID]; ok {
				if task.Phase == importPhaseScanning {
					task.Phase = importPhaseFailed
					task.Error = "scan_timeout"
				}
			}
			importTasksMu.Unlock()
			return
		case <-done:
		}

		if len(imagePaths) == 0 {
			log.Printf("import images: async scan found no images for project=%s mode=%s", req.ProjectID, req.Mode)
			importTasksMu.Lock()
			if task, ok := importTasks[taskID]; ok {
				if task.Phase == importPhaseScanning {
					task.Phase = importPhaseFailed
					task.Error = "no_images"
				}
			}
			importTasksMu.Unlock()
			return
		}

		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			if task.Phase == importPhaseScanning {
				task.Total = len(imagePaths)
			}
		}
		importTasksMu.Unlock()

		log.Printf("import images: async scan completed for task %s, found=%d, elapsed=%s", taskID, len(imagePaths), time.Since(scanStart))
		runImportImagesTask(cfg.DataPath, req.ProjectID, taskID, req.ImportMode, imagePaths)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(struct {
		TaskID string `json:"taskId"`
		Total  int    `json:"total"`
	}{
		TaskID: taskID,
		Total:  0,
	})
}

// handleImportTaskStatus 获取导入任务状态
func handleImportTaskStatus(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if strings.TrimSpace(id) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"id_required"}`))
		return
	}

	importTasksMu.RLock()
	task, ok := importTasks[id]
	importTasksMu.RUnlock()
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"task_not_found"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(task)
}

// handleProjectImages 获取项目图片列表
func handleProjectImages(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	projectID := strings.TrimSpace(r.URL.Query().Get("projectId"))
	if projectID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"project_required"}`))
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"config_unavailable"}`))
		return
	}

	projectRoot := filepath.Join(cfg.DataPath, "project_item", projectID)
	dbPath := filepath.Join(projectRoot, "db", "project.db")
	db, err := openProjectDB(dbPath)
	if err != nil {
		log.Printf("project images: open db failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"db_unavailable"}`))
		return
	}
	defer db.Close()

	_, _ = db.Exec(`ALTER TABLE image_index ADD COLUMN annotation_status TEXT NOT NULL DEFAULT 'none';`)

	rows, err := db.Query(`SELECT id, filename, original_rel_path, thumb_rel_path, COALESCE(annotation_status, 'none') FROM image_index WHERE deleted_in_project = 0 ORDER BY id ASC;`)
	if err != nil {
		log.Printf("project images: query failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"query_failed"}`))
		return
	}
	defer rows.Close()

	items := make([]projectImageListItem, 0, 256)
	var annotatedCount, unannotatedCount, negativeCount int64
	for rows.Next() {
		var id int64
		var filename string
		var originalRel string
		var thumbRel string
		var annotationStatus string
		if err := rows.Scan(&id, &filename, &originalRel, &thumbRel, &annotationStatus); err != nil {
			log.Printf("project images: scan failed: %v", err)
			continue
		}
		switch annotationStatus {
		case "annotated":
			annotatedCount++
		case "negative":
			negativeCount++
		default:
			unannotatedCount++
		}
		isExternal := !(strings.HasPrefix(originalRel, "images/") || strings.HasPrefix(originalRel, "./images/"))
		hasThumb := thumbRel != "" && thumbRel != originalRel
		originalPath := originalRel
		thumbPath := thumbRel
		if thumbPath == "" {
			thumbPath = originalPath
		}
		items = append(items, projectImageListItem{
			ID:               id,
			Filename:         filename,
			HasThumb:         hasThumb,
			IsExternal:       isExternal,
			ThumbPath:        thumbPath,
			OriginalPath:     originalPath,
			AnnotationStatus: annotationStatus,
		})
	}
	if err := rows.Err(); err != nil {
		log.Printf("project images: rows error: %v", err)
	}

	resp := projectImageListResponse{
		Items:            items,
		Total:            int64(len(items)),
		AnnotatedCount:   annotatedCount,
		UnannotatedCount: unannotatedCount,
		NegativeCount:    negativeCount,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// handleDeleteProjectImages 删除项目图片
func handleDeleteProjectImages(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type deleteImagesRequest struct {
		ProjectID string  `json:"projectId"`
		ImageIDs  []int64 `json:"imageIds"`
	}

	var req deleteImagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_json"}`))
		return
	}
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	if req.ProjectID == "" || len(req.ImageIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"project_and_images_required"}`))
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"config_unavailable"}`))
		return
	}

	taskID, err := generateProjectID()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"task_id_failed"}`))
		return
	}

	if !tryAcquireIOLock(taskID) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"io_task_busy","currentTaskId":"` + getCurrentIOTask() + `"}`))
		return
	}

	importTasksMu.Lock()
	importTasks[taskID] = &ImportTaskStatus{
		ID:        taskID,
		ProjectID: req.ProjectID,
		TaskType:  taskTypeDeleteImages,
		Phase:     importPhaseDeleting,
		Progress:  0,
		Imported:  0,
		Total:     len(req.ImageIDs),
	}
	importTasksMu.Unlock()

	go runDeleteImagesTask(cfg.DataPath, req.ProjectID, taskID, req.ImageIDs)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"taskId":   taskID,
		"imageIds": req.ImageIDs,
	})
}

// runDeleteImagesTask 异步执行删除图片任务
func runDeleteImagesTask(dataPath, projectID, taskID string, imageIDs []int64) {
	defer releaseIOLock(taskID)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[DeleteImages] Task %s panic: %v", taskID, r)
			importTasksMu.Lock()
			if task, ok := importTasks[taskID]; ok {
				task.Phase = importPhaseFailed
				task.Error = "panic_in_delete_task"
			}
			importTasksMu.Unlock()
		}
	}()

	log.Printf("[DeleteImages] Task %s started, deleting %d images", taskID, len(imageIDs))

	projectRoot := filepath.Join(dataPath, "project_item", projectID)
	dbPath := filepath.Join(projectRoot, "db", "project.db")
	db, err := openProjectDB(dbPath)
	if err != nil {
		log.Printf("[DeleteImages] Task %s open db failed: %v", taskID, err)
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
		log.Printf("[DeleteImages] Task %s set WAL failed: %v", taskID, err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 30000;`); err != nil {
		log.Printf("[DeleteImages] Task %s set busy_timeout failed: %v", taskID, err)
	}

	validIDs := make([]int64, 0, len(imageIDs))
	for _, id := range imageIDs {
		if id > 0 {
			validIDs = append(validIDs, id)
		}
	}

	if len(validIDs) == 0 {
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseCompleted
			task.Progress = 100
		}
		importTasksMu.Unlock()
		return
	}

	placeholders := make([]string, len(validIDs))
	args := make([]interface{}, len(validIDs))
	for i, id := range validIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	inClause := strings.Join(placeholders, ",")

	type imageInfo struct {
		ID          int64
		OriginalRel string
		ThumbRel    string
		RefCount    int64
	}
	imageInfos := make(map[int64]imageInfo)

	// 查询图片信息和引用计数（从新表）
	query := fmt.Sprintf(`SELECT i.id, i.original_rel_path, i.thumb_rel_path, COALESCE(r.ref_count, 0) 
		FROM image_index i LEFT JOIN image_ref_count r ON i.id = r.image_id 
		WHERE i.id IN (%s);`, inClause)
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("[DeleteImages] Task %s batch query failed: %v", taskID, err)
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseFailed
			task.Error = "query_failed"
		}
		importTasksMu.Unlock()
		return
	}
	for rows.Next() {
		var info imageInfo
		if err := rows.Scan(&info.ID, &info.OriginalRel, &info.ThumbRel, &info.RefCount); err != nil {
			continue
		}
		imageInfos[info.ID] = info
	}
	rows.Close()

	importTasksMu.Lock()
	if task, ok := importTasks[taskID]; ok {
		task.Progress = 20
	}
	importTasksMu.Unlock()

	// 从项目中删除图片：只设置 deleted_in_project = 1，不减少引用计数，不删除文件
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[DeleteImages] Task %s begin tx failed: %v", taskID, err)
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseFailed
			task.Error = "tx_begin_failed"
		}
		importTasksMu.Unlock()
		return
	}

	// 删除对应的标注数据
	annQuery := fmt.Sprintf(`DELETE FROM annotations WHERE image_id IN (%s);`, inClause)
	_, _ = tx.Exec(annQuery, args...)

	// 标记在项目中已删除
	updQuery := fmt.Sprintf(`UPDATE image_index SET deleted_in_project = 1 WHERE id IN (%s);`, inClause)
	if _, err := tx.Exec(updQuery, args...); err != nil {
		log.Printf("[DeleteImages] Task %s batch update failed: %v", taskID, err)
		tx.Rollback()
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseFailed
			task.Error = "update_failed"
		}
		importTasksMu.Unlock()
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[DeleteImages] Task %s commit failed: %v", taskID, err)
		importTasksMu.Lock()
		if task, ok := importTasks[taskID]; ok {
			task.Phase = importPhaseFailed
			task.Error = "tx_commit_failed"
		}
		importTasksMu.Unlock()
		return
	}

	importTasksMu.Lock()
	if task, ok := importTasks[taskID]; ok {
		task.Phase = importPhaseCompleted
		task.Progress = 100
		task.Total = len(validIDs)
		task.Imported = len(validIDs)
	}
	importTasksMu.Unlock()

	log.Printf("[DeleteImages] Task %s SUCCESS - marked %d images as deleted_in_project",
		taskID, len(validIDs))
}

// handleProjectImageFile 获取项目图片文件
func handleProjectImageFile(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	projectID := strings.TrimSpace(r.URL.Query().Get("projectId"))
	imageIDStr := strings.TrimSpace(r.URL.Query().Get("imageId"))
	kind := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("kind")))
	if kind == "" {
		kind = "thumb"
	}
	if projectID == "" || imageIDStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"project_and_image_required"}`))
		return
	}
	imageID, err := strconv.ParseInt(imageIDStr, 10, 64)
	if err != nil || imageID <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"image_invalid"}`))
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"config_unavailable"}`))
		return
	}

	projectRoot := filepath.Join(cfg.DataPath, "project_item", projectID)
	dbPath := filepath.Join(projectRoot, "db", "project.db")
	db, err := openProjectDB(dbPath)
	if err != nil {
		log.Printf("project image file: open db failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"db_unavailable"}`))
		return
	}
	defer db.Close()

	var originalRel string
	var thumbRel string
	row := db.QueryRow(`SELECT original_rel_path, thumb_rel_path FROM image_index WHERE id = ?;`, imageID)
	if err := row.Scan(&originalRel, &thumbRel); err != nil {
		if err == sql.ErrNoRows {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"image_not_found"}`))
			return
		}
		log.Printf("project image file: query failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"query_failed"}`))
		return
	}

	chosen := originalRel
	if kind == "thumb" {
		if thumbRel != "" {
			chosen = thumbRel
		}
	}
	if chosen == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"file_not_found"}`))
		return
	}

	var fullPath string
	if filepath.IsAbs(chosen) {
		fullPath = chosen
	} else {
		fullPath = filepath.Join(projectRoot, filepath.FromSlash(chosen))
	}

	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"file_not_found"}`))
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=31536000")
	http.ServeFile(w, r, fullPath)
}

// handleProjectImageFileByPath 通过路径获取项目图片文件
func handleProjectImageFileByPath(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	projectID := strings.TrimSpace(r.URL.Query().Get("projectId"))
	pathParam := strings.TrimSpace(r.URL.Query().Get("path"))
	if projectID == "" || pathParam == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"project_and_path_required"}`))
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"config_unavailable"}`))
		return
	}

	var fullPath string
	if filepath.IsAbs(pathParam) {
		fullPath = pathParam
	} else {
		if strings.Contains(pathParam, "..") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"path_invalid"}`))
			return
		}
		projectRoot := filepath.Join(cfg.DataPath, "project_item", projectID)
		fullPath = filepath.Join(projectRoot, filepath.FromSlash(pathParam))
	}

	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"file_not_found"}`))
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=31536000")
	imageFileSem <- struct{}{}
	defer func() { <-imageFileSem }()
	http.ServeFile(w, r, fullPath)
}
