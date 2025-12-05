package main

import (
	"log"
	"net/http"
	"strings"
)

func main() {
	// 初始化日志系统
	initLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/api/projects", handleProjects)
	mux.HandleFunc("/api/import-images", handleImportImages)
	mux.HandleFunc("/api/import-tasks", handleImportTaskStatus)
	mux.HandleFunc("/api/project-images", handleProjectImages)
	mux.HandleFunc("/api/project-images/delete", handleDeleteProjectImages)
	mux.HandleFunc("/api/project-image", handleProjectImageFile)
	mux.HandleFunc("/api/project-image-file", handleProjectImageFileByPath)
	mux.HandleFunc("/api/project-categories", handleProjectCategories)
	mux.HandleFunc("/api/project-categories/edit", handleEditProjectCategory)
	mux.HandleFunc("/api/project-categories/sort", handleSortProjectCategories)
	mux.HandleFunc("/api/project-annotations", handleProjectAnnotations)
	mux.HandleFunc("/api/project-annotations/save", handleSaveAnnotations)
	mux.HandleFunc("/api/annotations", handleAnnotations)
	mux.HandleFunc("/api/annotations/", handleAnnotations)
	// 插件管理
	mux.HandleFunc("/api/plugins", handlePlugins)
	mux.HandleFunc("/api/plugins/install", handlePluginInstall)
	mux.HandleFunc("/api/plugins/import-dataset", handleImportDatasetPlugins)
	// 插件详情（README、Logo、UI、文件列表、模型下载）- 使用前缀匹配
	mux.HandleFunc("/api/plugins/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/readme") {
			handlePluginReadme(w, r)
		} else if strings.HasSuffix(path, "/logo") {
			handlePluginLogo(w, r)
		} else if strings.Contains(path, "/ui/") {
			handlePluginUI(w, r)
		} else if strings.HasSuffix(path, "/files") {
			handlePluginFiles(w, r)
		} else if strings.HasSuffix(path, "/download-model") {
			handlePluginDownloadModel(w, r)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})
	// 数据集导入导出
	mux.HandleFunc("/api/dataset/detect", handleDatasetDetect)
	mux.HandleFunc("/api/dataset/import", handleDatasetImport)
	mux.HandleFunc("/api/dataset/export", handleDatasetExport)
	mux.HandleFunc("/api/dataset/export-status", handleExportStatus)
	mux.HandleFunc("/api/plugins/export", handleExportPlugins)
	mux.HandleFunc("/api/settings/paths", handleSettingsPaths)
	mux.HandleFunc("/api/shell/open-folder", handleOpenFolder)
	// 数据集版本管理
	mux.HandleFunc("/api/dataset-versions", handleDatasetVersions)
	mux.HandleFunc("/api/dataset-versions/create", handleCreateDatasetVersion)
	mux.HandleFunc("/api/dataset-versions/delete", handleDeleteDatasetVersion)
	mux.HandleFunc("/api/dataset-versions/rollback", handleRollbackDatasetVersion)
	mux.HandleFunc("/api/dataset-versions/update", handleUpdateDatasetVersion)
	// 系统资源监控
	mux.HandleFunc("/api/system/stats", handleSystemStats)
	// Python 环境管理（旧接口，训练页使用）
	mux.HandleFunc("/api/python/status", handlePythonStatus)
	mux.HandleFunc("/api/python/deploy", handlePythonDeploy)
	mux.HandleFunc("/api/python/undeploy", handlePythonUndeploy)
	mux.HandleFunc("/api/python/install", handlePythonInstall)
	mux.HandleFunc("/api/python/uninstall", handlePythonUninstall)
	mux.HandleFunc("/api/python/packages", handlePythonPackages)
	// 全局 WebSocket
	mux.HandleFunc("/api/ws", handleGlobalWS)
	// Python 环境管理（新接口，Python 环境页使用）
	mux.HandleFunc("/api/python/version", handlePythonVersion)
	mux.HandleFunc("/api/python/env-status", handlePythonEnvStatus)
	mux.HandleFunc("/api/python/create-venv", handlePythonCreateVenv)
	mux.HandleFunc("/api/python/delete-venv", handlePythonDeleteVenv)
	mux.HandleFunc("/api/python/install-deps", handlePythonInstallDeps)
	mux.HandleFunc("/api/python/stop-install", handlePythonStopInstall)
	mux.HandleFunc("/api/python/installing-tasks", handlePythonInstallingTasks)
	mux.HandleFunc("/api/python/uninstall-dep", handlePythonUninstallDep)
	mux.HandleFunc("/api/python/install-pytorch", handlePythonInstallPytorch)
	mux.HandleFunc("/api/python/run-command", handlePythonRunCommand)
	// Python 插件依赖汇总状态
	mux.HandleFunc("/api/python/plugins-deps-summary", handlePythonPluginsDepsSummary)
	mux.HandleFunc("/api/python/settings", handlePythonSettings)
	mux.HandleFunc("/api/system/gpu", handleSystemGpu)
	// 训练集管理
	mux.HandleFunc("/api/trainsets", handleTrainsets)
	mux.HandleFunc("/api/trainsets/save", handleTrainsetSave)
	mux.HandleFunc("/api/trainsets/delete", handleTrainsetDelete)
	// 训练插件与模型管理
	mux.HandleFunc("/api/training/plugins", handleTrainingPlugins)
	mux.HandleFunc("/api/training/check-deps", handleCheckPluginDeps)
	mux.HandleFunc("/api/training/download-model", handleDownloadModel)
	mux.HandleFunc("/api/training/model-status", handleModelStatus)
	mux.HandleFunc("/api/training/cancel-download", handleCancelDownload)
	// 训练任务管理
	mux.HandleFunc("/api/training/start", handleStartTraining)
	mux.HandleFunc("/api/training/stop", handleStopTraining)
	mux.HandleFunc("/api/training/tasks", handleTrainingTasks)
	mux.HandleFunc("/api/training/history", handleTrainingHistory)
	mux.HandleFunc("/api/training/delete", handleDeleteTrainingTask)
	mux.HandleFunc("/api/training/ws", handleTrainingWS)
	mux.HandleFunc("/api/training/outputs", handleTrainingOutputs)
	// 模型推理辅助
	mux.HandleFunc("/api/inference/ensure-dir", handleInferenceEnsureDir)
	mux.HandleFunc("/api/inference/models", handleInferenceModels)
	mux.HandleFunc("/api/inference/import-model", handleInferenceImportModel)
	mux.HandleFunc("/api/inference/upload-model", handleInferenceUploadModel)
	mux.HandleFunc("/api/inference/open-folder", handleInferenceOpenFolder)
	mux.HandleFunc("/api/inference/start", handleInferenceStart)
	mux.HandleFunc("/api/inference/stop", handleInferenceStop)
	mux.HandleFunc("/api/inference/status", handleInferenceStatus)

	addr := ":18080"
	log.Printf("Starting Go backend on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
