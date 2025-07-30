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

	"gateman.io/infrastructure/logger"
	"gocv.io/x/gocv"
)

// Face Matcher Configuration Constants
const (
	// Model Paths
	DEFAULT_YUNET_MODEL_PATH   = "./models/yunet.onnx"
	DEFAULT_ARCFACE_MODEL_PATH = "./models/arcface.onnx"

	// Image Processing Constants
	MIN_IMAGE_DIMENSION     = 50
	MAX_IMAGE_DIMENSION     = 4000
	MIN_ASPECT_RATIO        = 0.3
	MAX_ASPECT_RATIO        = 3.0
	MIN_FACE_SIZE           = 20
	MAX_DETECTION_DIMENSION = 640.0

	// Face Detection Thresholds
	HIGH_CONFIDENCE_THRESHOLD   = 0.9
	MEDIUM_CONFIDENCE_THRESHOLD = 0.5
	LOW_CONFIDENCE_THRESHOLD    = 0.3
	MIN_CONFIDENCE_THRESHOLD    = 0.1
	NMS_THRESHOLD               = 0.3

	// Image Quality Constants
	MIN_RESOLUTION             = 200
	RECOMMENDED_MIN_RESOLUTION = 400
	MIN_FACE_SIZE_PERCENT      = 5.0
	MAX_FACE_SIZE_PERCENT      = 80.0
	OPTIMAL_MIN_FACE_SIZE      = 15.0
	OPTIMAL_MAX_FACE_SIZE      = 60.0
	MAX_FACE_OFFSET_PERCENT    = 30.0
	BLUR_THRESHOLD             = 85.0
	MIN_BRIGHTNESS             = 50
	MAX_BRIGHTNESS             = 200
	QUALITY_SCORE_THRESHOLD    = 0.7
)

// Global FaceMatcher instance
var GlobalFaceMatcher *FaceMatcher

// InitializeFaceMatcherService initializes the global face matcher with model paths from environment variables
func InitializeFaceMatcherService() error {
	GlobalFaceMatcher = NewFaceMatcher()

	// Get model paths from environment variables with fallback defaults
	yunetModelPath := os.Getenv("YUNET_MODEL_PATH")
	if yunetModelPath == "" {
		yunetModelPath = DEFAULT_YUNET_MODEL_PATH
	}

	arcfaceModelPath := os.Getenv("ARCFACE_MODEL_PATH")
	if arcfaceModelPath == "" {
		arcfaceModelPath = DEFAULT_ARCFACE_MODEL_PATH
	}

	// Check if required model files exist
	if _, err := os.Stat(yunetModelPath); os.IsNotExist(err) {
		return fmt.Errorf("YuNet model file not found: %s", yunetModelPath)
	}
	if _, err := os.Stat(arcfaceModelPath); os.IsNotExist(err) {
		return fmt.Errorf("ArcFace model file not found: %s", arcfaceModelPath)
	}

	return GlobalFaceMatcher.Initialize(yunetModelPath, arcfaceModelPath)
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
	// Generate a request ID
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

	// Use the advanced implementation directly with standard verbosity
	advancedResult := fm.DetectAdvancedAntiSpoof(input, requestID, false)

	// Convert AdvancedAntiSpoofResult to AntiSpoofResult for compatibility
	return AntiSpoofResult{
		IsReal:       advancedResult.IsReal,
		SpoofScore:   advancedResult.SpoofScore,
		Confidence:   advancedResult.Confidence,
		HasFace:      advancedResult.HasFace,
		ProcessTime:  advancedResult.ProcessTime,
		SpoofReasons: advancedResult.SpoofReasons,
		Error:        advancedResult.Error,
	}
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
	if width < MIN_IMAGE_DIMENSION || height < MIN_IMAGE_DIMENSION {
		img.Close()
		return gocv.Mat{}, fmt.Errorf("image too small (%dx%d), minimum required: %dx%d", width, height, MIN_IMAGE_DIMENSION, MIN_IMAGE_DIMENSION)
	}

	// Check maximum dimensions (prevent DoS attacks)
	if width > MAX_IMAGE_DIMENSION || height > MAX_IMAGE_DIMENSION {
		img.Close()
		return gocv.Mat{}, fmt.Errorf("image too large (%dx%d), maximum allowed: %dx%d", width, height, MAX_IMAGE_DIMENSION, MAX_IMAGE_DIMENSION)
	}

	// Check aspect ratio (prevent extremely distorted images)
	aspectRatio := float64(width) / float64(height)
	if aspectRatio < MIN_ASPECT_RATIO || aspectRatio > MAX_ASPECT_RATIO {
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

	// Get original image dimensions
	imgSize := img.Size()
	if len(imgSize) < 2 {
		return gocv.Mat{}, errors.New("invalid image dimensions")
	}

	origHeight, origWidth := imgSize[0], imgSize[1]
	logger.Info(fmt.Sprintf("Original image: %dx%d (HxW)", origHeight, origWidth))

	// Calculate optimal size for YuNet detection
	maxDim := MAX_DETECTION_DIMENSION // Optimal size for YuNet
	scale := 1.0

	if origWidth > int(maxDim) || origHeight > int(maxDim) {
		// Scale down large images
		scale = maxDim / math.Max(float64(origWidth), float64(origHeight))
		logger.Info(fmt.Sprintf("Large image detected, scaling down by %.2fx for detection", scale))
	}

	// Create working image for detection
	var workingImg gocv.Mat
	var workingWidth, workingHeight int

	if scale < 1.0 {
		// Scale down the image
		workingWidth = int(float64(origWidth) * scale)
		workingHeight = int(float64(origHeight) * scale)

		workingImg = gocv.NewMat()
		gocv.Resize(img, &workingImg, image.Pt(workingWidth, workingHeight), 0, 0, gocv.InterpolationLinear)
		logger.Info(fmt.Sprintf("Scaled image for detection: %dx%d", workingHeight, workingWidth))
	} else {
		// Use original image
		workingImg = img.Clone()
		workingWidth = origWidth
		workingHeight = origHeight
		logger.Info("Using original size for detection")
	}
	defer workingImg.Close()

	// Set YuNet input size
	fm.yunetDetector.SetInputSize(image.Pt(workingWidth, workingHeight))

	// Try detection with different confidence thresholds
	thresholds := []float32{HIGH_CONFIDENCE_THRESHOLD, MEDIUM_CONFIDENCE_THRESHOLD, LOW_CONFIDENCE_THRESHOLD, MIN_CONFIDENCE_THRESHOLD}
	var faces gocv.Mat
	found := false

	for _, threshold := range thresholds {
		fm.yunetDetector.SetScoreThreshold(threshold)
		fm.yunetDetector.SetNMSThreshold(NMS_THRESHOLD)

		faces = gocv.NewMat()
		workingImg.CopyTo(&faces)
		faces.Close()
		faces = gocv.NewMat()

		fm.yunetDetector.Detect(workingImg, &faces)
		numFaces := faces.Rows()

		logger.Info(fmt.Sprintf("Detection with threshold %.1f: found %d faces", threshold, numFaces))

		if numFaces > 0 {
			found = true
			break
		}
		faces.Close()
	}

	if !found {
		return gocv.Mat{}, errors.New("no faces detected in image")
	}

	defer faces.Close()

	// Get the first face coordinates (from scaled image)
	x := int(faces.GetFloatAt(0, 0))
	y := int(faces.GetFloatAt(0, 1))
	w := int(faces.GetFloatAt(0, 2))
	h := int(faces.GetFloatAt(0, 3))

	logger.Info(fmt.Sprintf("Detected face in scaled image: x=%d, y=%d, w=%d, h=%d", x, y, w, h))

	// Scale coordinates back to original image size
	if scale < 1.0 {
		invScale := 1.0 / scale
		x = int(float64(x) * invScale)
		y = int(float64(y) * invScale)
		w = int(float64(w) * invScale)
		h = int(float64(h) * invScale)
		logger.Info(fmt.Sprintf("Scaled coordinates back to original: x=%d, y=%d, w=%d, h=%d", x, y, w, h))
	}

	// Validate coordinates against original image
	if x < 0 || y < 0 || x+w > origWidth || y+h > origHeight {
		logger.Info(fmt.Sprintf("Invalid coordinates after scaling: (%d,%d,%d,%d) vs image (%d,%d)", x, y, w, h, origWidth, origHeight))
		return gocv.Mat{}, errors.New("detected face coordinates are invalid")
	}

	// Validate face size
	if w < MIN_FACE_SIZE || h < MIN_FACE_SIZE {
		logger.Info(fmt.Sprintf("Face too small: %dx%d pixels", w, h))
		return gocv.Mat{}, errors.New("detected face is too small")
	}

	// Extract face region from original image
	faceRect := image.Rect(x, y, x+w, y+h)
	faceMat := img.Region(faceRect)

	// Calculate face size percentage of original image
	faceArea := w * h
	imageArea := origWidth * origHeight
	facePercentage := (float64(faceArea) / float64(imageArea)) * 100
	logger.Info(fmt.Sprintf("Extracted face: %dx%d (%.2f%% of original image)", w, h, facePercentage))

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

		// Check content type
		contentType := resp.Header.Get("Content-Type")
		log.Printf("Content-Type: %s", contentType)

		imageBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			result <- ImageData{Error: fmt.Errorf("failed to read image data: %w", err)}
			return
		}

		// Debug: Check what we downloaded
		log.Printf("Downloaded %d bytes from URL: %s", len(imageBytes), url)
		if len(imageBytes) == 0 {
			result <- ImageData{Error: errors.New("downloaded image data is empty")}
			return
		}

		// Check if data looks like an image by examining first few bytes
		if len(imageBytes) < 8 {
			result <- ImageData{Error: errors.New("downloaded data too small to be an image")}
			return
		}

		// Check for common image signatures
		imageType := detectImageType(imageBytes)
		logger.Info(fmt.Sprintf("Detected image type: %s", imageType))

		// Handle unsupported formats with specific error messages
		if imageType == "HEIC" {
			result <- ImageData{Error: errors.New("HEIC format not supported - please convert to JPEG or PNG. HEIC files are commonly from iPhones and need conversion before processing")}
			return
		}

		if imageType == "unknown" {
			logger.Info(fmt.Sprintf("Detected unknown image type: %s", imageType))
		}

		// Try different decoding approaches
		mat, err := fm.tryDecodeImage(imageBytes)
		if err != nil {
			result <- ImageData{Error: fmt.Errorf("failed to decode image (%s): %w", imageType, err)}
			return
		}

		if mat.Empty() {
			result <- ImageData{Error: fmt.Errorf("decoded image is empty (type: %s, size: %d bytes)", imageType, len(imageBytes))}
			return
		}

		log.Printf("Successfully decoded image: %dx%d", mat.Cols(), mat.Rows())
		result <- ImageData{Mat: mat}
	}()
}

// detectImageType detects the image type from the first few bytes
func detectImageType(data []byte) string {
	if len(data) < 8 {
		return "unknown"
	}

	// JPEG
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "JPEG"
	}

	// PNG
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "PNG"
	}

	// GIF
	if string(data[0:3]) == "GIF" {
		return "GIF"
	}

	// BMP
	if data[0] == 0x42 && data[1] == 0x4D {
		return "BMP"
	}

	// WEBP
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "WEBP"
	}

	// HEIC/HEIF detection (Apple's format)
	if len(data) >= 12 {
		// HEIC files start with specific patterns
		if string(data[4:8]) == "ftyp" {
			// Check for HEIC variants
			if len(data) >= 16 {
				brand := string(data[8:12])
				if brand == "heic" || brand == "mif1" || brand == "msf1" || brand == "hevc" || brand == "heix" {
					return "HEIC"
				}
			}
		}
	}

	// Check if it looks like HTML (common issue with URLs)
	if len(data) > 5 && (string(data[0:5]) == "<!DOC" || string(data[0:5]) == "<html" || string(data[0:4]) == "<HTM") {
		return "HTML"
	}

	return "unknown"
}

// tryDecodeImage tries multiple approaches to decode the image
func (fm *FaceMatcher) tryDecodeImage(imageBytes []byte) (gocv.Mat, error) {
	// Try 1: Standard decode with color
	mat, err := gocv.IMDecode(imageBytes, gocv.IMReadColor)
	if err == nil && !mat.Empty() {
		return mat, nil
	}
	if !mat.Empty() {
		mat.Close()
	}

	// Try 2: Decode as grayscale then convert to color
	mat, err = gocv.IMDecode(imageBytes, gocv.IMReadGrayScale)
	if err == nil && !mat.Empty() {
		colorMat := gocv.NewMat()
		gocv.CvtColor(mat, &colorMat, gocv.ColorGrayToBGR)
		mat.Close()
		if !colorMat.Empty() {
			return colorMat, nil
		}
		colorMat.Close()
	}
	if !mat.Empty() {
		mat.Close()
	}

	// Try 3: Decode with unchanged flag
	mat, err = gocv.IMDecode(imageBytes, gocv.IMReadUnchanged)
	if err == nil && !mat.Empty() {
		// Convert to BGR if needed
		if mat.Channels() == 1 {
			colorMat := gocv.NewMat()
			gocv.CvtColor(mat, &colorMat, gocv.ColorGrayToBGR)
			mat.Close()
			return colorMat, nil
		} else if mat.Channels() == 4 {
			colorMat := gocv.NewMat()
			gocv.CvtColor(mat, &colorMat, gocv.ColorBGRAToBGR)
			mat.Close()
			return colorMat, nil
		}
		return mat, nil
	}
	if !mat.Empty() {
		mat.Close()
	}

	return gocv.Mat{}, fmt.Errorf("all decode attempts failed: %v", err)
}

// loadImageFromBase64 loads an image from base64 string using goroutine
func (fm *FaceMatcher) loadImageFromBase64(base64Data string, result chan<- ImageData) {
	go func() {
		defer close(result)

		// Remove data URL prefix if present
		originalData := base64Data
		if strings.Contains(base64Data, ",") {
			parts := strings.Split(base64Data, ",")
			if len(parts) > 1 {
				base64Data = parts[1]
				log.Printf("Detected data URL format, extracted base64 part")
			}
		}

		imageBytes, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			result <- ImageData{Error: fmt.Errorf("failed to decode base64: %w", err)}
			return
		}

		// Debug: Check what we decoded
		log.Printf("Decoded %d bytes from base64", len(imageBytes))
		if len(imageBytes) == 0 {
			result <- ImageData{Error: errors.New("decoded base64 data is empty")}
			return
		}

		// Check if data looks like an image
		if len(imageBytes) < 8 {
			result <- ImageData{Error: errors.New("decoded data too small to be an image")}
			return
		}

		// Check for common image signatures
		imageType := detectImageType(imageBytes)
		log.Printf("Base64 image type: %s", imageType)

		// Handle unsupported formats with specific error messages
		if imageType == "HEIC" {
			result <- ImageData{Error: errors.New("HEIC format not supported - please convert to JPEG or PNG. HEIC files are commonly from iPhones and need conversion before processing")}
			return
		}

		if imageType == "unknown" {
			// Try to show first few bytes for debugging
			preview := imageBytes
			if len(preview) > 50 {
				preview = preview[:50]
			}
			log.Printf("First 50 bytes: %v", preview)
			log.Printf("First 50 bytes as string: %q", string(preview))

			// Also show the original data URL prefix if present
			if strings.Contains(originalData, ",") {
				parts := strings.Split(originalData, ",")
				log.Printf("Data URL prefix was: %s", parts[0])
			}
		}

		// Try different decoding approaches
		mat, err := fm.tryDecodeImage(imageBytes)
		if err != nil {
			result <- ImageData{Error: fmt.Errorf("failed to decode base64 image (%s): %w", imageType, err)}
			return
		}

		if mat.Empty() {
			result <- ImageData{Error: fmt.Errorf("decoded base64 image is empty (type: %s, size: %d bytes)", imageType, len(imageBytes))}
			return
		}

		log.Printf("Successfully decoded base64 image: %dx%d", mat.Cols(), mat.Rows())
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
	if width < MIN_RESOLUTION || height < MIN_RESOLUTION {
		result.Issues = append(result.Issues, fmt.Sprintf("Low resolution (%dx%d)", width, height))
		result.Recommendations = append(result.Recommendations, fmt.Sprintf("Use higher resolution image (minimum %dx%d)", RECOMMENDED_MIN_RESOLUTION, RECOMMENDED_MIN_RESOLUTION))
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
	minFaceSize := MIN_FACE_SIZE_PERCENT    // 5% of image
	maxFaceSize := MAX_FACE_SIZE_PERCENT    // 80% of image
	optimalMinSize := OPTIMAL_MIN_FACE_SIZE // 15% is better
	optimalMaxSize := OPTIMAL_MAX_FACE_SIZE // 60% is better

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

	if offsetX > MAX_FACE_OFFSET_PERCENT || offsetY > MAX_FACE_OFFSET_PERCENT {
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
	blurThreshold := BLUR_THRESHOLD
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

	if grayMean < MIN_BRIGHTNESS {
		result.Issues = append(result.Issues, "Image is too dark")
		result.Recommendations = append(result.Recommendations, "Improve lighting conditions")
	} else if grayMean > MAX_BRIGHTNESS {
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
	if grayMean < MIN_BRIGHTNESS || grayMean > MAX_BRIGHTNESS {
		score -= 0.2
	}
	if width < RECOMMENDED_MIN_RESOLUTION || height < RECOMMENDED_MIN_RESOLUTION {
		score -= 0.2
	}

	// Ensure score is between 0 and 1
	if score < 0 {
		score = 0
	}

	result.QualityScore = score

	// Determine if image is good quality (threshold: 0.7)
	result.IsGoodQuality = score >= QUALITY_SCORE_THRESHOLD && len(result.Issues) <= 1

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

// Compare compares two faces from either URLs or base64 images with improved threshold
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

	// Improve threshold for better accuracy (minimum 0.8 to reduce false positives)
	adjustedThreshold := math.Max(threshold, 0.8)

	return CompareResult{
		Similarity: similarity,
		Match:      similarity >= adjustedThreshold,
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
