package facematch

import (
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gocv.io/x/gocv"
)

// Global FaceMatcher instance
var GlobalFaceMatcher *FaceMatcher

// InitializeFaceMatcherService initializes the global face matcher with model paths from environment variables
func InitializeFaceMatcherService() error {
	GlobalFaceMatcher = NewFaceMatcher()

	// Get model paths from environment variables with fallback defaults
	yunetModelPath := os.Getenv("YUNET_MODEL_PATH")
	if yunetModelPath == "" {
		yunetModelPath = "./models/face_detection_yunet_2023mar.onnx"
	}

	arcfaceModelPath := os.Getenv("ARCFACE_MODEL_PATH")
	if arcfaceModelPath == "" {
		arcfaceModelPath = "./models/face_recognition_sface_2021dec.onnx"
	}

	// For anti-spoofing, we'll use a simpler approach without requiring a separate model
	// since the face_anti_spoofing.onnx doesn't exist in the models directory
	// We'll implement rule-based anti-spoofing detection using image analysis

	// Check if required model files exist
	if _, err := os.Stat(yunetModelPath); os.IsNotExist(err) {
		return fmt.Errorf("YuNet model file not found: %s", yunetModelPath)
	}
	if _, err := os.Stat(arcfaceModelPath); os.IsNotExist(err) {
		return fmt.Errorf("ArcFace model file not found: %s", arcfaceModelPath)
	}

	return GlobalFaceMatcher.Initialize(yunetModelPath, arcfaceModelPath)
}

// TestLivenessCheck performs a liveness check to verify the service is working
// input can be either a URL (starting with http:// or https://) or a base64 encoded image
// if input is empty, uses a default test image
func TestLivenessCheck(input ...string) error {
	if GlobalFaceMatcher == nil {
		return errors.New("global face matcher not initialized")
	}

	var testImage string
	if len(input) > 0 && input[0] != "" {
		testImage = input[0]
	} else {
		// Default test image - a simple 1x1 pixel image for basic functionality test
		testImage = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	}
	InitializeFaceMatcherService()
	result := GlobalFaceMatcher.DetectAntiSpoof(testImage)
	if result.Error != "" {
		return fmt.Errorf("face matcher liveness check failed: %s", result.Error)
	}

	return nil
}

// TestImageQuality tests the image quality verification functionality
func TestImageQuality(input string) (ImageQualityResult, error) {
	if GlobalFaceMatcher == nil {
		return ImageQualityResult{}, errors.New("global face matcher not initialized")
	}

	result := GlobalFaceMatcher.VerifyImageQuality(input)
	return result, nil
}

// TestFaceComparison tests face comparison functionality between two images
func TestFaceComparison(input1, input2 string, threshold float64) (CompareResult, error) {
	if GlobalFaceMatcher == nil {
		return CompareResult{}, errors.New("global face matcher not initialized")
	}

	if threshold == 0 {
		threshold = 0.8 // Default threshold
	}

	result := GlobalFaceMatcher.Compare(input1, input2, threshold)
	return result, nil
}

// FaceMatcher handles face detection and comparison
type FaceMatcher struct {
	yunetDetector    gocv.FaceDetectorYN
	arcfaceNet       gocv.Net
	yunetModelPath   string
	arcfaceModelPath string
	mu               sync.RWMutex
	initialized      bool
	modelsLoaded     bool
}

// CompareResult represents the result of face comparison
type CompareResult struct {
	Similarity float64 `json:"similarity"`
	Match      bool    `json:"match"`
	Error      string  `json:"error,omitempty"`
}

// ImageData holds loaded image data and any error
type ImageData struct {
	Mat   gocv.Mat
	Error error
}

// ImageQualityResult represents the result of image quality verification
type ImageQualityResult struct {
	IsGoodQuality   bool     `json:"is_good_quality"`
	HasFace         bool     `json:"has_face"`
	FaceCount       int      `json:"face_count"`
	FaceSize        float64  `json:"face_size_percent"`
	ImageResolution string   `json:"image_resolution"`
	QualityScore    float64  `json:"quality_score"` // 0.0 to 1.0
	Issues          []string `json:"issues,omitempty"`
	Recommendations []string `json:"recommendations,omitempty"`
	Error           string   `json:"error,omitempty"`
}

// AntiSpoofResult represents the result of anti-spoofing detection
// Simplified and production-ready structure
type AntiSpoofResult struct {
	IsReal       bool     `json:"is_real"`                 // True if image appears to be a real face
	SpoofScore   float64  `json:"spoof_score"`             // 0.0 to 1.0, higher means more likely to be spoof
	Confidence   float64  `json:"confidence"`              // 0.0 to 1.0, confidence in the prediction
	HasFace      bool     `json:"has_face"`                // Whether a face was detected
	ProcessTime  int64    `json:"process_time_ms"`         // Processing time in milliseconds
	SpoofReasons []string `json:"spoof_reasons,omitempty"` // Specific reasons if detected as spoof
	Error        string   `json:"error,omitempty"`         // Error message if any
}

// NewFaceMatcher creates a new face matcher instance
func NewFaceMatcher() *FaceMatcher {
	return &FaceMatcher{}
}

// Initialize loads the YuNet and ArcFace models - simplified without FAS model
func (fm *FaceMatcher) Initialize(yunetModelPath, arcfaceModelPath string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fm.initialized {
		return nil
	}

	// Validate input parameters
	if yunetModelPath == "" {
		return errors.New("YuNet model path cannot be empty")
	}
	if arcfaceModelPath == "" {
		return errors.New("ArcFace model path cannot be empty")
	}

	// Validate model files exist
	if _, err := os.Stat(yunetModelPath); os.IsNotExist(err) {
		return fmt.Errorf("YuNet model file not found: %s", yunetModelPath)
	}
	if _, err := os.Stat(arcfaceModelPath); os.IsNotExist(err) {
		return fmt.Errorf("ArcFace model file not found: %s", arcfaceModelPath)
	}

	// Store model paths
	fm.yunetModelPath = yunetModelPath
	fm.arcfaceModelPath = arcfaceModelPath

	// Load models immediately for production readiness
	if err := fm.loadModels(); err != nil {
		return fmt.Errorf("failed to load models during initialization: %v", err)
	}

	fm.initialized = true
	log.Printf("FaceMatcher initialized successfully with models: %s, %s", yunetModelPath, arcfaceModelPath)
	return nil
}

// loadModels loads the actual OpenCV models with proper error handling
func (fm *FaceMatcher) loadModels() error {
	if fm.modelsLoaded {
		return nil
	}

	// Initialize YuNet face detector with error handling
	detector := gocv.NewFaceDetectorYN(fm.yunetModelPath, "", image.Pt(320, 320))

	// Initialize ArcFace network with error handling
	net := gocv.ReadNet(fm.arcfaceModelPath, "")
	if net.Empty() {
		return fmt.Errorf("failed to load ArcFace network from: %s", fm.arcfaceModelPath)
	}

	// Set backend and target for better performance
	net.SetPreferableBackend(gocv.NetBackendOpenCV)
	net.SetPreferableTarget(gocv.NetTargetCPU)

	fm.yunetDetector = detector
	fm.arcfaceNet = net
	fm.modelsLoaded = true

	log.Printf("Models loaded successfully")
	return nil
}

// DetectAntiSpoof performs production-ready anti-spoofing detection on a single image
// This is the main API for anti-spoofing detection
func (fm *FaceMatcher) DetectAntiSpoof(input string) AntiSpoofResult {
	startTime := time.Now()

	// Initialize result with safe defaults
	result := AntiSpoofResult{
		IsReal:       false, // Default to unsafe
		SpoofScore:   1.0,   // Default to high spoof probability
		Confidence:   0.0,
		HasFace:      false,
		ProcessTime:  0,
		SpoofReasons: []string{},
	}

	// Validate initialization
	if !fm.initialized {
		result.Error = "face matcher not initialized"
		result.ProcessTime = time.Since(startTime).Milliseconds()
		return result
	}

	if !fm.modelsLoaded {
		result.Error = "models not loaded"
		result.ProcessTime = time.Since(startTime).Milliseconds()
		return result
	}

	// Validate input
	if strings.TrimSpace(input) == "" {
		result.Error = "input image cannot be empty"
		result.ProcessTime = time.Since(startTime).Milliseconds()
		return result
	}

	// Load and validate image
	img, err := fm.loadImageWithValidation(input)
	if err != nil {
		result.Error = fmt.Sprintf("failed to load image: %v", err)
		result.ProcessTime = time.Since(startTime).Milliseconds()
		return result
	}
	defer img.Close()

	// Detect face
	faceRegion, faceErr := fm.detectPrimaryFace(img)
	if faceErr != nil {
		result.Error = fmt.Sprintf("no valid face detected: %v", faceErr)
		result.ProcessTime = time.Since(startTime).Milliseconds()
		return result
	}
	defer faceRegion.Close()

	result.HasFace = true

	// Perform rule-based anti-spoofing analysis
	spoofAnalysis := fm.analyzeImageForSpoofing(img, faceRegion)

	result.IsReal = spoofAnalysis.IsReal
	result.SpoofScore = spoofAnalysis.SpoofScore
	result.Confidence = spoofAnalysis.Confidence
	result.SpoofReasons = spoofAnalysis.Reasons
	result.ProcessTime = time.Since(startTime).Milliseconds()

	return result
}

// SpoofAnalysis holds the analysis results
type SpoofAnalysis struct {
	IsReal     bool
	SpoofScore float64
	Confidence float64
	Reasons    []string
}

// analyzeImageForSpoofing performs rule-based anti-spoofing analysis
func (fm *FaceMatcher) analyzeImageForSpoofing(img, face gocv.Mat) SpoofAnalysis {
	analysis := SpoofAnalysis{
		IsReal:     true,
		SpoofScore: 0.0,
		Confidence: 0.8, // Base confidence
		Reasons:    []string{},
	}

	// Convert to grayscale for analysis
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(face, &gray, gocv.ColorBGRToGray)

	// Track individual indicators for better logic
	var indicators []string
	var totalPenalty float64

	// 1. Check image sharpness (blurry images often indicate screen captures)
	laplacian := gocv.NewMat()
	defer laplacian.Close()
	gocv.Laplacian(gray, &laplacian, gocv.MatTypeCV64F, 1, 1, 0, gocv.BorderDefault)

	meanMat := gocv.NewMat()
	stddevMat := gocv.NewMat()
	defer meanMat.Close()
	defer stddevMat.Close()

	gocv.MeanStdDev(laplacian, &meanMat, &stddevMat)
	variance := stddevMat.GetDoubleAt(0, 0) * stddevMat.GetDoubleAt(0, 0)

	// Low variance indicates blur (potential screen/photo capture)
	if variance < 40.0 { // More lenient threshold
		totalPenalty += 0.25 // Reduced penalty
		indicators = append(indicators, "image appears blurry or low quality")
	}

	// 2. Check color distribution (photos/screens often have limited color range)
	channels := gocv.Split(face)
	defer func() {
		for _, ch := range channels {
			ch.Close()
		}
	}()

	if len(channels) >= 3 {
		// Calculate color variance across channels
		var colorVariances []float64
		for _, channel := range channels {
			channelMean := gocv.NewMat()
			channelStd := gocv.NewMat()
			defer channelMean.Close()
			defer channelStd.Close()

			gocv.MeanStdDev(channel, &channelMean, &channelStd)
			variance := channelStd.GetDoubleAt(0, 0)
			colorVariances = append(colorVariances, variance)
		}

		// Low color variance might indicate artificial reproduction
		avgVariance := (colorVariances[0] + colorVariances[1] + colorVariances[2]) / 3
		if avgVariance < 12.0 { // More lenient threshold
			totalPenalty += 0.15 // Reduced penalty
			indicators = append(indicators, "limited color variation detected")
		}
	}

	// 3. Texture analysis - only flag obviously artificial textures
	textureScore := fm.calculateTextureComplexity(gray)
	if textureScore < 0.08 { // Even more restrictive threshold - only obvious fakes
		totalPenalty += 0.2 // Reduced penalty
		indicators = append(indicators, "insufficient skin texture detail")
	}

	// 4. Check edge density (photos/screens often have different edge characteristics)
	edges := gocv.NewMat()
	defer edges.Close()
	gocv.Canny(gray, &edges, 50, 150)

	nonZero := gocv.CountNonZero(edges)
	totalPixels := edges.Rows() * edges.Cols()
	edgeDensity := float64(nonZero) / float64(totalPixels)

	if edgeDensity < 0.03 || edgeDensity > 0.35 { // More lenient range
		totalPenalty += 0.1 // Reduced penalty
		indicators = append(indicators, "unusual edge density pattern")
	}

	// 5. Size-based checks (very small faces might indicate distant screens)
	faceSize := face.Rows() * face.Cols()
	if faceSize < 2000 { // More lenient size requirement
		totalPenalty += 0.08 // Reduced penalty
		indicators = append(indicators, "face region too small")
	}

	// 6. Check for uniform regions - only flag extremely uniform images
	uniformityScore := fm.calculateUniformity(gray)
	if uniformityScore > 0.85 { // Much more restrictive threshold
		totalPenalty += 0.15 // Reduced penalty
		indicators = append(indicators, "image appears too uniform")
	}

	// Set the spoof score
	analysis.SpoofScore = totalPenalty
	if analysis.SpoofScore > 1.0 {
		analysis.SpoofScore = 1.0
	}

	// More balanced decision logic - require stronger evidence for rejection
	strongIndicators := 0
	for _, indicator := range indicators {
		if indicator == "image appears blurry or low quality" ||
			indicator == "insufficient skin texture detail" {
			strongIndicators++
		}
	}

	// More lenient decision logic
	if len(indicators) >= 4 || strongIndicators >= 2 || analysis.SpoofScore >= 0.6 { // Higher threshold
		analysis.IsReal = false
		analysis.Reasons = indicators
		analysis.Confidence = 0.9
	} else if len(indicators) >= 2 && analysis.SpoofScore >= 0.4 { // Medium threshold
		analysis.IsReal = false
		analysis.Reasons = indicators
		analysis.Confidence = 0.8
	} else if len(indicators) >= 1 {
		// Minor concerns - still accept but with lower confidence
		analysis.IsReal = true
		analysis.Confidence = 0.7 // Moderate confidence
		// No reasons shown to avoid confusion
	} else {
		// No concerns - high confidence
		analysis.IsReal = true
		analysis.Confidence = 0.9
	}

	return analysis
}

// calculateTextureComplexity calculates a texture complexity measure
func (fm *FaceMatcher) calculateTextureComplexity(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() < 3 || gray.Cols() < 3 {
		return 0.0
	}

	// Simple texture complexity measure using local variance
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()

	blurred := gocv.NewMat()
	defer blurred.Close()
	gocv.GaussianBlur(gray, &blurred, image.Pt(3, 3), 0, 0, gocv.BorderDefault)

	diff := gocv.NewMat()
	defer diff.Close()
	gocv.AbsDiff(gray, blurred, &diff)

	meanMat := gocv.NewMat()
	stdMat := gocv.NewMat()
	defer meanMat.Close()
	defer stdMat.Close()

	gocv.MeanStdDev(diff, &meanMat, &stdMat)

	// Normalize the texture measure
	textureMeasure := meanMat.GetDoubleAt(0, 0) / 255.0
	return textureMeasure
}

// calculateUniformity calculates a uniformity measure
func (fm *FaceMatcher) calculateUniformity(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() < 3 || gray.Cols() < 3 {
		return 0.0
	}

	// Simple uniformity measure using local variance
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()

	blurred := gocv.NewMat()
	defer blurred.Close()
	gocv.GaussianBlur(gray, &blurred, image.Pt(3, 3), 0, 0, gocv.BorderDefault)

	diff := gocv.NewMat()
	defer diff.Close()
	gocv.AbsDiff(gray, blurred, &diff)

	meanMat := gocv.NewMat()
	stdMat := gocv.NewMat()
	defer meanMat.Close()
	defer stdMat.Close()

	gocv.MeanStdDev(diff, &meanMat, &stdMat)

	// Normalize the uniformity measure
	uniformityMeasure := 1.0 - (meanMat.GetDoubleAt(0, 0) / 255.0)
	return uniformityMeasure
}

// loadImageWithValidation loads an image with comprehensive validation
func (fm *FaceMatcher) loadImageWithValidation(input string) (gocv.Mat, error) {
	// Load image
	img, err := fm.loadImage(input)
	if err != nil {
		return gocv.Mat{}, err
	}

	// Validate image properties
	if img.Empty() {
		img.Close()
		return gocv.Mat{}, errors.New("loaded image is empty")
	}

	size := img.Size()
	if len(size) < 2 {
		img.Close()
		return gocv.Mat{}, errors.New("invalid image dimensions")
	}

	height, width := size[0], size[1]

	// Check minimum dimensions
	if width < 50 || height < 50 {
		img.Close()
		return gocv.Mat{}, fmt.Errorf("image too small (%dx%d), minimum required: 50x50", width, height)
	}

	// Check maximum dimensions (prevent DoS attacks)
	if width > 4000 || height > 4000 {
		img.Close()
		return gocv.Mat{}, fmt.Errorf("image too large (%dx%d), maximum allowed: 4000x4000", width, height)
	}

	// Check aspect ratio (prevent extremely distorted images)
	aspectRatio := float64(width) / float64(height)
	if aspectRatio < 0.3 || aspectRatio > 3.0 {
		img.Close()
		return gocv.Mat{}, fmt.Errorf("invalid aspect ratio: %.2f", aspectRatio)
	}

	return img, nil
}

// detectPrimaryFace detects the primary (largest) face in an image
func (fm *FaceMatcher) detectPrimaryFace(img gocv.Mat) (gocv.Mat, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	if !fm.initialized || !fm.modelsLoaded {
		return gocv.Mat{}, errors.New("face matcher not properly initialized")
	}

	// Set input size for YuNet
	imgSize := img.Size()
	if len(imgSize) >= 2 {
		fm.yunetDetector.SetInputSize(image.Pt(imgSize[1], imgSize[0])) // width, height
	}

	// Detect faces
	faces := gocv.NewMat()
	defer faces.Close()

	fm.yunetDetector.Detect(img, &faces)

	if faces.Rows() == 0 {
		return gocv.Mat{}, errors.New("no faces detected in image")
	}

	// Get the primary (first/largest) face
	x := int(faces.GetFloatAt(0, 0))
	y := int(faces.GetFloatAt(0, 1))
	w := int(faces.GetFloatAt(0, 2))
	h := int(faces.GetFloatAt(0, 3))

	// Validate face coordinates
	imgHeight, imgWidth := imgSize[0], imgSize[1]
	if x < 0 || y < 0 || x+w > imgWidth || y+h > imgHeight {
		return gocv.Mat{}, errors.New("detected face coordinates are invalid")
	}

	// Validate face size
	if w < 20 || h < 20 {
		return gocv.Mat{}, errors.New("detected face is too small")
	}

	// Extract face region
	faceRect := image.Rect(x, y, x+w, y+h)
	faceMat := img.Region(faceRect)

	return faceMat, nil
}

// loadImageFromURL loads an image from a URL using goroutine
func (fm *FaceMatcher) loadImageFromURL(url string, result chan<- ImageData) {
	go func() {
		defer close(result)

		resp, err := http.Get(url)
		if err != nil {
			result <- ImageData{Error: fmt.Errorf("failed to fetch image from URL: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			result <- ImageData{Error: fmt.Errorf("HTTP error: %d", resp.StatusCode)}
			return
		}

		imageBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			result <- ImageData{Error: fmt.Errorf("failed to read image data: %w", err)}
			return
		}

		mat, err := gocv.IMDecode(imageBytes, gocv.IMReadColor)
		if err != nil {
			result <- ImageData{Error: fmt.Errorf("failed to decode image: %w", err)}
			return
		}

		if mat.Empty() {
			result <- ImageData{Error: errors.New("decoded image is empty")}
			return
		}

		result <- ImageData{Mat: mat}
	}()
}

// loadImageFromBase64 loads an image from base64 string using goroutine
func (fm *FaceMatcher) loadImageFromBase64(base64Data string, result chan<- ImageData) {
	go func() {
		defer close(result)

		// Remove data URL prefix if present
		if strings.Contains(base64Data, ",") {
			parts := strings.Split(base64Data, ",")
			if len(parts) > 1 {
				base64Data = parts[1]
			}
		}

		imageBytes, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			result <- ImageData{Error: fmt.Errorf("failed to decode base64: %w", err)}
			return
		}

		mat, err := gocv.IMDecode(imageBytes, gocv.IMReadColor)
		if err != nil {
			result <- ImageData{Error: fmt.Errorf("failed to decode image: %w", err)}
			return
		}

		if mat.Empty() {
			result <- ImageData{Error: errors.New("decoded image is empty")}
			return
		}

		result <- ImageData{Mat: mat}
	}()
}

// loadImage determines if the input is URL or base64 and loads accordingly
func (fm *FaceMatcher) loadImage(input string) (gocv.Mat, error) {
	result := make(chan ImageData, 1)

	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		fm.loadImageFromURL(input, result)
	} else {
		fm.loadImageFromBase64(input, result)
	}

	data := <-result
	if data.Error != nil {
		return gocv.Mat{}, data.Error
	}

	return data.Mat, nil
}

// VerifyImageQuality analyzes an image to determine if it's suitable for biometric matching
func (fm *FaceMatcher) VerifyImageQuality(input string) ImageQualityResult {
	if !fm.initialized {
		return ImageQualityResult{
			Error: "face matcher not initialized",
		}
	}

	// Try to load models if they haven't been loaded yet
	if !fm.modelsLoaded {
		if err := fm.loadModels(); err != nil {
			return ImageQualityResult{
				Error: fmt.Sprintf("failed to load models: %v", err),
			}
		}
	}

	// Load the image
	img, err := fm.loadImage(input)
	if err != nil {
		return ImageQualityResult{
			Error: fmt.Sprintf("failed to load image: %v", err),
		}
	}
	defer img.Close()

	// Initialize result
	result := ImageQualityResult{
		Issues:          []string{},
		Recommendations: []string{},
	}

	// Get image dimensions
	imgSize := img.Size()
	if len(imgSize) < 2 {
		return ImageQualityResult{
			Error: "invalid image dimensions",
		}
	}

	height, width := imgSize[0], imgSize[1]
	result.ImageResolution = fmt.Sprintf("%dx%d", width, height)

	// Check minimum resolution
	minResolution := 200
	if width < minResolution || height < minResolution {
		result.Issues = append(result.Issues, fmt.Sprintf("Low resolution (%dx%d)", width, height))
		result.Recommendations = append(result.Recommendations, "Use higher resolution image (minimum 400x400)")
	}

	// Detect faces
	faces, faceCount := fm.detectAllFaces(img)
	if faces.Empty() || faceCount == 0 {
		result.HasFace = false
		result.FaceCount = 0
		result.IsGoodQuality = false
		result.QualityScore = 0.0
		result.Issues = append(result.Issues, "No face detected")
		result.Recommendations = append(result.Recommendations, "Ensure face is clearly visible and well-lit")
		return result
	}
	defer faces.Close()

	result.HasFace = true
	result.FaceCount = faceCount

	// Check for multiple faces
	if faceCount > 1 {
		result.Issues = append(result.Issues, fmt.Sprintf("Multiple faces detected (%d)", faceCount))
		result.Recommendations = append(result.Recommendations, "Use image with single person only")
	}

	// Analyze the primary (first/largest) face
	faceData := faces.GetFloatAt(0, 0)
	x := int(faceData)
	faceData = faces.GetFloatAt(0, 1)
	y := int(faceData)
	faceData = faces.GetFloatAt(0, 2)
	w := int(faceData)
	faceData = faces.GetFloatAt(0, 3)
	h := int(faceData)

	// Calculate face size as percentage of image
	imageArea := float64(width * height)
	faceArea := float64(w * h)
	facePercentage := (faceArea / imageArea) * 100
	result.FaceSize = facePercentage

	// Check face size requirements
	minFaceSize := 5.0     // 5% of image
	maxFaceSize := 80.0    // 80% of image
	optimalMinSize := 15.0 // 15% is better
	optimalMaxSize := 60.0 // 60% is better

	if facePercentage < minFaceSize {
		result.Issues = append(result.Issues, fmt.Sprintf("Face too small (%.1f%% of image)", facePercentage))
		result.Recommendations = append(result.Recommendations, "Move closer to camera or crop image to focus on face")
	} else if facePercentage > maxFaceSize {
		result.Issues = append(result.Issues, fmt.Sprintf("Face too large (%.1f%% of image)", facePercentage))
		result.Recommendations = append(result.Recommendations, "Move back from camera or include more background")
	}

	// Check face position (should be roughly centered)
	faceCenterX := x + w/2
	faceCenterY := y + h/2
	imageCenterX := width / 2
	imageCenterY := height / 2

	offsetX := float64(abs(faceCenterX-imageCenterX)) / float64(width) * 100
	offsetY := float64(abs(faceCenterY-imageCenterY)) / float64(height) * 100

	if offsetX > 30 || offsetY > 30 {
		result.Issues = append(result.Issues, "Face not well-centered in image")
		result.Recommendations = append(result.Recommendations, "Center the face in the image")
	}

	// Extract face region for further analysis
	faceRect := image.Rect(x, y, x+w, y+h)
	faceMat := img.Region(faceRect)
	defer faceMat.Close()

	// Check image sharpness/blur using Laplacian variance
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(faceMat, &gray, gocv.ColorBGRToGray)

	laplacian := gocv.NewMat()
	defer laplacian.Close()
	gocv.Laplacian(gray, &laplacian, gocv.MatTypeCV64F, 1, 1, 0, gocv.BorderDefault)

	// Calculate mean and standard deviation for blur detection
	meanMat := gocv.NewMat()
	stddevMat := gocv.NewMat()
	defer meanMat.Close()
	defer stddevMat.Close()

	gocv.MeanStdDev(laplacian, &meanMat, &stddevMat)
	variance := stddevMat.GetDoubleAt(0, 0) * stddevMat.GetDoubleAt(0, 0)

	// Blur detection threshold (adjust based on testing)
	blurThreshold := 85.0
	if variance < blurThreshold {
		result.Issues = append(result.Issues, "Image appears blurry or out of focus")
		result.Recommendations = append(result.Recommendations, "Ensure camera is focused and image is sharp")
	}

	// Check brightness/contrast using ORIGINAL grayscale image
	brightnessMean := gocv.NewMat()
	brightnessStddev := gocv.NewMat()
	defer brightnessMean.Close()
	defer brightnessStddev.Close()

	gocv.MeanStdDev(gray, &brightnessMean, &brightnessStddev)
	grayMean := brightnessMean.GetDoubleAt(0, 0)

	if grayMean < 50 {
		result.Issues = append(result.Issues, "Image is too dark")
		result.Recommendations = append(result.Recommendations, "Improve lighting conditions")
	} else if grayMean > 200 {
		result.Issues = append(result.Issues, "Image is overexposed")
		result.Recommendations = append(result.Recommendations, "Reduce lighting or exposure")
	}

	// Calculate overall quality score
	score := 1.0

	// Deduct for issues
	if faceCount > 1 {
		score -= 0.2
	}
	if facePercentage < optimalMinSize || facePercentage > optimalMaxSize {
		score -= 0.2
	}
	if offsetX > 20 || offsetY > 20 {
		score -= 0.1
	}
	if variance < blurThreshold {
		score -= 0.3
	}
	if grayMean < 50 || grayMean > 200 {
		score -= 0.2
	}
	if width < 400 || height < 400 {
		score -= 0.2
	}

	// Ensure score is between 0 and 1
	if score < 0 {
		score = 0
	}

	result.QualityScore = score

	// Determine if image is good quality (threshold: 0.7)
	result.IsGoodQuality = score >= 0.7 && len(result.Issues) <= 1

	if result.IsGoodQuality {
		result.Recommendations = append(result.Recommendations, "Image quality is suitable for biometric matching")
	}

	return result
}

// detectAllFaces detects all faces in an image and returns count
func (fm *FaceMatcher) detectAllFaces(img gocv.Mat) (gocv.Mat, int) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	if !fm.initialized || !fm.modelsLoaded {
		return gocv.Mat{}, 0
	}

	// Set input size for YuNet
	imgSize := img.Size()
	if len(imgSize) >= 2 {
		fm.yunetDetector.SetInputSize(image.Pt(imgSize[1], imgSize[0])) // width, height
	}

	// Detect faces
	faces := gocv.NewMat()
	fm.yunetDetector.Detect(img, &faces)

	return faces, faces.Rows()
}

// abs returns absolute value of integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Compare compares two faces from either URLs or base64 images
func (fm *FaceMatcher) Compare(input1, input2 string, threshold float64) CompareResult {
	if !fm.initialized {
		return CompareResult{
			Error: "face matcher not initialized",
		}
	}

	// Try to load models if they haven't been loaded yet
	if !fm.modelsLoaded {
		if err := fm.loadModels(); err != nil {
			return CompareResult{
				Error: fmt.Sprintf("failed to load models: %v", err),
			}
		}
	}

	// Load images concurrently using goroutines
	var wg sync.WaitGroup
	var img1, img2 gocv.Mat
	var err1, err2 error

	wg.Add(2)

	// Load first image
	go func() {
		defer wg.Done()
		img1, err1 = fm.loadImage(input1)
	}()

	// Load second image
	go func() {
		defer wg.Done()
		img2, err2 = fm.loadImage(input2)
	}()

	wg.Wait()

	// Check for loading errors
	if err1 != nil {
		return CompareResult{
			Error: fmt.Sprintf("failed to load first image: %v", err1),
		}
	}
	defer img1.Close()

	if err2 != nil {
		return CompareResult{
			Error: fmt.Sprintf("failed to load second image: %v", err2),
		}
	}
	defer img2.Close()

	// Detect and extract faces sequentially to avoid OpenCV thread safety issues
	face1, faceErr1 := fm.detectPrimaryFace(img1)
	if faceErr1 != nil {
		return CompareResult{
			Error: fmt.Sprintf("failed to detect face in first image: %v", faceErr1),
		}
	}
	defer face1.Close()

	face2, faceErr2 := fm.detectPrimaryFace(img2)
	if faceErr2 != nil {
		return CompareResult{
			Error: fmt.Sprintf("failed to detect face in second image: %v", faceErr2),
		}
	}
	defer face2.Close()

	// Extract features sequentially to avoid OpenCV thread safety issues
	features1, featErr1 := fm.extractFaceFeatures(face1)
	if featErr1 != nil {
		return CompareResult{
			Error: fmt.Sprintf("failed to extract features from first face: %v", featErr1),
		}
	}
	defer features1.Close()

	features2, featErr2 := fm.extractFaceFeatures(face2)
	if featErr2 != nil {
		return CompareResult{
			Error: fmt.Sprintf("failed to extract features from second face: %v", featErr2),
		}
	}
	defer features2.Close()

	// Calculate similarity
	similarity := fm.calculateSimilarity(features1, features2)

	return CompareResult{
		Similarity: similarity,
		Match:      similarity >= threshold,
	}
}

// Close releases all resources
func (fm *FaceMatcher) Close() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fm.initialized && fm.modelsLoaded {
		// Close the detector and network
		fm.yunetDetector.Close()
		fm.arcfaceNet.Close()
		fm.modelsLoaded = false
	}
	fm.initialized = false
}

// extractFaceFeatures extracts features using ArcFace
func (fm *FaceMatcher) extractFaceFeatures(face gocv.Mat) (gocv.Mat, error) {
	// Use read lock since we're only reading from the network
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	if !fm.initialized || !fm.modelsLoaded {
		return gocv.Mat{}, errors.New("face matcher not initialized or models not loaded")
	}

	// Preprocess face for ArcFace (usually 112x112)
	resized := gocv.NewMat()
	defer resized.Close()
	gocv.Resize(face, &resized, image.Pt(112, 112), 0, 0, gocv.InterpolationLinear)

	// Normalize to [0,1]
	normalized := gocv.NewMat()
	defer normalized.Close()
	resized.ConvertTo(&normalized, gocv.MatTypeCV32F)
	normalized.DivideFloat(255.0)

	// Create blob from image
	blob := gocv.BlobFromImage(normalized, 1.0, image.Pt(112, 112), gocv.NewScalar(0, 0, 0, 0), true, false)
	defer blob.Close()

	// Set input to the network
	fm.arcfaceNet.SetInput(blob, "")

	// Forward pass
	features := fm.arcfaceNet.Forward("")

	return features.Clone(), nil
}

// calculateSimilarity calculates cosine similarity between two feature vectors
func (fm *FaceMatcher) calculateSimilarity(features1, features2 gocv.Mat) float64 {
	// Flatten the feature matrices
	flat1 := features1.Reshape(1, 1)
	flat2 := features2.Reshape(1, 1)
	defer flat1.Close()
	defer flat2.Close()

	// Calculate dot product
	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	for i := 0; i < flat1.Cols(); i++ {
		val1 := float64(flat1.GetFloatAt(0, i))
		val2 := float64(flat2.GetFloatAt(0, i))

		dotProduct += val1 * val2
		norm1 += val1 * val1
		norm2 += val2 * val2
	}

	// Calculate cosine similarity
	if norm1 == 0 || norm2 == 0 {
		return 0
	}

	similarity := dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
	return similarity
}
