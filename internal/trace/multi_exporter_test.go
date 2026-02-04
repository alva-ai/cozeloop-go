// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"errors"
	"testing"

	"github.com/coze-dev/cozeloop-go/entity"
	. "github.com/smartystreets/goconvey/convey"
)

// mockExporter is a mock implementation of Exporter for testing
type mockExporter struct {
	exportSpansCalled  bool
	exportFilesCalled  bool
	exportSpansErr     error
	exportFilesErr     error
	exportedSpans      []*entity.UploadSpan
	exportedFiles      []*entity.UploadFile
}

func (m *mockExporter) ExportSpans(ctx context.Context, spans []*entity.UploadSpan) error {
	m.exportSpansCalled = true
	m.exportedSpans = spans
	return m.exportSpansErr
}

func (m *mockExporter) ExportFiles(ctx context.Context, files []*entity.UploadFile) error {
	m.exportFilesCalled = true
	m.exportedFiles = files
	return m.exportFilesErr
}

func TestNewMultiExporter(t *testing.T) {
	Convey("NewMultiExporter", t, func() {
		Convey("should filter out nil exporters", func() {
			exp1 := &mockExporter{}
			exp2 := &mockExporter{}
			multi := NewMultiExporter(exp1, nil, exp2, nil)
			So(multi.ExporterCount(), ShouldEqual, 2)
		})

		Convey("should handle all nil exporters", func() {
			multi := NewMultiExporter(nil, nil)
			So(multi.ExporterCount(), ShouldEqual, 0)
		})

		Convey("should handle empty input", func() {
			multi := NewMultiExporter()
			So(multi.ExporterCount(), ShouldEqual, 0)
		})
	})
}

func TestMultiExporter_ExportSpans(t *testing.T) {
	Convey("MultiExporter.ExportSpans", t, func() {
		ctx := context.Background()

		Convey("with no exporters should return nil", func() {
			multi := NewMultiExporter()
			err := multi.ExportSpans(ctx, []*entity.UploadSpan{{}})
			So(err, ShouldBeNil)
		})

		Convey("should call all exporters", func() {
			exp1 := &mockExporter{}
			exp2 := &mockExporter{}
			multi := NewMultiExporter(exp1, exp2)

			spans := []*entity.UploadSpan{{SpanID: "test"}}
			err := multi.ExportSpans(ctx, spans)

			So(err, ShouldBeNil)
			So(exp1.exportSpansCalled, ShouldBeTrue)
			So(exp2.exportSpansCalled, ShouldBeTrue)
			So(exp1.exportedSpans, ShouldResemble, spans)
			So(exp2.exportedSpans, ShouldResemble, spans)
		})

		Convey("should continue if one exporter fails", func() {
			exp1 := &mockExporter{exportSpansErr: errors.New("exp1 error")}
			exp2 := &mockExporter{}
			multi := NewMultiExporter(exp1, exp2)

			spans := []*entity.UploadSpan{{SpanID: "test"}}
			err := multi.ExportSpans(ctx, spans)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "exp1 error")
			So(exp1.exportSpansCalled, ShouldBeTrue)
			So(exp2.exportSpansCalled, ShouldBeTrue)
		})

		Convey("should return first error when multiple exporters fail", func() {
			exp1 := &mockExporter{exportSpansErr: errors.New("first error")}
			exp2 := &mockExporter{exportSpansErr: errors.New("second error")}
			multi := NewMultiExporter(exp1, exp2)

			spans := []*entity.UploadSpan{{SpanID: "test"}}
			err := multi.ExportSpans(ctx, spans)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "first error")
		})
	})
}

func TestMultiExporter_ExportFiles(t *testing.T) {
	Convey("MultiExporter.ExportFiles", t, func() {
		ctx := context.Background()

		Convey("with no exporters should return nil", func() {
			multi := NewMultiExporter()
			err := multi.ExportFiles(ctx, []*entity.UploadFile{{}})
			So(err, ShouldBeNil)
		})

		Convey("should call all exporters", func() {
			exp1 := &mockExporter{}
			exp2 := &mockExporter{}
			multi := NewMultiExporter(exp1, exp2)

			files := []*entity.UploadFile{{TosKey: "test"}}
			err := multi.ExportFiles(ctx, files)

			So(err, ShouldBeNil)
			So(exp1.exportFilesCalled, ShouldBeTrue)
			So(exp2.exportFilesCalled, ShouldBeTrue)
			So(exp1.exportedFiles, ShouldResemble, files)
			So(exp2.exportedFiles, ShouldResemble, files)
		})

		Convey("should continue if one exporter fails", func() {
			exp1 := &mockExporter{exportFilesErr: errors.New("exp1 error")}
			exp2 := &mockExporter{}
			multi := NewMultiExporter(exp1, exp2)

			files := []*entity.UploadFile{{TosKey: "test"}}
			err := multi.ExportFiles(ctx, files)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "exp1 error")
			So(exp1.exportFilesCalled, ShouldBeTrue)
			So(exp2.exportFilesCalled, ShouldBeTrue)
		})
	})
}

func TestMultiExporter_AddExporter(t *testing.T) {
	Convey("MultiExporter.AddExporter", t, func() {
		Convey("should add exporter", func() {
			multi := NewMultiExporter()
			So(multi.ExporterCount(), ShouldEqual, 0)

			exp := &mockExporter{}
			multi.AddExporter(exp)
			So(multi.ExporterCount(), ShouldEqual, 1)
		})

		Convey("should not add nil exporter", func() {
			multi := NewMultiExporter()
			multi.AddExporter(nil)
			So(multi.ExporterCount(), ShouldEqual, 0)
		})
	})
}
