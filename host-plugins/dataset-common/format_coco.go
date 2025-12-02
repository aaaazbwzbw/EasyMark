package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ==================== COCO Format Structures ====================

type COCODataset struct {
	Images      []COCOImage      `json:"images"`
	Annotations []COCOAnnotation `json:"annotations"`
	Categories  []COCOCategory   `json:"categories"`
}

type COCOImage struct {
	ID       int    `json:"id"`
	FileName string `json:"file_name"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type COCOAnnotation struct {
	ID           int         `json:"id"`
	ImageID      int         `json:"image_id"`
	CategoryID   int         `json:"category_id"`
	BBox         []float64   `json:"bbox,omitempty"`
	Keypoints    []float64   `json:"keypoints,omitempty"`
	Segmentation interface{} `json:"segmentation,omitempty"`
	Area         float64     `json:"area,omitempty"`
	IsCrowd      int         `json:"iscrowd,omitempty"`
}

type COCOCategory struct {
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	Supercategory string   `json:"supercategory,omitempty"`
	Keypoints     []string `json:"keypoints,omitempty"`
	Skeleton      [][]int  `json:"skeleton,omitempty"`
}

// ==================== COCO Detection ====================

func DetectCOCO(rootPath string) FormatScore {
	result := FormatScore{Format: "coco", Score: 0}

	files := findCOCOAnnotationFiles(rootPath)
	if len(files) == 0 {
		result.Reason = "No COCO annotation files found"
		return result
	}

	for _, f := range files {
		if validateCOCOFile(f) {
			result.Score = 0.9
			result.Reason = fmt.Sprintf("Found COCO annotation: %s", filepath.Base(f))
			return result
		}
	}

	result.Reason = "JSON files found but not valid COCO format"
	return result
}

func findCOCOAnnotationFiles(rootPath string) []string {
	var files []string
	visited := make(map[string]bool)

	addJSONFiles := func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
				fullPath := filepath.Join(dir, e.Name())
				if !visited[fullPath] {
					visited[fullPath] = true
					files = append(files, fullPath)
				}
			}
		}
	}

	addJSONFiles(rootPath)
	// 支持多种目录结构
	for _, d := range []string{
		"annotations", "labels", "annotation", "label", "json",
		"train/annotations", "val/annotations", "test/annotations", // train/val/test 子目录结构
	} {
		addJSONFiles(filepath.Join(rootPath, d))
	}

	// 检查根目录下的子目录
	entries, _ := os.ReadDir(rootPath)
	for _, e := range entries {
		if e.IsDir() {
			subDir := filepath.Join(rootPath, e.Name())
			addJSONFiles(subDir)
			// 也检查子目录下的 annotations 目录
			addJSONFiles(filepath.Join(subDir, "annotations"))
		}
	}

	return files
}

func validateCOCOFile(filePath string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	var dataset map[string]interface{}
	if err := json.Unmarshal(data, &dataset); err != nil {
		return false
	}

	_, hasImages := dataset["images"]
	_, hasAnnotations := dataset["annotations"]
	_, hasCategories := dataset["categories"]

	return hasImages && hasAnnotations && hasCategories
}

// ==================== COCO Import ====================

func ImportCOCO(req ImportRequest, resp *ImportResponse) {
	annotationFile := ""
	if v, ok := req.Params["annotationFile"].(string); ok && v != "" {
		annotationFile = v
		if !filepath.IsAbs(annotationFile) {
			annotationFile = filepath.Join(req.RootPath, annotationFile)
		}
	} else {
		files := findCOCOAnnotationFiles(req.RootPath)
		if len(files) > 0 {
			annotationFile = files[0]
		}
	}

	if annotationFile == "" {
		resp.Errors = append(resp.Errors, ErrorItem{
			Code:    "missing_annotation_file",
			Message: "No COCO annotation file found",
		})
		return
	}

	dataset, err := parseCOCOFile(annotationFile)
	if err != nil {
		resp.Errors = append(resp.Errors, ErrorItem{
			Code:    "invalid_annotation_file",
			Message: err.Error(),
			Details: map[string]interface{}{"file": annotationFile},
		})
		return
	}

	// Determine images directory
	imagesDir := ""
	if v, ok := req.Params["imagesDir"].(string); ok && v != "" {
		imagesDir = v
	} else {
		possibleDirs := []string{
			"images", "train2017", "val2017", "test2017",
			"train", "val", "test", "train2014", "val2014",
			"JPEGImages", "img", "imgs", "data", "photos",
		}
		for _, d := range possibleDirs {
			testPath := filepath.Join(req.RootPath, d)
			if info, err := os.Stat(testPath); err == nil && info.IsDir() {
				imagesDir = d
				break
			}
		}
	}

	// Build image ID to key mapping
	imageIDMap := make(map[int]string)
	imageSizeMap := make(map[int][2]int)
	for _, img := range dataset.Images {
		relPath := img.FileName
		if imagesDir != "" {
			relPath = filepath.Join(imagesDir, img.FileName)
		}
		key := relPath
		imageIDMap[img.ID] = key
		imageSizeMap[img.ID] = [2]int{img.Width, img.Height}

		resp.Images = append(resp.Images, ImageRef{
			Key:          key,
			RelativePath: relPath,
			Meta: map[string]interface{}{
				"width":  img.Width,
				"height": img.Height,
			},
		})
	}
	resp.Stats.ImageCount = len(resp.Images)

	// Build category mapping
	// categoryIDMap: COCO category ID -> bbox category key
	// keypointCategoryMap: COCO category ID -> keypoint category key (if has keypoints)
	categoryIDMap := make(map[int]string)
	keypointCategoryMap := make(map[int]string)
	colorIdx := 0

	// First pass: create all keypoint categories
	for _, cat := range dataset.Categories {
		if len(cat.Keypoints) == 0 {
			continue
		}
		kpKey := fmt.Sprintf("cat_%d_kp", cat.ID)
		keypointCategoryMap[cat.ID] = kpKey

		keypoints := []map[string]interface{}{}
		for j, kpName := range cat.Keypoints {
			keypoints = append(keypoints, map[string]interface{}{
				"id":   j + 1,
				"name": kpName,
			})
		}
		meta := map[string]interface{}{
			"keypoints": keypoints,
		}
		if len(cat.Skeleton) > 0 {
			meta["skeleton"] = cat.Skeleton
		}

		resp.Categories = append(resp.Categories, CategoryDef{
			Key:       kpKey,
			Name:      cat.Name + "_keypoints",
			Type:      "keypoint",
			Color:     GetColor(colorIdx),
			SortOrder: colorIdx + 1,
			Meta:      meta,
		})
		colorIdx++
	}

	// Second pass: create bbox categories with keypoint binding
	for _, cat := range dataset.Categories {
		bboxKey := fmt.Sprintf("cat_%d", cat.ID)
		categoryIDMap[cat.ID] = bboxKey

		// Build meta with keypoint category binding
		meta := map[string]interface{}{}
		if kpKey, hasKp := keypointCategoryMap[cat.ID]; hasKp {
			meta["keypointCategoryKey"] = kpKey // Backend will resolve to ID
		}

		resp.Categories = append(resp.Categories, CategoryDef{
			Key:       bboxKey,
			Name:      cat.Name,
			Type:      "bbox",
			Color:     GetColor(colorIdx),
			SortOrder: colorIdx + 1,
			Meta:      meta,
		})
		colorIdx++
	}

	// Process annotations
	for _, ann := range dataset.Annotations {
		imageKey, ok := imageIDMap[ann.ImageID]
		if !ok {
			resp.Stats.SkippedAnnotations++
			continue
		}
		categoryKey, ok := categoryIDMap[ann.CategoryID]
		if !ok {
			resp.Stats.SkippedAnnotations++
			continue
		}

		size := imageSizeMap[ann.ImageID]
		imgWidth := float64(size[0])
		imgHeight := float64(size[1])
		if imgWidth == 0 {
			imgWidth = 1
		}
		if imgHeight == 0 {
			imgHeight = 1
		}

		// Always create bbox annotation
		data := map[string]interface{}{}

		if len(ann.BBox) == 4 {
			data["x"] = ann.BBox[0] / imgWidth
			data["y"] = ann.BBox[1] / imgHeight
			data["width"] = ann.BBox[2] / imgWidth
			data["height"] = ann.BBox[3] / imgHeight
		}

		// If has keypoints, embed them in bbox annotation
		if len(ann.Keypoints) > 0 && len(ann.Keypoints)%3 == 0 {
			points := [][]float64{}
			for i := 0; i < len(ann.Keypoints); i += 3 {
				x := ann.Keypoints[i] / imgWidth
				y := ann.Keypoints[i+1] / imgHeight
				v := ann.Keypoints[i+2]
				points = append(points, []float64{x, y, v})
			}
			data["keypoints"] = points
			// Reference the keypoint category by key (backend will resolve to ID)
			if kpKey, hasKp := keypointCategoryMap[ann.CategoryID]; hasKp {
				data["keypointCategoryKey"] = kpKey
			}
		}

		resp.Annotations = append(resp.Annotations, AnnotationDef{
			ImageKey:    imageKey,
			CategoryKey: categoryKey,
			Type:        "bbox",
			Data:        data,
		})
	}
	resp.Stats.AnnotationCount = len(resp.Annotations)
}

func parseCOCOFile(filePath string) (*COCODataset, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var dataset COCODataset
	if err := json.Unmarshal(data, &dataset); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return &dataset, nil
}
