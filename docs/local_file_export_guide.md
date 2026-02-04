# Local File Export Guide

This guide explains how to use the local file export feature in cozeloop-go to export traces to both the CozeLoop server and a local markdown file simultaneously.

## Overview

The local file export feature allows you to:

- **Dual Export**: Send traces to the CozeLoop server while simultaneously writing them to a local file
- **Human-Readable Format**: Local traces are written in markdown format for easy reading and debugging
- **Flexible Configuration**: Enable via environment variables or programmatic options
- **Zero Code Changes**: Existing tracing code works without modification when this feature is enabled

This is particularly useful for:
- Local development and debugging
- Offline trace analysis
- Compliance and audit logging
- Troubleshooting LLM application behavior

## Quick Start

### Option 1: Environment Variables

The simplest way to enable local file export is via environment variables:

```bash
# Enable local file export
export COZELOOP_LOCAL_FILE_EXPORT_ENABLED=true

# Optional: Set custom file path (defaults to ./cozeloop_traces.md)
export COZELOOP_LOCAL_FILE_EXPORT_PATH=/var/log/myapp/traces.md

# Your existing environment variables
export COZELOOP_WORKSPACE_ID=your_workspace_id
export COZELOOP_API_TOKEN=your_api_token
```

Then run your application normally - no code changes required!

### Option 2: Programmatic Configuration

```go
package main

import (
    "context"
    "github.com/coze-dev/cozeloop-go"
)

func main() {
    client, err := cozeloop.NewClient(
        cozeloop.WithLocalFileExport(true),                        // Enable local file export
        cozeloop.WithLocalFileExportPath("./traces/app_traces.md"), // Custom path (optional)
    )
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    
    // Your existing tracing code works unchanged
    ctx, span := client.StartSpan(ctx, "my_operation", "custom")
    span.SetInput(ctx, "user query")
    span.SetOutput(ctx, "response")
    span.Finish(ctx)
    
    // Flush to ensure traces are written
    client.Flush(ctx)
}
```

## Configuration Options

| Option | Environment Variable | Default | Description |
|--------|---------------------|---------|-------------|
| `WithLocalFileExport(bool)` | `COZELOOP_LOCAL_FILE_EXPORT_ENABLED` | `false` | Enable/disable local file export |
| `WithLocalFileExportPath(string)` | `COZELOOP_LOCAL_FILE_EXPORT_PATH` | `./cozeloop_traces.md` | Path to the local trace file |

## Integration with LLM Applications

### Basic LLM Call Tracing

```go
package main

import (
    "context"
    "github.com/coze-dev/cozeloop-go"
    "github.com/coze-dev/cozeloop-go/spec/tracespec"
)

func main() {
    // Initialize client with local file export
    client, _ := cozeloop.NewClient(
        cozeloop.WithLocalFileExport(true),
        cozeloop.WithLocalFileExportPath("./llm_traces.md"),
    )
    defer client.Close(context.Background())
    
    ctx := context.Background()
    
    // Trace an LLM call
    response, err := callLLM(ctx, client, "What is the weather in Tokyo?")
    if err != nil {
        // Error is automatically traced
    }
    
    // Flush traces
    client.Flush(ctx)
}

func callLLM(ctx context.Context, client cozeloop.Client, prompt string) (string, error) {
    // Create a model span
    ctx, span := client.StartSpan(ctx, "openai_chat", tracespec.VModelSpanType)
    defer span.Finish(ctx)
    
    // Set model information
    span.SetModelProvider(ctx, "openai")
    span.SetModelName(ctx, "gpt-4o")
    span.SetInput(ctx, prompt)
    
    // Make your LLM call here
    // response, err := openaiClient.CreateChatCompletion(...)
    
    // Simulated response for demo
    response := "The weather in Tokyo is sunny with 22°C."
    
    // Set output and metrics
    span.SetOutput(ctx, response)
    span.SetInputTokens(ctx, 15)
    span.SetOutputTokens(ctx, 25)
    
    return response, nil
}
```

### RAG (Retrieval-Augmented Generation) Pipeline

```go
package main

import (
    "context"
    "github.com/coze-dev/cozeloop-go"
    "github.com/coze-dev/cozeloop-go/spec/tracespec"
)

func main() {
    client, _ := cozeloop.NewClient(
        cozeloop.WithLocalFileExport(true),
        cozeloop.WithLocalFileExportPath("./rag_traces.md"),
    )
    defer client.Close(context.Background())
    
    ctx := context.Background()
    
    // Run RAG pipeline
    runRAGPipeline(ctx, client, "How do I configure logging?")
    
    client.Flush(ctx)
}

func runRAGPipeline(ctx context.Context, client cozeloop.Client, query string) string {
    // Root span for the entire RAG pipeline
    ctx, rootSpan := client.StartSpan(ctx, "rag_pipeline", "rag")
    defer rootSpan.Finish(ctx)
    
    rootSpan.SetInput(ctx, query)
    rootSpan.SetUserID(ctx, "user_123")
    
    // Step 1: Retrieve relevant documents
    docs := retrieveDocuments(ctx, client, query)
    
    // Step 2: Generate response with context
    response := generateResponse(ctx, client, query, docs)
    
    rootSpan.SetOutput(ctx, response)
    return response
}

func retrieveDocuments(ctx context.Context, client cozeloop.Client, query string) []string {
    ctx, span := client.StartSpan(ctx, "document_retrieval", tracespec.VRetrieverSpanType)
    defer span.Finish(ctx)
    
    span.SetInput(ctx, query)
    
    // Simulate retrieval
    docs := []string{
        "Document 1: Logging can be configured via environment variables...",
        "Document 2: Use SetLogLevel() to change log verbosity...",
    }
    
    span.SetOutput(ctx, docs)
    span.SetTags(ctx, map[string]interface{}{
        "num_documents": len(docs),
        "retriever":     "vector_db",
    })
    
    return docs
}

func generateResponse(ctx context.Context, client cozeloop.Client, query string, docs []string) string {
    ctx, span := client.StartSpan(ctx, "llm_generation", tracespec.VModelSpanType)
    defer span.Finish(ctx)
    
    span.SetModelProvider(ctx, "openai")
    span.SetModelName(ctx, "gpt-4o")
    span.SetInput(ctx, tracespec.ModelInput{
        Messages: []*tracespec.ModelMessage{
            {Parts: []*tracespec.ModelMessagePart{{Type: "text", Text: "Context: " + docs[0]}}},
            {Parts: []*tracespec.ModelMessagePart{{Type: "text", Text: "Query: " + query}}},
        },
    })
    
    // Simulate LLM response
    response := "To configure logging, you can use environment variables or call SetLogLevel()..."
    
    span.SetOutput(ctx, response)
    span.SetInputTokens(ctx, 150)
    span.SetOutputTokens(ctx, 50)
    
    return response
}
```

### Multi-Agent System

```go
package main

import (
    "context"
    "github.com/coze-dev/cozeloop-go"
)

func main() {
    client, _ := cozeloop.NewClient(
        cozeloop.WithLocalFileExport(true),
        cozeloop.WithLocalFileExportPath("./agent_traces.md"),
    )
    defer client.Close(context.Background())
    
    ctx := context.Background()
    
    // Run multi-agent workflow
    runAgentWorkflow(ctx, client, "Research and summarize AI trends")
    
    client.Flush(ctx)
}

func runAgentWorkflow(ctx context.Context, client cozeloop.Client, task string) {
    // Orchestrator span
    ctx, orchestratorSpan := client.StartSpan(ctx, "orchestrator", "agent")
    defer orchestratorSpan.Finish(ctx)
    
    orchestratorSpan.SetInput(ctx, task)
    orchestratorSpan.SetTags(ctx, map[string]interface{}{
        "workflow_type": "research_summarize",
    })
    
    // Research Agent
    researchResult := runResearchAgent(ctx, client, task)
    
    // Summarizer Agent
    summary := runSummarizerAgent(ctx, client, researchResult)
    
    orchestratorSpan.SetOutput(ctx, summary)
}

func runResearchAgent(ctx context.Context, client cozeloop.Client, task string) string {
    ctx, span := client.StartSpan(ctx, "research_agent", "agent")
    defer span.Finish(ctx)
    
    span.SetInput(ctx, task)
    span.SetTags(ctx, map[string]interface{}{
        "agent_role": "researcher",
        "tools":      []string{"web_search", "arxiv"},
    })
    
    // Simulate research
    result := "Research findings: AI trends include LLMs, agents, multimodal models..."
    
    span.SetOutput(ctx, result)
    return result
}

func runSummarizerAgent(ctx context.Context, client cozeloop.Client, content string) string {
    ctx, span := client.StartSpan(ctx, "summarizer_agent", "agent")
    defer span.Finish(ctx)
    
    span.SetInput(ctx, content)
    span.SetTags(ctx, map[string]interface{}{
        "agent_role":   "summarizer",
        "output_format": "bullet_points",
    })
    
    // Simulate summarization
    summary := "Summary:\n- LLMs continue to grow\n- Agent frameworks emerging\n- Multimodal is key"
    
    span.SetOutput(ctx, summary)
    return summary
}
```

## Output Format

The local file uses a human-readable markdown format. Each span is written as:

```markdown
# Trace: abc123def456789012345678901234

## Span: my_operation
- **Type:** model
- **Span ID:** span1234567890ab
- **Parent ID:** parent12345678
- **Start Time:** 2026-02-04 10:30:45.123
- **Duration:** 1.234s
- **Status:** OK (0)
- **Service:** my_service
- **Workspace ID:** ws_123

### Input
```
What is the weather in Tokyo?
```

### Output
```
The weather in Tokyo is sunny with 22°C.
```

### Tags
| Key | Value |
|-----|-------|
| model_name | gpt-4o |
| model_provider | openai |
| input_tokens | 15 |
| output_tokens | 25 |

### System Tags
| Key | Value |
|-----|-------|
| _runtime | {"language":"go","loop_sdk_version":"0.1.0"} |

---
```

## Best Practices

### 1. Use Meaningful Span Names

```go
// Good: Descriptive names
ctx, span := client.StartSpan(ctx, "openai_chat_completion", tracespec.VModelSpanType)
ctx, span := client.StartSpan(ctx, "vector_search_documents", tracespec.VRetrieverSpanType)

// Avoid: Generic names
ctx, span := client.StartSpan(ctx, "step1", "custom")
```

### 2. Set Appropriate Tags

```go
span.SetTags(ctx, map[string]interface{}{
    "model_name":     "gpt-4o",
    "temperature":    0.7,
    "max_tokens":     1000,
    "retry_count":    0,
    "cache_hit":      false,
})
```

### 3. Handle Errors Properly

```go
func callLLM(ctx context.Context, client cozeloop.Client) error {
    ctx, span := client.StartSpan(ctx, "llm_call", tracespec.VModelSpanType)
    defer span.Finish(ctx)
    
    response, err := makeLLMRequest()
    if err != nil {
        span.SetError(ctx, err)
        span.SetStatusCode(ctx, 500)
        return err
    }
    
    span.SetOutput(ctx, response)
    return nil
}
```

### 4. Flush Before Exit

```go
func main() {
    client, _ := cozeloop.NewClient(
        cozeloop.WithLocalFileExport(true),
    )
    
    // Use defer to ensure flush on exit
    defer func() {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        client.Flush(ctx)
        client.Close(ctx)
    }()
    
    // Your application code
}
```

### 5. Organize Traces by Environment

```go
// Development
client, _ := cozeloop.NewClient(
    cozeloop.WithLocalFileExport(true),
    cozeloop.WithLocalFileExportPath("./traces/dev_traces.md"),
)

// Or use environment-based paths
tracePath := fmt.Sprintf("./traces/%s_traces.md", os.Getenv("ENV"))
client, _ := cozeloop.NewClient(
    cozeloop.WithLocalFileExport(true),
    cozeloop.WithLocalFileExportPath(tracePath),
)
```

### 6. Log Rotation (External)

For production use, consider using external log rotation tools like `logrotate`:

```bash
# /etc/logrotate.d/cozeloop-traces
/var/log/myapp/traces.md {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
}
```

## Troubleshooting

### Traces Not Written to File

1. Check that local file export is enabled:
   ```go
   cozeloop.WithLocalFileExport(true)
   ```

2. Verify the file path is writable:
   ```bash
   touch /path/to/traces.md && rm /path/to/traces.md
   ```

3. Ensure `Flush()` is called before the program exits:
   ```go
   client.Flush(ctx)
   ```

### File Permission Errors

The SDK automatically creates parent directories, but the process needs write permissions:

```bash
# Ensure directory is writable
chmod 755 /path/to/traces/
```

### Large File Size

The trace file grows continuously. Consider:
- Using log rotation
- Periodically archiving old traces
- Setting up a cron job to clean up old trace files

## FAQ

**Q: Does enabling local file export affect server reporting?**

A: No. When local file export is enabled, traces are sent to both the CozeLoop server AND the local file. Server reporting continues normally.

**Q: What happens if the file write fails?**

A: The SDK logs an error but continues to export to the server. File export failures don't block server reporting.

**Q: Can I use a custom exporter instead?**

A: Yes. Use `WithExporter()` to provide a completely custom exporter that implements the `Exporter` interface.

**Q: Is the file format configurable?**

A: Currently, only markdown format is supported. The format is designed for human readability.

**Q: Are binary files (images, etc.) written to the local file?**

A: No. Binary content like images from multimodal inputs are not written to the local file to keep it readable. Only text content and metadata are included.
