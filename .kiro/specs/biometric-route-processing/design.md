# Design Document

## Overview

This design implements the actual processing logic in the production liveness routes by connecting the HTTP endpoint handlers to the existing processing functions. The current implementation has all the processing functions defined but they are not being called from the route handlers.

## Architecture

### Current State Analysis

The current route implementation has:
1. **Route handlers defined** with proper validation and request binding
2. **Processing functions implemented** with full biometric processing logic
3. **Missing connection** between route handlers and processing functions
4. **Incomplete error handling** in route validation sections

### Target Architecture

```
HTTP Request → Route Handler → Validation → Processing Function → Response
```

The design will:
1. Connect route handlers to their corresponding processing functions
2. Add proper error handling for validation failures
3. Ensure consistent response formatting
4. Add proper middleware integration

## Components and Interfaces

### 1. Route Handler Enhancement

Each route handler needs to:
- Get the application context from Gin context
- Handle validation errors properly with structured responses
- Call the appropriate processing function
- Use consistent error response format

### 2. Error Handling Enhancement

Current validation functions return errors but don't handle them properly. The design will:
- Create a centralized error handling function
- Ensure all validation errors return proper HTTP responses
- Use consistent error codes and messages

### 3. Processing Function Integration

The existing processing functions need to be called with proper parameters:
- Extract application context from Gin context
- Pass validated request bodies to processing functions
- Handle any additional context requirements

### 4. Response Consistency

Ensure all endpoints use the same response format through the serverResponse.Responder.

## Data Models

### Request Flow Data Structure

```go
type RequestContext struct {
    GinContext     *gin.Context
    AppContext     *interfaces.ApplicationContext[any]
    RequestID      string
    StartTime      time.Time
}
```

### Error Response Structure

```go
type ValidationErrorResponse struct {
    Error     string    `json:"error"`
    Message   string    `json:"message"`
    Details   string    `json:"details,omitempty"`
    RequestID string    `json:"request_id,omitempty"`
    Timestamp time.Time `json:"timestamp"`
}
```

## Implementation Details

### 1. Route Handler Updates

Each route handler will follow this pattern:

```go
livenessRouter.POST("/detect", func(ctx *gin.Context) {
    // Get application context
    appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
    
    // Bind and validate request
    var body ProductionLivenessRequest
    if err := bindAndValidateProductionRequest(ctx, &body); err != nil {
        handleValidationError(ctx, "Request binding failed", err)
        return
    }
    
    // Validate image input
    if err := validateProductionImageInput(body.Image); err != nil {
        handleValidationError(ctx, "Image validation failed", err)
        return
    }
    
    // Process the request
    processProductionLivenessDetection(ctx, appContext, body)
})
```

### 2. Centralized Error Handling

Create a unified error handling function:

```go
func handleValidationError(ctx *gin.Context, message string, err error) {
    requestID := generateRequestID("error")
    
    serverResponse.Responder.Respond(
        ctx,
        http.StatusBadRequest,
        message,
        ValidationErrorResponse{
            Error:     "VALIDATION_ERROR",
            Message:   err.Error(),
            RequestID: requestID,
            Timestamp: time.Now(),
        },
        nil,
        nil,
        nil,
    )
}
```

### 3. Middleware Integration

The routes will need to ensure proper middleware is applied:
- Application context middleware
- Request validation middleware
- Audit logging middleware

### 4. Processing Function Calls

Each endpoint will call its corresponding processing function:

- `/detect` → `processProductionLivenessDetection`
- `/detect/verbose` → `processProductionLivenessDetection` (with verbose=true)
- `/compare` → `processProductionFaceComparison`
- `/quality` → `processProductionImageQuality`
- `/batch/detect` → `processProductionBatchLiveness`

## Error Handling

### Validation Error Flow

1. **Request Binding Errors**: Invalid JSON, missing required fields
2. **Image Validation Errors**: Invalid format, size limits, security checks
3. **Business Logic Errors**: Threshold validation, batch size limits
4. **Service Errors**: FaceMatcher not initialized, processing failures

### Error Response Format

All validation errors will use consistent format:
```json
{
  "error": "VALIDATION_ERROR",
  "message": "Specific error description",
  "details": "Additional context if available",
  "request_id": "generated_or_provided_id",
  "timestamp": "2025-01-27T10:30:00Z"
}
```

## Testing Strategy

### Unit Testing
- Test each route handler with valid and invalid inputs
- Test error handling paths
- Test processing function integration

### Integration Testing
- Test complete request/response flow
- Test with actual biometric processing
- Test error scenarios with FaceMatcher

### Performance Testing
- Test processing times under load
- Test batch processing performance
- Test concurrent request handling

## Security Considerations

### Input Validation
- Comprehensive image input validation
- URL security checks (no localhost)
- Base64 format validation
- Size limit enforcement

### Error Information Disclosure
- Avoid exposing internal system details in error messages
- Use generic error codes for external responses
- Log detailed errors internally for debugging

### Request Tracking
- Generate unique request IDs for all requests
- Include request IDs in all responses
- Use request IDs for audit logging

## Performance Considerations

### Processing Optimization
- Reuse existing processing functions without modification
- Maintain concurrent processing capabilities
- Preserve existing performance characteristics

### Memory Management
- Proper cleanup of image processing resources
- Efficient batch processing implementation
- Avoid memory leaks in error paths

### Response Time
- Maintain current processing times
- Add processing time metrics to responses
- Optimize error response paths

## Deployment Considerations

### Backward Compatibility
- Maintain existing API contract
- Preserve response formats
- Keep existing error codes

### Configuration
- Use existing environment variable patterns
- Maintain development vs production behavior
- Preserve existing security settings

### Monitoring
- Maintain existing health check endpoints
- Preserve metrics collection
- Keep audit logging functionality