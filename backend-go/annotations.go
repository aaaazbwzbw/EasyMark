package main

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// AnnotationData 标注数据
type AnnotationData struct {
	ID         int64  `json:"id"`
	ImageID    int64  `json:"imageId"`
	CategoryID int64  `json:"categoryId"`
	Type       string `json:"type"`
	Data       string `json:"data"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

// SaveAnnotationsRequest 保存标注请求
type SaveAnnotationsRequest struct {
	ProjectID   string `json:"projectId"`
	ImageID     int64  `json:"imageId"`
	IsNegative  bool   `json:"isNegative"`
	Annotations []struct {
		ID         string `json:"id,omitempty"`
		CategoryID int64  `json:"categoryId"`
		Type       string `json:"type"`
		Data       string `json:"data"`
	} `json:"annotations"`
}

// handleProjectAnnotations 处理项目标注的 GET 请求
func handleProjectAnnotations(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method == http.MethodGet {
		projectID := strings.TrimSpace(r.URL.Query().Get("projectId"))
		imageIDStr := strings.TrimSpace(r.URL.Query().Get("imageId"))
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
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"db_unavailable"}`))
			return
		}
		defer db.Close()

		rows, err := db.Query(`SELECT id, image_id, category_id, type, data, created_at, updated_at FROM annotations WHERE image_id = ?;`, imageID)
		if err != nil {
			log.Printf("annotations get: query failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"query_failed"}`))
			return
		}
		defer rows.Close()

		annotations := []AnnotationData{}
		for rows.Next() {
			var a AnnotationData
			if err := rows.Scan(&a.ID, &a.ImageID, &a.CategoryID, &a.Type, &a.Data, &a.CreatedAt, &a.UpdatedAt); err != nil {
				continue
			}
			annotations = append(annotations, a)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"annotations": annotations})
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

// handleSaveAnnotations 保存标注
func handleSaveAnnotations(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req SaveAnnotationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_request"}`))
		return
	}

	if req.ProjectID == "" || req.ImageID <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"project_and_image_required"}`))
		return
	}

	cfg, err := loadPathsConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"config_unavailable"}`))
		return
	}

	projectRoot := filepath.Join(cfg.DataPath, "project_item", req.ProjectID)
	dbPath := filepath.Join(projectRoot, "db", "project.db")
	db, err := openProjectDB(dbPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"db_unavailable"}`))
		return
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"tx_begin_failed"}`))
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM annotations WHERE image_id = ?;`, req.ImageID)
	if err != nil {
		log.Printf("save annotations: delete old failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"delete_failed"}`))
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)

	for _, ann := range req.Annotations {
		_, err = tx.Exec(`INSERT INTO annotations (image_id, category_id, type, data, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?);`,
			req.ImageID, ann.CategoryID, ann.Type, ann.Data, now, now)
		if err != nil {
			log.Printf("save annotations: insert failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"insert_failed"}`))
			return
		}
	}

	var newStatus string
	if len(req.Annotations) > 0 {
		newStatus = "annotated"
	} else if req.IsNegative {
		newStatus = "negative"
	} else {
		newStatus = "none"
	}
	_, err = tx.Exec(`UPDATE image_index SET annotation_status = ? WHERE id = ?;`, newStatus, req.ImageID)
	if err != nil {
		log.Printf("save annotations: update status failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"tx_commit_failed"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success":true,"status":"` + newStatus + `"}`))
}
