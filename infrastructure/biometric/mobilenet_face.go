package biometric

import (
	"fmt"
	"image"
	"os"
	"sync"
	"time"

	"gateman.io/infrastructure/logger"
	"gocv.io/x/gocv"
)

// MobileNetFaceService provides face detection using MobileNet-SSD
type MobileNetFaceService struct {
	net                 gocv.Net
	inputSize           image.Point
	confidenceThreshold float32
	nmsThreshold        float32
	modelsLoaded        bool
	processingStats     ProcessingStats
	mutex               sync.RWMutex
}

// MobileNetConfig holds configuration for MobileNet service
type MobileNetConfig struct {
	ModelPath           string
	ConfigPath          string
	InputSize           image.Point
	ConfidenceThreshold float32
	NMSThreshold        float32
	Backend             gocv.NetBackendType
	Target              gocv.NetTargetType
}

// MobileNetDetectionResult holds detection results
type MobileNetDetectionResult struct {
	Faces          []image.Rectangle
	Confidences    []float32
	ProcessingTime time.Duration
	Method         string
}

// NewMobileNetFaceService creates a new MobileNet face service
func NewMobileNetFaceService(config MobileNetConfig) *MobileNetFaceService {
	service := &MobileNetFaceService{
		inputSize:           config.InputSize,
		confidenceThreshold: config.ConfidenceThreshold,
		nmsThreshold:        config.NMSThreshold,
		processingStats:     ProcessingStats{},
	}

	if err := service.loadModel(config); err != nil {
		logger.Error("Failed to load MobileNet model", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return service
	}

	service.modelsLoaded = true
	logger.Info("MobileNet face service initialized successfully")
	return service
}

// loadModel loads the MobileNet model
func (mfs *MobileNetFaceService) loadModel(config MobileNetConfig) error {
	// Check if model files exist
	if _, err := os.Stat(config.ModelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file not found: %s", config.ModelPath)
	}
	if _, err := os.Stat(config.ConfigPath); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", config.ConfigPath)
	}

	// Load the pre-trained model using generic ReadNet (supports both .pb and .pbtxt)
	mfs.net = gocv.ReadNet(config.ModelPath, config.ConfigPath)

	// Check if network is empty after loading
	if mfs.net.Empty() {
		return fmt.Errorf("failed to load MobileNet model from %s and %s", config.ModelPath, config.ConfigPath)
	}

	// Set backend and target for CPU optimization
	mfs.net.SetPreferableBackend(config.Backend)
	mfs.net.SetPreferableTarget(config.Target)

	// Verify the network is still ready after configuration
	if mfs.net.Empty() {
		return fmt.Errorf("MobileNet model became empty after configuration")
	}

	// Note: EnableFusion is not available in this version of gocv
	// CPU optimizations are handled by backend and target settings

	logger.Info("MobileNet model loaded successfully", logger.LoggerOptions{
		Key: "model_info",
		Data: map[string]interface{}{
			"model_path":  config.ModelPath,
			"config_path": config.ConfigPath,
			"input_size":  fmt.Sprintf("%dx%d", config.InputSize.X, config.InputSize.Y),
			"backend":     config.Backend.String(),
			"target":      config.Target.String(),
		},
	})

	return nil
}

// DetectFaces performs face detection using MobileNet
func (mfs *MobileNetFaceService) DetectFaces(img gocv.Mat) (*MobileNetDetectionResult, error) {
	startTime := time.Now()

	if !mfs.modelsLoaded {
		return nil, fmt.Errorf("MobileNet model not loaded")
	}

	if mfs.net.Empty() {
		return nil, fmt.Errorf("MobileNet network is empty")
	}

	mfs.mutex.Lock()
	defer mfs.mutex.Unlock()

	// Preprocess image
	blob := gocv.BlobFromImage(img, 1.0, mfs.inputSize, gocv.NewScalar(104, 117, 123, 0), false, false)
	defer blob.Close()

	// Set input
	mfs.net.SetInput(blob, "")

	// Run inference
	detections := mfs.net.Forward("")
	defer detections.Close()

	// Parse detections
	faces, confidences := mfs.parseDetections(detections, img)

	// Apply Non-Maximum Suppression
	indices := gocv.NMSBoxes(faces, confidences, mfs.confidenceThreshold, mfs.nmsThreshold)

	var finalFaces []image.Rectangle
	var finalConfidences []float32

	for _, idx := range indices {
		finalFaces = append(finalFaces, faces[idx])
		finalConfidences = append(finalConfidences, confidences[idx])
	}

	processingTime := time.Since(startTime)

	// Update statistics
	mfs.updateStats(processingTime, len(finalFaces) > 0)

	result := &MobileNetDetectionResult{
		Faces:          finalFaces,
		Confidences:    finalConfidences,
		ProcessingTime: processingTime,
		Method:         "MobileNet-SSD",
	}

	logger.Info("MobileNet face detection completed", logger.LoggerOptions{
		Key: "detection_result",
		Data: map[string]interface{}{
			"faces_detected":     len(finalFaces),
			"processing_time_ms": processingTime.Milliseconds(),
			"max_confidence": func() float32 {
				if len(finalConfidences) > 0 {
					max := finalConfidences[0]
					for _, conf := range finalConfidences {
						if conf > max {
							max = conf
						}
					}
					return max
				}
				return 0
			}(),
		},
	})

	return result, nil
}

// parseDetections parses the detection results from the neural network
func (mfs *MobileNetFaceService) parseDetections(detections gocv.Mat, img gocv.Mat) ([]image.Rectangle, []float32) {
	var faces []image.Rectangle
	var confidences []float32

	for i := 0; i < detections.Rows(); i++ {
		confidence := detections.GetFloatAt(i, 2)

		if confidence > mfs.confidenceThreshold {
			x1 := int(detections.GetFloatAt(i, 3) * float32(img.Cols()))
			y1 := int(detections.GetFloatAt(i, 4) * float32(img.Rows()))
			x2 := int(detections.GetFloatAt(i, 5) * float32(img.Cols()))
			y2 := int(detections.GetFloatAt(i, 6) * float32(img.Rows()))

			// Ensure coordinates are within image bounds
			if x1 >= 0 && y1 >= 0 && x2 <= img.Cols() && y2 <= img.Rows() && x2 > x1 && y2 > y1 {
				face := image.Rect(x1, y1, x2, y2)
				faces = append(faces, face)
				confidences = append(confidences, confidence)
			}
		}
	}

	return faces, confidences
}

// updateStats updates processing statistics
func (mfs *MobileNetFaceService) updateStats(processingTime time.Duration, success bool) {
	mfs.processingStats.TotalRequests++
	if success {
		mfs.processingStats.SuccessfulRequests++
	}
	mfs.processingStats.TotalTime += processingTime.Milliseconds()
	mfs.processingStats.AverageTime = float64(mfs.processingStats.TotalTime) / float64(mfs.processingStats.TotalRequests)
}

// GetStats returns processing statistics
func (mfs *MobileNetFaceService) GetStats() ProcessingStats {
	mfs.mutex.RLock()
	defer mfs.mutex.RUnlock()
	return mfs.processingStats
}

// IsHealthy checks if the service is healthy
func (mfs *MobileNetFaceService) IsHealthy() bool {
	return mfs.modelsLoaded && !mfs.net.Empty()
}

// Close releases resources
func (mfs *MobileNetFaceService) Close() {
	if !mfs.net.Empty() {
		mfs.net.Close()
	}
}

// GetDefaultConfig returns default MobileNet configuration
func GetDefaultMobileNetConfig() MobileNetConfig {
	return MobileNetConfig{
		ModelPath:           "./models/mobilenet/opencv_face_detector_uint8.pb",
		ConfigPath:          "./models/mobilenet/opencv_face_detector.pbtxt",
		InputSize:           image.Pt(300, 300),
		ConfidenceThreshold: 0.5,
		NMSThreshold:        0.4,
		Backend:             gocv.NetBackendOpenCV,
		Target:              gocv.NetTargetCPU,
	}
}
