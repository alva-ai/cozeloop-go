# CozeLoop Go SDK - Technical Specification

## 1. Overview

**CozeLoop Go SDK** is an official Go client library for interacting with the [CozeLoop platform](https://loop.coze.cn).
It provides developers with tools to integrate observability tracing and prompt management into their LLM-powered
applications.

### 1.1 Key Features

- **Distributed Tracing**: Report traces with rich metadata for LLM operations
- **Prompt Management**: Fetch, cache, and format prompts from CozeLoop's prompt hub
- **Multi-modality Support**: Handle text, images, and file content in traces
- **Streaming Support**: Execute prompts with streaming responses
- **Cross-service Correlation**: Propagate trace context across service boundaries using W3C Trace Context

### 1.2 Requirements

- Go 1.18 or higher
- CozeLoop workspace credentials (JWT OAuth or API Token)

---

## 2. Architecture

### 2.1 Package Structure

```
cozeloop-go/
├── client.go           # Main client interface and implementation
├── trace.go            # Trace client interface and options
├── span.go             # Span interface for trace operations
├── prompt.go           # Prompt client interface
├── const.go            # Public constants and configuration types
├── error.go            # Public error types
├── noop.go             # No-operation implementations for graceful degradation
├── entity/             # Public data models
│   ├── prompt.go       # Prompt-related entities
│   └── stream.go       # Stream reader interface
├── spec/               # Trace specification module
│   └── tracespec/      # Standard trace data models
├── internal/           # Internal implementation
│   ├── consts/         # Internal constants
│   ├── httpclient/     # HTTP client with auth
│   ├── trace/          # Trace implementation
│   ├── prompt/         # Prompt implementation
│   ├── logger/         # Logging utilities
│   ├── stream/         # SSE stream handling
│   ├── idgen/          # ID generation
│   └── util/           # Utility functions
└── examples/           # Usage examples
```

### 2.2 Core Components

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Interface                         │
├─────────────────────────────────────────────────────────────────┤
│  PromptClient          │           TraceClient                   │
│  - GetPrompt()         │           - StartSpan()                 │
│  - PromptFormat()      │           - GetSpanFromContext()        │
│  - Execute()           │           - GetSpanFromHeader()         │
│  - ExecuteStreaming()  │           - Flush()                     │
└────────────┬───────────┴──────────────────┬─────────────────────┘
             │                              │
             ▼                              ▼
┌────────────────────────┐    ┌────────────────────────────────────┐
│    Prompt Provider     │    │         Trace Provider              │
│  - Cache management    │    │  - Span creation & lifecycle        │
│  - Template rendering  │    │  - Batch processing                 │
│  - OpenAPI client      │    │  - Export to CozeLoop server        │
└────────────────────────┘    └────────────────────────────────────┘
             │                              │
             └──────────────┬───────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP Client                               │
│  - JWT OAuth / Token Auth                                        │
│  - Request/Response handling                                     │
│  - Retry with exponential backoff                                │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. Client Interface

### 3.1 Client Creation

```go
type Client interface {
    PromptClient
    TraceClient
    GetWorkspaceID() string
    Close(ctx context.Context)
}
```

#### Configuration Options

| Option                              | Description                 | Default               |
| ----------------------------------- | --------------------------- | --------------------- |
| `WithAPIBaseURL(url)`               | API base URL                | `https://api.coze.cn` |
| `WithWorkspaceID(id)`               | Workspace identifier        | Required (from env)   |
| `WithHTTPClient(client)`            | Custom HTTP client          | `http.DefaultClient`  |
| `WithTimeout(duration)`             | Request timeout             | 3 seconds             |
| `WithUploadTimeout(duration)`       | File upload timeout         | 30 seconds            |
| `WithJWTOAuthClientID(id)`          | JWT OAuth client ID         | From env              |
| `WithJWTOAuthPrivateKey(key)`       | JWT OAuth private key       | From env              |
| `WithJWTOAuthPublicKeyID(id)`       | JWT OAuth public key ID     | From env              |
| `WithAPIToken(token)`               | API token (testing only)    | From env              |
| `WithPromptCacheMaxCount(n)`        | Max cached prompts          | 100                   |
| `WithPromptCacheRefreshInterval(d)` | Cache refresh interval      | 1 minute              |
| `WithPromptTrace(enable)`           | Enable prompt tracing       | false                 |
| `WithUltraLargeTraceReport(enable)` | Enable large content upload | false                 |
| `WithExporter(exporter)`            | Custom trace exporter       | Built-in exporter     |
| `WithTraceFinishEventProcessor(fn)` | Custom finish event handler | Default logger        |
| `WithTraceTagTruncateConf(conf)`    | Tag truncation limits       | Default limits        |
| `WithTraceQueueConf(conf)`          | Queue configuration         | Default config        |

#### Environment Variables

| Variable                           | Description              |
| ---------------------------------- | ------------------------ |
| `COZELOOP_API_BASE_URL`            | API base URL             |
| `COZELOOP_WORKSPACE_ID`            | Workspace ID             |
| `COZELOOP_JWT_OAUTH_CLIENT_ID`     | JWT OAuth client ID      |
| `COZELOOP_JWT_OAUTH_PRIVATE_KEY`   | JWT OAuth private key    |
| `COZELOOP_JWT_OAUTH_PUBLIC_KEY_ID` | JWT OAuth public key ID  |
| `COZELOOP_API_TOKEN`               | API token (testing only) |

### 3.2 Thread Safety

- The client is **thread-safe** and should be created once per application
- Creating multiple clients with the same configuration returns a cached instance
- Automatic graceful shutdown on SIGINT/SIGTERM signals

---

## 4. Tracing System

### 4.1 Span Interface

```go
type Span interface {
    SpanContext

    // Core operations
    SetInput(ctx context.Context, input interface{})
    SetOutput(ctx context.Context, output interface{})
    SetError(ctx context.Context, err error)
    SetStatusCode(ctx context.Context, code int)
    SetTags(ctx context.Context, tagKVs map[string]interface{})
    SetBaggage(ctx context.Context, baggageItems map[string]string)
    Finish(ctx context.Context)

    // LLM-specific setters
    SetModelProvider(ctx context.Context, provider string)
    SetModelName(ctx context.Context, name string)
    SetModelCallOptions(ctx context.Context, options interface{})
    SetInputTokens(ctx context.Context, tokens int)
    SetOutputTokens(ctx context.Context, tokens int)
    SetStartTimeFirstResp(ctx context.Context, timestamp int64)

    // Identifiers
    SetUserID(ctx context.Context, userID string)
    SetMessageID(ctx context.Context, messageID string)
    SetThreadID(ctx context.Context, threadID string)
    SetPrompt(ctx context.Context, prompt entity.Prompt)

    // Metadata
    SetServiceName(ctx context.Context, serviceName string)
    SetLogID(ctx context.Context, logID string)
    SetRuntime(ctx context.Context, runtime tracespec.Runtime)
    SetDeploymentEnv(ctx context.Context, env string)

    // Cross-service propagation
    ToHeader() (map[string]string, error)
    GetStartTime() time.Time
}

type SpanContext interface {
    GetSpanID() string
    GetTraceID() string
    GetBaggage() map[string]string
}
```

### 4.2 Span Types

The SDK supports various span types for different operations:

| Span Type         | Description               |
| ----------------- | ------------------------- |
| `model`           | LLM model invocation      |
| `tool`            | Tool/function call        |
| `retriever`       | Data retrieval operation  |
| `prompt_hub`      | Prompt fetch from hub     |
| `prompt_template` | Prompt template rendering |
| `custom`          | Custom user-defined span  |

### 4.3 Span Lifecycle

```go
// 1. Start a span
ctx, span := loop.StartSpan(ctx, "my-operation", "model")

// 2. Set attributes
span.SetInput(ctx, input)
span.SetModelName(ctx, "gpt-4")
span.SetModelProvider(ctx, "openai")

// 3. Perform operation
result, err := doOperation()

// 4. Set results
span.SetOutput(ctx, result)
if err != nil {
    span.SetError(ctx, err)
    span.SetStatusCode(ctx, -1)
}
span.SetInputTokens(ctx, 100)
span.SetOutputTokens(ctx, 50)

// 5. Finish span
span.Finish(ctx)
```

### 4.4 Span Options

| Option                    | Description                   |
| ------------------------- | ----------------------------- |
| `WithStartTime(t)`        | Custom start time             |
| `WithChildOf(spanCtx)`    | Set parent span               |
| `WithStartNewTrace()`     | Start a new trace             |
| `WithSpanID(id)`          | Custom span ID (16 hex chars) |
| `WithSpanWorkspaceID(id)` | Override workspace ID         |

### 4.5 Cross-Service Propagation

The SDK implements W3C Trace Context compatible headers:

```go
// Service A: Export trace context
headers, _ := span.ToHeader()
// Headers: X-Cozeloop-Traceparent, X-Cozeloop-Tracestate

// Service B: Import trace context
spanCtx := loop.GetSpanFromHeader(ctx, headers)
ctx, childSpan := loop.StartSpan(ctx, "child-op", "custom",
    loop.WithChildOf(spanCtx))
```

### 4.6 Trace Data Model

#### Span Structure

```go
type Span struct {
    SpanID       string            // 16 hex characters
    TraceID      string            // 32 hex characters
    ParentSpanID string            // "0" for root spans
    SpanType     string            // Type identifier
    Name         string            // Span name
    WorkspaceID  string            // Workspace identifier
    ServiceName  string            // Service name
    LogID        string            // Custom log ID
    StartTime    time.Time         // Start timestamp
    Duration     time.Duration     // Duration in microseconds
    StatusCode   int32             // 0 = success, non-zero = error
    TagMap       map[string]any    // Custom tags
    SystemTagMap map[string]any    // System tags
    Baggage      map[string]string // Propagated context
}
```

### 4.7 Tag Constraints

| Constraint                   | Limit           |
| ---------------------------- | --------------- |
| Max tags per span            | 50              |
| Max tag key size             | 1024 bytes      |
| Max tag value size (default) | 1024 bytes      |
| Max input/output size        | 1 MB            |
| Text truncation length       | 1000 characters |

### 4.8 Multi-modality Support

The SDK supports multi-modal content in input/output:

```go
input := &tracespec.ModelInput{
    Messages: []*tracespec.ModelMessage{{
        Role: "user",
        Parts: []*tracespec.ModelMessagePart{
            {Type: "text", Text: "Describe this image"},
            {Type: "image_url", ImageURL: &tracespec.ModelImageURL{
                URL: "data:image/png;base64,..." // or URL
            }},
        },
    }},
}
span.SetInput(ctx, input)
```

Supported content types:

- `text` - Plain text content
- `image_url` - Image URL or base64 data
- `file_url` - File URL or base64 data
- `audio_url` - Audio URL
- `video_url` - Video URL

### 4.9 Ultra Large Report

When enabled, content exceeding 1MB is uploaded as separate files:

```go
client, _ := cozeloop.NewClient(
    cozeloop.WithUltraLargeTraceReport(true),
)
```

### 4.10 Trace Reporting Pipeline

The SDK uses an asynchronous, batched reporting pipeline to efficiently upload traces to the CozeLoop server.

#### 4.10.1 Architecture Overview

```
┌──────────────┐     ┌──────────────┐     ┌──────────────────┐
│ span.Finish()│────▶│  Span Queue  │────▶│ ExportSpans()    │──▶ POST /v1/loop/traces/ingest
└──────────────┘     │  (1024 max)  │     │ (batch of 100)   │
                     └──────────────┘     └────────┬─────────┘
                                                   │ on failure
                                                   ▼
                     ┌──────────────┐     ┌──────────────────┐
                     │ Retry Queue  │◀────│ Re-queue spans   │
                     │  (512 max)   │     └──────────────────┘
                     └──────────────┘
                                                   │ on success (multimodal/large)
                                                   ▼
                     ┌──────────────┐     ┌──────────────────┐
                     │  File Queue  │────▶│ ExportFiles()    │──▶ POST /v1/loop/files/upload
                     │  (512 max)   │     └──────────────────┘
                     └──────────────┘
```

#### 4.10.2 Queue Configuration

| Queue        | Max Length | Batch Size | Batch Byte Size | Flush Interval |
| ------------ | ---------- | ---------- | --------------- | -------------- |
| Span         | 1024       | 100        | 4 MB            | 1 second       |
| Span Retry   | 512        | 50         | 4 MB            | 1 second       |
| File         | 512        | 1          | 100 MB          | 5 seconds      |
| File Retry   | 512        | 1          | 100 MB          | 5 seconds      |

Custom queue configuration:

```go
client, _ := cozeloop.NewClient(
    cozeloop.WithTraceQueueConf(&cozeloop.TraceQueueConf{
        SpanQueueLength:          2048,  // Max spans in queue
        SpanMaxExportBatchLength: 200,   // Spans per batch
    }),
)
```

#### 4.10.3 Reporting Flow

**Step 1: Span Finish → Queue Entry**

When `span.Finish(ctx)` is called, the span is enqueued for async processing:

```go
// internal/trace/span_processor.go
func (b *BatchSpanProcessor) OnSpanEnd(ctx context.Context, s *Span) {
    b.spanQM.Enqueue(ctx, s, s.bytesSize)
}
```

**Step 2: Batch Processing**

A background goroutine processes the queue, batching spans based on:
- **Timeout**: Flush every 1 second (configurable)
- **Count**: Flush when batch reaches 100 spans
- **Size**: Flush when batch reaches 4 MB

```go
// internal/trace/queue_manager.go
func (b *BatchQueueManager) processQueue() {
    for {
        select {
        case <-b.timer.C:
            b.doExport(ctx)  // Timeout triggered
        case sd := <-b.queue:
            b.batch = append(b.batch, sd)
            if b.isShouldExport() {
                b.doExport(ctx)  // Batch full
            }
        }
    }
}
```

**Step 3: Transform and Export**

Spans are transformed to upload format and sent to the server:

```go
// internal/trace/span_processor.go
func newExportSpansFunc(...) exportFunc {
    return func(ctx context.Context, l []interface{}) {
        // Transform spans to upload format
        uploadSpans, uploadFiles := transferToUploadSpanAndFile(ctx, spans)
        
        // Send to server
        err := exporter.ExportSpans(ctx, uploadSpans)
        
        if err != nil {
            // On failure: re-queue to retry queue
            for _, span := range spans {
                spanRetryQueue.Enqueue(ctx, span, span.bytesSize)
            }
        } else {
            // On success: queue files for upload
            for _, file := range uploadFiles {
                fileQueue.Enqueue(ctx, file, int64(len(file.Data)))
            }
        }
    }
}
```

**Step 4: HTTP Upload**

The exporter makes the actual HTTP calls:

```go
// internal/trace/exporter.go
func (e *SpanExporter) ExportSpans(ctx context.Context, ss []*entity.UploadSpan) error {
    return e.client.Post(ctx, "/v1/loop/traces/ingest", UploadSpanData{ss}, &resp)
}

func (e *SpanExporter) ExportFiles(ctx context.Context, files []*entity.UploadFile) error {
    for _, file := range files {
        e.client.UploadFile(ctx, "/v1/loop/files/upload", file.TosKey,
            bytes.NewReader([]byte(file.Data)),
            map[string]string{"workspace_id": file.SpaceID}, &resp)
    }
}
```

#### 4.10.4 API Endpoints

| Endpoint                    | Method | Description                                      |
| --------------------------- | ------ | ------------------------------------------------ |
| `/v1/loop/traces/ingest`    | POST   | Batch upload of spans (JSON payload)             |
| `/v1/loop/files/upload`     | POST   | Upload large files (multipart, images, base64)   |

#### 4.10.5 Upload Span Data Structure

```go
type UploadSpan struct {
    StartedATMicros  int64              // Start time in microseconds
    LogID            string             // Custom log ID
    SpanID           string             // 16 hex char span ID
    ParentID         string             // Parent span ID ("0" for root)
    TraceID          string             // 32 hex char trace ID
    DurationMicros   int64              // Duration in microseconds
    ServiceName      string             // Service identifier
    WorkspaceID      string             // Workspace identifier
    SpanName         string             // Span name
    SpanType         string             // Span type
    StatusCode       int32              // Status code (0 = success)
    Input            string             // Input content (JSON string)
    Output           string             // Output content (JSON string)
    ObjectStorage    string             // File references (JSON)
    SystemTagsString map[string]string  // System string tags
    SystemTagsLong   map[string]int64   // System numeric tags
    SystemTagsDouble map[string]float64 // System float tags
    TagsString       map[string]string  // Custom string tags
    TagsLong         map[string]int64   // Custom numeric tags
    TagsDouble       map[string]float64 // Custom float tags
    TagsBool         map[string]bool    // Custom boolean tags
}
```

#### 4.10.6 File Upload Types

| Upload Type      | Description                                    | Key Format                                     |
| ---------------- | ---------------------------------------------- | ---------------------------------------------- |
| Large Text       | Text content > 1MB (with UltraLargeReport)     | `{traceID}_{spanID}_{tagKey}_text_large_text`  |
| Multi-modality   | Base64 images/files from input/output          | `{traceID}_{spanID}_{tagKey}_{type}_{randomID}`|

#### 4.10.7 Retry Behavior

- **Automatic retry**: Failed exports are re-queued to a retry queue
- **Single retry**: Each span/file is retried once before being dropped
- **Non-blocking**: Queue full results in dropped items (logged as warning)
- **Graceful shutdown**: `Close(ctx)` drains all queues before exit

#### 4.10.8 Custom Exporter

You can implement a custom exporter for alternative backends:

```go
type Exporter interface {
    ExportSpans(ctx context.Context, spans []*entity.UploadSpan) error
    ExportFiles(ctx context.Context, files []*entity.UploadFile) error
}

// Example: Custom exporter
type MyExporter struct{}

func (e *MyExporter) ExportSpans(ctx context.Context, spans []*entity.UploadSpan) error {
    // Send to your own backend
    return nil
}

func (e *MyExporter) ExportFiles(ctx context.Context, files []*entity.UploadFile) error {
    // Handle file uploads
    return nil
}

// Use custom exporter
client, _ := cozeloop.NewClient(
    cozeloop.WithExporter(&MyExporter{}),
)
```

#### 4.10.9 Finish Event Processor

Monitor trace reporting events with a custom processor:

```go
client, _ := cozeloop.NewClient(
    cozeloop.WithTraceFinishEventProcessor(func(ctx context.Context, info *cozeloop.FinishEventInfo) {
        switch info.EventType {
        case cozeloop.SpanFinishEventSpanQueueEntryRate:
            // Span enqueued
        case cozeloop.SpanFinishEventFlushSpanRate:
            // Batch exported
            if info.IsEventFail {
                log.Printf("Export failed: %s", info.DetailMsg)
            }
        case cozeloop.SpanFinishEventFileQueueEntryRate:
            // File enqueued
        case cozeloop.SpanFinishEventFlushFileRate:
            // File uploaded
        }
    }),
)
```

---

## 5. Prompt Management

### 5.1 PromptClient Interface

```go
type PromptClient interface {
    GetPrompt(ctx context.Context, param GetPromptParam, options ...GetPromptOption) (*entity.Prompt, error)
    PromptFormat(ctx context.Context, prompt *entity.Prompt, variables map[string]any, options ...PromptFormatOption) ([]*entity.Message, error)
    Execute(ctx context.Context, param *entity.ExecuteParam, options ...ExecuteOption) (entity.ExecuteResult, error)
    ExecuteStreaming(ctx context.Context, param *entity.ExecuteParam, options ...ExecuteStreamingOption) (entity.StreamReader[entity.ExecuteResult], error)
}
```

### 5.2 Prompt Entity

```go
type Prompt struct {
    WorkspaceID    string
    PromptKey      string
    Version        string
    PromptTemplate *PromptTemplate
    Tools          []*Tool
    ToolCallConfig *ToolCallConfig
    LLMConfig      *LLMConfig
}

type PromptTemplate struct {
    TemplateType TemplateType    // "normal" or "jinja2"
    Messages     []*Message
    VariableDefs []*VariableDef
}

type Message struct {
    Role             Role           // system, user, assistant, tool, placeholder
    Content          *string
    Parts            []*ContentPart // Multi-modal content
    ReasoningContent *string
    ToolCallID       *string
    ToolCalls        []*ToolCall
}
```

### 5.3 Template Types

#### Normal Template (Mustache-style)

```
Hello {{name}}, welcome to {{service}}!
```

#### Jinja2 Template

```jinja2
{% for item in items %}
- {{ item.name }}: {{ item.value }}
{% endfor %}
```

### 5.4 Variable Types

| Type          | Go Type                 |
| ------------- | ----------------------- |
| `string`      | `string`                |
| `boolean`     | `bool`                  |
| `integer`     | `int`, `int32`, `int64` |
| `float`       | `float32`, `float64`    |
| `object`      | Any struct              |
| `placeholder` | `[]*entity.Message`     |
| `multi_part`  | `[]*entity.ContentPart` |
| `array<T>`    | `[]T`                   |

### 5.5 Prompt Caching

- **LRU Cache**: Configurable max size (default: 100)
- **Async Refresh**: Background refresh at configurable interval (default: 1 minute)
- **Cache Key**: Composite of `PromptKey + Version + Label`
- **Deep Copy**: Cached prompts are deep-copied to prevent mutation

### 5.6 Usage Examples

```go
// Get prompt
prompt, err := loop.GetPrompt(ctx, cozeloop.GetPromptParam{
    PromptKey: "my-prompt",
    Version:   "v1.0",  // Optional, defaults to latest
    Label:     "prod",  // Optional
})

// Format with variables
messages, err := loop.PromptFormat(ctx, prompt, map[string]any{
    "user_name": "Alice",
    "context":   "Hello world",
})

// Execute prompt (PTaaS - Prompt Template as a Service)
result, err := loop.Execute(ctx, &entity.ExecuteParam{
    PromptKey:    "my-prompt",
    VariableVals: map[string]any{"name": "Bob"},
})

// Streaming execution
stream, err := loop.ExecuteStreaming(ctx, &entity.ExecuteParam{
    PromptKey:    "my-prompt",
    VariableVals: map[string]any{"name": "Charlie"},
})
for {
    chunk, err := stream.Recv()
    if err == io.EOF {
        break
    }
    fmt.Print(chunk.Message.Content)
}
```

---

## 6. Authentication

### 6.1 JWT OAuth (Recommended for Production)

```go
client, _ := cozeloop.NewClient(
    cozeloop.WithJWTOAuthClientID("your-client-id"),
    cozeloop.WithJWTOAuthPrivateKey("-----BEGIN RSA PRIVATE KEY-----\n..."),
    cozeloop.WithJWTOAuthPublicKeyID("your-public-key-id"),
)
```

Features:

- Automatic token refresh
- Token caching with singleflight deduplication
- Configurable TTL (default: 900 seconds)
- Advance refresh (60 seconds before expiry)

### 6.2 API Token (Testing Only)

```go
client, _ := cozeloop.NewClient(
    cozeloop.WithAPIToken("your-api-token"),
)
```

---

## 7. HTTP Client

### 7.1 Features

- **Connection pooling**: Uses Go's `http.DefaultClient` by default
- **Automatic retry**: Exponential backoff with jitter
- **Timeout management**: Configurable request and upload timeouts
- **Header enrichment**: Automatic trace context injection
- **User-Agent**: SDK version identification

### 7.2 Retry Strategy

- Initial delay: 100ms
- Max delay: 10s
- Multiplier: 2x
- Jitter: Random factor
- Retryable status codes: 429, 500, 502, 503, 504

---

## 8. Error Handling

### 8.1 Error Types

```go
var (
    ErrInvalidParam     // Invalid parameter error
    ErrHeaderParent     // Invalid trace header
    ErrRemoteService    // Remote service error
    ErrAuthInfoRequired // Missing authentication
    ErrParsePrivateKey  // Invalid private key
)

type AuthError struct {
    Code int
    Msg  string
}

type RemoteServiceError struct {
    Code int
    Msg  string
}
```

### 8.2 Graceful Degradation

The SDK implements a `NoopClient` for graceful degradation:

- Returns immediately without errors for all operations
- Logs warnings about initialization failures
- Prevents application crashes due to SDK issues

---

## 9. Specification Models (tracespec)

### 9.1 Model Input/Output

```go
type ModelInput struct {
    Messages        []*ModelMessage
    Tools           []*ModelTool
    ModelToolChoice *ModelToolChoice
}

type ModelOutput struct {
    ID      string
    Choices []*ModelChoice
}

type ModelCallOption struct {
    Temperature      float32
    MaxTokens        int64
    Stop             []string
    TopP             float32
    TopK             *int64
    PresencePenalty  *float32
    FrequencyPenalty *float32
    ReasoningEffort  string
}
```

### 9.2 Runtime Information

```go
type Runtime struct {
    Language       string // "go"
    Library        string
    Scene          string // "custom", "prompt_hub", etc.
    LoopSDKVersion string
}
```

### 9.3 Standard Tag Keys

| Key                  | Description                   |
| -------------------- | ----------------------------- |
| `input`              | Input data (JSON)             |
| `output`             | Output data (JSON)            |
| `error`              | Error message                 |
| `model_provider`     | LLM provider (e.g., "openai") |
| `model_name`         | Model name (e.g., "gpt-4")    |
| `input_tokens`       | Input token count             |
| `output_tokens`      | Output token count            |
| `tokens`             | Total token count             |
| `call_options`       | Model call options            |
| `prompt_key`         | Prompt identifier             |
| `prompt_version`     | Prompt version                |
| `runtime`            | Runtime information           |
| `latency_first_resp` | First response latency        |

---

## 10. Examples

### 10.1 Basic Tracing

```go
package main

import (
    "context"
    loop "github.com/coze-dev/cozeloop-go"
)

func main() {
    ctx := context.Background()

    // Start root span
    ctx, span := loop.StartSpan(ctx, "chat-completion", "model")
    defer span.Finish(ctx)

    span.SetInput(ctx, "What is the weather?")
    span.SetModelName(ctx, "gpt-4")
    span.SetModelProvider(ctx, "openai")

    // Your LLM call here
    output := "The weather is sunny."

    span.SetOutput(ctx, output)
    span.SetInputTokens(ctx, 10)
    span.SetOutputTokens(ctx, 5)

    // Ensure traces are flushed
    loop.Close(ctx)
}
```

### 10.2 Parent-Child Spans

```go
func processRequest(ctx context.Context) {
    ctx, parentSpan := loop.StartSpan(ctx, "process-request", "custom")
    defer parentSpan.Finish(ctx)

    // Child span automatically links to parent
    ctx, childSpan := loop.StartSpan(ctx, "llm-call", "model")
    // ... do work ...
    childSpan.Finish(ctx)

    // Another child
    ctx, anotherChild := loop.StartSpan(ctx, "tool-call", "tool")
    // ... do work ...
    anotherChild.Finish(ctx)
}
```

### 10.3 Cross-Service Tracing

```go
// Service A
func handleRequest(w http.ResponseWriter, r *http.Request) {
    ctx, span := loop.StartSpan(r.Context(), "service-a", "custom")
    defer span.Finish(ctx)

    headers, _ := span.ToHeader()

    // Call Service B
    req, _ := http.NewRequest("POST", "http://service-b/api", nil)
    for k, v := range headers {
        req.Header.Set(k, v)
    }
    http.DefaultClient.Do(req)
}

// Service B
func handleAPI(w http.ResponseWriter, r *http.Request) {
    headers := make(map[string]string)
    for k := range r.Header {
        headers[k] = r.Header.Get(k)
    }

    spanCtx := loop.GetSpanFromHeader(r.Context(), headers)
    ctx, span := loop.StartSpan(r.Context(), "service-b", "custom",
        loop.WithChildOf(spanCtx))
    defer span.Finish(ctx)

    // Process request...
}
```

---

## 11. Performance Considerations

### 11.1 Batching

- Spans are batched before upload
- Configurable queue size and flush intervals
- Background processing to avoid blocking

### 11.2 Memory Management

- Prompt cache with configurable size limit
- Tag value truncation to prevent memory issues
- Deep copy of cached objects to prevent mutation

### 11.3 Best Practices

1. **Create one client**: Reuse the same client instance
2. **Call Close()**: Ensure proper cleanup on shutdown
3. **Use context**: Pass context for cancellation support
4. **Finish spans**: Always call `Finish()` to report spans
5. **Set baggage wisely**: Baggage is a set of key-value pairs attached to a span's context that is automatically
   propagated to all child spans and across service boundaries. Use baggage for small, essential metadata needed
   throughout a trace, but avoid placing large or sensitive data to minimize performance overhead.
6. **Handle errors**: Check errors from `GetPrompt()` and `PromptFormat()`

---

## 12. Versioning

The SDK follows semantic versioning. Current version information is embedded at build time and included in trace runtime
metadata.

---

## 13. License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

---

## 14. Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

---

## 15. Support

- **Security Issues**: Report via [ByteDance Security Center](https://security.bytedance.com/src) or sec@bytedance.com
- **Bug Reports**: Create a GitHub issue
- **Documentation**: https://loop.coze.cn
