package main

import (
	"database/sql"
	"net/http"
)

// withCORS 添加 CORS 头（保留在 utils.go，其他函数已移至 remaining.go）
func withCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
}

// openProjectDB 打开项目数据库并设置 busy_timeout 以避免并发锁定问题
// 调用者需要负责 defer db.Close()
func openProjectDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	// 设置 busy_timeout，单位毫秒，等待 5 秒
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
