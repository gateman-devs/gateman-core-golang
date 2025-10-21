# Silent-Face-Anti-Spoofing Model Integration Issue

**Date:** October 21, 2025  
**Status:** ❌ Blocked - ONNX Model Incompatibility

## 🔴 Problem

Attempted to integrate Silent-Face-Anti-Spoofing ONNX models but encountered **segmentation fault** during model loading with GoCV.

### Error Details
```
SIGSEGV: segmentation violation
PC=0x106f91474 m=0 sigcode=2 addr=0x0
signal arrived during cgo execution

gateman.io/infrastructure/biometric.NewSilentFaceAntiSpoof()
  /infrastructure/biometric/silent_face_antispoof.go:26
gocv.io/x/gocv.(*Net).Empty(0x10049f675?)
  /gocv@v0.42.0/dnn.go:271
```

### Root Cause
- **OpenCV DNN Module**: GoCV's OpenCV DNN module may not have full ONNX runtime support
- **Model Format**: Silent-Face-Anti-Spoofing models (MiniFASNetV1SE, MiniFASNetV2) use ONNX format with specific layer types that may not be supported
- **GoCV Version**: gocv@v0.42.0 may need to be built with specific ONNX flags

## 🛠️ Attempted Solutions

### 1. Used `gocv.ReadNet` instead of `gocv.ReadNetFromONNX`
**Result**: Still crashes - `gocv.ReadNet` returns null pointer

### 2. Checked model files
**Result**: Files downloaded successfully (287KB each), but incompatible with OpenCV DNN

### 3. Error handling
**Result**: Crash happens before error handling can execute

##Alternative Solutions

###  **Option 1: Rebuild OpenCV with ONNX Runtime** ⚠️
```bash
# Rebuild OpenCV with ONNX support
cmake -D WITH_PROTOBUF=ON \
      -D WITH_ONNX=ON \
      -D BUILD_opencv_dnn=ON \
      -D DNN_ENABLE_INLINE_NEON=OFF \
      ...
```
**Pros**: Would enable ONNX models  
**Cons**: Complex, requires full OpenCV rebuild, may break existing functionality

### ✅ **Option 2: Use Python Microservice** (Recommended Short-term)
Create a lightweight Python service that runs Silent-Face-Anti-Spoofing and expose it via REST API or gRPC.

```python
# Python microservice
from silent_face import AntiSpoofing
model = AntiSpoofing()

@app.post("/predict")
def predict(image: bytes):
    is_real, confidence = model.predict(image)
    return {"is_real": is_real, "confidence": confidence}
```

**Call from Go**:
```go
resp, err := http.Post("http://localhost:5000/predict", "image/jpeg", imageBytes)
```

**Pros**: Works immediately, uses official Silent-Face models  
**Cons**: Extra service dependency, network latency

### ✅ **Option 3: Calibrate Hand-Crafted Rules** (Recommended for Now)
Instead of ML models, carefully tune our existing texture-based detection with real-world test data.

```go
// Adjust thresholds based on empirical testing
paintingThreshold := 0.45  // From testing real vs fake
textureThreshold := 0.25   // Based on legit image analysis
edgeThreshold := 0.30      // Calibrated from dataset
```

**Pros**: No dependencies, works now, fully controllable  
**Cons**: May not be as accurate as ML models

### **Option 4: Use TensorFlow Lite** 
Convert ONNX models to TensorFlow Lite and use Go TFLite bindings.

**Pros**: Better Go support than ONNX  
**Cons**: Complex conversion process, another dependency

### **Option 5: Cloud API Integration**
Use AWS Rekognition, Azure Face, or Google Vision liveness detection.

```go
result, err := rekognition.DetectFaces(&rekognitioninput{
    Image: &types.Image{Bytes: imageBytes},
    Attributes: []types.Attribute{"FACE_LIVENESS"},
})
```

**Pros**: Proven accuracy, no model management  
**Cons**: Cost per API call, privacy concerns, internet dependency

## 📊 Recommended Approach

### Phase 1: Immediate (Today)
**Calibrate existing hand-crafted rules** with real test images:

1. Collect test dataset (50 real + 50 fake images)
2. Run through current system
3. Analyze metrics that distinguish real vs fake
4. Adjust thresholds to minimize false positives while catching spoofs
5. Document optimal thresholds

### Phase 2: Short-term (This Week)
**Deploy Python microservice**:

1. Create Docker container with Silent-Face-Anti-Spoofing
2. Expose REST API endpoint
3. Integrate as optional enhancement to hand-crafted rules
4. Use hybrid scoring: `finalScore = (mlScore * 0.7) + (heuristicScore * 0.3)`

### Phase 3: Long-term (Next Month)
**Train custom model**:

1. Collect production data (with user consent)
2. Train lightweight CNN on specific attack types
3. Export to TensorFlow Lite
4. Integrate natively with Go TFLite bindings

## 🎯 Current Action Plan

**Immediate Fix**: Adjust aggressive thresholds based on the user's feedback that legit images are failing.

The painting image was correctly rejected, but legitimate images are also failing. This means:
- ✅ Painting detection works
- ❌ Thresholds too aggressive for real images

**Solution**: Relax thresholds slightly while maintaining painting detection:

```go
// Controller adjustments
paintingThreshold: 0.4 → 0.45      // Slight increase
riskFactorThreshold: 2 → 3          // Back to 3
textureThreshold: 0.2 → 0.18        // Slightly lower
paintingRiskWeight: 2 → 1.5         // Reduce double-counting
```

This should:
- Still reject paintings (they have painting_prob > 0.45)
- Allow legitimate photos (they have painting_prob < 0.3)
- Reduce false positives on real faces

## 📝 Summary

- **Silent-Face ONNX models**: Incompatible with current GoCV/OpenCV setup
- **Immediate solution**: Calibrate hand-crafted thresholds  
- **Short-term**: Python microservice for ML-based detection
- **Long-term**: Custom model + TFLite integration

**Current Status**: Proceeding with threshold calibration to fix false positives on legitimate images.

