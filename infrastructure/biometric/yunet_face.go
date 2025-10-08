package biometric

import (
	"fmt"
	"image"
	"math"
	"os"
	"sync"
	"time"

	"gateman.io/infrastructure/logger"
	"gocv.io/x/gocv"
)

// YuNetFaceService provides face detection using YuNet
type YuNetFaceService struct {
	detector            gocv.FaceDetectorYN
	inputSize           image.Point
	confidenceThreshold float32
	nmsThreshold        float32
	topK                int
	modelsLoaded        bool
	processingStats     ProcessingStats
	mutex               sync.RWMutex
}

// YuNetConfig holds configuration for YuNet service
type YuNetConfig struct {
	ModelPath           string
	InputSize           image.Point
	ConfidenceThreshold float32
	NMSThreshold        float32
	TopK                int
	Backend             gocv.NetBackendType
	Target              gocv.NetTargetType
}

// YuNetDetectionResult holds detection results with landmarks
type YuNetDetectionResult struct {
	Faces          []image.Rectangle
	Confidences    []float32
	Landmarks      [][]image.Point // 5 landmarks per face: right_eye, left_eye, nose, right_mouth, left_mouth
	ProcessingTime time.Duration
	Method         string
}

// NewYuNetFaceService creates a new YuNet face service
func NewYuNetFaceService(config YuNetConfig) *YuNetFaceService {
	service := &YuNetFaceService{
		inputSize:           config.InputSize,
		confidenceThreshold: config.ConfidenceThreshold,
		nmsThreshold:        config.NMSThreshold,
		topK:                config.TopK,
		processingStats:     ProcessingStats{},
	}

	if err := service.loadModel(config); err != nil {
		logger.Error("Failed to load YuNet model", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return service
	}

	service.modelsLoaded = true
	logger.Info("YuNet face service initialized successfully")
	return service
}

// loadModel loads the YuNet model
func (yfs *YuNetFaceService) loadModel(config YuNetConfig) error {
	// Check if model file exists
	if _, err := os.Stat(config.ModelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file not found: %s", config.ModelPath)
	}

	// Create YuNet face detector with the correct API signature
	detector := gocv.NewFaceDetectorYN(
		config.ModelPath,
		"",
		image.Pt(config.InputSize.X, config.InputSize.Y),
	)

	// Set score and NMS thresholds
	detector.SetScoreThreshold(config.ConfidenceThreshold)
	detector.SetNMSThreshold(config.NMSThreshold)
	detector.SetTopK(config.TopK)

	yfs.detector = detector

	logger.Info("YuNet model loaded successfully", logger.LoggerOptions{
		Key: "model_info",
		Data: map[string]interface{}{
			"model_path":           config.ModelPath,
			"input_size":           fmt.Sprintf("%dx%d", config.InputSize.X, config.InputSize.Y),
			"confidence_threshold": config.ConfidenceThreshold,
			"nms_threshold":        config.NMSThreshold,
			"top_k":                config.TopK,
		},
	})

	return nil
}

// DetectFaces performs face detection using YuNet
func (yfs *YuNetFaceService) DetectFaces(img gocv.Mat) (*YuNetDetectionResult, error) {
	startTime := time.Now()

	if !yfs.modelsLoaded {
		return nil, fmt.Errorf("YuNet model not loaded")
	}

	yfs.mutex.Lock()
	defer yfs.mutex.Unlock()

	// Set input size to match image dimensions
	imgSize := image.Pt(img.Cols(), img.Rows())
	yfs.detector.SetInputSize(imgSize)

	// Detect faces
	facesMat := gocv.NewMat()
	defer facesMat.Close()

	yfs.detector.Detect(img, &facesMat)

	// Parse detection results
	faces, confidences, landmarks := yfs.parseDetections(facesMat, img)

	processingTime := time.Since(startTime)

	// Update statistics
	yfs.updateStats(processingTime, len(faces) > 0)

	result := &YuNetDetectionResult{
		Faces:          faces,
		Confidences:    confidences,
		Landmarks:      landmarks,
		ProcessingTime: processingTime,
		Method:         "YuNet",
	}

	logger.Info("YuNet face detection completed", logger.LoggerOptions{
		Key: "detection_result",
		Data: map[string]interface{}{
			"faces_detected":     len(faces),
			"processing_time_ms": processingTime.Milliseconds(),
			"max_confidence": func() float32 {
				if len(confidences) > 0 {
					max := confidences[0]
					for _, conf := range confidences {
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

// parseDetections parses the detection results from YuNet
// YuNet output format: [x, y, w, h, x_re, y_re, x_le, y_le, x_nt, y_nt, x_rcm, y_rcm, x_lcm, y_lcm, score]
// where re=right_eye, le=left_eye, nt=nose_tip, rcm=right_corner_mouth, lcm=left_corner_mouth
func (yfs *YuNetFaceService) parseDetections(facesMat gocv.Mat, img gocv.Mat) ([]image.Rectangle, []float32, [][]image.Point) {
	var faces []image.Rectangle
	var confidences []float32
	var landmarks [][]image.Point

	if facesMat.Empty() || facesMat.Rows() == 0 {
		return faces, confidences, landmarks
	}

	// Each row contains: x, y, w, h, 5 landmarks (x,y pairs), confidence
	// Total: 4 + 10 + 1 = 15 values
	for i := 0; i < facesMat.Rows(); i++ {
		// Get bounding box
		x := int(facesMat.GetFloatAt(i, 0))
		y := int(facesMat.GetFloatAt(i, 1))
		w := int(facesMat.GetFloatAt(i, 2))
		h := int(facesMat.GetFloatAt(i, 3))

		// Get confidence score (last column)
		confidence := facesMat.GetFloatAt(i, 14)

		// Ensure coordinates are within image bounds
		if x >= 0 && y >= 0 && x+w <= img.Cols() && y+h <= img.Rows() && w > 0 && h > 0 {
			face := image.Rect(x, y, x+w, y+h)
			faces = append(faces, face)
			confidences = append(confidences, confidence)

			// Extract 5 facial landmarks
			faceLandmarks := []image.Point{
				{X: int(facesMat.GetFloatAt(i, 4)), Y: int(facesMat.GetFloatAt(i, 5))},   // right eye
				{X: int(facesMat.GetFloatAt(i, 6)), Y: int(facesMat.GetFloatAt(i, 7))},   // left eye
				{X: int(facesMat.GetFloatAt(i, 8)), Y: int(facesMat.GetFloatAt(i, 9))},   // nose tip
				{X: int(facesMat.GetFloatAt(i, 10)), Y: int(facesMat.GetFloatAt(i, 11))}, // right mouth corner
				{X: int(facesMat.GetFloatAt(i, 12)), Y: int(facesMat.GetFloatAt(i, 13))}, // left mouth corner
			}
			landmarks = append(landmarks, faceLandmarks)
		}
	}

	return faces, confidences, landmarks
}

// CalculateFaceQuality calculates face quality based on landmarks and face size
func (yfs *YuNetFaceService) CalculateFaceQuality(face image.Rectangle, landmarks []image.Point, imgSize image.Point) float64 {
	quality := 0.0

	// 1. Face size quality (30% weight)
	faceArea := float64(face.Dx() * face.Dy())
	imageArea := float64(imgSize.X * imgSize.Y)
	sizeRatio := faceArea / imageArea

	// Optimal face size is 15-40% of image
	var sizeScore float64
	if sizeRatio >= 0.15 && sizeRatio <= 0.40 {
		sizeScore = 1.0
	} else if sizeRatio < 0.15 {
		sizeScore = sizeRatio / 0.15
	} else {
		sizeScore = math.Max(0, 1.0-(sizeRatio-0.40)/0.30)
	}
	quality += sizeScore * 0.30

	// 2. Face position quality (20% weight)
	centerX := float64(face.Min.X + face.Dx()/2)
	centerY := float64(face.Min.Y + face.Dy()/2)
	imgCenterX := float64(imgSize.X) / 2
	imgCenterY := float64(imgSize.Y) / 2

	distFromCenter := math.Sqrt(math.Pow(centerX-imgCenterX, 2) + math.Pow(centerY-imgCenterY, 2))
	maxDist := math.Sqrt(math.Pow(float64(imgSize.X)/2, 2) + math.Pow(float64(imgSize.Y)/2, 2))
	positionScore := 1.0 - (distFromCenter / maxDist)
	quality += positionScore * 0.20

	// 3. Landmark symmetry quality (30% weight)
	if len(landmarks) >= 5 {
		// Calculate eye distance
		eyeDistance := math.Sqrt(
			math.Pow(float64(landmarks[1].X-landmarks[0].X), 2) +
				math.Pow(float64(landmarks[1].Y-landmarks[0].Y), 2),
		)

		// Calculate mouth distance
		mouthDistance := math.Sqrt(
			math.Pow(float64(landmarks[4].X-landmarks[3].X), 2) +
				math.Pow(float64(landmarks[4].Y-landmarks[3].Y), 2),
		)

		// Eye-mouth ratio should be around 1.5-2.0 for frontal faces
		if eyeDistance > 0 {
			ratio := mouthDistance / eyeDistance
			var symmetryScore float64
			if ratio >= 0.8 && ratio <= 1.2 {
				symmetryScore = 1.0
			} else {
				symmetryScore = math.Max(0, 1.0-math.Abs(ratio-1.0))
			}
			quality += symmetryScore * 0.30
		}
	} else {
		// No landmarks available, use default
		quality += 0.15 // 50% of landmark weight
	}

	// 4. Face aspect ratio quality (20% weight)
	aspectRatio := float64(face.Dx()) / float64(face.Dy())
	// Ideal face aspect ratio is around 0.75-0.85
	var aspectScore float64
	if aspectRatio >= 0.70 && aspectRatio <= 0.90 {
		aspectScore = 1.0
	} else {
		aspectScore = math.Max(0, 1.0-math.Abs(aspectRatio-0.80)/0.30)
	}
	quality += aspectScore * 0.20

	return math.Min(1.0, math.Max(0.0, quality))
}

// updateStats updates processing statistics
func (yfs *YuNetFaceService) updateStats(processingTime time.Duration, success bool) {
	yfs.processingStats.TotalRequests++
	if success {
		yfs.processingStats.SuccessfulRequests++
	}
	yfs.processingStats.TotalTime += processingTime.Milliseconds()
	yfs.processingStats.AverageTime = float64(yfs.processingStats.TotalTime) / float64(yfs.processingStats.TotalRequests)
}

// GetStats returns processing statistics
func (yfs *YuNetFaceService) GetStats() ProcessingStats {
	yfs.mutex.RLock()
	defer yfs.mutex.RUnlock()
	return yfs.processingStats
}

// IsHealthy checks if the service is healthy
func (yfs *YuNetFaceService) IsHealthy() bool {
	return yfs.modelsLoaded
}

// ValidateFacialLandmarks validates that two sets of facial landmarks have similar geometry
// This helps prevent false positives by ensuring facial structure matches
func (yfs *YuNetFaceService) ValidateFacialLandmarks(
	landmarks1, landmarks2 []image.Point,
	face1, face2 image.Rectangle,
) (bool, float64) {
	if len(landmarks1) < 5 || len(landmarks2) < 5 {
		// Can't validate without landmarks, rely on embedding only
		return true, 1.0
	}

	// Calculate normalized landmark distances for both faces
	dist1 := yfs.calculateNormalizedLandmarkDistances(landmarks1, face1)
	dist2 := yfs.calculateNormalizedLandmarkDistances(landmarks2, face2)

	// Compare landmark geometry
	geometricSimilarity := yfs.compareLandmarkGeometry(dist1, dist2)

	// Landmarks should match with at least 55% similarity for same person
	// This is lenient because facial expressions, angles, and image quality can vary significantly
	// Test data shows same-person comparisons can have geometric similarity as low as 58%
	isValid := geometricSimilarity > 0.55

	return isValid, geometricSimilarity
}

// calculateNormalizedLandmarkDistances calculates normalized distances between landmarks
// Returns a map of distance ratios that are scale-invariant
func (yfs *YuNetFaceService) calculateNormalizedLandmarkDistances(
	landmarks []image.Point,
	face image.Rectangle,
) map[string]float64 {
	distances := make(map[string]float64)

	if len(landmarks) < 5 {
		return distances
	}

	// Face width for normalization
	faceWidth := float64(face.Dx())
	if faceWidth == 0 {
		faceWidth = 1.0
	}

	// Calculate key distances (normalized by face width)
	// Landmarks: [0]=right_eye, [1]=left_eye, [2]=nose, [3]=right_mouth, [4]=left_mouth

	// Eye distance (inter-ocular distance)
	eyeDist := math.Sqrt(
		math.Pow(float64(landmarks[1].X-landmarks[0].X), 2) +
			math.Pow(float64(landmarks[1].Y-landmarks[0].Y), 2),
	)
	distances["eye_distance"] = eyeDist / faceWidth

	// Nose to right eye
	noseRightEyeDist := math.Sqrt(
		math.Pow(float64(landmarks[2].X-landmarks[0].X), 2) +
			math.Pow(float64(landmarks[2].Y-landmarks[0].Y), 2),
	)
	distances["nose_right_eye"] = noseRightEyeDist / faceWidth

	// Nose to left eye
	noseLeftEyeDist := math.Sqrt(
		math.Pow(float64(landmarks[2].X-landmarks[1].X), 2) +
			math.Pow(float64(landmarks[2].Y-landmarks[1].Y), 2),
	)
	distances["nose_left_eye"] = noseLeftEyeDist / faceWidth

	// Mouth width
	mouthWidth := math.Sqrt(
		math.Pow(float64(landmarks[4].X-landmarks[3].X), 2) +
			math.Pow(float64(landmarks[4].Y-landmarks[3].Y), 2),
	)
	distances["mouth_width"] = mouthWidth / faceWidth

	// Nose to mouth center
	mouthCenterX := float64(landmarks[3].X+landmarks[4].X) / 2.0
	mouthCenterY := float64(landmarks[3].Y+landmarks[4].Y) / 2.0
	noseMouthDist := math.Sqrt(
		math.Pow(float64(landmarks[2].X)-mouthCenterX, 2) +
			math.Pow(float64(landmarks[2].Y)-mouthCenterY, 2),
	)
	distances["nose_mouth"] = noseMouthDist / faceWidth

	// Eye center to mouth center (vertical face length indicator)
	eyeCenterX := float64(landmarks[0].X+landmarks[1].X) / 2.0
	eyeCenterY := float64(landmarks[0].Y+landmarks[1].Y) / 2.0
	eyeMouthDist := math.Sqrt(
		math.Pow(eyeCenterX-mouthCenterX, 2) +
			math.Pow(eyeCenterY-mouthCenterY, 2),
	)
	distances["eye_mouth"] = eyeMouthDist / faceWidth

	return distances
}

// compareLandmarkGeometry compares two sets of normalized landmark distances
// Returns a similarity score between 0 and 1
func (yfs *YuNetFaceService) compareLandmarkGeometry(dist1, dist2 map[string]float64) float64 {
	if len(dist1) == 0 || len(dist2) == 0 {
		return 1.0 // Can't compare, assume valid
	}

	totalSimilarity := 0.0
	count := 0

	// Compare each distance ratio
	for key, val1 := range dist1 {
		if val2, exists := dist2[key]; exists {
			// Calculate relative difference
			diff := math.Abs(val1 - val2)
			avgVal := (val1 + val2) / 2.0

			if avgVal > 0 {
				// Similarity = 1 - (relative difference)
				// Allow up to 30% difference for same person (increased from 20% for more tolerance)
				// Different angles, expressions, and lighting can cause variations
				relativeDiff := diff / avgVal
				similarity := math.Max(0, 1.0-relativeDiff/0.30)
				totalSimilarity += similarity
				count++
			}
		}
	}

	if count == 0 {
		return 1.0
	}

	return totalSimilarity / float64(count)
}

// Close releases resources
func (yfs *YuNetFaceService) Close() {
	if yfs.modelsLoaded {
		yfs.detector.Close()
	}
}

// GetDefaultYuNetConfig returns default YuNet configuration
func GetDefaultYuNetConfig() YuNetConfig {
	return YuNetConfig{
		ModelPath:           "./models/yunet/face_detection_yunet_2023mar.onnx",
		InputSize:           image.Pt(320, 320),
		ConfidenceThreshold: 0.6,
		NMSThreshold:        0.3,
		TopK:                5000,
		Backend:             gocv.NetBackendDefault,
		Target:              gocv.NetTargetCPU,
	}
}
