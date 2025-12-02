package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ==================== YOLO Detection ====================

func DetectYOLO(rootPath string) FormatScore {
	result := FormatScore{Format: "yolo", Score: 0}

	labelFiles := findYOLOLabelFiles(rootPath)
	if len(labelFiles) == 0 {
		result.Reason = "No YOLO label files found"
		return result
	}

	for _, f := range labelFiles {
		if isValidYOLOLabelFile(f) {
			result.Score = 0.85
			result.Reason = fmt.Sprintf("Found YOLO label: %s", filepath.Base(f))
			return result
		}
	}

	result.Reason = "TXT files found but not valid YOLO format"
	return result
}

func findYOLOLabelFiles(rootPath string) []string {
	var files []string
	visited := make(map[string]bool)

	tryDir := func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if e.IsDir() || strings.ToLower(filepath.Ext(e.Name())) != ".txt" {
				continue
			}
			// Skip common non-label files
			name := strings.ToLower(e.Name())
			if name == "classes.txt" || name == "data.names" || name == "obj.names" {
				continue
			}
			full := filepath.Join(dir, e.Name())
			if !visited[full] {
				visited[full] = true
				files = append(files, full)
			}
		}
	}

	tryDir(rootPath)
	// 支持多种目录结构
	for _, d := range []string{
		"labels", "labels/train", "labels/val",
		"train/labels", "val/labels", "test/labels", // train/val/test 子目录结构
		"label",
	} {
		tryDir(filepath.Join(rootPath, d))
	}

	// 检查根目录下的子目录
	entries, _ := os.ReadDir(rootPath)
	for _, e := range entries {
		if e.IsDir() {
			subDir := filepath.Join(rootPath, e.Name())
			tryDir(subDir)
			// 也检查子目录下的 labels 目录
			tryDir(filepath.Join(subDir, "labels"))
		}
	}

	return files
}

func isValidYOLOLabelFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 5 {
			return false
		}
		if _, err := strconv.Atoi(parts[0]); err != nil {
			return false
		}
		for i := 1; i < 5; i++ {
			if _, err := strconv.ParseFloat(parts[i], 64); err != nil {
				return false
			}
		}
		return true
	}
	return false
}

// ==================== YOLO Import ====================

func ImportYOLO(req ImportRequest, resp *ImportResponse) {
	root := req.RootPath

	// Resolve labels directory
	labelsDirAbs := ""
	if v, ok := req.Params["labelsDir"].(string); ok && v != "" {
		if filepath.IsAbs(v) {
			labelsDirAbs = v
		} else {
			labelsDirAbs = filepath.Join(root, v)
		}
	} else {
		labelsDirAbs = guessYOLOLabelsDir(root)
	}

	if labelsDirAbs == "" {
		resp.Errors = append(resp.Errors, ErrorItem{
			Code:    "missing_labels_dir",
			Message: "No labels directory found",
		})
		return
	}

	// Collect label files
	var labelFiles []string
	filepath.WalkDir(labelsDirAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(d.Name())) == ".txt" {
			name := strings.ToLower(d.Name())
			if name != "classes.txt" && name != "data.names" && name != "obj.names" {
				labelFiles = append(labelFiles, path)
			}
		}
		return nil
	})

	if len(labelFiles) == 0 {
		resp.Errors = append(resp.Errors, ErrorItem{
			Code:    "no_label_files",
			Message: "No YOLO label files found",
			Details: map[string]interface{}{"labelsDir": labelsDirAbs},
		})
		return
	}

	// First pass: find used class IDs and keypoint count per class
	usedClassIDs := make(map[int]bool)      // only class IDs that actually appear in labels
	classKeypointCount := make(map[int]int) // class ID -> max keypoint count
	maxClassID := -1
	for _, lf := range labelFiles {
		boxes := parseYOLOLabelFile(lf, resp)
		for _, b := range boxes {
			usedClassIDs[b.classID] = true
			if b.classID > maxClassID {
				maxClassID = b.classID
			}
			if len(b.keypoints) > classKeypointCount[b.classID] {
				classKeypointCount[b.classID] = len(b.keypoints)
			}
		}
	}

	if len(usedClassIDs) == 0 {
		return
	}

	// Build class names (read from classes.txt or similar)
	classNames := buildYOLOClassNames(root, req.Params, maxClassID+1)

	// Build categories - only for actually used class IDs
	classKeyMap := make(map[int]string)         // class ID -> bbox category key
	keypointCategoryMap := make(map[int]string) // class ID -> keypoint category key
	colorIdx := 0

	// First: create keypoint categories for used classes that have keypoints
	for id := range usedClassIDs {
		if kpCount := classKeypointCount[id]; kpCount > 0 {
			kpKey := fmt.Sprintf("class_%d_kp", id)
			keypointCategoryMap[id] = kpKey

			name := classNames[id]
			if name == "" {
				name = fmt.Sprintf("class_%d", id)
			}

			// Generate keypoint semantics (1, 2, 3, ...)
			keypoints := make([]map[string]interface{}, kpCount)
			for i := 0; i < kpCount; i++ {
				keypoints[i] = map[string]interface{}{
					"id":   i + 1,
					"name": fmt.Sprintf("%d", i+1),
				}
			}

			resp.Categories = append(resp.Categories, CategoryDef{
				Key:       kpKey,
				Name:      name + "_keypoints",
				Type:      "keypoint",
				Color:     GetColor(colorIdx),
				SortOrder: colorIdx + 1,
				Meta: map[string]interface{}{
					"keypoints": keypoints,
				},
			})
			colorIdx++
		}
	}

	// Second: create bbox categories only for used class IDs
	for id := range usedClassIDs {
		bboxKey := fmt.Sprintf("class_%d", id)
		classKeyMap[id] = bboxKey
		name := classNames[id]
		if name == "" {
			name = fmt.Sprintf("class_%d", id)
		}

		// Build meta with keypoint category binding
		meta := map[string]interface{}{}
		if kpKey, hasKp := keypointCategoryMap[id]; hasKp {
			meta["keypointCategoryKey"] = kpKey // Backend will resolve to ID
		}

		resp.Categories = append(resp.Categories, CategoryDef{
			Key:       bboxKey,
			Name:      name,
			Type:      "bbox",
			Color:     GetColor(colorIdx),
			SortOrder: colorIdx + 1,
			Meta:      meta,
		})
		colorIdx++
	}

	// Resolve images directory
	imagesDirAbs := resolveYOLOImagesDir(root, req.Params)

	// Build image index
	imageIndex, images := buildYOLOImageIndex(root, imagesDirAbs)
	resp.Images = images
	resp.Stats.ImageCount = len(images)

	// Second pass: build annotations
	for _, lf := range labelFiles {
		boxes := parseYOLOLabelFile(lf, resp)
		if len(boxes) == 0 {
			continue
		}
		base := strings.TrimSuffix(filepath.Base(lf), filepath.Ext(lf))
		imageKey, ok := imageIndex[base]
		if !ok {
			resp.Stats.SkippedAnnotations += len(boxes)
			continue
		}

		for _, b := range boxes {
			catKey, ok := classKeyMap[b.classID]
			if !ok {
				resp.Stats.SkippedAnnotations++
				continue
			}
			// Convert from center-based to top-left
			x := b.xCenter - b.width/2.0
			y := b.yCenter - b.height/2.0

			data := map[string]interface{}{
				"x":      x,
				"y":      y,
				"width":  b.width,
				"height": b.height,
			}

			// If has keypoints, embed them in the bbox annotation
			if len(b.keypoints) > 0 {
				data["keypoints"] = b.keypoints
				// Reference the keypoint category by key (backend will resolve to ID)
				if kpKey, hasKp := keypointCategoryMap[b.classID]; hasKp {
					data["keypointCategoryKey"] = kpKey
				}
			}

			resp.Annotations = append(resp.Annotations, AnnotationDef{
				ImageKey:    imageKey,
				CategoryKey: catKey,
				Type:        "bbox",
				Data:        data,
			})
		}
	}
	resp.Stats.AnnotationCount = len(resp.Annotations)
}

type yoloBox struct {
	classID   int
	xCenter   float64
	yCenter   float64
	width     float64
	height    float64
	keypoints [][]float64 // [[x, y, visibility], ...]
}

func parseYOLOLabelFile(path string, resp *ImportResponse) []yoloBox {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var boxes []yoloBox
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 5 {
			resp.Stats.SkippedAnnotations++
			continue
		}

		classID, err := strconv.Atoi(parts[0])
		if err != nil {
			resp.Stats.SkippedAnnotations++
			continue
		}

		vals := make([]float64, 4)
		ok := true
		for i := 0; i < 4; i++ {
			v, err := strconv.ParseFloat(parts[i+1], 64)
			if err != nil {
				ok = false
				break
			}
			vals[i] = v
		}
		if !ok {
			resp.Stats.SkippedAnnotations++
			continue
		}

		box := yoloBox{
			classID: classID,
			xCenter: vals[0],
			yCenter: vals[1],
			width:   vals[2],
			height:  vals[3],
		}

		// Parse keypoints if present (format: x y visibility, repeated)
		// After bbox (5 values), remaining values are keypoints in groups of 3
		remaining := parts[5:]
		if len(remaining) >= 3 && len(remaining)%3 == 0 {
			numKeypoints := len(remaining) / 3
			keypoints := make([][]float64, 0, numKeypoints)
			for i := 0; i < numKeypoints; i++ {
				x, err1 := strconv.ParseFloat(remaining[i*3], 64)
				y, err2 := strconv.ParseFloat(remaining[i*3+1], 64)
				v, err3 := strconv.ParseFloat(remaining[i*3+2], 64)
				if err1 != nil || err2 != nil || err3 != nil {
					break
				}
				keypoints = append(keypoints, []float64{x, y, v})
			}
			if len(keypoints) == numKeypoints {
				box.keypoints = keypoints
			}
		}

		boxes = append(boxes, box)
	}

	return boxes
}

func guessYOLOLabelsDir(root string) string {
	// 支持多种目录结构
	candidates := []string{
		"labels",
		"train/labels", // train/val/test 子目录结构
		"val/labels",
		"labels/train",
		"labels/val",
		"label",
	}
	for _, rel := range candidates {
		p := filepath.Join(root, rel)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	return ""
}

func resolveYOLOImagesDir(root string, params map[string]interface{}) string {
	if v, ok := params["imagesDir"].(string); ok && v != "" {
		if filepath.IsAbs(v) {
			return v
		}
		return filepath.Join(root, v)
	}

	// 支持多种目录结构
	candidates := []string{
		"images",
		"train/images", // train/val/test 子目录结构
		"val/images",
		"images/train",
		"images/val",
		"img", "imgs", "JPEGImages",
	}
	for _, rel := range candidates {
		p := filepath.Join(root, rel)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	return ""
}

func buildYOLOImageIndex(root, imagesDirAbs string) (map[string]string, []ImageRef) {
	imageIndex := make(map[string]string)
	var images []ImageRef

	searchRoot := imagesDirAbs
	if searchRoot == "" {
		searchRoot = root
	}

	validExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true,
		".bmp": true, ".webp": true,
	}

	filepath.WalkDir(searchRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if !validExts[ext] {
			return nil
		}
		base := strings.TrimSuffix(d.Name(), ext)
		if _, exists := imageIndex[base]; exists {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = d.Name()
		}

		imageIndex[base] = rel
		images = append(images, ImageRef{Key: rel, RelativePath: rel})
		return nil
	})

	return imageIndex, images
}

func buildYOLOClassNames(root string, params map[string]interface{}, count int) []string {
	names := make([]string, count)

	classesPath := ""
	if v, ok := params["classesFile"].(string); ok && v != "" {
		if filepath.IsAbs(v) {
			classesPath = v
		} else {
			classesPath = filepath.Join(root, v)
		}
	} else {
		candidates := []string{"classes.txt", "data.names", "obj.names", "names.txt"}
		for _, name := range candidates {
			p := filepath.Join(root, name)
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				classesPath = p
				break
			}
		}
	}

	if classesPath == "" {
		return names
	}

	f, err := os.Open(classesPath)
	if err != nil {
		return names
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	idx := 0
	for scanner.Scan() && idx < count {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		names[idx] = line
		idx++
	}

	return names
}
