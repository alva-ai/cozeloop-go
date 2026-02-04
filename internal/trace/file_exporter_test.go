// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coze-dev/cozeloop-go/entity"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewFileExporter(t *testing.T) {
	Convey("NewFileExporter", t, func() {
		Convey("with empty path should use default", func() {
			exporter := NewFileExporter("")
			So(exporter.filePath, ShouldEqual, DefaultLocalExportPath)
		})

		Convey("with custom path should use provided path", func() {
			customPath := "/tmp/custom_traces.md"
			exporter := NewFileExporter(customPath)
			So(exporter.filePath, ShouldEqual, customPath)
		})
	})
}

func TestFileExporter_ExportSpans(t *testing.T) {
	Convey("FileExporter.ExportSpans", t, func() {
		ctx := context.Background()

		Convey("with empty spans should return nil", func() {
			exporter := NewFileExporter("")
			err := exporter.ExportSpans(ctx, []*entity.UploadSpan{})
			So(err, ShouldBeNil)
		})

		Convey("with valid spans should write to file", func() {
			// Create temp file
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test_traces.md")
			exporter := NewFileExporter(filePath)

			// Create test spans
			now := time.Now()
			spans := []*entity.UploadSpan{
				{
					TraceID:         "trace123456789012345678901234",
					SpanID:          "span1234567890ab",
					ParentID:        "0",
					SpanName:        "test_span",
					SpanType:        "test_type",
					StartedATMicros: now.UnixMicro(),
					DurationMicros:  1000000, // 1 second
					StatusCode:      0,
					WorkspaceID:     "ws123",
					ServiceName:     "test_service",
					Input:           "test input",
					Output:          "test output",
					TagsString:      map[string]string{"key1": "value1"},
					TagsLong:        map[string]int64{"count": 42},
					TagsDouble:      map[string]float64{"score": 0.95},
					TagsBool:        map[string]bool{"enabled": true},
				},
			}

			err := exporter.ExportSpans(ctx, spans)
			So(err, ShouldBeNil)

			// Read the file and verify content
			content, err := os.ReadFile(filePath)
			So(err, ShouldBeNil)
			contentStr := string(content)

			So(contentStr, ShouldContainSubstring, "# Trace: trace123456789012345678901234")
			So(contentStr, ShouldContainSubstring, "## Span: test_span")
			So(contentStr, ShouldContainSubstring, "**Type:** test_type")
			So(contentStr, ShouldContainSubstring, "**Span ID:** span1234567890ab")
			So(contentStr, ShouldContainSubstring, "test input")
			So(contentStr, ShouldContainSubstring, "test output")
			So(contentStr, ShouldContainSubstring, "key1")
			So(contentStr, ShouldContainSubstring, "value1")
		})

		Convey("should append to existing file", func() {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test_traces.md")
			exporter := NewFileExporter(filePath)

			// Write first batch
			spans1 := []*entity.UploadSpan{
				{
					TraceID:         "trace1",
					SpanID:          "span1",
					SpanName:        "first_span",
					SpanType:        "test",
					StartedATMicros: time.Now().UnixMicro(),
					DurationMicros:  1000,
				},
			}
			err := exporter.ExportSpans(ctx, spans1)
			So(err, ShouldBeNil)

			// Write second batch
			spans2 := []*entity.UploadSpan{
				{
					TraceID:         "trace2",
					SpanID:          "span2",
					SpanName:        "second_span",
					SpanType:        "test",
					StartedATMicros: time.Now().UnixMicro(),
					DurationMicros:  2000,
				},
			}
			err = exporter.ExportSpans(ctx, spans2)
			So(err, ShouldBeNil)

			// Read the file and verify both spans are present
			content, err := os.ReadFile(filePath)
			So(err, ShouldBeNil)
			contentStr := string(content)

			So(contentStr, ShouldContainSubstring, "first_span")
			So(contentStr, ShouldContainSubstring, "second_span")
		})

		Convey("should create directory if not exists", func() {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "subdir", "nested", "traces.md")
			exporter := NewFileExporter(filePath)

			spans := []*entity.UploadSpan{
				{
					TraceID:         "trace1",
					SpanID:          "span1",
					SpanName:        "test",
					SpanType:        "test",
					StartedATMicros: time.Now().UnixMicro(),
					DurationMicros:  1000,
				},
			}

			err := exporter.ExportSpans(ctx, spans)
			So(err, ShouldBeNil)

			// Verify file exists
			_, err = os.Stat(filePath)
			So(err, ShouldBeNil)
		})
	})
}

func TestFileExporter_ExportFiles(t *testing.T) {
	Convey("FileExporter.ExportFiles", t, func() {
		ctx := context.Background()
		exporter := NewFileExporter("")

		Convey("should return nil (no-op)", func() {
			files := []*entity.UploadFile{
				{TosKey: "key1", Data: "data1"},
			}
			err := exporter.ExportFiles(ctx, files)
			So(err, ShouldBeNil)
		})
	})
}

func TestFormatDuration(t *testing.T) {
	Convey("formatDuration", t, func() {
		Convey("microseconds", func() {
			d := 500 * time.Microsecond
			So(formatDuration(d), ShouldEqual, "500μs")
		})

		Convey("milliseconds", func() {
			d := 500 * time.Millisecond
			So(formatDuration(d), ShouldEqual, "500.00ms")
		})

		Convey("seconds", func() {
			d := 2500 * time.Millisecond
			So(formatDuration(d), ShouldEqual, "2.500s")
		})

		Convey("minutes", func() {
			d := 90 * time.Second
			So(formatDuration(d), ShouldEqual, "1.50m")
		})
	})
}

func TestTruncateString(t *testing.T) {
	Convey("truncateString", t, func() {
		Convey("short string should not be truncated", func() {
			s := "short"
			result := truncateString(s, 10)
			So(result, ShouldEqual, "short")
		})

		Convey("long string should be truncated", func() {
			s := "this is a very long string"
			result := truncateString(s, 10)
			So(result, ShouldEqual, "this is a ... (truncated)")
		})
	})
}

func TestEscapeMarkdown(t *testing.T) {
	Convey("escapeMarkdown", t, func() {
		Convey("should escape pipe character", func() {
			s := "value|with|pipes"
			result := escapeMarkdown(s)
			So(result, ShouldEqual, "value\\|with\\|pipes")
		})

		Convey("should replace newlines with spaces", func() {
			s := "line1\nline2\r\nline3"
			result := escapeMarkdown(s)
			So(strings.Contains(result, "\n"), ShouldBeFalse)
			So(strings.Contains(result, "\r"), ShouldBeFalse)
		})
	})
}
