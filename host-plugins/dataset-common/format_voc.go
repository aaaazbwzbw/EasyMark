package main

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ==================== VOC Format Structures ====================

type VOCAnnotation struct {
	XMLName  xml.Name    `xml:"annotation"`
	Folder   string      `xml:"folder"`
	Filename string      `xml:"filename"`
	Size     VOCSize     `xml:"size"`
	Objects  []VOCObject `xml:"object"`
}

type VOCSize struct {
	Width  int `xml:"width"`
	Height int `xml:"height"`
	Depth  int `xml:"depth"`
}

type VOCObject struct {
	Name      string  `xml:"name"`
	Pose      string  `xml:"pose"`
	Truncated int     `xml:"truncated"`
	Difficult int     `xml:"difficult"`
	Bndbox    VOCBBox `xml:"bndbox"`
}

type VOCBBox struct {
	Xmin float64 `xml:"xmin"`
	Ymin float64 `xml:"ymin"`
	Xmax float64 `xml:"xmax"`
	Ymax float64 `xml:"ymax"`
}

// ==================== VOC Detection ====================

func DetectVOC(rootPath string) FormatScore {
	result := FormatScore{Format: "voc", Score: 0}

	xmlFiles := findVOCAnnotationFiles(rootPath)
	if len(xmlFiles) == 0 {
		result.Reason = "No VOC annotation files found"
		return result
	}

	for _, f := range xmlFiles {
		if isValidVOCFile(f) {
			result.Score = 0.88
			result.Reason = fmt.Sprintf("Found VOC annotation: %s", filepath.Base(f))
			return result
		}
	}

	result.Reason = "XML files found but not valid VOC format"
	return result
}

func findVOCAnnotationFiles(rootPath string) []string {
	var files []string
	visited := make(map[string]bool)

	addXMLFiles := func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".xml") {
				fullPath := filepath.Join(dir, e.Name())
				if !visited[fullPath] {
					visited[fullPath] = true
					files = append(files, fullPath)
				}
			}
		}
	}

	addXMLFiles(rootPath)
	// 支持多种目录结构
	for _, d := range []string{
		"Annotations", "annotations", "labels", "xml",
		"train/Annotations", "val/Annotations", "test/Annotations", // train/val/test 子目录结构
		"train/annotations", "val/annotations",
	} {
		addXMLFiles(filepath.Join(rootPath, d))
	}

	// 检查根目录下的子目录
	entries, _ := os.ReadDir(rootPath)
	for _, e := range entries {
		if e.IsDir() {
			subDir := filepath.Join(rootPath, e.Name())
			addXMLFiles(subDir)
			// 也检查子目录下的 Annotations 目录
			addXMLFiles(filepath.Join(subDir, "Annotations"))
			addXMLFiles(filepath.Join(subDir, "annotations"))
		}
	}

	return files
}

func isValidVOCFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var ann VOCAnnotation
	if err := xml.Unmarshal(data, &ann); err != nil {
		return false
	}

	// Valid VOC file should have filename and at least recognize the structure
	return ann.Filename != "" || len(ann.Objects) > 0
}

// ==================== VOC Import ====================

func ImportVOC(req ImportRequest, resp *ImportResponse) {
	root := req.RootPath

	// Find annotations directory
	annotationsDirAbs := ""
	if v, ok := req.Params["annotationFile"].(string); ok && v != "" {
		if filepath.IsAbs(v) {
			annotationsDirAbs = v
		} else {
			annotationsDirAbs = filepath.Join(root, v)
		}
	} else {
		annotationsDirAbs = guessVOCAnnotationsDir(root)
	}

	if annotationsDirAbs == "" {
		resp.Errors = append(resp.Errors, ErrorItem{
			Code:    "missing_annotations_dir",
			Message: "No VOC annotations directory found",
		})
		return
	}

	// Collect XML files
	var xmlFiles []string
	filepath.WalkDir(annotationsDirAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(d.Name())) == ".xml" {
			xmlFiles = append(xmlFiles, path)
		}
		return nil
	})

	if len(xmlFiles) == 0 {
		resp.Errors = append(resp.Errors, ErrorItem{
			Code:    "no_annotation_files",
			Message: "No VOC annotation XML files found",
			Details: map[string]interface{}{"annotationsDir": annotationsDirAbs},
		})
		return
	}

	// Find images directory
	imagesDirAbs := ""
	if v, ok := req.Params["imagesDir"].(string); ok && v != "" {
		if filepath.IsAbs(v) {
			imagesDirAbs = v
		} else {
			imagesDirAbs = filepath.Join(root, v)
		}
	} else {
		imagesDirAbs = guessVOCImagesDir(root)
	}

	// First pass: collect all class names
	classNames := make(map[string]bool)
	for _, xmlFile := range xmlFiles {
		ann, err := parseVOCFile(xmlFile)
		if err != nil {
			continue
		}
		for _, obj := range ann.Objects {
			classNames[obj.Name] = true
		}
	}

	// Build categories (VOC format does not support keypoints)
	classKeyMap := make(map[string]string)
	idx := 0
	for name := range classNames {
		key := fmt.Sprintf("class_%d", idx)
		classKeyMap[name] = key
		resp.Categories = append(resp.Categories, CategoryDef{
			Key:       key,
			Name:      name,
			Type:      "bbox",
			Color:     GetColor(idx),
			SortOrder: idx + 1,
			Meta:      map[string]interface{}{},
		})
		idx++
	}

	// Second pass: build images and annotations
	imageKeyMap := make(map[string]bool)
	for _, xmlFile := range xmlFiles {
		ann, err := parseVOCFile(xmlFile)
		if err != nil {
			resp.Stats.SkippedImages++
			continue
		}

		// Determine image path
		imageName := ann.Filename
		if imageName == "" {
			base := strings.TrimSuffix(filepath.Base(xmlFile), ".xml")
			imageName = base + ".jpg"
		}

		var imageRelPath string
		if imagesDirAbs != "" {
			// Try to find the actual image file
			imageRelPath = findVOCImage(root, imagesDirAbs, imageName)
		}
		if imageRelPath == "" {
			// Use folder info from annotation if available
			if ann.Folder != "" {
				imageRelPath = filepath.Join(ann.Folder, imageName)
			} else {
				imageRelPath = imageName
			}
		}

		imageKey := imageRelPath
		if !imageKeyMap[imageKey] {
			imageKeyMap[imageKey] = true
			resp.Images = append(resp.Images, ImageRef{
				Key:          imageKey,
				RelativePath: imageRelPath,
				Meta: map[string]interface{}{
					"width":  ann.Size.Width,
					"height": ann.Size.Height,
				},
			})
		}

		// Process objects
		imgWidth := float64(ann.Size.Width)
		imgHeight := float64(ann.Size.Height)
		if imgWidth == 0 {
			imgWidth = 1
		}
		if imgHeight == 0 {
			imgHeight = 1
		}

		for _, obj := range ann.Objects {
			catKey, ok := classKeyMap[obj.Name]
			if !ok {
				resp.Stats.SkippedAnnotations++
				continue
			}

			// Normalize coordinates
			x := obj.Bndbox.Xmin / imgWidth
			y := obj.Bndbox.Ymin / imgHeight
			w := (obj.Bndbox.Xmax - obj.Bndbox.Xmin) / imgWidth
			h := (obj.Bndbox.Ymax - obj.Bndbox.Ymin) / imgHeight

			resp.Annotations = append(resp.Annotations, AnnotationDef{
				ImageKey:    imageKey,
				CategoryKey: catKey,
				Type:        "bbox",
				Data: map[string]interface{}{
					"x":      x,
					"y":      y,
					"width":  w,
					"height": h,
				},
			})
		}
	}

	resp.Stats.ImageCount = len(resp.Images)
	resp.Stats.AnnotationCount = len(resp.Annotations)
}

func parseVOCFile(path string) (*VOCAnnotation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var ann VOCAnnotation
	if err := xml.Unmarshal(data, &ann); err != nil {
		return nil, err
	}

	return &ann, nil
}

func guessVOCAnnotationsDir(root string) string {
	candidates := []string{"Annotations", "annotations", "labels", "xml"}
	for _, rel := range candidates {
		p := filepath.Join(root, rel)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	// Check if root itself contains XML files
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".xml") {
			return root
		}
	}
	return ""
}

func guessVOCImagesDir(root string) string {
	candidates := []string{"JPEGImages", "images", "imgs", "img", "photos"}
	for _, rel := range candidates {
		p := filepath.Join(root, rel)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	return ""
}

func findVOCImage(root, imagesDir, imageName string) string {
	// Try exact match first
	fullPath := filepath.Join(imagesDir, imageName)
	if _, err := os.Stat(fullPath); err == nil {
		rel, _ := filepath.Rel(root, fullPath)
		return rel
	}

	// Try different extensions
	baseName := strings.TrimSuffix(imageName, filepath.Ext(imageName))
	exts := []string{".jpg", ".jpeg", ".png", ".bmp", ".webp"}
	for _, ext := range exts {
		fullPath := filepath.Join(imagesDir, baseName+ext)
		if _, err := os.Stat(fullPath); err == nil {
			rel, _ := filepath.Rel(root, fullPath)
			return rel
		}
	}

	return ""
}
