// Common Dataset Import Plugin for EasyMark
// Supports COCO, YOLO, and Pascal VOC formats
package main

// ==================== Protocol Types ====================

type DetectRequest struct {
	RootPath   string                 `json:"rootPath"`
	HintFormat string                 `json:"hintFormat,omitempty"`
	Params     map[string]interface{} `json:"params,omitempty"`
}

type DetectResponse struct {
	Supported bool    `json:"supported"`
	Score     float64 `json:"score"`
	Reason    string  `json:"reason"`
	FormatID  string  `json:"formatId"`
}

type ImportRequest struct {
	RootPath string                 `json:"rootPath"`
	FormatID string                 `json:"formatId"`
	Params   map[string]interface{} `json:"params"`
}

type ImportResponse struct {
	Images      []ImageRef      `json:"images"`
	Categories  []CategoryDef   `json:"categories"`
	Annotations []AnnotationDef `json:"annotations"`
	Stats       Stats           `json:"stats"`
	Errors      []ErrorItem     `json:"errors"`
}

type ImageRef struct {
	Key          string                 `json:"key"`
	RelativePath string                 `json:"relativePath"`
	Meta         map[string]interface{} `json:"meta,omitempty"`
}

type CategoryDef struct {
	Key       string                 `json:"key"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Color     string                 `json:"color,omitempty"`
	SortOrder int                    `json:"sortOrder,omitempty"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
}

type AnnotationDef struct {
	ImageKey    string                 `json:"imageKey"`
	CategoryKey string                 `json:"categoryKey"`
	Type        string                 `json:"type"`
	Data        map[string]interface{} `json:"data"`
}

type Stats struct {
	ImageCount         int `json:"imageCount"`
	AnnotationCount    int `json:"annotationCount"`
	SkippedImages      int `json:"skippedImages"`
	SkippedAnnotations int `json:"skippedAnnotations"`
}

type ErrorItem struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ==================== Format Detection Result ====================

type FormatScore struct {
	Format string
	Score  float64
	Reason string
}

// ==================== Export Types ====================

type ExportRequest struct {
	Format      string             `json:"format"`
	OutputDir   string             `json:"outputDir"`
	Images      []ExportImage      `json:"images"`
	Categories  []ExportCategory   `json:"categories"`
	Annotations []ExportAnnotation `json:"annotations"`
	Split       SplitConfig        `json:"split"`
}

type ExportImage struct {
	Key          string `json:"key"`
	RelativePath string `json:"relativePath"`
	AbsolutePath string `json:"absolutePath"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Split        string `json:"split"` // train, val, test
}

type ExportCategory struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Color string `json:"color"`
	Mate  string `json:"mate,omitempty"` // 关键点配置等元数据 (JSON 字符串)
}

type ExportAnnotation struct {
	ID         int                    `json:"id,omitempty"` // 标注ID，用于关键点引用矩形框
	ImageKey   string                 `json:"imageKey"`
	CategoryID int                    `json:"categoryId"`
	Type       string                 `json:"type"`
	Data       map[string]interface{} `json:"data"`
}

type SplitConfig struct {
	Train int `json:"train"`
	Val   int `json:"val"`
	Test  int `json:"test"`
}

type ExportResponse struct {
	Success   bool            `json:"success"`
	Structure ExportStructure `json:"structure"`
	Stats     ExportStats     `json:"stats"`
	Errors    []ErrorItem     `json:"errors"`
}

type ExportStructure struct {
	Directories []string     `json:"directories"`
	Files       []FileOutput `json:"files"`
	CopyImages  []CopyTask   `json:"copyImages"`
}

type FileOutput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type CopyTask struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type ExportStats struct {
	ImageCount      int `json:"imageCount"`
	AnnotationCount int `json:"annotationCount"`
	TrainCount      int `json:"trainCount"`
	ValCount        int `json:"valCount"`
	TestCount       int `json:"testCount"`
}

// ==================== Color Palette ====================

var DefaultColors = []string{
	"#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4",
	"#FFEAA7", "#DDA0DD", "#98D8C8", "#F7DC6F",
	"#74B9FF", "#A29BFE", "#FD79A8", "#00CEC9",
}

func GetColor(index int) string {
	return DefaultColors[index%len(DefaultColors)]
}
