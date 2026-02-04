// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/logger"
)

const (
	DefaultLocalExportPath = "./cozeloop_traces.md"
)

var _ Exporter = (*FileExporter)(nil)

// FileExporter exports spans to a local markdown file
type FileExporter struct {
	filePath string
	mu       sync.Mutex
}

// NewFileExporter creates a new FileExporter with the given file path
func NewFileExporter(filePath string) *FileExporter {
	if filePath == "" {
		filePath = DefaultLocalExportPath
	}
	return &FileExporter{
		filePath: filePath,
	}
}

// ExportSpans writes spans to the markdown file
func (e *FileExporter) ExportSpans(ctx context.Context, spans []*entity.UploadSpan) error {
	if len(spans) == 0 {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(e.filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.CtxErrorf(ctx, "failed to create directory for trace file: %v", err)
			return err
		}
	}

	// Open file in append mode
	f, err := os.OpenFile(e.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.CtxErrorf(ctx, "failed to open trace file: %v", err)
		return err
	}
	defer f.Close()

	// Write spans to file
	for _, span := range spans {
		if span == nil {
			continue
		}
		md := e.spanToMarkdown(span)
		if _, err := f.WriteString(md); err != nil {
			logger.CtxErrorf(ctx, "failed to write span to file: %v", err)
			return err
		}
	}

	logger.CtxDebugf(ctx, "exported %d spans to file: %s", len(spans), e.filePath)
	return nil
}

// ExportFiles is a no-op for file exporter as we don't need to handle file uploads locally
func (e *FileExporter) ExportFiles(ctx context.Context, files []*entity.UploadFile) error {
	// File uploads are not written to the local markdown file
	// They would be too large and are typically binary data
	return nil
}

// spanToMarkdown converts a span to markdown format
func (e *FileExporter) spanToMarkdown(span *entity.UploadSpan) string {
	var sb strings.Builder

	// Header with trace info
	sb.WriteString(fmt.Sprintf("# Trace: %s\n\n", span.TraceID))

	// Span section
	sb.WriteString(fmt.Sprintf("## Span: %s\n\n", span.SpanName))

	// Basic info
	sb.WriteString(fmt.Sprintf("- **Type:** %s\n", span.SpanType))
	sb.WriteString(fmt.Sprintf("- **Span ID:** %s\n", span.SpanID))
	sb.WriteString(fmt.Sprintf("- **Parent ID:** %s\n", span.ParentID))

	// Time info
	startTime := time.UnixMicro(span.StartedATMicros)
	sb.WriteString(fmt.Sprintf("- **Start Time:** %s\n", startTime.Format("2006-01-02 15:04:05.000")))

	// Duration in human readable format
	duration := time.Duration(span.DurationMicros) * time.Microsecond
	sb.WriteString(fmt.Sprintf("- **Duration:** %s\n", formatDuration(duration)))

	// Status
	statusText := "OK"
	if span.StatusCode != 0 {
		statusText = "ERROR"
	}
	sb.WriteString(fmt.Sprintf("- **Status:** %s (%d)\n", statusText, span.StatusCode))

	// Service and workspace info
	if span.ServiceName != "" {
		sb.WriteString(fmt.Sprintf("- **Service:** %s\n", span.ServiceName))
	}
	if span.WorkspaceID != "" {
		sb.WriteString(fmt.Sprintf("- **Workspace ID:** %s\n", span.WorkspaceID))
	}
	if span.LogID != "" {
		sb.WriteString(fmt.Sprintf("- **Log ID:** %s\n", span.LogID))
	}

	sb.WriteString("\n")

	// Input section
	if span.Input != "" {
		sb.WriteString("### Input\n\n")
		sb.WriteString("```\n")
		sb.WriteString(truncateString(span.Input, 2000))
		sb.WriteString("\n```\n\n")
	}

	// Output section
	if span.Output != "" {
		sb.WriteString("### Output\n\n")
		sb.WriteString("```\n")
		sb.WriteString(truncateString(span.Output, 2000))
		sb.WriteString("\n```\n\n")
	}

	// Tags section
	hasTags := len(span.TagsString) > 0 || len(span.TagsLong) > 0 ||
		len(span.TagsDouble) > 0 || len(span.TagsBool) > 0

	if hasTags {
		sb.WriteString("### Tags\n\n")
		sb.WriteString("| Key | Value |\n")
		sb.WriteString("|-----|-------|\n")

		// String tags
		writeTagsToTable(&sb, span.TagsString)

		// Long tags
		for k, v := range span.TagsLong {
			sb.WriteString(fmt.Sprintf("| %s | %d |\n", escapeMarkdown(k), v))
		}

		// Double tags
		for k, v := range span.TagsDouble {
			sb.WriteString(fmt.Sprintf("| %s | %.4f |\n", escapeMarkdown(k), v))
		}

		// Bool tags
		for k, v := range span.TagsBool {
			sb.WriteString(fmt.Sprintf("| %s | %t |\n", escapeMarkdown(k), v))
		}

		sb.WriteString("\n")
	}

	// System tags section
	hasSystemTags := len(span.SystemTagsString) > 0 || len(span.SystemTagsLong) > 0 ||
		len(span.SystemTagsDouble) > 0

	if hasSystemTags {
		sb.WriteString("### System Tags\n\n")
		sb.WriteString("| Key | Value |\n")
		sb.WriteString("|-----|-------|\n")

		writeTagsToTable(&sb, span.SystemTagsString)

		for k, v := range span.SystemTagsLong {
			sb.WriteString(fmt.Sprintf("| %s | %d |\n", escapeMarkdown(k), v))
		}

		for k, v := range span.SystemTagsDouble {
			sb.WriteString(fmt.Sprintf("| %s | %.4f |\n", escapeMarkdown(k), v))
		}

		sb.WriteString("\n")
	}

	// Separator
	sb.WriteString("---\n\n")

	return sb.String()
}

// writeTagsToTable writes string tags to markdown table in sorted order
func writeTagsToTable(sb *strings.Builder, tags map[string]string) {
	if len(tags) == 0 {
		return
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := tags[k]
		// Truncate long values for readability
		displayValue := truncateString(v, 100)
		sb.WriteString(fmt.Sprintf("| %s | %s |\n", escapeMarkdown(k), escapeMarkdown(displayValue)))
	}
}

// formatDuration formats a duration in human readable format
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dμs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.3fs", d.Seconds())
	}
	return fmt.Sprintf("%.2fm", d.Minutes())
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}

// escapeMarkdown escapes special markdown characters in table cells
func escapeMarkdown(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
