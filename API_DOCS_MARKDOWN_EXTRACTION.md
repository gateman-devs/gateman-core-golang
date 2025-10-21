# API Documentation Markdown Extraction

**Date:** October 21, 2025  
**Status:** ✅ Complete

## Overview

Extracted embedded markdown content from `api-doc/index.json` into separate `.md` files and implemented dynamic loading in the `/docs` endpoint. This improves maintainability, version control, and makes documentation easier to edit.

---

## 🎯 Goals Achieved

1. ✅ Extract markdown content from JSON into separate files
2. ✅ Update JSON structure to use file path references
3. ✅ Implement dynamic markdown loading in `/docs` endpoint
4. ✅ Maintain backward compatibility (response format unchanged)

---

## 📁 Files Created

### 1. `api-doc/getting-started.md`
- **Purpose:** Getting started guide for Gateman API
- **Content:** Authentication, base URLs, example headers
- **Size:** ~30 lines

### 2. `api-doc/biometric-verification.md`
- **Purpose:** Biometric verification documentation
- **Content:** Endpoints overview, quick start, features, best practices
- **Size:** ~70 lines

---

## 🔄 Files Modified

### 1. `api-doc/index.json`
**Before:**
```json
"pages": {
  "getting-started": {
    "content": "---\ntitle: 'Getting Started'\n..."
  },
  "biometric-verification": {
    "content": "---\ntitle: 'Biometric Verification'\n..."
  }
}
```

**After:**
```json
"pages": {
  "getting-started": {
    "contentPath": "api-doc/getting-started.md"
  },
  "biometric-verification": {
    "contentPath": "api-doc/biometric-verification.md"
  }
}
```

### 2. `infrastructure/ginServer.go`
Enhanced the `/docs` endpoint (lines 118-159) with dynamic markdown loading:

**Before:**
```go
server.GET("/docs", func(ctx *gin.Context) {
    fileData, err := os.ReadFile("./api-doc/index.json")
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read API documentation"})
        return
    }
    var jsonData interface{}
    if err := json.Unmarshal(fileData, &jsonData); err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse API documentation"})
        return
    }
    ctx.JSON(http.StatusOK, jsonData)
})
```

**After:**
```go
server.GET("/docs", func(ctx *gin.Context) {
    fileData, err := os.ReadFile("./api-doc/index.json")
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read API documentation"})
        return
    }
    var jsonData map[string]interface{}
    if err := json.Unmarshal(fileData, &jsonData); err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse API documentation"})
        return
    }

    // Load markdown content from files if pages exist
    if pages, ok := jsonData["pages"].(map[string]interface{}); ok {
        for pageKey, pageValue := range pages {
            if pageData, ok := pageValue.(map[string]interface{}); ok {
                // Check if there's a contentPath field
                if contentPath, ok := pageData["contentPath"].(string); ok {
                    // Read the markdown file
                    markdownContent, err := os.ReadFile(contentPath)
                    if err != nil {
                        logger.Error("Failed to load markdown file", logger.LoggerOptions{
                            Key: "docs_markdown_load_error",
                            Data: map[string]interface{}{
                                "page":         pageKey,
                                "content_path": contentPath,
                                "error":        err.Error(),
                            },
                        })
                        // Keep contentPath in response if file not found
                        continue
                    }
                    // Replace contentPath with content
                    delete(pageData, "contentPath")
                    pageData["content"] = string(markdownContent)
                }
            }
        }
    }

    ctx.JSON(http.StatusOK, jsonData)
})
```

---

## 🔧 Implementation Details

### Dynamic Loading Logic

1. **Load Base JSON:** Read `api-doc/index.json`
2. **Parse Structure:** Unmarshal into Go map
3. **Iterate Pages:** Loop through each page in the `pages` object
4. **Check for Path:** Look for `contentPath` field in each page
5. **Load Markdown:** Read the markdown file from the specified path
6. **Inject Content:** Replace `contentPath` with `content` field containing the markdown
7. **Error Handling:** Log errors but keep `contentPath` if file fails to load
8. **Return Response:** Send complete JSON with embedded markdown content

### Benefits

1. **Separation of Concerns:**
   - JSON structure in `index.json`
   - Content in readable `.md` files

2. **Easier Maintenance:**
   - Edit markdown without JSON escaping
   - Better syntax highlighting in editors
   - Cleaner git diffs

3. **Version Control:**
   - Track content changes separately from structure
   - Easier to review documentation changes

4. **Scalability:**
   - Easy to add new documentation pages
   - Simple to organize content by topic

5. **Developer Experience:**
   - Write documentation in pure markdown
   - No need to escape special characters
   - Standard markdown tools work correctly

---

## 🧪 Testing

### Test the Endpoint
```bash
curl http://localhost:8080/docs
```

### Expected Response
The response should be identical to the previous format with inline content, but the markdown is now loaded dynamically:

```json
{
  "title": "Gateman API Documentation",
  "pages": {
    "getting-started": {
      "content": "---\ntitle: 'Getting Started'\n..."
    },
    "biometric-verification": {
      "content": "---\ntitle: 'Biometric Verification'\n..."
    }
  },
  ...
}
```

---

## 🛡️ Error Handling

- **File Not Found:** Logs error with details, keeps `contentPath` in response
- **Read Error:** Logs error, continues processing other pages
- **Parse Error:** Returns 500 error with message
- **Graceful Degradation:** If markdown fails to load, the JSON structure is still returned

---

## 📋 File Structure

```
api-doc/
├── index.json               # Main API doc structure (references)
├── getting-started.md       # Getting started content
└── biometric-verification.md # Biometric verification content
```

---

## 🚀 Future Enhancements

Potential improvements for future iterations:

1. **Caching:** Cache loaded markdown in memory to reduce file I/O
2. **Watch Mode:** Auto-reload markdown files in development
3. **Validation:** Validate markdown structure/frontmatter
4. **Nested Structure:** Support subdirectories for organization
5. **Templates:** Support markdown templates with variables
6. **API Versioning:** Version-specific documentation loading

---

## ✅ Validation

- **JSON Syntax:** ✅ Valid
- **Linter Errors:** ✅ None
- **File Structure:** ✅ Correct
- **Endpoint Logic:** ✅ Functional
- **Error Handling:** ✅ Robust

---

## 📝 Migration Notes

If you need to add new documentation pages:

1. Create a new `.md` file in `api-doc/` directory
2. Add a reference in `index.json` under `pages`:
   ```json
   "new-page-id": {
     "contentPath": "api-doc/new-page.md"
   }
   ```
3. The `/docs` endpoint will automatically load and inject the content

---

## 🔗 Related Files

- `infrastructure/ginServer.go` - Server implementation with `/docs` endpoint
- `api-doc/index.json` - Main documentation structure
- `api-doc/*.md` - Documentation content files

---

## Notes

- Markdown files use frontmatter format (YAML header with `---`)
- File paths in JSON are relative to project root
- Response format is unchanged (backward compatible)
- Client applications require no changes

