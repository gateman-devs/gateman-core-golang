# API Documentation Architecture

**Date:** October 21, 2025

## 📐 System Architecture

### Before: Monolithic JSON
```
┌─────────────────────────────────────┐
│      api-doc/index.json             │
│  ┌───────────────────────────────┐  │
│  │  Structure + Embedded Content │  │
│  │  - Endpoints                  │  │
│  │  - Schemas                    │  │
│  │  - Pages with inline markdown │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
              ↓
    GET /docs endpoint
              ↓
    Raw JSON response
```

**Issues:**
- ❌ Hard to edit markdown (JSON escaping)
- ❌ Poor git diffs (everything in one file)
- ❌ No syntax highlighting for markdown
- ❌ Difficult to maintain

---

### After: Modular Architecture
```
┌──────────────────────────────────────────────────────────┐
│                    api-doc/                               │
│  ┌────────────────────┐  ┌──────────────────────────┐   │
│  │   index.json       │  │   Markdown Files          │   │
│  │  - Structure       │  │  • getting-started.md     │   │
│  │  - Endpoints       │  │  • biometric-verification │   │
│  │  - Schemas         │  │    .md                    │   │
│  │  - Path references │  │  • [future pages...]      │   │
│  └────────────────────┘  └──────────────────────────┘   │
└──────────────────────────────────────────────────────────┘
                            ↓
              GET /docs endpoint (ginServer.go)
                            ↓
              ┌─────────────────────────┐
              │  Dynamic Loading Logic  │
              │  1. Load index.json     │
              │  2. Find contentPath    │
              │  3. Read .md files      │
              │  4. Inject as content   │
              └─────────────────────────┘
                            ↓
            Complete JSON response
         (identical to before)
```

**Benefits:**
- ✅ Easy to edit markdown (pure .md files)
- ✅ Clean git diffs (separate files)
- ✅ Full syntax highlighting
- ✅ Maintainable and scalable

---

## 🔄 Request Flow

```
Client Request: GET /docs
       ↓
┌──────────────────────────────────────┐
│  1. Load Base Structure              │
│     Read: api-doc/index.json         │
│     Parse: JSON → Go map             │
└──────────────────────────────────────┘
       ↓
┌──────────────────────────────────────┐
│  2. Scan for Content References      │
│     Check: pages[*].contentPath      │
└──────────────────────────────────────┘
       ↓
┌──────────────────────────────────────┐
│  3. Load Markdown Files              │
│     For each contentPath:            │
│     • Read file from disk            │
│     • Handle errors gracefully       │
└──────────────────────────────────────┘
       ↓
┌──────────────────────────────────────┐
│  4. Transform Structure              │
│     Replace:                         │
│     contentPath → content            │
│     With markdown string             │
└──────────────────────────────────────┘
       ↓
┌──────────────────────────────────────┐
│  5. Return Complete Response         │
│     JSON with embedded content       │
│     (backward compatible)            │
└──────────────────────────────────────┘
       ↓
Client receives full documentation
```

---

## 📂 File Organization

```
gateman-backend/
├── api-doc/
│   ├── index.json                    # Main structure (11KB)
│   ├── getting-started.md            # Getting started content (966B)
│   └── biometric-verification.md     # Biometric docs (1.5KB)
│
├── infrastructure/
│   └── ginServer.go                  # Server with /docs endpoint
│
└── Documentation files:
    ├── API_DOCS_MARKDOWN_EXTRACTION.md
    ├── API_DOCUMENTATION_UPDATES.md
    └── DOCS_ARCHITECTURE.md (this file)
```

---

## 🔍 Code Example

### index.json Structure
```json
{
  "title": "Gateman API Documentation",
  "version": "1.0.0",
  "endpoints": [...],
  "pages": {
    "getting-started": {
      "contentPath": "api-doc/getting-started.md"
    },
    "biometric-verification": {
      "contentPath": "api-doc/biometric-verification.md"
    }
  }
}
```

### Markdown File Format
```markdown
---
title: 'Getting Started'
description: 'Quick guide to start using Gateman API'
---

## Authentication

All Gateman API endpoints require authentication...

### Example Request Headers

\```json
{
  "x-api-key": "your-api-key",
  "x-app-id": "your-app-id"
}
\```
```

### Endpoint Implementation
```go
server.GET("/docs", func(ctx *gin.Context) {
    // 1. Load base JSON
    fileData, err := os.ReadFile("./api-doc/index.json")
    var jsonData map[string]interface{}
    json.Unmarshal(fileData, &jsonData)
    
    // 2. Load markdown files dynamically
    if pages, ok := jsonData["pages"].(map[string]interface{}); ok {
        for _, pageValue := range pages {
            if pageData, ok := pageValue.(map[string]interface{}); ok {
                if contentPath, ok := pageData["contentPath"].(string); ok {
                    // 3. Read and inject markdown
                    markdownContent, _ := os.ReadFile(contentPath)
                    delete(pageData, "contentPath")
                    pageData["content"] = string(markdownContent)
                }
            }
        }
    }
    
    // 4. Return complete response
    ctx.JSON(http.StatusOK, jsonData)
})
```

---

## 🎯 Response Format

### API Response (Unchanged)
```json
{
  "title": "Gateman API Documentation",
  "version": "1.0.0",
  "pages": {
    "getting-started": {
      "content": "---\ntitle: 'Getting Started'\n..."
    },
    "biometric-verification": {
      "content": "---\ntitle: 'Biometric Verification'\n..."
    }
  }
}
```

**Note:** Response format is identical to before - clients require no changes!

---

## 🚦 Error Handling

### File Not Found
```
contentPath: "api-doc/missing.md"
       ↓
File read fails
       ↓
Log error with details
       ↓
Keep contentPath in response
       ↓
API still returns (graceful degradation)
```

### Error Log Example
```json
{
  "key": "docs_markdown_load_error",
  "data": {
    "page": "getting-started",
    "content_path": "api-doc/getting-started.md",
    "error": "file not found"
  }
}
```

---

## 🧪 Testing

### Local Testing
```bash
# Start server
go run main.go

# Test endpoint
curl http://localhost:8080/docs | jq '.pages'

# Expected output:
{
  "getting-started": {
    "content": "---\ntitle: 'Getting Started'..."
  },
  "biometric-verification": {
    "content": "---\ntitle: 'Biometric Verification'..."
  }
}
```

### Verify Markdown Files
```bash
# List files
ls -lh api-doc/

# Check content
cat api-doc/getting-started.md
cat api-doc/biometric-verification.md
```

---

## 🔮 Future Enhancements

### 1. Caching Layer
```go
var docsCache map[string]string
var cacheMutex sync.RWMutex

func loadMarkdownWithCache(path string) string {
    cacheMutex.RLock()
    if content, ok := docsCache[path]; ok {
        cacheMutex.RUnlock()
        return content
    }
    cacheMutex.RUnlock()
    
    // Load and cache
    content := loadMarkdown(path)
    cacheMutex.Lock()
    docsCache[path] = content
    cacheMutex.Unlock()
    return content
}
```

### 2. Hot Reload (Dev Mode)
```go
if os.Getenv("APP_ENV") == "dev" {
    // Always read from disk
} else {
    // Use cache
}
```

### 3. Nested Structure
```
api-doc/
├── index.json
├── guides/
│   ├── getting-started.md
│   └── authentication.md
├── endpoints/
│   ├── biometric.md
│   └── webhooks.md
└── examples/
    ├── face-comparison.md
    └── liveness-check.md
```

---

## ✅ Validation Checklist

- [x] Markdown files created
- [x] JSON structure updated
- [x] Endpoint implementation complete
- [x] Error handling implemented
- [x] Backward compatibility maintained
- [x] Documentation created
- [x] No linter errors
- [x] Testing guide provided

---

## 📚 Related Documentation

1. `API_DOCS_MARKDOWN_EXTRACTION.md` - Detailed implementation guide
2. `API_DOCUMENTATION_UPDATES.md` - Response structure changes
3. `RESPONSE_STRUCTURE_CLEANUP.md` - DTO cleanup
4. `ACCURACY_IMPROVEMENTS_SUMMARY.md` - Biometric improvements

---

## 🎉 Summary

The API documentation system is now:
- **Modular:** Separate structure and content
- **Maintainable:** Easy to edit pure markdown
- **Scalable:** Simple to add new pages
- **Robust:** Graceful error handling
- **Compatible:** No breaking changes

**Zero client impact** - The response format remains identical!

