// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/coze-dev/cozeloop-go"
	"github.com/coze-dev/cozeloop-go/internal/logger"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

func main() {
	// This example demonstrates how to export traces to both the CozeLoop server
	// AND a local markdown file simultaneously.
	//
	// You can enable local file export in two ways:
	//
	// Option 1: Environment variables
	//   COZELOOP_LOCAL_FILE_EXPORT_ENABLED=true
	//   COZELOOP_LOCAL_FILE_EXPORT_PATH=/path/to/traces.md
	//
	// Option 2: Programmatic configuration (shown below)

	// Set the following environment variables first (Assuming you are using a PAT token.)
	// COZELOOP_WORKSPACE_ID=your workspace id
	// COZELOOP_API_TOKEN=your token

	logger.SetLogLevel(logger.LogLevelInfo)

	// Create client with local file export enabled
	// The local file will contain human-readable markdown format traces
	client, err := cozeloop.NewClient(
		cozeloop.WithLocalFileExport(true),                          // Enable local file export
		cozeloop.WithLocalFileExportPath("./traces/my_traces.md"),   // Custom file path (optional)
	)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	fmt.Println("Starting trace example with local file export...")
	fmt.Println("Traces will be written to: ./traces/my_traces.md")

	// Create a root span
	ctx, rootSpan := client.StartSpan(ctx, "local_file_export_example", "main")

	// Set some tags
	rootSpan.SetTags(ctx, map[string]interface{}{
		"example_type": "local_file_export",
		"environment":  "development",
	})
	rootSpan.SetUserID(ctx, "user_123")

	// Simulate some work with a child span
	simulateLLMCall(ctx, client)

	// Finish the root span
	rootSpan.Finish(ctx)

	// Force flush to ensure all spans are written
	client.Flush(ctx)

	fmt.Println("\nTrace completed!")
	fmt.Println("Check the local file for the trace data.")

	// Print the file content if it exists
	printLocalTraceFile("./traces/my_traces.md")
}

func simulateLLMCall(ctx context.Context, client cozeloop.Client) {
	ctx, span := client.StartSpan(ctx, "llm_call", tracespec.VModelSpanType)
	defer span.Finish(ctx)

	// Set LLM-specific tags
	span.SetModelName(ctx, "gpt-4o")
	span.SetModelProvider(ctx, "openai")
	span.SetInput(ctx, "What is the weather like in Shanghai?")

	// Simulate processing time
	time.Sleep(500 * time.Millisecond)

	// Set output and token counts
	span.SetOutput(ctx, "The weather in Shanghai is sunny with temperatures around 25°C.")
	span.SetInputTokens(ctx, 15)
	span.SetOutputTokens(ctx, 20)
	span.SetStartTimeFirstResp(ctx, time.Now().UnixMicro())
}

func printLocalTraceFile(path string) {
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Could not read trace file: %v\n", err)
		return
	}

	fmt.Println("\n=== Local Trace File Content ===")
	fmt.Println(string(content))
	fmt.Println("=== End of Trace File ===")
}
