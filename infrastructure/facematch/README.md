# Production-Ready Anti-Spoofing System

## Overview

This anti-spoofing system is designed for production use with enhanced security, reliability, and ease of use. It detects whether a face image is from a real person or a spoof attempt (photo, screen, video, etc.).

## Key Features

✅ **Production Ready**: Robust error handling, proper validation, and comprehensive logging  
✅ **Simple API**: Single method call with clear results  
✅ **Multiple Input Types**: Supports both URLs and base64 encoded images  
✅ **Rule-Based Detection**: Advanced image analysis without requiring external model files  
✅ **Detailed Feedback**: Specific reasons for spoof detection  
✅ **Performance Optimized**: Fast processing with proper resource management  

## Quick Start

### 1. Initialize the System

```go
import "your-project/infrastructure/facematch"

// Initialize the face matcher service
err := facematch.InitializeFaceMatcherService()
if err != nil {
    log.Fatalf("Failed to initialize: %v", err)
}
```

### 2. Detect Anti-Spoofing

```go
// Check a URL image
result := facematch.GlobalFaceMatcher.DetectAntiSpoof("https://example.com/image.jpg")

// Check a base64 image
result := facematch.GlobalFaceMatcher.DetectAntiSpoof("data:image/jpeg;base64,/9j/4AAQ...")

// Check the result
if result.Error != "" {
    log.Printf("Error: %s", result.Error)
} else if result.IsReal && result.Confidence > 0.8 {
    fmt.Println("✅ Real face detected")
} else {
    fmt.Println("❌ Potential spoof detected")
}
```

## API Reference

### Main Function

#### `DetectAntiSpoof(input string) AntiSpoofResult`

Performs anti-spoofing detection on a single image.

**Parameters:**
- `input`: Image URL (http/https) or base64 encoded image string

**Returns:** `AntiSpoofResult` structure with the following fields:

```go
type AntiSpoofResult struct {
    IsReal       bool     `json:"is_real"`         // True if image appears to be a real face
    SpoofScore   float64  `json:"spoof_score"`     // 0.0-1.0, higher = more likely spoof
    Confidence   float64  `json:"confidence"`      // 0.0-1.0, confidence in prediction
    HasFace      bool     `json:"has_face"`        // Whether a face was detected
    ProcessTime  int64    `json:"process_time_ms"` // Processing time in milliseconds
    SpoofReasons []string `json:"spoof_reasons"`   // Specific reasons if spoof detected
    Error        string   `json:"error"`           // Error message if any
}
```

### Helper Function

#### `ProductionAntiSpoofCheck(imageInput string) (bool, string)`

Production-ready wrapper that returns simple pass/fail with explanation.

**Returns:**
- `bool`: true if image passes anti-spoofing check
- `string`: explanation message

## Detection Methods

The system uses multiple rule-based detection methods:

### 1. **Sharpness Analysis**
- Detects blurry images that often indicate screen captures
- Uses Laplacian variance to measure image sharpness

### 2. **Color Distribution Analysis**  
- Analyzes color variance across RGB channels
- Screens/photos often have limited color ranges

### 3. **Texture Complexity**
- Real faces have complex skin textures
- Uses simplified Local Binary Pattern analysis

### 4. **Edge Density Analysis**
- Analyzes edge patterns using Canny detection
- Photos/screens have different edge characteristics

### 5. **Size Validation**
- Very small faces might indicate distant screens
- Validates minimum face size requirements

## Production Usage Patterns

### Basic Usage
```go
result := GlobalFaceMatcher.DetectAntiSpoof(imageInput)
if result.IsReal && result.Confidence > 0.8 {
    // Proceed with face matching
} else {
    // Reject or request new image
}
```

### With Error Handling
```go
result := GlobalFaceMatcher.DetectAntiSpoof(imageInput)

if result.Error != "" {
    log.Printf("Anti-spoofing failed: %s", result.Error)
    return
}

if !result.HasFace {
    log.Printf("No face detected")
    return
}

if result.IsReal && result.Confidence > 0.8 {
    log.Printf("✅ Verified real face (confidence: %.3f)", result.Confidence)
} else {
    log.Printf("❌ Potential spoof detected (score: %.3f)", result.SpoofScore)
    for _, reason := range result.SpoofReasons {
        log.Printf("  - %s", reason)
    }
}
```

### Using the Helper Function
```go
passed, message := facematch.ProductionAntiSpoofCheck(imageInput)
if passed {
    fmt.Printf("✅ %s", message)
} else {
    fmt.Printf("❌ %s", message)
}
```

## Configuration

### Environment Variables

```bash
# Optional: Custom model paths
export YUNET_MODEL_PATH="./models/face_detection_yunet_2023mar.onnx"
export ARCFACE_MODEL_PATH="./models/face_recognition_sface_2021dec.onnx"
```

### Thresholds

You can adjust detection sensitivity by modifying thresholds in your application:

```go
// Conservative (fewer false positives, may miss some spoofs)
if result.IsReal && result.Confidence > 0.9 && result.SpoofScore < 0.3 {
    // Accept
}

// Balanced (recommended for production)
if result.IsReal && result.Confidence > 0.8 && result.SpoofScore < 0.5 {
    // Accept  
}

// Aggressive (catches more spoofs, may have false positives)
if result.IsReal && result.Confidence > 0.7 && result.SpoofScore < 0.4 {
    // Accept
}
```

## Performance Considerations

### Image Requirements
- **Minimum size**: 50x50 pixels
- **Maximum size**: 4000x4000 pixels  
- **Aspect ratio**: 0.3 to 3.0
- **Formats**: JPEG, PNG, and other OpenCV-supported formats

### Processing Time
- Typical processing time: 50-200ms per image
- Depends on image size and complexity
- Processing time is included in the result

### Memory Usage
- Images are properly closed after processing
- No memory leaks with proper resource management
- Concurrent processing is thread-safe

## Error Handling

The system provides detailed error messages:

```go
// Common error types
"face matcher not initialized"           // Service not initialized
"input image cannot be empty"            // Empty input
"failed to load image: ..."              // Image format/network issues
"no valid face detected: ..."            // No face found
"image too small (WxH), minimum..."      // Size validation
"image too large (WxH), maximum..."      // Size validation  
"invalid aspect ratio: X.XX"             // Distorted image
```

## Security Considerations

### Input Validation
- All inputs are validated before processing
- Maximum image size limits prevent DoS attacks
- Aspect ratio validation prevents malformed images

### Safe Defaults
- Default to "not real" for safety
- High spoof scores for uncertain cases
- Conservative confidence calculations

### Resource Management
- Automatic cleanup of OpenCV matrices
- Timeout handling for network requests
- Memory leak prevention

## Testing

```go
// Test the system
err := facematch.TestLivenessCheck()
if err != nil {
    log.Printf("System test failed: %v", err)
} else {
    log.Printf("✅ Anti-spoofing system is working")
}
```

## Migration from Old API

If you were using the old `CheckAntiSpoof` method:

```go
// Old API
result := matcher.CheckAntiSpoof(input)

// New API (same interface, enhanced functionality)
result := matcher.DetectAntiSpoof(input)
```

The new API maintains backward compatibility while adding:
- Better error handling
- More detailed spoof reasons
- Enhanced validation
- Improved performance

## Support

For issues or questions about the anti-spoofing system:

1. Check the error messages for specific guidance
2. Verify model files are present and accessible
3. Ensure input images meet the requirements
4. Review the processing time for performance issues 