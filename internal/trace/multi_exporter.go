// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/logger"
)

var _ Exporter = (*MultiExporter)(nil)

// MultiExporter wraps multiple exporters and calls them all
// This allows exporting spans to multiple destinations (e.g., server + local file)
type MultiExporter struct {
	exporters []Exporter
}

// NewMultiExporter creates a new MultiExporter with the given exporters
func NewMultiExporter(exporters ...Exporter) *MultiExporter {
	// Filter out nil exporters
	validExporters := make([]Exporter, 0, len(exporters))
	for _, e := range exporters {
		if e != nil {
			validExporters = append(validExporters, e)
		}
	}
	return &MultiExporter{
		exporters: validExporters,
	}
}

// ExportSpans exports spans to all wrapped exporters
// It continues exporting even if one exporter fails, but returns the first error encountered
func (m *MultiExporter) ExportSpans(ctx context.Context, spans []*entity.UploadSpan) error {
	if len(m.exporters) == 0 {
		return nil
	}

	var firstErr error
	for _, exporter := range m.exporters {
		if err := exporter.ExportSpans(ctx, spans); err != nil {
			logger.CtxErrorf(ctx, "multi-exporter: failed to export spans: %v", err)
			if firstErr == nil {
				firstErr = err
			}
			// Continue to try other exporters even if one fails
		}
	}

	return firstErr
}

// ExportFiles exports files to all wrapped exporters
// It continues exporting even if one exporter fails, but returns the first error encountered
func (m *MultiExporter) ExportFiles(ctx context.Context, files []*entity.UploadFile) error {
	if len(m.exporters) == 0 {
		return nil
	}

	var firstErr error
	for _, exporter := range m.exporters {
		if err := exporter.ExportFiles(ctx, files); err != nil {
			logger.CtxErrorf(ctx, "multi-exporter: failed to export files: %v", err)
			if firstErr == nil {
				firstErr = err
			}
			// Continue to try other exporters even if one fails
		}
	}

	return firstErr
}

// AddExporter adds an exporter to the multi-exporter
func (m *MultiExporter) AddExporter(exporter Exporter) {
	if exporter != nil {
		m.exporters = append(m.exporters, exporter)
	}
}

// ExporterCount returns the number of exporters
func (m *MultiExporter) ExporterCount() int {
	return len(m.exporters)
}
