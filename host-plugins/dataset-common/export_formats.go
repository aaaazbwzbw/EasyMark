// Export format implementations for YOLO, COCO, and Pascal VOC
package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// getMapKeys 获取 map 的所有键（用于调试）
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// ==================== YOLO Export ====================

func ExportYOLO(req ExportRequest, resp *ExportResponse) {
	// Build category index map (name -> index)
	catIndex := make(map[int]int) // categoryID -> YOLO class index
	catNames := []string{}
	for i, cat := range req.Categories {
		catIndex[cat.ID] = i
		catNames = append(catNames, cat.Name)
	}

	// Create directories
	dirs := []string{
		"train/images", "train/labels",
		"val/images", "val/labels",
		"test/images", "test/labels",
	}
	resp.Structure.Directories = dirs

	// Group annotations by image
	imgAnnotations := make(map[string][]ExportAnnotation)
	for _, ann := range req.Annotations {
		imgAnnotations[ann.ImageKey] = append(imgAnnotations[ann.ImageKey], ann)
	}

	// ★ 预先检测是否有关键点，确定 kptCount（所有 bbox 必须输出相同数量的关键点）
	hasKeypoints := false
	kptCount := 0
	fmt.Fprintf(os.Stderr, "[DEBUG] Checking keypoints in %d annotations\n", len(req.Annotations))
	for i, ann := range req.Annotations {
		if ann.Type == "bbox" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Ann %d: Type=bbox, Data keys=%v\n", i, getMapKeys(ann.Data))
			if keypoints, ok := ann.Data["keypoints"].([]interface{}); ok && len(keypoints) > 0 {
				hasKeypoints = true
				fmt.Fprintf(os.Stderr, "[DEBUG] Ann %d: Found %d keypoints\n", i, len(keypoints))
				if len(keypoints) > kptCount {
					kptCount = len(keypoints)
				}
			} else {
				fmt.Fprintf(os.Stderr, "[DEBUG] Ann %d: No keypoints found, raw value type=%T\n", i, ann.Data["keypoints"])
			}
		}
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] Result: hasKeypoints=%v, kptCount=%d\n", hasKeypoints, kptCount)
	// 也从类别 mate 中获取
	if hasKeypoints && kptCount == 0 {
		for _, cat := range req.Categories {
			if cat.Mate != "" {
				cnt := parseKeypointCountFromMate(cat.Mate)
				if cnt > kptCount {
					kptCount = cnt
				}
			}
		}
	}

	// Process each image
	trainCount, valCount, testCount := 0, 0, 0
	for _, img := range req.Images {
		split := img.Split
		if split == "" {
			split = "train"
		}

		// Count splits
		switch split {
		case "train":
			trainCount++
		case "val":
			valCount++
		case "test":
			testCount++
		}

		// Generate label file content
		// 新格式：关键点作为 bbox 的扩展字段 (data.keypoints)
		var lines []string
		imgAnns := imgAnnotations[img.Key]

		for _, ann := range imgAnns {
			classIdx, ok := catIndex[ann.CategoryID]
			if !ok {
				continue
			}
			data := ann.Data

			switch ann.Type {
			case "bbox":
				x, _ := toFloat64(data["x"])
				y, _ := toFloat64(data["y"])
				w, _ := toFloat64(data["width"])
				h, _ := toFloat64(data["height"])
				centerX := x + w/2
				centerY := y + h/2

				// ★ 如果数据集有关键点，所有 bbox 都必须输出相同数量的关键点
				if hasKeypoints && kptCount > 0 {
					// YOLO Pose format: class_id center_x center_y width height kpt1_x kpt1_y kpt1_v ...
					var keypointCoords []string
					keypoints, _ := data["keypoints"].([]interface{})
					for i := 0; i < kptCount; i++ {
						if i < len(keypoints) {
							if point, ok := keypoints[i].([]interface{}); ok && len(point) >= 3 {
								px, _ := toFloat64(point[0])
								py, _ := toFloat64(point[1])
								v, _ := toFloat64(point[2])
								keypointCoords = append(keypointCoords, fmt.Sprintf("%.6f %.6f %d", px, py, int(v)))
								continue
							}
						}
						// 缺失的关键点用 0 0 0 填充
						keypointCoords = append(keypointCoords, "0 0 0")
					}
					line := fmt.Sprintf("%d %.6f %.6f %.6f %.6f %s", classIdx, centerX, centerY, w, h, strings.Join(keypointCoords, " "))
					lines = append(lines, line)
				} else {
					// YOLO Detection format: class_id center_x center_y width height
					lines = append(lines, fmt.Sprintf("%d %.6f %.6f %.6f %.6f", classIdx, centerX, centerY, w, h))
				}

			case "polygon":
				// YOLO Segmentation format: class_id x1 y1 x2 y2 x3 y3 ...
				points, ok := data["points"].([]interface{})
				if !ok || len(points) < 3 {
					continue
				}
				var coords []string
				coords = append(coords, fmt.Sprintf("%d", classIdx))
				for _, pt := range points {
					point, ok := pt.([]interface{})
					if !ok || len(point) < 2 {
						continue
					}
					px, _ := toFloat64(point[0])
					py, _ := toFloat64(point[1])
					coords = append(coords, fmt.Sprintf("%.6f", px), fmt.Sprintf("%.6f", py))
				}
				if len(coords) >= 7 { // class_id + at least 3 points
					lines = append(lines, strings.Join(coords, " "))
				}
			}
		}

		// Get image filename
		imgName := filepath.Base(img.RelativePath)
		labelName := strings.TrimSuffix(imgName, filepath.Ext(imgName)) + ".txt"

		// Add label file
		if len(lines) > 0 {
			resp.Structure.Files = append(resp.Structure.Files, FileOutput{
				Path:    fmt.Sprintf("%s/labels/%s", split, labelName),
				Content: strings.Join(lines, "\n") + "\n",
			})
		}

		// Add image copy task
		resp.Structure.CopyImages = append(resp.Structure.CopyImages, CopyTask{
			From: img.AbsolutePath,
			To:   fmt.Sprintf("%s/images/%s", split, imgName),
		})
	}

	// Generate data.yaml (使用上面已经检测的 hasKeypoints 和 kptCount)
	var dataYaml string
	if hasKeypoints && kptCount > 0 {
		// YOLO Pose format with kpt_shape
		dataYaml = fmt.Sprintf(`train: ./train/images
val: ./val/images
test: ./test/images

nc: %d
names: [%s]
kpt_shape: [%d, 3]
`, len(catNames), formatStringList(catNames), kptCount)
	} else {
		dataYaml = fmt.Sprintf(`train: ./train/images
val: ./val/images
test: ./test/images

nc: %d
names: [%s]
`, len(catNames), formatStringList(catNames))
	}

	resp.Structure.Files = append(resp.Structure.Files, FileOutput{
		Path:    "data.yaml",
		Content: dataYaml,
	})

	// Update stats
	resp.Stats = ExportStats{
		ImageCount:      len(req.Images),
		AnnotationCount: len(req.Annotations),
		TrainCount:      trainCount,
		ValCount:        valCount,
		TestCount:       testCount,
	}
}

// parseKeypointCountFromMate 从类别 mate JSON 中解析关键点数量
func parseKeypointCountFromMate(mate string) int {
	var mateData struct {
		Keypoints []struct {
			Name string `json:"name"`
		} `json:"keypoints"`
	}
	if err := json.Unmarshal([]byte(mate), &mateData); err != nil {
		return 0
	}
	return len(mateData.Keypoints)
}

// ==================== COCO Export ====================
// 使用已有的 COCODataset, COCOImage, COCOAnnotation, COCOCategory 类型（定义在 format_coco.go）

func ExportCOCO(req ExportRequest, resp *ExportResponse) {
	// Create directories
	resp.Structure.Directories = []string{
		"train", "val", "test",
		"annotations",
	}

	// Build image key to ID map
	imgKeyToID := make(map[string]int)
	for i, img := range req.Images {
		imgKeyToID[img.Key] = i + 1
	}

	// Group by split
	splitData := map[string]*COCODataset{
		"train": {Images: []COCOImage{}, Annotations: []COCOAnnotation{}, Categories: []COCOCategory{}},
		"val":   {Images: []COCOImage{}, Annotations: []COCOAnnotation{}, Categories: []COCOCategory{}},
		"test":  {Images: []COCOImage{}, Annotations: []COCOAnnotation{}, Categories: []COCOCategory{}},
	}

	// Add categories to all splits
	for _, cat := range req.Categories {
		cocoCat := COCOCategory{ID: cat.ID, Name: cat.Name}
		for _, data := range splitData {
			data.Categories = append(data.Categories, cocoCat)
		}
	}

	// Process images and annotations
	annID := 1
	trainCount, valCount, testCount := 0, 0, 0

	for _, img := range req.Images {
		split := img.Split
		if split == "" {
			split = "train"
		}

		switch split {
		case "train":
			trainCount++
		case "val":
			valCount++
		case "test":
			testCount++
		}

		imgID := imgKeyToID[img.Key]
		imgName := filepath.Base(img.RelativePath)

		// Add image
		splitData[split].Images = append(splitData[split].Images, COCOImage{
			ID:       imgID,
			FileName: imgName,
			Width:    img.Width,
			Height:   img.Height,
		})

		// Add image copy task
		resp.Structure.CopyImages = append(resp.Structure.CopyImages, CopyTask{
			From: img.AbsolutePath,
			To:   fmt.Sprintf("%s/%s", split, imgName),
		})
	}

	// Build image dimensions map
	imgDimensions := make(map[string][2]int)
	for _, img := range req.Images {
		imgDimensions[img.Key] = [2]int{img.Width, img.Height}
	}

	// Build image split map
	imgSplitMap := make(map[string]string)
	for _, img := range req.Images {
		split := img.Split
		if split == "" {
			split = "train"
		}
		imgSplitMap[img.Key] = split
	}

	// Process annotations
	for _, ann := range req.Annotations {
		imgID, ok := imgKeyToID[ann.ImageKey]
		if !ok {
			continue
		}

		imgSplit := imgSplitMap[ann.ImageKey]
		dims := imgDimensions[ann.ImageKey]
		imgWidth, imgHeight := dims[0], dims[1]

		data := ann.Data
		var cocoAnn COCOAnnotation

		switch ann.Type {
		case "bbox":
			x, _ := toFloat64(data["x"])
			y, _ := toFloat64(data["y"])
			w, _ := toFloat64(data["width"])
			h, _ := toFloat64(data["height"])

			absX := x * float64(imgWidth)
			absY := y * float64(imgHeight)
			absW := w * float64(imgWidth)
			absH := h * float64(imgHeight)

			cocoAnn = COCOAnnotation{
				ID:         annID,
				ImageID:    imgID,
				CategoryID: ann.CategoryID,
				BBox:       []float64{absX, absY, absW, absH},
				Area:       absW * absH,
				IsCrowd:    0,
			}

		case "polygon":
			points, ok := data["points"].([]interface{})
			if !ok || len(points) < 3 {
				continue
			}

			// Convert to COCO segmentation format [[x1,y1,x2,y2,...]]
			var segPoints []float64
			var minX, minY, maxX, maxY float64 = 1, 1, 0, 0
			for _, pt := range points {
				point, ok := pt.([]interface{})
				if !ok || len(point) < 2 {
					continue
				}
				px, _ := toFloat64(point[0])
				py, _ := toFloat64(point[1])
				absX := px * float64(imgWidth)
				absY := py * float64(imgHeight)
				segPoints = append(segPoints, absX, absY)
				if px < minX {
					minX = px
				}
				if px > maxX {
					maxX = px
				}
				if py < minY {
					minY = py
				}
				if py > maxY {
					maxY = py
				}
			}

			// Calculate bounding box
			absMinX := minX * float64(imgWidth)
			absMinY := minY * float64(imgHeight)
			absW := (maxX - minX) * float64(imgWidth)
			absH := (maxY - minY) * float64(imgHeight)

			cocoAnn = COCOAnnotation{
				ID:           annID,
				ImageID:      imgID,
				CategoryID:   ann.CategoryID,
				BBox:         []float64{absMinX, absMinY, absW, absH},
				Segmentation: [][]float64{segPoints},
				Area:         absW * absH,
				IsCrowd:      0,
			}

		case "keypoint":
			points, ok := data["points"].([]interface{})
			if !ok || len(points) == 0 {
				continue
			}

			// Convert to COCO keypoints format [x1,y1,v1,x2,y2,v2,...]
			var kpPoints []float64
			var minX, minY, maxX, maxY float64 = 1, 1, 0, 0
			numKeypoints := 0
			for _, pt := range points {
				point, ok := pt.([]interface{})
				if !ok || len(point) < 3 {
					continue
				}
				px, _ := toFloat64(point[0])
				py, _ := toFloat64(point[1])
				v, _ := toFloat64(point[2])
				absX := px * float64(imgWidth)
				absY := py * float64(imgHeight)
				kpPoints = append(kpPoints, absX, absY, v)
				if v > 0 {
					numKeypoints++
					if px < minX {
						minX = px
					}
					if px > maxX {
						maxX = px
					}
					if py < minY {
						minY = py
					}
					if py > maxY {
						maxY = py
					}
				}
			}

			// Calculate bounding box
			absMinX := minX * float64(imgWidth)
			absMinY := minY * float64(imgHeight)
			absW := (maxX - minX) * float64(imgWidth)
			absH := (maxY - minY) * float64(imgHeight)
			if absW < 1 {
				absW = 10
			}
			if absH < 1 {
				absH = 10
			}

			cocoAnn = COCOAnnotation{
				ID:         annID,
				ImageID:    imgID,
				CategoryID: ann.CategoryID,
				BBox:       []float64{absMinX, absMinY, absW, absH},
				Keypoints:  kpPoints,
				Area:       absW * absH,
				IsCrowd:    0,
			}

		default:
			continue
		}

		splitData[imgSplit].Annotations = append(splitData[imgSplit].Annotations, cocoAnn)
		annID++
	}

	// Generate annotation files
	for split, data := range splitData {
		if len(data.Images) == 0 {
			continue
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		resp.Structure.Files = append(resp.Structure.Files, FileOutput{
			Path:    fmt.Sprintf("annotations/%s.json", split),
			Content: string(jsonData),
		})
	}

	resp.Stats = ExportStats{
		ImageCount:      len(req.Images),
		AnnotationCount: len(req.Annotations),
		TrainCount:      trainCount,
		ValCount:        valCount,
		TestCount:       testCount,
	}
}

// ==================== VOC Export ====================
// 使用已有的 VOCAnnotation, VOCSize, VOCObject, VOCBBox 类型（定义在 format_voc.go）

func ExportVOC(req ExportRequest, resp *ExportResponse) {
	// Create directories
	resp.Structure.Directories = []string{
		"train/JPEGImages", "train/Annotations",
		"val/JPEGImages", "val/Annotations",
		"test/JPEGImages", "test/Annotations",
	}

	// Build category ID to name map
	catNames := make(map[int]string)
	for _, cat := range req.Categories {
		catNames[cat.ID] = cat.Name
	}

	// Group annotations by image
	imgAnnotations := make(map[string][]ExportAnnotation)
	for _, ann := range req.Annotations {
		imgAnnotations[ann.ImageKey] = append(imgAnnotations[ann.ImageKey], ann)
	}

	trainCount, valCount, testCount := 0, 0, 0

	for _, img := range req.Images {
		split := img.Split
		if split == "" {
			split = "train"
		}

		switch split {
		case "train":
			trainCount++
		case "val":
			valCount++
		case "test":
			testCount++
		}

		imgName := filepath.Base(img.RelativePath)
		xmlName := strings.TrimSuffix(imgName, filepath.Ext(imgName)) + ".xml"

		// Build VOC annotation
		vocAnn := VOCAnnotation{
			Folder:   split,
			Filename: imgName,
			Size: VOCSize{
				Width:  img.Width,
				Height: img.Height,
				Depth:  3,
			},
			Objects: []VOCObject{},
		}

		for _, ann := range imgAnnotations[img.Key] {
			if ann.Type != "bbox" {
				continue
			}

			catName, ok := catNames[ann.CategoryID]
			if !ok {
				continue
			}

			data := ann.Data
			x, _ := toFloat64(data["x"])
			y, _ := toFloat64(data["y"])
			w, _ := toFloat64(data["width"])
			h, _ := toFloat64(data["height"])

			// Convert normalized to absolute
			xmin := x * float64(img.Width)
			ymin := y * float64(img.Height)
			xmax := (x + w) * float64(img.Width)
			ymax := (y + h) * float64(img.Height)

			vocAnn.Objects = append(vocAnn.Objects, VOCObject{
				Name:      catName,
				Pose:      "Unspecified",
				Truncated: 0,
				Difficult: 0,
				Bndbox: VOCBBox{
					Xmin: xmin,
					Ymin: ymin,
					Xmax: xmax,
					Ymax: ymax,
				},
			})
		}

		// Generate XML
		xmlData, _ := xml.MarshalIndent(vocAnn, "", "  ")
		resp.Structure.Files = append(resp.Structure.Files, FileOutput{
			Path:    fmt.Sprintf("%s/Annotations/%s", split, xmlName),
			Content: xml.Header + string(xmlData),
		})

		// Add image copy task
		resp.Structure.CopyImages = append(resp.Structure.CopyImages, CopyTask{
			From: img.AbsolutePath,
			To:   fmt.Sprintf("%s/JPEGImages/%s", split, imgName),
		})
	}

	resp.Stats = ExportStats{
		ImageCount:      len(req.Images),
		AnnotationCount: len(req.Annotations),
		TrainCount:      trainCount,
		ValCount:        valCount,
		TestCount:       testCount,
	}
}

// ==================== Helper Functions ====================

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

func formatStringList(items []string) string {
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("'%s'", item)
	}
	return strings.Join(quoted, ", ")
}
