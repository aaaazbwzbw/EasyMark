package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

// handleProjectCategories 处理项目类别的 CRUD 操作
func handleProjectCategories(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method == http.MethodGet {
		projectID := strings.TrimSpace(r.URL.Query().Get("projectId"))
		typeStr := strings.TrimSpace(r.URL.Query().Get("type"))
		versionStr := strings.TrimSpace(r.URL.Query().Get("version"))
		if projectID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"project_required"}`))
			return
		}
		if typeStr != "" {
			switch typeStr {
			case "bbox", "keypoint", "polygon", "category":
			default:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"category_type_invalid"}`))
				return
			}
		}

		cfg, err := loadPathsConfig()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"config_unavailable"}`))
			return
		}

		projectRoot := filepath.Join(cfg.DataPath, "project_item", projectID)
		var dbPath string
		if versionStr != "" {
			dbPath = filepath.Join(projectRoot, "db", "versions", "v"+versionStr, "project.db")
		} else {
			dbPath = filepath.Join(projectRoot, "db", "project.db")
		}
		db, err := openProjectDB(dbPath)
		if err != nil {
			log.Printf("project categories get: open db failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"db_unavailable"}`))
			return
		}
		defer db.Close()

		if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
			log.Printf("project categories get: set busy_timeout failed: %v", err)
		}
		if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS categories (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	color TEXT NOT NULL,
	sort_order INTEGER NOT NULL DEFAULT 0,
	mate TEXT NOT NULL DEFAULT ''
);
`); err != nil {
			log.Printf("project categories get: ensure table failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"schema_unavailable"}`))
			return
		}

		hasMateColumn := false
		colRows, colErr := db.Query(`PRAGMA table_info(categories);`)
		if colErr == nil {
			defer colRows.Close()
			for colRows.Next() {
				var cid int
				var cname string
				var ctype string
				var notnull int
				var dfltValue interface{}
				var pk int
				if err := colRows.Scan(&cid, &cname, &ctype, &notnull, &dfltValue, &pk); err == nil {
					if cname == "mate" {
						hasMateColumn = true
						break
					}
				}
			}
		}

		var rows *sql.Rows
		if hasMateColumn {
			if typeStr == "" {
				rows, err = db.Query(`SELECT id, name, type, color, sort_order, mate FROM categories ORDER BY sort_order ASC, id ASC;`)
			} else {
				rows, err = db.Query(`SELECT id, name, type, color, sort_order, mate FROM categories WHERE type = ? ORDER BY sort_order ASC, id ASC;`, typeStr)
			}
		} else {
			if typeStr == "" {
				rows, err = db.Query(`SELECT id, name, type, color, sort_order FROM categories ORDER BY sort_order ASC, id ASC;`)
			} else {
				rows, err = db.Query(`SELECT id, name, type, color, sort_order FROM categories WHERE type = ? ORDER BY sort_order ASC, id ASC;`, typeStr)
			}
		}
		if err != nil {
			log.Printf("project categories get: query failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"query_failed"}`))
			return
		}
		defer rows.Close()

		type categoryItem struct {
			ID              int64  `json:"id"`
			Name            string `json:"name"`
			Type            string `json:"type"`
			Color           string `json:"color"`
			SortOrder       int64  `json:"sortOrder"`
			Mate            string `json:"mate"`
			AnnotationCount int64  `json:"annotationCount"`
			ImageCount      int64  `json:"imageCount"`
		}
		items := make([]categoryItem, 0, 32)
		for rows.Next() {
			var it categoryItem
			if hasMateColumn {
				if err := rows.Scan(&it.ID, &it.Name, &it.Type, &it.Color, &it.SortOrder, &it.Mate); err != nil {
					log.Printf("project categories get: scan failed: %v", err)
					continue
				}
			} else {
				if err := rows.Scan(&it.ID, &it.Name, &it.Type, &it.Color, &it.SortOrder); err != nil {
					log.Printf("project categories get: scan failed: %v", err)
					continue
				}
				it.Mate = ""
			}
			items = append(items, it)
		}
		if err := rows.Err(); err != nil {
			log.Printf("project categories get: rows error: %v", err)
		}

		for i := range items {
			catID := items[i].ID
			catType := items[i].Type

			if catType == "keypoint" {
				// 关键点类别：关键点数据嵌入在 bbox 标注的 data 字段中
				// 方式1：通过 keypointCategoryId 直接匹配
				// 方式2：通过 bbox 类别的 mate 字段找到绑定的 bbox 类别，统计该类别下带 keypoints 的标注

				// 先找到绑定此关键点类别的 bbox 类别
				var boundBboxCatID int64 = -1
				bboxRows, _ := db.Query(`SELECT id, mate FROM categories WHERE type = 'bbox'`)
				if bboxRows != nil {
					for bboxRows.Next() {
						var bboxID int64
						var mate string
						if err := bboxRows.Scan(&bboxID, &mate); err == nil && mate != "" {
							// 检查 mate 中是否有 keypointCategoryId 指向当前关键点类别
							if strings.Contains(mate, fmt.Sprintf(`"keypointCategoryId":%d`, catID)) ||
								strings.Contains(mate, fmt.Sprintf(`"keypointCategoryId": %d`, catID)) {
								boundBboxCatID = bboxID
								break
							}
						}
					}
					bboxRows.Close()
				}

				var annCount int64
				var imgCount int64

				if boundBboxCatID > 0 {
					// 统计该 bbox 类别下带 keypoints 数据的标注
					if err := db.QueryRow(`
						SELECT COUNT(*) FROM annotations 
						WHERE category_id = ? 
						AND type = 'bbox' 
						AND json_extract(data, '$.keypoints') IS NOT NULL
					`, boundBboxCatID).Scan(&annCount); err == nil {
						items[i].AnnotationCount = annCount
					}
					if err := db.QueryRow(`
						SELECT COUNT(DISTINCT image_id) FROM annotations 
						WHERE category_id = ? 
						AND type = 'bbox' 
						AND json_extract(data, '$.keypoints') IS NOT NULL
					`, boundBboxCatID).Scan(&imgCount); err == nil {
						items[i].ImageCount = imgCount
					}
				} else {
					// 回退：直接通过 keypointCategoryId 匹配
					if err := db.QueryRow(`
						SELECT COUNT(*) FROM annotations 
						WHERE type = 'bbox' 
						AND CAST(json_extract(data, '$.keypointCategoryId') AS INTEGER) = ?
					`, catID).Scan(&annCount); err == nil {
						items[i].AnnotationCount = annCount
					}
					if err := db.QueryRow(`
						SELECT COUNT(DISTINCT image_id) FROM annotations 
						WHERE type = 'bbox' 
						AND CAST(json_extract(data, '$.keypointCategoryId') AS INTEGER) = ?
					`, catID).Scan(&imgCount); err == nil {
						items[i].ImageCount = imgCount
					}
				}
			} else {
				// 其他类别（bbox、polygon 等）：直接按 category_id 统计
				var annCount int64
				if err := db.QueryRow(`SELECT COUNT(*) FROM annotations WHERE category_id = ?`, catID).Scan(&annCount); err == nil {
					items[i].AnnotationCount = annCount
				}
				var imgCount int64
				if err := db.QueryRow(`SELECT COUNT(DISTINCT image_id) FROM annotations WHERE category_id = ?`, catID).Scan(&imgCount); err == nil {
					items[i].ImageCount = imgCount
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(struct {
			Items []categoryItem `json:"items"`
		}{Items: items})
		return
	}
	if r.Method == http.MethodPut {
		var rawReq map[string]interface{}
		bodyBytes, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(bodyBytes, &rawReq); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_json"}`))
			return
		}

		log.Printf("PUT /api/project-categories: rawReq=%v", rawReq)
		if mateStr, hasMate := rawReq["mate"].(string); hasMate {
			projectID, _ := rawReq["projectId"].(string)
			projectID = strings.TrimSpace(projectID)
			categoryID, _ := rawReq["id"].(float64)
			log.Printf("PUT mate format: projectId=%s, categoryID=%v, mate=%s", projectID, categoryID, mateStr)
			if projectID == "" || categoryID <= 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"projectId_and_id_required"}`))
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
				log.Printf("project categories put: open db failed: %v", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"db_unavailable"}`))
				return
			}
			defer db.Close()

			if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
				log.Printf("project categories put: set busy_timeout failed: %v", err)
			}

			res, err := db.Exec(`UPDATE categories SET mate = ? WHERE id = ?;`, mateStr, int64(categoryID))
			if err != nil {
				log.Printf("project categories put: update mate failed: %v", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"update_failed"}`))
				return
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"category_not_found"}`))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success":true}`))
			return
		}

		type updateKeypointsRequest struct {
			ProjectID  string `json:"projectId"`
			CategoryID int64  `json:"categoryId"`
			Keypoints  []struct {
				Name string `json:"name"`
			} `json:"keypoints"`
		}
		var req updateKeypointsRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_json"}`))
			return
		}
		req.ProjectID = strings.TrimSpace(req.ProjectID)
		if req.ProjectID == "" || req.CategoryID <= 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"project_and_category_required"}`))
			return
		}
		if len(req.Keypoints) > 64 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"keypoints_too_many"}`))
			return
		}
		validKeypoints := make([]struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}, 0, len(req.Keypoints))
		for i, kp := range req.Keypoints {
			name := strings.TrimSpace(kp.Name)
			if name == "" {
				continue
			}
			if len(name) > 100 {
				name = name[:100]
			}
			validKeypoints = append(validKeypoints, struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}{ID: i + 1, Name: name})
		}
		if len(validKeypoints) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"keypoints_empty"}`))
			return
		}
		for i := range validKeypoints {
			validKeypoints[i].ID = i + 1
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
			log.Printf("project categories put: open db failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"db_unavailable"}`))
			return
		}
		defer db.Close()
		if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
			log.Printf("project categories put: set busy_timeout failed: %v", err)
		}

		hasMateColumn := false
		colRows, colErr := db.Query(`PRAGMA table_info(categories);`)
		if colErr == nil {
			defer colRows.Close()
			for colRows.Next() {
				var cid int
				var cname string
				var ctype string
				var notnull int
				var dfltValue interface{}
				var pk int
				if err := colRows.Scan(&cid, &cname, &ctype, &notnull, &dfltValue, &pk); err == nil {
					if cname == "mate" {
						hasMateColumn = true
						break
					}
				}
			}
		}
		if !hasMateColumn {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"mate_unsupported"}`))
			return
		}

		var catType string
		row := db.QueryRow(`SELECT type FROM categories WHERE id = ?;`, req.CategoryID)
		if err := row.Scan(&catType); err != nil {
			log.Printf("project categories put: category not found: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"category_not_found"}`))
			return
		}
		if catType != "keypoint" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"category_type_not_keypoint"}`))
			return
		}

		mateObj := struct {
			Keypoints []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"keypoints"`
		}{Keypoints: validKeypoints}
		mateBytes, err := json.Marshal(mateObj)
		if err != nil {
			log.Printf("project categories put: marshal mate failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"internal_error"}`))
			return
		}

		_, err = db.Exec(`UPDATE categories SET mate = ? WHERE id = ?;`, string(mateBytes), req.CategoryID)
		if err != nil {
			log.Printf("project categories put: update mate failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"update_failed"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
		return
	}

	if r.Method == http.MethodDelete {
		type deleteCategoryRequest struct {
			ProjectID  string `json:"projectId"`
			CategoryID int64  `json:"categoryId"`
		}
		var req deleteCategoryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_json"}`))
			return
		}
		req.ProjectID = strings.TrimSpace(req.ProjectID)
		if req.ProjectID == "" || req.CategoryID <= 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"project_and_category_required"}`))
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
			log.Printf("project categories delete: open db failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"db_unavailable"}`))
			return
		}
		defer db.Close()
		if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
			log.Printf("project categories delete: set busy_timeout failed: %v", err)
		}

		var catName string
		row := db.QueryRow(`SELECT name FROM categories WHERE id = ?;`, req.CategoryID)
		if err := row.Scan(&catName); err != nil {
			log.Printf("project categories delete: category not found: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"category_not_found"}`))
			return
		}

		// 删除类别前先删除相关标注
		_, _ = db.Exec(`DELETE FROM annotations WHERE category_id = ?;`, req.CategoryID)

		// 更新没有任何标注的图片状态为未标注
		_, _ = db.Exec(`UPDATE image_index SET annotation_status = 'none' 
			WHERE annotation_status = 'annotated' AND id NOT IN (SELECT DISTINCT image_id FROM annotations);`)

		res, err := db.Exec(`DELETE FROM categories WHERE id = ?;`, req.CategoryID)
		if err != nil {
			log.Printf("project categories delete: delete failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"delete_failed"}`))
			return
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"category_not_found"}`))
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

	type createCategoryRequest struct {
		ProjectID string `json:"projectId"`
		Name      string `json:"name"`
		Type      string `json:"type"`
		Color     string `json:"color"`
		Mate      string `json:"mate"`
	}

	var req createCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_json"}`))
		return
	}
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	name := strings.TrimSpace(req.Name)
	typeStr := strings.TrimSpace(req.Type)
	color := strings.TrimSpace(req.Color)
	mate := strings.TrimSpace(req.Mate)
	if req.ProjectID == "" || name == "" || typeStr == "" || color == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"project_name_type_color_required"}`))
		return
	}

	switch typeStr {
	case "bbox", "keypoint", "polygon", "category":
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"category_type_invalid"}`))
		return
	}

	if len(color) != 7 || color[0] != '#' {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"color_invalid"}`))
		return
	}
	for i := 1; i < 7; i++ {
		c := color[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"color_invalid"}`))
			return
		}
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
		log.Printf("project categories: open db failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"db_unavailable"}`))
		return
	}
	defer db.Close()

	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		log.Printf("project categories: set busy_timeout failed: %v", err)
	}

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS categories (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	color TEXT NOT NULL,
	sort_order INTEGER NOT NULL DEFAULT 0,
	mate TEXT NOT NULL DEFAULT ''
);
`); err != nil {
		log.Printf("project categories: ensure table failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"schema_unavailable"}`))
		return
	}
	if _, err := db.Exec(`DROP INDEX IF EXISTS idx_categories_name;`); err != nil {
		log.Printf("project categories: drop old index failed: %v", err)
	}

	var sortOrder int64
	row := db.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM categories WHERE type = ?;`, typeStr)
	if err := row.Scan(&sortOrder); err != nil {
		log.Printf("project categories: query sort_order failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"query_failed"}`))
		return
	}

	res, err := db.Exec(`INSERT INTO categories (name, type, color, sort_order, mate) VALUES (?, ?, ?, ?, ?);`, name, typeStr, color, sortOrder, mate)
	if err != nil {
		log.Printf("project categories: insert failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"insert_failed"}`))
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("project categories: lastInsertId failed: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	type createCategoryResponse struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		Type      string `json:"type"`
		Color     string `json:"color"`
		SortOrder int64  `json:"sortOrder"`
	}
	_ = json.NewEncoder(w).Encode(createCategoryResponse{
		ID:        id,
		Name:      name,
		Type:      typeStr,
		Color:     color,
		SortOrder: sortOrder,
	})
}

// handleEditProjectCategory 更新类别的名称和颜色
func handleEditProjectCategory(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type editCategoryRequest struct {
		ProjectID     string `json:"projectId"`
		CategoryID    int64  `json:"categoryId"`
		Name          string `json:"name"`
		Color         string `json:"color"`
		Merge         bool   `json:"merge"`
		MergeTargetID int64  `json:"mergeTargetId"`
	}

	var req editCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_json"}`))
		return
	}
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	name := strings.TrimSpace(req.Name)
	color := strings.TrimSpace(req.Color)
	if req.ProjectID == "" || req.CategoryID <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"project_and_category_required"}`))
		return
	}
	if name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"name_required"}`))
		return
	}
	if color == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"color_required"}`))
		return
	}
	if len(name) > 100 {
		name = name[:100]
	}
	if len(color) > 100 {
		color = color[:100]
	}
	if len(color) != 7 || color[0] != '#' {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"color_invalid"}`))
		return
	}
	for i := 1; i < 7; i++ {
		c := color[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"color_invalid"}`))
			return
		}
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
		log.Printf("project categories edit: open db failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"db_unavailable"}`))
		return
	}
	defer db.Close()
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		log.Printf("project categories edit: set busy_timeout failed: %v", err)
	}

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS categories (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	color TEXT NOT NULL,
	sort_order INTEGER NOT NULL DEFAULT 0,
	mate TEXT NOT NULL DEFAULT ''
);
`); err != nil {
		log.Printf("project categories edit: ensure table failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"schema_unavailable"}`))
		return
	}

	var existingID int64
	row := db.QueryRow(`SELECT id FROM categories WHERE id = ?;`, req.CategoryID)
	if err := row.Scan(&existingID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"category_not_found"}`))
			return
		}
		log.Printf("project categories edit: query failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"query_failed"}`))
		return
	}

	// 如果是合并操作
	if req.Merge && req.MergeTargetID > 0 {
		// 将所有标注从当前类别转移到目标类别
		_, err := db.Exec(`UPDATE annotations SET category_id = ? WHERE category_id = ?;`, req.MergeTargetID, req.CategoryID)
		if err != nil {
			log.Printf("project categories merge: update annotations failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"merge_failed"}`))
			return
		}
		// 删除当前类别
		_, err = db.Exec(`DELETE FROM categories WHERE id = ?;`, req.CategoryID)
		if err != nil {
			log.Printf("project categories merge: delete category failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"merge_failed"}`))
			return
		}
		log.Printf("project categories merge: merged category %d into %d", req.CategoryID, req.MergeTargetID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"merged":true}`))
		return
	}

	res, err := db.Exec(`UPDATE categories SET name = ?, color = ? WHERE id = ?;`, name, color, req.CategoryID)
	if err != nil {
		log.Printf("project categories edit: update failed: %v", err)
		low := strings.ToLower(err.Error())
		status := http.StatusInternalServerError
		code := "update_failed"
		if strings.Contains(low, "unique constraint failed") {
			status = http.StatusConflict
			code = "category_exists"
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":"%s"}`, code)))
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"category_not_found"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success":true}`))
}

// handleSortProjectCategories 更新类别排序
func handleSortProjectCategories(w http.ResponseWriter, r *http.Request) {
	withCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type sortCategoriesRequest struct {
		ProjectID   string  `json:"projectId"`
		Type        string  `json:"type"`
		CategoryIDs []int64 `json:"categoryIds"`
	}

	var req sortCategoriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_json"}`))
		return
	}
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	typeStr := strings.TrimSpace(req.Type)
	if req.ProjectID == "" || typeStr == "" || len(req.CategoryIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"project_type_ids_required"}`))
		return
	}

	switch typeStr {
	case "bbox", "keypoint", "polygon", "category":
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"category_type_invalid"}`))
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
		log.Printf("project categories sort: open db failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"db_unavailable"}`))
		return
	}
	defer db.Close()
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		log.Printf("project categories sort: set busy_timeout failed: %v", err)
	}

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS categories (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	color TEXT NOT NULL,
	sort_order INTEGER NOT NULL DEFAULT 0,
	mate TEXT NOT NULL DEFAULT ''
);
`); err != nil {
		log.Printf("project categories sort: ensure table failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"schema_unavailable"}`))
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("project categories sort: begin tx failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"tx_begin_failed"}`))
		return
	}

	sortOrder := int64(1)
	for _, id := range req.CategoryIDs {
		if id <= 0 {
			continue
		}
		if _, err := tx.Exec(`UPDATE categories SET sort_order = ? WHERE id = ? AND type = ?;`, sortOrder, id, typeStr); err != nil {
			_ = tx.Rollback()
			log.Printf("project categories sort: update failed for id=%d: %v", id, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"update_failed"}`))
			return
		}
		sortOrder++
	}
	if err := tx.Commit(); err != nil {
		log.Printf("project categories sort: commit failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"tx_commit_failed"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success":true}`))
}
