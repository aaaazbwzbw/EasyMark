// Common Dataset Import Plugin for EasyMark
// Supports COCO, YOLO, and Pascal VOC formats
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <detect|import>\n", os.Args[0])
		os.Exit(1)
	}

	action := os.Args[1]
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
		os.Exit(1)
	}

	switch action {
	case "detect":
		handleDetect(input)
	case "import":
		handleImport(input)
	case "export":
		handleExport(input)
	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %s\n", action)
		os.Exit(1)
	}
}

// ==================== Detect ====================

func handleDetect(input []byte) {
	var req DetectRequest
	if err := json.Unmarshal(input, &req); err != nil {
		outputDetectResponse(DetectResponse{
			Supported: false,
			Reason:    "Failed to parse request",
			FormatID:  "dataset.common",
		})
		return
	}

	// Detect all formats and pick the best match
	scores := []FormatScore{
		DetectCOCO(req.RootPath),
		DetectYOLO(req.RootPath),
		DetectVOC(req.RootPath),
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	best := scores[0]
	if best.Score > 0 {
		outputDetectResponse(DetectResponse{
			Supported: true,
			Score:     best.Score,
			Reason:    fmt.Sprintf("[%s] %s", formatDisplayName(best.Format), best.Reason),
			FormatID:  "dataset.common:" + best.Format,
		})
	} else {
		outputDetectResponse(DetectResponse{
			Supported: false,
			Score:     0,
			Reason:    "No supported dataset format detected (COCO/YOLO/VOC)",
			FormatID:  "dataset.common",
		})
	}
}

func formatDisplayName(format string) string {
	switch format {
	case "coco":
		return "COCO"
	case "yolo":
		return "YOLO"
	case "voc":
		return "Pascal VOC"
	default:
		return format
	}
}

func outputDetectResponse(resp DetectResponse) {
	out, _ := json.Marshal(resp)
	fmt.Println(string(out))
}

// ==================== Import ====================

func handleImport(input []byte) {
	var req ImportRequest
	if err := json.Unmarshal(input, &req); err != nil {
		outputImportError("invalid_request", "Failed to parse request")
		return
	}

	resp := ImportResponse{
		Images:      []ImageRef{},
		Categories:  []CategoryDef{},
		Annotations: []AnnotationDef{},
		Stats:       Stats{},
		Errors:      []ErrorItem{},
	}

	// Determine format from formatId or params
	format := ""
	if v, ok := req.Params["format"].(string); ok && v != "" && v != "auto" {
		format = v
	} else {
		// Extract format from formatId (e.g., "dataset.common:coco")
		if len(req.FormatID) > 15 && req.FormatID[:15] == "dataset.common:" {
			format = req.FormatID[15:]
		}
	}

	// If still unknown, auto-detect
	if format == "" {
		scores := []FormatScore{
			DetectCOCO(req.RootPath),
			DetectYOLO(req.RootPath),
			DetectVOC(req.RootPath),
		}
		sort.Slice(scores, func(i, j int) bool {
			return scores[i].Score > scores[j].Score
		})
		if scores[0].Score > 0 {
			format = scores[0].Format
		}
	}

	// Import based on format
	switch format {
	case "coco":
		ImportCOCO(req, &resp)
	case "yolo":
		ImportYOLO(req, &resp)
	case "voc":
		ImportVOC(req, &resp)
	default:
		resp.Errors = append(resp.Errors, ErrorItem{
			Code:    "unknown_format",
			Message: "Could not determine dataset format. Please specify format parameter.",
		})
	}

	outputImportResponse(resp)
}

func outputImportResponse(resp ImportResponse) {
	out, _ := json.Marshal(resp)
	fmt.Println(string(out))
}

func outputImportError(code, message string) {
	resp := ImportResponse{
		Errors: []ErrorItem{{Code: code, Message: message}},
	}
	outputImportResponse(resp)
}

// ==================== Export ====================

func handleExport(input []byte) {
	var req ExportRequest
	if err := json.Unmarshal(input, &req); err != nil {
		outputExportError("invalid_request", "Failed to parse export request")
		return
	}
	
	// Debug: print categories
	fmt.Fprintf(os.Stderr, "[DEBUG] Categories count: %d\n", len(req.Categories))
	for i, cat := range req.Categories {
		fmt.Fprintf(os.Stderr, "[DEBUG] Category %d: ID=%d, Name=%s, Type=%s\n", i, cat.ID, cat.Name, cat.Type)
	}

	resp := ExportResponse{
		Success: true,
		Structure: ExportStructure{
			Directories: []string{},
			Files:       []FileOutput{},
			CopyImages:  []CopyTask{},
		},
		Stats:  ExportStats{},
		Errors: []ErrorItem{},
	}

	// Export based on format
	switch req.Format {
	case "yolo":
		ExportYOLO(req, &resp)
	case "coco":
		ExportCOCO(req, &resp)
	case "voc":
		ExportVOC(req, &resp)
	default:
		resp.Success = false
		resp.Errors = append(resp.Errors, ErrorItem{
			Code:    "unsupported_format",
			Message: fmt.Sprintf("Unsupported export format: %s", req.Format),
		})
	}

	outputExportResponse(resp)
}

func outputExportResponse(resp ExportResponse) {
	out, _ := json.Marshal(resp)
	fmt.Println(string(out))
}

func outputExportError(code, message string) {
	resp := ExportResponse{
		Success: false,
		Errors:  []ErrorItem{{Code: code, Message: message}},
	}
	outputExportResponse(resp)
}
