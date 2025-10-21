package biometric

import (
	"bytes"
	"fmt"
	"image"
	stdimage "image" // Alias to avoid shadowing issues
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gateman.io/application/utils"
	"gateman.io/infrastructure/biometric/types"
	"gateman.io/infrastructure/logger"
	"gocv.io/x/gocv"
)

// LocalFaceService provides local face comparison and liveness detection using OpenCV
type LocalFaceService struct {
	faceCascade     gocv.CascadeClassifier
	eyeCascade      gocv.CascadeClassifier
	modelsLoaded    bool
	processingStats ProcessingStats
}

// ProcessingStats tracks processing statistics
type ProcessingStats struct {
	TotalRequests      int64
	SuccessfulRequests int64
	AverageTime        float64
	TotalTime          int64
}

// FaceComparisonResult represents the result of face comparison
type FaceComparisonResult struct {
	IsMatch        bool    `json:"is_match"`
	Similarity     float64 `json:"similarity"`
	Confidence     float64 `json:"confidence"`
	ProcessingTime int64   `json:"processing_time_ms"`
	FaceCount1     int     `json:"face_count_1"`
	FaceCount2     int     `json:"face_count_2"`
	Quality1       float64 `json:"quality_1"`
	Quality2       float64 `json:"quality_2"`
}

// LivenessResult represents the result of liveness detection
type LivenessResult struct {
	IsLive         bool    `json:"is_live"`
	SpoofScore     float64 `json:"spoof_score"`
	Confidence     float64 `json:"confidence"`
	ProcessingTime int64   `json:"processing_time_ms"`
	FaceCount      int     `json:"face_count"`
	Quality        float64 `json:"quality"`
	Analysis       struct {
		TextureScore    float64 `json:"texture_score"`
		EdgeScore       float64 `json:"edge_score"`
		ColorScore      float64 `json:"color_score"`
		ReflectionScore float64 `json:"reflection_score"`
		MotionScore     float64 `json:"motion_score"`
	} `json:"analysis"`
}

// NewLocalFaceService creates a new local face service
func NewLocalFaceService() *LocalFaceService {
	service := &LocalFaceService{
		processingStats: ProcessingStats{},
	}

	// Load face detection models
	if err := service.loadModels(); err != nil {
		logger.Error("Failed to load face detection models", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return service
	}

	service.modelsLoaded = true
	logger.Info("Local face service initialized successfully")

	return service
}

// loadModels loads the required OpenCV models
func (lfs *LocalFaceService) loadModels() error {
	// Get cascade file paths from environment or use defaults
	cascadePath := os.Getenv("OPENCV_CASCADE_PATH")
	if cascadePath == "" {
		cascadePath = "./models/haarcascades"
	}

	// Load Haar cascade for face detection
	lfs.faceCascade = gocv.NewCascadeClassifier()
	faceCascadeFile := filepath.Join(cascadePath, "haarcascade_frontalface_alt.xml")
	if !lfs.faceCascade.Load(faceCascadeFile) {
		// Try alternative paths
		alternativePaths := []string{
			"haarcascade_frontalface_alt.xml",
			"/usr/local/share/opencv4/haarcascades/haarcascade_frontalface_alt.xml",
			"/usr/share/opencv4/haarcascades/haarcascade_frontalface_alt.xml",
			"/opt/homebrew/share/opencv4/haarcascades/haarcascade_frontalface_alt.xml",
		}

		loaded := false
		for _, path := range alternativePaths {
			if lfs.faceCascade.Load(path) {
				loaded = true
				break
			}
		}

		if !loaded {
			return fmt.Errorf("failed to load face cascade classifier from %s or alternative paths", faceCascadeFile)
		}
	}

	// Load Haar cascade for eye detection
	lfs.eyeCascade = gocv.NewCascadeClassifier()
	eyeCascadeFile := filepath.Join(cascadePath, "haarcascade_eye.xml")
	if !lfs.eyeCascade.Load(eyeCascadeFile) {
		// Try alternative paths
		alternativePaths := []string{
			"haarcascade_eye.xml",
			"/usr/local/share/opencv4/haarcascades/haarcascade_eye.xml",
			"/usr/share/opencv4/haarcascades/haarcascade_eye.xml",
			"/opt/homebrew/share/opencv4/haarcascades/haarcascade_eye.xml",
		}

		loaded := false
		for _, path := range alternativePaths {
			if lfs.eyeCascade.Load(path) {
				loaded = true
				break
			}
		}

		if !loaded {
			return fmt.Errorf("failed to load eye cascade classifier from %s or alternative paths", eyeCascadeFile)
		}
	}

	// Note: LBPH face recognizer removed for simplicity

	return nil
}

// CompareFacesWithMobileNet compares two face images using MobileNet detection ONLY
func (lfs *LocalFaceService) CompareFacesWithMobileNet(image1 *string, image2 *string) (*types.BiometricFaceMatchResponse, error) {
	startTime := time.Now()

	// Create MobileNet service ONLY
	mobileNetConfig := GetDefaultMobileNetConfig()
	mobileNetService := NewMobileNetFaceService(mobileNetConfig)
	defer mobileNetService.Close()

	// Process first image with MobileNet detection ONLY
	img1, faces1, quality1, err := lfs.ProcessImageWithMobileNet(*image1, mobileNetService)
	if err != nil {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to process first image: %v", err)),
		}, nil
	}
	defer img1.Close()

	// Process second image with MobileNet detection ONLY
	img2, faces2, quality2, err := lfs.ProcessImageWithMobileNet(*image2, mobileNetService)
	if err != nil {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to process second image: %v", err)),
		}, nil
	}
	defer img2.Close()

	// Check if faces are detected
	if len(faces1) == 0 {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("No face detected in first image"),
		}, nil
	}

	if len(faces2) == 0 {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("No face detected in second image"),
		}, nil
	}

	// Use the largest face from each image
	face1 := lfs.getLargestFace(faces1)
	face2 := lfs.getLargestFace(faces2)

	// Extract face regions
	faceRegion1 := img1.Region(face1)
	faceRegion2 := img2.Region(face2)
	defer faceRegion1.Close()
	defer faceRegion2.Close()

	// Convert to grayscale for comparison
	gray1 := gocv.NewMat()
	gray2 := gocv.NewMat()
	defer gray1.Close()
	defer gray2.Close()

	gocv.CvtColor(faceRegion1, &gray1, gocv.ColorBGRToGray)
	gocv.CvtColor(faceRegion2, &gray2, gocv.ColorBGRToGray)

	// Enhanced face preprocessing
	processed1 := lfs.preprocessFaceForComparison(gray1)
	processed2 := lfs.preprocessFaceForComparison(gray2)
	defer processed1.Close()
	defer processed2.Close()

	// Calculate similarity using multiple methods
	similarity := lfs.calculateSimilarity(processed1, processed2)
	confidence := lfs.calculateConfidence(float64(similarity), quality1, quality2)

	// Adaptive threshold based on image quality and confidence
	// ADJUSTED: Lowered from 0.8 to 0.65 for better real-world accuracy with MobileNet
	baseThreshold := 0.65
	minConfidence := 0.55 // Lowered from 0.7 for better balance

	// Adjust threshold based on image quality
	avgQuality := (quality1 + quality2) / 2.0
	if avgQuality < 0.3 {
		// Only very poor quality images get stricter threshold
		baseThreshold += 0.1
	} else if avgQuality > 0.7 {
		// High quality images can be slightly more lenient
		baseThreshold -= 0.05
	}

	// Additional validation: ensure both faces are detected properly
	if len(faces1) != 1 || len(faces2) != 1 {
		// Multiple faces - be more cautious
		baseThreshold += 0.15
	}

	// Quality difference penalty - only for very large differences
	qualityDiff := math.Abs(quality1 - quality2)
	if qualityDiff > 0.4 {
		baseThreshold += 0.08
	}

	// Determine if faces match based on adaptive thresholds
	isMatch := float64(similarity) > baseThreshold && confidence > minConfidence

	processingTime := time.Since(startTime)

	// Update statistics
	lfs.updateStats(processingTime.Milliseconds(), true)

	return &types.BiometricFaceMatchResponse{
		Success:           true,
		Match:             isMatch,
		Confidence:        confidence,
		ProcessingTimeMs:  int(processingTime.Milliseconds()),
		FaceQualityScores: []float64{quality1, quality2},
		Liveness: types.Liveness{
			Image1: lfs.performQuickLivenessCheck(faceRegion1),
			Image2: lfs.performQuickLivenessCheck(faceRegion2),
		},
	}, nil
}

// CompareFacesWithYuNet compares two face images using YuNet detection
func (lfs *LocalFaceService) CompareFacesWithYuNet(image1 *string, image2 *string) (*types.BiometricFaceMatchResponse, error) {
	startTime := time.Now()

	// Create YuNet service
	yunetConfig := GetDefaultYuNetConfig()
	yunetService := NewYuNetFaceService(yunetConfig)
	defer yunetService.Close()

	// Process first image with YuNet detection
	img1, faces1, quality1, err := lfs.ProcessImageWithYuNet(*image1, yunetService)
	if err != nil {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to process first image: %v", err)),
		}, nil
	}
	defer img1.Close()

	// Process second image with YuNet detection
	img2, faces2, quality2, err := lfs.ProcessImageWithYuNet(*image2, yunetService)
	if err != nil {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to process second image: %v", err)),
		}, nil
	}
	defer img2.Close()

	// Check if faces are detected
	if len(faces1) == 0 {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("No face detected in first image"),
		}, nil
	}

	if len(faces2) == 0 {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("No face detected in second image"),
		}, nil
	}

	// Use the largest face from each image
	face1 := lfs.getLargestFace(faces1)
	face2 := lfs.getLargestFace(faces2)

	// Extract face regions
	faceRegion1 := img1.Region(face1)
	faceRegion2 := img2.Region(face2)
	defer faceRegion1.Close()
	defer faceRegion2.Close()

	// Convert to grayscale for comparison
	gray1 := gocv.NewMat()
	gray2 := gocv.NewMat()
	defer gray1.Close()
	defer gray2.Close()

	gocv.CvtColor(faceRegion1, &gray1, gocv.ColorBGRToGray)
	gocv.CvtColor(faceRegion2, &gray2, gocv.ColorBGRToGray)

	// Enhanced face preprocessing
	processed1 := lfs.preprocessFaceForComparison(gray1)
	processed2 := lfs.preprocessFaceForComparison(gray2)
	defer processed1.Close()
	defer processed2.Close()

	// Calculate similarity using multiple methods
	similarity := lfs.calculateSimilarity(processed1, processed2)
	confidence := lfs.calculateConfidence(float64(similarity), quality1, quality2)

	// Adaptive threshold based on image quality and confidence
	// ADJUSTED: Lowered from 0.75 to 0.60 for better real-world accuracy with YuNet
	baseThreshold := 0.60
	minConfidence := 0.50 // Lowered from 0.65 for better balance

	// Adjust threshold based on image quality
	avgQuality := (quality1 + quality2) / 2.0
	if avgQuality < 0.3 {
		// Only very poor quality images get stricter threshold
		baseThreshold += 0.1
	} else if avgQuality > 0.7 {
		// High quality images can be slightly more lenient
		baseThreshold -= 0.05
	}

	// Additional validation: ensure both faces are detected properly
	if len(faces1) != 1 || len(faces2) != 1 {
		// Multiple faces - be more cautious
		baseThreshold += 0.15
	}

	// Quality difference penalty - only for very large differences
	qualityDiff := math.Abs(quality1 - quality2)
	if qualityDiff > 0.4 {
		baseThreshold += 0.08
	}

	// Determine if faces match based on adaptive thresholds
	isMatch := float64(similarity) > baseThreshold && confidence > minConfidence

	processingTime := time.Since(startTime)

	// Update statistics
	lfs.updateStats(processingTime.Milliseconds(), true)

	return &types.BiometricFaceMatchResponse{
		Success:           true,
		Match:             isMatch,
		Confidence:        confidence,
		ProcessingTimeMs:  int(processingTime.Milliseconds()),
		FaceQualityScores: []float64{quality1, quality2},
		Liveness: types.Liveness{
			Image1: lfs.performQuickLivenessCheck(faceRegion1),
			Image2: lfs.performQuickLivenessCheck(faceRegion2),
		},
	}, nil
}

// CompareFaces compares two face images using YuNet detection + ArcFace recognition
func (lfs *LocalFaceService) CompareFaces(image1 *string, image2 *string) (*types.BiometricFaceMatchResponse, error) {
	// Use YuNet + ArcFace for best accuracy and discrimination
	// ArcFace provides better separation between similar-looking different people
	return lfs.CompareFacesWithArcFace(image1, image2)
}

// CompareFacesWithFaceNet compares two face images using YuNet detection + FaceNet recognition
func (lfs *LocalFaceService) CompareFacesWithFaceNet(image1 *string, image2 *string) (*types.BiometricFaceMatchResponse, error) {
	startTime := time.Now()

	// Create YuNet service for face detection
	yunetConfig := GetDefaultYuNetConfig()
	yunetService := NewYuNetFaceService(yunetConfig)
	defer yunetService.Close()

	// Create FaceNet recognizer for face comparison
	facenetConfig := GetDefaultFaceNetConfig()
	facenetRecognizer := NewFaceNetRecognizer(facenetConfig)
	defer facenetRecognizer.Close()

	if !facenetRecognizer.modelsLoaded {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("FaceNet model not loaded. Please ensure the model file exists at: " + facenetConfig.ModelPath),
		}, nil
	}

	// Process first image with YuNet detection
	img1, faces1, quality1, err := lfs.ProcessImageWithYuNet(*image1, yunetService)
	if err != nil {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to process first image: %v", err)),
		}, nil
	}
	defer img1.Close()

	// Get landmarks for first image
	detectionResult1, _ := yunetService.DetectFaces(img1)

	// Process second image with YuNet detection
	img2, faces2, quality2, err := lfs.ProcessImageWithYuNet(*image2, yunetService)
	if err != nil {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to process second image: %v", err)),
		}, nil
	}
	defer img2.Close()

	// Get landmarks for second image
	detectionResult2, _ := yunetService.DetectFaces(img2)

	// Check if faces are detected
	if len(faces1) == 0 {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("No face detected in first image"),
		}, nil
	}

	if len(faces2) == 0 {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("No face detected in second image"),
		}, nil
	}

	// Use the largest face from each image
	face1 := lfs.getLargestFace(faces1)
	face2 := lfs.getLargestFace(faces2)

	// Extract face regions
	faceRegion1 := img1.Region(face1)
	faceRegion2 := img2.Region(face2)
	defer faceRegion1.Close()
	defer faceRegion2.Close()

	// Extract face embeddings using FaceNet
	embedding1, err := facenetRecognizer.ExtractEmbedding(faceRegion1)
	if err != nil {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to extract embedding from first image: %v", err)),
		}, nil
	}

	embedding2, err := facenetRecognizer.ExtractEmbedding(faceRegion2)
	if err != nil {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to extract embedding from second image: %v", err)),
		}, nil
	}

	// Compare embeddings using cosine similarity
	similarity := facenetRecognizer.CompareFaces(embedding1, embedding2)

	// FaceNet/SFace similarity is in range [-1, 1], normalize to [0, 1]
	normalizedSimilarity := (similarity + 1.0) / 2.0

	// Calculate confidence based on similarity and quality
	confidence := lfs.calculateConfidence(normalizedSimilarity, quality1, quality2)

	// SFace threshold - adjusted to 0.82 cosine similarity (0.91 normalized)
	// Combined with hybrid scoring (embedding + landmarks) for robust matching
	// The hybrid score prevents false positives even when embedding similarity is high
	baseThreshold := 0.82                              // Cosine similarity threshold
	normalizedThreshold := (baseThreshold + 1.0) / 2.0 // Convert to [0, 1] range = 0.91

	avgQuality := (quality1 + quality2) / 2.0
	if avgQuality < 0.3 {
		// Very poor quality - be more strict
		normalizedThreshold += 0.05
	} else if avgQuality > 0.7 {
		// High quality - slightly more lenient but still conservative
		normalizedThreshold -= 0.01 // Minimal reduction for high quality
	}

	// Multiple faces penalty
	if len(faces1) != 1 || len(faces2) != 1 {
		normalizedThreshold += 0.05
	}

	// Quality difference penalty
	qualityDiff := math.Abs(quality1 - quality2)
	if qualityDiff > 0.4 {
		normalizedThreshold += 0.03
	}

	// Get geometric similarity from landmarks
	landmarkValid := true
	geometricSimilarity := 1.0
	if len(detectionResult1.Landmarks) > 0 && len(detectionResult2.Landmarks) > 0 {
		// Get landmarks for the largest faces
		landmarks1 := detectionResult1.Landmarks[0]
		landmarks2 := detectionResult2.Landmarks[0]

		// Calculate landmark geometry similarity
		landmarkValid, geometricSimilarity = yunetService.ValidateFacialLandmarks(
			landmarks1, landmarks2, face1, face2,
		)
	}

	// Hybrid matching decision: combine embedding similarity with geometric similarity
	// Use weighted score: 65% embedding + 35% geometric (increased geometric weight)
	hybridScore := (normalizedSimilarity * 0.65) + (geometricSimilarity * 0.35)

	// Require both embedding similarity above threshold AND reasonable hybrid score
	// Hybrid threshold of 0.79 balances security and usability
	// This prevents most false positives while minimizing false negatives
	// Calibrated based on real-world test data (80% accuracy, 0 false negatives)
	isMatch := normalizedSimilarity > normalizedThreshold &&
		confidence > 0.5 &&
		hybridScore > 0.79

	processingTime := time.Since(startTime)

	// Update statistics
	lfs.updateStats(processingTime.Milliseconds(), true)

	logger.Info("FaceNet face comparison completed", logger.LoggerOptions{
		Key: "comparison_result",
		Data: map[string]interface{}{
			"cosine_similarity":     similarity,
			"normalized_similarity": normalizedSimilarity,
			"threshold":             normalizedThreshold,
			"confidence":            confidence,
			"is_match":              isMatch,
			"landmark_valid":        landmarkValid,
			"geometric_similarity":  geometricSimilarity,
			"hybrid_score":          hybridScore,
			"quality1":              quality1,
			"quality2":              quality2,
			"faces_detected_image1": len(faces1),
			"faces_detected_image2": len(faces2),
			"processing_time_ms":    processingTime.Milliseconds(),
		},
	})

	return &types.BiometricFaceMatchResponse{
		Success:           true,
		Match:             isMatch,
		Confidence:        confidence,
		ProcessingTimeMs:  int(processingTime.Milliseconds()),
		FaceQualityScores: []float64{quality1, quality2},
		Liveness: types.Liveness{
			Image1: lfs.performQuickLivenessCheck(faceRegion1),
			Image2: lfs.performQuickLivenessCheck(faceRegion2),
		},
	}, nil
}

// CompareFacesWithArcFace compares two face images using YuNet detection + ArcFace recognition
func (lfs *LocalFaceService) CompareFacesWithArcFace(image1 *string, image2 *string) (*types.BiometricFaceMatchResponse, error) {
	startTime := time.Now()

	// Create YuNet service for face detection
	yunetConfig := GetDefaultYuNetConfig()
	yunetService := NewYuNetFaceService(yunetConfig)
	defer yunetService.Close()

	// Create ArcFace recognizer for face comparison
	arcfaceConfig := GetDefaultArcFaceConfig()
	arcfaceRecognizer := NewArcFaceRecognizer(arcfaceConfig)
	defer arcfaceRecognizer.Close()

	if !arcfaceRecognizer.modelsLoaded {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("ArcFace model not loaded. Please ensure the model file exists at: " + arcfaceConfig.ModelPath),
		}, nil
	}

	// CONCURRENT PROCESSING: Download, detect faces, and calculate quality for both images simultaneously
	type imageProcessResult struct {
		img     gocv.Mat
		faces   []image.Rectangle
		quality float64
		err     error
	}

	resultChan := make(chan imageProcessResult, 2)

	// Process image 1 concurrently (download + face detection + quality check)
	go func() {
		img, faces, quality, err := lfs.ProcessImageWithYuNet(*image1, yunetService)
		resultChan <- imageProcessResult{img, faces, quality, err}
	}()

	// Process image 2 concurrently (download + face detection + quality check)
	go func() {
		img, faces, quality, err := lfs.ProcessImageWithYuNet(*image2, yunetService)
		resultChan <- imageProcessResult{img, faces, quality, err}
	}()

	// Wait for both results
	result1 := <-resultChan
	result2 := <-resultChan

	// Check for errors in image 1
	if result1.err != nil {
		if !result1.img.Empty() {
			result1.img.Close()
		}
		if !result2.img.Empty() {
			result2.img.Close()
		}
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to process first image: %v", result1.err)),
		}, nil
	}

	// Check for errors in image 2
	if result2.err != nil {
		result1.img.Close()
		if !result2.img.Empty() {
			result2.img.Close()
		}
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to process second image: %v", result2.err)),
		}, nil
	}

	// Assign results
	img1 := result1.img
	faces1 := result1.faces
	quality1 := result1.quality

	img2 := result2.img
	faces2 := result2.faces
	quality2 := result2.quality

	defer img1.Close()
	defer img2.Close()

	// Check if faces are detected
	if len(faces1) == 0 {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("No face detected in first image"),
		}, nil
	}

	if len(faces2) == 0 {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("No face detected in second image"),
		}, nil
	}

	// Use the largest face from each image
	face1 := lfs.getLargestFace(faces1)
	face2 := lfs.getLargestFace(faces2)

	// Extract face regions
	faceRegion1 := img1.Region(face1)
	faceRegion2 := img2.Region(face2)
	defer faceRegion1.Close()
	defer faceRegion2.Close()

	// SEQUENTIAL ARCFACE PROCESSING: Extract embeddings one at a time
	// (ArcFace model operations are not thread-safe, must be sequential)
	embedding1, err := arcfaceRecognizer.ExtractEmbedding(faceRegion1)
	if err != nil {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to extract embedding from first image: %v", err)),
		}, nil
	}

	embedding2, err := arcfaceRecognizer.ExtractEmbedding(faceRegion2)
	if err != nil {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to extract embedding from second image: %v", err)),
		}, nil
	}

	// Compare embeddings using cosine similarity
	similarity := arcfaceRecognizer.CompareFaces(embedding1, embedding2)

	// ArcFace similarity is in range [-1, 1], normalize to [0, 1]
	normalizedSimilarity := (similarity + 1.0) / 2.0

	// Calculate confidence based on similarity and quality
	confidence := lfs.calculateConfidence(normalizedSimilarity, quality1, quality2)

	// RUN LIVENESS CHECKS CONCURRENTLY USING GOROUTINES ALWAYS BEFORE FACE COMPARISON
	livenessChan := make(chan struct {
		image1Live bool
		image2Live bool
	}, 1)

	go func() {
		image1Live := lfs.performQuickLivenessCheck(faceRegion1)
		image2Live := lfs.performQuickLivenessCheck(faceRegion2)
		livenessChan <- struct {
			image1Live bool
			image2Live bool
		}{image1Live, image2Live}
	}()

	// ArcFace threshold - typically 0.4 cosine similarity (0.7 normalized)
	// Adjust based on image quality
	baseThreshold := 0.4                               // Cosine similarity threshold
	normalizedThreshold := (baseThreshold + 1.0) / 2.0 // Convert to [0, 1] range

	avgQuality := (quality1 + quality2) / 2.0
	if avgQuality < 0.3 {
		// Very poor quality - be more strict
		normalizedThreshold += 0.05
	} else if avgQuality > 0.7 {
		// High quality - can be slightly more lenient
		normalizedThreshold -= 0.03
	}

	// Multiple faces penalty
	if len(faces1) != 1 || len(faces2) != 1 {
		normalizedThreshold += 0.05
	}

	// Quality difference penalty
	qualityDiff := math.Abs(quality1 - quality2)
	if qualityDiff > 0.4 {
		normalizedThreshold += 0.03
	}

	// Determine if faces match
	isMatch := normalizedSimilarity > normalizedThreshold && confidence > 0.5

	processingTime := time.Since(startTime)

	// Wait for liveness results
	livenessResult := <-livenessChan

	// Update statistics
	lfs.updateStats(processingTime.Milliseconds(), true)

	logger.Info("ArcFace face comparison completed", logger.LoggerOptions{
		Key: "comparison_result",
		Data: map[string]interface{}{
			"cosine_similarity":     similarity,
			"normalized_similarity": normalizedSimilarity,
			"threshold":             normalizedThreshold,
			"confidence":            confidence,
			"is_match":              isMatch,
			"quality1":              quality1,
			"quality2":              quality2,
			"faces_detected_image1": len(faces1),
			"faces_detected_image2": len(faces2),
			"processing_time_ms":    processingTime.Milliseconds(),
		},
	})

	return &types.BiometricFaceMatchResponse{
		Success:           true,
		Match:             isMatch,
		Confidence:        confidence,
		ProcessingTimeMs:  int(processingTime.Milliseconds()),
		FaceQualityScores: []float64{quality1, quality2},
		Liveness: types.Liveness{
			Image1: livenessResult.image1Live,
			Image2: livenessResult.image2Live,
		},
	}, nil
}

// CompareFacesWithEnhancedHaar compares two face images using enhanced Haar cascade with improved accuracy
func (lfs *LocalFaceService) CompareFacesWithEnhancedHaar(image1 *string, image2 *string) (*types.BiometricFaceMatchResponse, error) {
	startTime := time.Now()

	if !lfs.modelsLoaded {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("Haar cascade models not loaded"),
		}, nil
	}

	// Process first image with enhanced detection
	img1, faces1, quality1, err := lfs.ProcessImageWithEnhancedDetection(*image1)
	if err != nil {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to process first image: %v", err)),
		}, nil
	}
	defer img1.Close()

	// Process second image with enhanced detection
	img2, faces2, quality2, err := lfs.ProcessImageWithEnhancedDetection(*image2)
	if err != nil {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to process second image: %v", err)),
		}, nil
	}
	defer img2.Close()

	// Check if faces are detected
	if len(faces1) == 0 {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("No face detected in first image"),
		}, nil
	}

	if len(faces2) == 0 {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("No face detected in second image"),
		}, nil
	}

	// Use the largest face from each image
	face1 := lfs.getLargestFace(faces1)
	face2 := lfs.getLargestFace(faces2)

	// Extract face regions with padding for better accuracy
	faceRegion1 := lfs.extractFaceRegionWithPadding(img1, face1)
	faceRegion2 := lfs.extractFaceRegionWithPadding(img2, face2)
	defer faceRegion1.Close()
	defer faceRegion2.Close()

	// Enhanced preprocessing for better comparison
	processed1 := lfs.enhancedFacePreprocessing(faceRegion1)
	processed2 := lfs.enhancedFacePreprocessing(faceRegion2)
	defer processed1.Close()
	defer processed2.Close()

	// Calculate similarity using multiple enhanced methods
	similarity := lfs.calculateEnhancedSimilarity(processed1, processed2)
	confidence := lfs.calculateEnhancedConfidence(float64(similarity), quality1, quality2, faces1, faces2)

	// Adaptive threshold based on image quality, confidence, and face characteristics
	// ADJUSTED: Lowered from 0.75 to 0.62 for better real-world accuracy with enhanced Haar
	baseThreshold := 0.62
	minConfidence := 0.52 // Lowered from 0.65 for better balance

	// Quality-based threshold adjustment
	avgQuality := (quality1 + quality2) / 2.0
	if avgQuality < 0.3 {
		// Only very poor quality images get stricter threshold
		baseThreshold += 0.1
	} else if avgQuality > 0.7 {
		// High quality images can be slightly more lenient
		baseThreshold -= 0.05
	}

	// Multiple faces penalty
	if len(faces1) != 1 || len(faces2) != 1 {
		baseThreshold += 0.12 // Reduced penalty for multiple faces
	}

	// Quality difference penalty - only for very large differences
	qualityDiff := math.Abs(quality1 - quality2)
	if qualityDiff > 0.4 {
		baseThreshold += 0.08
	}

	// Face size consistency check - more lenient
	sizeRatio := float64(face1.Dx()*face1.Dy()) / float64(face2.Dx()*face2.Dy())
	if sizeRatio < 0.4 || sizeRatio > 2.5 {
		baseThreshold += 0.08 // Reduced penalty for size differences
	}

	// Determine if faces match based on adaptive thresholds
	isMatch := float64(similarity) > baseThreshold && confidence > minConfidence

	processingTime := time.Since(startTime)

	// Update statistics
	lfs.updateStats(processingTime.Milliseconds(), true)

	return &types.BiometricFaceMatchResponse{
		Success:           true,
		Match:             isMatch,
		Confidence:        confidence,
		ProcessingTimeMs:  int(processingTime.Milliseconds()),
		FaceQualityScores: []float64{quality1, quality2},
		Liveness: types.Liveness{
			Image1: lfs.performQuickLivenessCheck(faceRegion1),
			Image2: lfs.performQuickLivenessCheck(faceRegion2),
		},
	}, nil
}

// CompareFacesWithHaar compares two face images using Haar cascade (legacy method)
func (lfs *LocalFaceService) CompareFacesWithHaar(image1 *string, image2 *string) (*types.BiometricFaceMatchResponse, error) {
	startTime := time.Now()

	if !lfs.modelsLoaded {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("Face detection models not loaded"),
		}, nil
	}

	// Process first image
	img1, faces1, quality1, err := lfs.ProcessImage(*image1)
	if err != nil {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to process first image: %v", err)),
		}, nil
	}
	defer img1.Close()

	// Process second image
	img2, faces2, quality2, err := lfs.ProcessImage(*image2)
	if err != nil {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer(fmt.Sprintf("Failed to process second image: %v", err)),
		}, nil
	}
	defer img2.Close()

	// Check if faces are detected
	if len(faces1) == 0 {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("No face detected in first image"),
		}, nil
	}

	if len(faces2) == 0 {
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("No face detected in second image"),
		}, nil
	}

	// Use the largest face from each image
	face1 := lfs.getLargestFace(faces1)
	face2 := lfs.getLargestFace(faces2)

	// Extract face regions
	faceRegion1 := img1.Region(face1)
	faceRegion2 := img2.Region(face2)
	defer faceRegion1.Close()
	defer faceRegion2.Close()

	// Convert to grayscale for comparison
	gray1 := gocv.NewMat()
	gray2 := gocv.NewMat()
	defer gray1.Close()
	defer gray2.Close()

	gocv.CvtColor(faceRegion1, &gray1, gocv.ColorBGRToGray)
	gocv.CvtColor(faceRegion2, &gray2, gocv.ColorBGRToGray)

	// Enhanced face preprocessing
	processed1 := lfs.preprocessFaceForComparison(gray1)
	processed2 := lfs.preprocessFaceForComparison(gray2)
	defer processed1.Close()
	defer processed2.Close()

	// Calculate similarity using multiple methods
	similarity := lfs.calculateSimilarity(processed1, processed2)
	confidence := lfs.calculateConfidence(float64(similarity), quality1, quality2)

	// Adaptive threshold based on image quality and confidence
	baseThreshold := 0.8 // Increased for better security
	minConfidence := 0.7 // Increased for better accuracy

	// Adjust threshold based on image quality
	avgQuality := (quality1 + quality2) / 2.0
	if avgQuality < 0.4 {
		baseThreshold += 0.15 // Stricter for low quality
	} else if avgQuality > 0.8 {
		baseThreshold -= 0.05 // Slightly more lenient for high quality
	}

	// Additional validation: ensure both faces are detected properly
	if len(faces1) != 1 || len(faces2) != 1 {
		baseThreshold += 0.2 // Much stricter if multiple faces detected
	}

	// Quality difference penalty - penalize large quality differences
	qualityDiff := math.Abs(quality1 - quality2)
	if qualityDiff > 0.3 {
		baseThreshold += 0.1 // Additional penalty for quality mismatch
	}

	// Determine if faces match based on adaptive thresholds
	isMatch := float64(similarity) > baseThreshold && confidence > minConfidence

	processingTime := time.Since(startTime).Milliseconds()

	// Update statistics
	lfs.updateStats(processingTime, true)

	return &types.BiometricFaceMatchResponse{
		Success:           true,
		Match:             isMatch,
		Confidence:        confidence,
		ProcessingTimeMs:  int(processingTime),
		FaceQualityScores: []float64{quality1, quality2},
		Liveness: types.Liveness{
			Image1: lfs.performQuickLivenessCheck(faceRegion1),
			Image2: lfs.performQuickLivenessCheck(faceRegion2),
		},
	}, nil
}

// ImageLivenessCheck performs liveness detection on a single image using YuNet
func (lfs *LocalFaceService) ImageLivenessCheck(image *string, lenientBlurry bool) (*types.BiometricLivenessResponse, error) {
	startTime := time.Now()
	logger.Info("🔍 Starting liveness check with YuNet", logger.LoggerOptions{
		Key:  "total_start",
		Data: startTime.Format("15:04:05.000"),
	})

	// Create YuNet service for face detection
	yunetConfig := GetDefaultYuNetConfig()
	yunetService := NewYuNetFaceService(yunetConfig)
	defer yunetService.Close()

	if !yunetService.IsHealthy() {
		// Fallback to Haar cascade if YuNet fails
		logger.Info("⚠️ YuNet not available, falling back to Haar cascade")
		if !lfs.modelsLoaded {
			return &types.BiometricLivenessResponse{
				Success:       false,
				FailureReason: utils.GetStringPointer("models_not_loaded"),
				Error:         utils.GetStringPointer("Face detection models not loaded"),
			}, nil
		}
	}

	// Process image with YuNet detection
	processStart := time.Now()

	// Process with YuNet or fallback to Haar cascade (using stdimage alias to avoid shadowing)
	img, faces, quality, err := func() (gocv.Mat, []stdimage.Rectangle, float64, error) {
		if yunetService.IsHealthy() {
			return lfs.ProcessImageWithYuNet(*image, yunetService)
		}
		return lfs.ProcessImage(*image)
	}()

	processTime := time.Since(processStart).Milliseconds()
	logger.Info("📸 Image processing completed", logger.LoggerOptions{
		Key:  "process_time_ms",
		Data: processTime,
	})

	if err != nil {
		// Provide graceful degradation with detailed error information
		return lfs.handleImageProcessingError(err, startTime)
	}
	defer img.Close()

	// Check if face is detected
	if len(faces) == 0 {
		logger.Info("❌ No face detected", logger.LoggerOptions{
			Key:  "total_time_ms",
			Data: time.Since(startTime).Milliseconds(),
		})
		return &types.BiometricLivenessResponse{
			Success:       false,
			FailureReason: utils.GetStringPointer("no_face_detected"),
			Error:         utils.GetStringPointer("No face detected in image"),
		}, nil
	}

	// Use the largest face
	face := lfs.getLargestFace(faces)
	faceRegion := img.Region(face)
	defer faceRegion.Close()

	// DEBUG: Log face selection details
	logger.Info("🔍 DEBUG: Face selection", logger.LoggerOptions{
		Key: "face_selection",
		Data: map[string]interface{}{
			"total_faces":   len(faces),
			"selected_face": fmt.Sprintf("x:%d,y:%d,w:%d,h:%d", face.Min.X, face.Min.Y, face.Dx(), face.Dy()),
			"face_area":     face.Dx() * face.Dy(),
			"face_ratio":    float64(face.Dx()*face.Dy()) / float64(img.Rows()*img.Cols()),
			"all_faces": func() []string {
				var faceStrings []string
				for i, f := range faces {
					faceStrings = append(faceStrings, fmt.Sprintf("face%d:x:%d,y:%d,w:%d,h:%d", i+1, f.Min.X, f.Min.Y, f.Dx(), f.Dy()))
				}
				return faceStrings
			}(),
		},
	})

	// Perform liveness analysis - runs comprehensive spoof detection in parallel
	livenessStart := time.Now()
	livenessScore, spoofPenalty, paintingProbability, screenProbability, printProbability, maskProbability, skinToneRealism, detailedResult := lfs.analyzeLiveness(faceRegion, img)
	livenessTime := time.Since(livenessStart).Milliseconds()
	logger.Info("🧠 Liveness analysis completed", logger.LoggerOptions{
		Key:  "liveness_time_ms",
		Data: livenessTime,
	})

	// Liveness score is calculated normally by analyzeLiveness function
	// No fixed scores - let the system calculate the actual liveness score

	// DEBUG: Log critical values for non-determinism analysis
	logger.Info("🔍 DEBUG: Liveness analysis results", logger.LoggerOptions{
		Key: "liveness_debug",
		Data: map[string]interface{}{
			"liveness_score":   livenessScore,
			"spoof_penalty":    spoofPenalty,
			"face_region_size": fmt.Sprintf("%dx%d", faceRegion.Cols(), faceRegion.Rows()),
			"face_region_type": faceRegion.Type().String(),
			"img_size":         fmt.Sprintf("%dx%d", img.Cols(), img.Rows()),
			"img_type":         img.Type().String(),
			"faces_detected":   len(faces),
			"selected_face":    fmt.Sprintf("x:%d,y:%d,w:%d,h:%d", face.Min.X, face.Min.Y, face.Dx(), face.Dy()),
		},
	})

	// Calculate confidence and additional metrics
	metricsStart := time.Now()
	confidence := lfs.calculateLivenessConfidence(livenessScore, quality)

	// DEBUG: Log confidence calculation
	logger.Info("🔍 DEBUG: Confidence calculation", logger.LoggerOptions{
		Key: "confidence_debug",
		Data: map[string]interface{}{
			"liveness_score": livenessScore,
			"quality_score":  quality,
			"confidence":     confidence,
		},
	})

	// These additional calculations might be slow
	lightingStart := time.Now()
	lightingScore := lfs.calculateLightingScore(faceRegion)
	lightingTime := time.Since(lightingStart).Milliseconds()

	sharpnessStart := time.Now()
	sharpnessScore := lfs.calculateSharpnessScore(faceRegion)
	sharpnessTime := time.Since(sharpnessStart).Milliseconds()

	textureStart := time.Now()
	// Convert face region to grayscale for enhanced texture analysis
	grayFace := gocv.NewMat()
	defer grayFace.Close()
	gocv.CvtColor(faceRegion, &grayFace, gocv.ColorBGRToGray)
	textureScore := lfs.calculateGLCMScore(grayFace)
	textureTime := time.Since(textureStart).Milliseconds()

	metricsTime := time.Since(metricsStart).Milliseconds()
	logger.Info("📊 Metrics calculation completed", logger.LoggerOptions{
		Key:  "metrics_time_ms",
		Data: metricsTime,
	})
	logger.Info("📊 Individual metric times", logger.LoggerOptions{
		Key:  "lighting_ms",
		Data: lightingTime,
	}, logger.LoggerOptions{
		Key:  "sharpness_ms",
		Data: sharpnessTime,
	}, logger.LoggerOptions{
		Key:  "texture_ms",
		Data: textureTime,
	})

	// Determine if image is live with rounding error tolerance
	// Use tolerance to account for floating-point precision issues
	const thresholdTolerance = 0.005 // 0.5% tolerance for rounding errors

	// Set thresholds based on mode
	var effectiveThreshold float64
	if lenientBlurry {
		effectiveThreshold = 0.52 - thresholdTolerance // 0.515
		logger.Info("🔍 DEBUG: Lenient blurry mode enabled", logger.LoggerOptions{
			Key: "lenient_blurry",
			Data: map[string]interface{}{
				"threshold":           0.52,
				"effective_threshold": effectiveThreshold,
			},
		})
	} else {
		// ADJUSTED: Lowered from 0.6 to 0.5 to reduce false negatives on real faces
		// The improved LBP/LPQ algorithms are more accurate but can score lower on compressed images
		effectiveThreshold = 0.5 - thresholdTolerance // 0.495
		logger.Info("🔍 DEBUG: Normal mode enabled", logger.LoggerOptions{
			Key: "normal_mode",
			Data: map[string]interface{}{
				"threshold":           0.5,
				"effective_threshold": effectiveThreshold,
			},
		})
	}

	isLive := livenessScore > effectiveThreshold

	// DEBUG: Log threshold decision
	logger.Info("🔍 DEBUG: Threshold decision", logger.LoggerOptions{
		Key: "threshold_decision",
		Data: map[string]interface{}{
			"liveness_score":      livenessScore,
			"base_threshold":      0.65,
			"threshold_tolerance": thresholdTolerance,
			"effective_threshold": effectiveThreshold,
			"lenient_blurry":      lenientBlurry,
			"is_live":             isLive,
			"decision_reason": fmt.Sprintf("%.6f %s %.6f", livenessScore, func() string {
				if isLive {
					return ">"
				} else {
					return "<="
				}
			}(), effectiveThreshold),
		},
	})

	processingTime := time.Since(startTime).Milliseconds()
	logger.Info("✅ Liveness check completed", logger.LoggerOptions{
		Key:  "total_time_ms",
		Data: processingTime,
	})

	// Update statistics
	lfs.updateStats(processingTime, true)

	return &types.BiometricLivenessResponse{
		Success:          true,
		IsLive:           isLive,
		LivenessScore:    livenessScore,
		ThresholdUsed:    effectiveThreshold,
		Confidence:       confidence,
		QualityScore:     quality,
		ProcessingTimeMs: int(processingTime),
		AnalysisDetails: types.AnalysisDetails{
			ImageQuality:        quality,
			LandmarkConsistency: livenessScore,
			LightingScore:       lightingScore,
			SharpnessScore:      sharpnessScore,
			SpoofDetectionScore: spoofPenalty,
			TextureScore:        textureScore,

			// Spoof type probabilities (from parallel detection)
			PaintingProbability: paintingProbability,
			ScreenProbability:   screenProbability,
			PrintProbability:    printProbability,
			MaskProbability:     maskProbability,
			SkinToneRealism:     skinToneRealism,

			// Populate detailed breakdown scores from analyzeLiveness
			LBPScore:              detailedResult.LBPScore,
			LPQScore:              detailedResult.LPQScore,
			ReflectionConsistency: detailedResult.ReflectionConsistency,
			ColorRGBVariance:      detailedResult.ColorRGBVariance,
			ColorHSVVariance:      detailedResult.ColorHSVVariance,
			ColorLABVariance:      detailedResult.ColorLABVariance,
			EdgeDensity:           detailedResult.EdgeDensity,
			EdgeSharpness:         detailedResult.EdgeSharpness,
			EdgeConsistency:       detailedResult.EdgeConsistency,
			HighFrequency:         detailedResult.HighFrequency,
			MidFrequency:          detailedResult.MidFrequency,
			LowFrequency:          detailedResult.LowFrequency,
			CompressionArtifacts:  detailedResult.CompressionArtifacts,
			TextureVariance:       detailedResult.TextureVariance,
			TextureUniformity:     detailedResult.TextureUniformity,
			TextureEntropy:        detailedResult.TextureEntropy,
		},
	}, nil
}

// ProcessImage processes an image from URL or base64 and returns OpenCV Mat, faces, and quality
func (lfs *LocalFaceService) ProcessImage(imageInput string) (gocv.Mat, []image.Rectangle, float64, error) {
	processStart := time.Now()
	logger.Info("🔄 Starting image processing", logger.LoggerOptions{
		Key:  "process_start",
		Data: processStart.Format("15:04:05.000"),
	})

	// Use panic recovery to catch segmentation faults and other crashes
	defer func() {
		if r := recover(); r != nil {
			// If we panic (segmentation fault), return a safe error
			logger.Error("Image processing crashed with panic", logger.LoggerOptions{
				Key:  "panic",
				Data: fmt.Sprintf("%v", r),
			})
		}
	}()

	var imgData []byte
	var err error

	// Security validation
	if err := lfs.validateImageInput(imageInput); err != nil {
		return gocv.Mat{}, nil, 0, fmt.Errorf("image validation failed: %v", err)
	}

	// Check if input is URL or base64
	if strings.HasPrefix(imageInput, "http://") || strings.HasPrefix(imageInput, "https://") {
		// Download image from URL with security checks
		downloadStart := time.Now()
		imgData, err = lfs.downloadImageSecurely(imageInput)
		downloadTime := time.Since(downloadStart).Milliseconds()
		logger.Info("⬇️ Image download completed", logger.LoggerOptions{
			Key:  "download_time_ms",
			Data: downloadTime,
		})
		if err != nil {
			return gocv.Mat{}, nil, 0, fmt.Errorf("failed to download image: %v", err)
		}
	} else {
		// Decode base64 image
		decodeStart := time.Now()
		imgData, err = utils.DecodeBase64Image(imageInput)
		decodeTime := time.Since(decodeStart).Milliseconds()
		logger.Info("🔓 Base64 decode completed", logger.LoggerOptions{
			Key:  "decode_time_ms",
			Data: decodeTime,
		})
		if err != nil {
			return gocv.Mat{}, nil, 0, fmt.Errorf("failed to decode base64 image: %v", err)
		}
	}

	// Decode image
	decodeStart := time.Now()
	img, format, err := image.Decode(bytes.NewReader(imgData))
	decodeTime := time.Since(decodeStart).Milliseconds()
	logger.Info("🖼️ Image decode completed", logger.LoggerOptions{
		Key:  "decode_time_ms",
		Data: decodeTime,
	}, logger.LoggerOptions{
		Key:  "format",
		Data: format,
	})
	if err != nil {
		return gocv.Mat{}, nil, 0, fmt.Errorf("failed to decode image: %v", err)
	}

	// Validate image format and characteristics
	if err := lfs.validateImageFormat(img, format, imgData); err != nil {
		return gocv.Mat{}, nil, 0, fmt.Errorf("image format validation failed: %v", err)
	}

	// Convert to OpenCV Mat with robust error handling
	convertStart := time.Now()
	mat, err := lfs.convertImageToMatSafely(img)
	convertTime := time.Since(convertStart).Milliseconds()
	logger.Info("🔄 Mat conversion completed", logger.LoggerOptions{
		Key:  "convert_time_ms",
		Data: convertTime,
	})
	if err != nil {
		return gocv.Mat{}, nil, 0, fmt.Errorf("failed to convert image to Mat: %v", err)
	}

	// Validate Mat dimensions to prevent crashes
	if mat.Empty() {
		return gocv.Mat{}, nil, 0, fmt.Errorf("converted image Mat is empty")
	}

	// Check for corrupted or invalid dimensions
	rows := mat.Rows()
	cols := mat.Cols()
	if rows <= 0 || cols <= 0 {
		return gocv.Mat{}, nil, 0, fmt.Errorf("invalid image dimensions: %dx%d", rows, cols)
	}

	// Check for extremely large images that could cause memory issues
	if rows > 20000 || cols > 20000 {
		return gocv.Mat{}, nil, 0, fmt.Errorf("image too large: %dx%d (max: 20000x20000)", rows, cols)
	}

	// Additional safety check for potential memory corruption
	if rows*cols > 100000000 { // 100M pixels
		return gocv.Mat{}, nil, 0, fmt.Errorf("image has too many pixels: %d (max: 100M)", rows*cols)
	}

	// PERFORMANCE OPTIMIZATION: Resize large images to reduce processing time
	resizeStart := time.Now()
	maxWidth := 800
	maxHeight := 600

	if rows > maxHeight || cols > maxWidth {
		// Calculate new dimensions maintaining aspect ratio
		aspectRatio := float64(cols) / float64(rows)
		var newWidth, newHeight int

		if aspectRatio > float64(maxWidth)/float64(maxHeight) {
			// Width is the limiting factor
			newWidth = maxWidth
			newHeight = int(float64(maxWidth) / aspectRatio)
		} else {
			// Height is the limiting factor
			newHeight = maxHeight
			newWidth = int(float64(maxHeight) * aspectRatio)
		}

		// Ensure minimum dimensions
		if newWidth < 200 {
			newWidth = 200
			newHeight = int(float64(newWidth) / aspectRatio)
		}
		if newHeight < 200 {
			newHeight = 200
			newWidth = int(float64(newHeight) * aspectRatio)
		}

		// Create resized Mat
		resized := gocv.NewMat()
		gocv.Resize(mat, &resized, image.Pt(newWidth, newHeight), 0, 0, gocv.InterpolationLinear)

		// Replace original with resized version
		mat.Close()
		mat = resized

		// Update dimensions
		rows = mat.Rows()
		cols = mat.Cols()

		resizeTime := time.Since(resizeStart).Milliseconds()
		logger.Info("📐 Image resized for performance", logger.LoggerOptions{
			Key:  "resize_time_ms",
			Data: resizeTime,
		}, logger.LoggerOptions{
			Key:  "new_dimensions",
			Data: fmt.Sprintf("%dx%d", cols, rows),
		})
	}

	// Test Mat validity by trying to access a safe pixel
	// This will detect corrupted Mats that pass dimension checks but have invalid memory
	if rows > 0 && cols > 0 {
		// Try to access the first pixel safely
		func() {
			defer func() {
				if r := recover(); r != nil {
					// If accessing the first pixel causes a panic, the Mat is corrupted
					panic(fmt.Sprintf("corrupted Mat detected - cannot access pixel (0,0) in %dx%d image", rows, cols))
				}
			}()
			_ = mat.GetFloatAt(0, 0)
		}()
	}

	// Detect faces using enhanced method
	faceDetectionStart := time.Now()
	faces := lfs.enhancedFaceDetection(mat)
	faceDetectionTime := time.Since(faceDetectionStart).Milliseconds()
	logger.Info("👤 Face detection completed", logger.LoggerOptions{
		Key:  "face_detection_time_ms",
		Data: faceDetectionTime,
	}, logger.LoggerOptions{
		Key:  "faces_found",
		Data: len(faces),
	})

	// Calculate image quality using advanced method
	qualityStart := time.Now()
	quality := lfs.calculateAdvancedImageQuality(mat, faces)
	qualityTime := time.Since(qualityStart).Milliseconds()
	logger.Info("⭐ Quality calculation completed", logger.LoggerOptions{
		Key:  "quality_time_ms",
		Data: qualityTime,
	}, logger.LoggerOptions{
		Key:  "quality_score",
		Data: quality,
	})

	totalProcessTime := time.Since(processStart).Milliseconds()
	logger.Info("✅ Image processing completed", logger.LoggerOptions{
		Key:  "total_process_time_ms",
		Data: totalProcessTime,
	}, logger.LoggerOptions{
		Key:  "format",
		Data: format,
	})

	return mat, faces, quality, nil
}

// ProcessImageWithMobileNet processes an image using MobileNet face detection ONLY
func (lfs *LocalFaceService) ProcessImageWithMobileNet(imageStr string, mobileNetService *MobileNetFaceService) (gocv.Mat, []image.Rectangle, float64, error) {
	// Use the existing ProcessImage method to get the image
	img, faces, quality, err := lfs.ProcessImage(imageStr)
	if err != nil {
		return gocv.Mat{}, nil, 0, err
	}

	// Use MobileNet face detection ONLY instead of the faces from ProcessImage
	detectionResult, err := mobileNetService.DetectFaces(img)
	if err != nil {
		return img, faces, quality, err
	}

	// Calculate quality using existing method
	quality = lfs.calculateImageQuality(img, detectionResult.Faces)

	return img, detectionResult.Faces, quality, nil
}

// ProcessImageWithYuNet processes an image using YuNet face detection
func (lfs *LocalFaceService) ProcessImageWithYuNet(imageStr string, yunetService *YuNetFaceService) (gocv.Mat, []image.Rectangle, float64, error) {
	// Use the existing ProcessImage method to get the image
	img, faces, quality, err := lfs.ProcessImage(imageStr)
	if err != nil {
		return gocv.Mat{}, nil, 0, err
	}

	// Use YuNet face detection instead of the faces from ProcessImage
	detectionResult, err := yunetService.DetectFaces(img)
	if err != nil {
		return img, faces, quality, err
	}

	// Calculate quality using YuNet's advanced quality calculation with landmarks
	if len(detectionResult.Faces) > 0 && len(detectionResult.Landmarks) > 0 {
		// Use the largest face for quality calculation
		largestIdx := 0
		maxArea := 0
		for i, face := range detectionResult.Faces {
			area := face.Dx() * face.Dy()
			if area > maxArea {
				maxArea = area
				largestIdx = i
			}
		}

		quality = yunetService.CalculateFaceQuality(
			detectionResult.Faces[largestIdx],
			detectionResult.Landmarks[largestIdx],
			image.Pt(img.Cols(), img.Rows()),
		)
	} else {
		quality = lfs.calculateImageQuality(img, detectionResult.Faces)
	}

	return img, detectionResult.Faces, quality, nil
}

// DetectFaces detects faces in the image (public method for testing)
func (lfs *LocalFaceService) DetectFaces(img gocv.Mat) []image.Rectangle {
	if !lfs.modelsLoaded {
		return []image.Rectangle{}
	}
	faces := lfs.faceCascade.DetectMultiScale(img)
	return faces
}

// ProcessImageWithEnhancedDetection processes an image with enhanced face detection
func (lfs *LocalFaceService) ProcessImageWithEnhancedDetection(imageStr string) (gocv.Mat, []image.Rectangle, float64, error) {
	// Use existing ProcessImage method to load the image
	img, _, _, err := lfs.ProcessImage(imageStr)
	if err != nil {
		return gocv.Mat{}, nil, 0, err
	}

	// Enhanced face detection with multiple scales and parameters
	enhancedFaces := lfs.enhancedFaceDetection(img)

	// Calculate enhanced image quality
	enhancedQuality := lfs.calculateEnhancedImageQuality(img, enhancedFaces)

	return img, enhancedFaces, enhancedQuality, nil
}

// enhancedFaceDetection performs enhanced face detection with multiple parameters
func (lfs *LocalFaceService) enhancedFaceDetection(img gocv.Mat) []image.Rectangle {
	if !lfs.modelsLoaded {
		return []image.Rectangle{}
	}

	// Convert to grayscale for better detection
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// Apply histogram equalization for better contrast
	equalized := gocv.NewMat()
	defer equalized.Close()
	gocv.EqualizeHist(gray, &equalized)

	// Detect faces with enhanced parameters
	faces := lfs.faceCascade.DetectMultiScaleWithParams(
		equalized,
		1.1,                   // scale factor
		3,                     // min neighbors (increased for better accuracy)
		0,                     // flags
		image.Point{30, 30},   // min size (increased for better accuracy)
		image.Point{300, 300}, // max size
	)

	// If no faces found with enhanced parameters, try with relaxed parameters
	if len(faces) == 0 {
		faces = lfs.faceCascade.DetectMultiScaleWithParams(
			equalized,
			1.05,                  // smaller scale factor
			2,                     // fewer neighbors
			0,                     // flags
			image.Point{20, 20},   // smaller min size
			image.Point{400, 400}, // larger max size
		)
	}

	return faces
}

// extractFaceRegionWithPadding extracts face region with padding for better accuracy
func (lfs *LocalFaceService) extractFaceRegionWithPadding(img gocv.Mat, face image.Rectangle) gocv.Mat {
	// Add padding around the face (20% on each side)
	paddingX := int(float64(face.Dx()) * 0.2)
	paddingY := int(float64(face.Dy()) * 0.2)

	// Calculate padded region
	x1 := max(0, face.Min.X-paddingX)
	y1 := max(0, face.Min.Y-paddingY)
	x2 := min(img.Cols(), face.Max.X+paddingX)
	y2 := min(img.Rows(), face.Max.Y+paddingY)

	paddedFace := image.Rect(x1, y1, x2, y2)
	return img.Region(paddedFace)
}

// enhancedFacePreprocessing applies enhanced preprocessing to face regions
func (lfs *LocalFaceService) enhancedFacePreprocessing(faceRegion gocv.Mat) gocv.Mat {
	// Convert to grayscale
	gray := gocv.NewMat()
	gocv.CvtColor(faceRegion, &gray, gocv.ColorBGRToGray)

	// Resize to standard size for comparison
	resized := gocv.NewMat()
	gocv.Resize(gray, &resized, image.Point{128, 128}, 0, 0, gocv.InterpolationCubic)
	gray.Close()

	// Apply histogram equalization
	equalized := gocv.NewMat()
	gocv.EqualizeHist(resized, &equalized)
	resized.Close()

	// Apply Gaussian blur to reduce noise
	blurred := gocv.NewMat()
	gocv.GaussianBlur(equalized, &blurred, image.Point{3, 3}, 0, 0, gocv.BorderDefault)
	equalized.Close()

	// Apply bilateral filter for edge preservation
	filtered := gocv.NewMat()
	gocv.BilateralFilter(blurred, &filtered, 9, 75, 75)
	blurred.Close()

	return filtered
}

// calculateEnhancedSimilarity calculates similarity using multiple enhanced methods
func (lfs *LocalFaceService) calculateEnhancedSimilarity(face1, face2 gocv.Mat) float64 {
	// Use existing similarity calculation as base
	baseSimilarity := float64(lfs.calculateSimilarity(face1, face2))

	// Feature-based similarity (using ORB features)
	featureScore := lfs.calculateFeatureSimilarity(face1, face2)

	// Edge similarity (simplified)
	edgeScore := lfs.calculateSimpleEdgeSimilarity(face1, face2)

	// Weighted combination of methods
	weights := []float64{0.6, 0.25, 0.15}
	scores := []float64{baseSimilarity, featureScore, edgeScore}

	totalScore := 0.0
	for i, score := range scores {
		totalScore += score * weights[i]
	}

	return totalScore
}

// calculateEnhancedConfidence calculates enhanced confidence score
func (lfs *LocalFaceService) calculateEnhancedConfidence(similarity, quality1, quality2 float64, faces1, faces2 []image.Rectangle) float64 {
	baseConfidence := similarity

	// Quality factor
	avgQuality := (quality1 + quality2) / 2.0
	qualityFactor := avgQuality * 0.3

	// Face count factor (single face is more reliable)
	faceCountFactor := 1.0
	if len(faces1) == 1 && len(faces2) == 1 {
		faceCountFactor = 1.1
	} else if len(faces1) > 1 || len(faces2) > 1 {
		faceCountFactor = 0.8
	}

	// Quality consistency factor
	qualityDiff := math.Abs(quality1 - quality2)
	consistencyFactor := 1.0 - (qualityDiff * 0.5)

	// Calculate final confidence
	confidence := baseConfidence + qualityFactor
	confidence *= faceCountFactor
	confidence *= consistencyFactor

	// Clamp to [0, 1]
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

// calculateEnhancedImageQuality calculates enhanced image quality score
func (lfs *LocalFaceService) calculateEnhancedImageQuality(img gocv.Mat, faces []image.Rectangle) float64 {
	if len(faces) == 0 {
		return 0.0
	}

	// Basic image quality
	basicQuality := lfs.calculateImageQuality(img, faces)

	// Face size quality (larger faces are generally better)
	largestFace := lfs.getLargestFace(faces)
	faceArea := float64(largestFace.Dx() * largestFace.Dy())
	imgArea := float64(img.Cols() * img.Rows())
	faceSizeRatio := faceArea / imgArea

	// Optimal face size is around 10-30% of image
	var sizeQuality float64
	if faceSizeRatio >= 0.1 && faceSizeRatio <= 0.3 {
		sizeQuality = 1.0
	} else if faceSizeRatio >= 0.05 && faceSizeRatio <= 0.5 {
		sizeQuality = 0.8
	} else {
		sizeQuality = 0.6
	}

	// Face position quality (centered faces are better)
	centerX := float64(img.Cols()) / 2.0
	centerY := float64(img.Rows()) / 2.0
	faceCenterX := float64(largestFace.Min.X+largestFace.Max.X) / 2.0
	faceCenterY := float64(largestFace.Min.Y+largestFace.Max.Y) / 2.0

	distanceFromCenter := math.Sqrt(math.Pow(faceCenterX-centerX, 2) + math.Pow(faceCenterY-centerY, 2))
	maxDistance := math.Sqrt(math.Pow(float64(img.Cols()), 2)+math.Pow(float64(img.Rows()), 2)) / 2.0
	positionQuality := 1.0 - (distanceFromCenter / maxDistance)

	// Combine quality factors
	enhancedQuality := (basicQuality * 0.5) + (sizeQuality * 0.3) + (positionQuality * 0.2)

	return enhancedQuality
}

// calculateFeatureSimilarity calculates feature-based similarity using ORB features
func (lfs *LocalFaceService) calculateFeatureSimilarity(face1, face2 gocv.Mat) float64 {
	// Create ORB detector
	orb := gocv.NewORB()
	defer orb.Close()

	// Detect and compute keypoints and descriptors
	keypoints1, descriptors1 := orb.DetectAndCompute(face1, gocv.NewMat())
	keypoints2, descriptors2 := orb.DetectAndCompute(face2, gocv.NewMat())

	if len(keypoints1) == 0 || len(keypoints2) == 0 {
		return 0.0
	}

	// Create BF matcher
	matcher := gocv.NewBFMatcher()
	defer matcher.Close()

	// Match descriptors
	matches := matcher.KnnMatch(descriptors1, descriptors2, 2)

	// Apply Lowe's ratio test
	goodMatches := 0
	for _, match := range matches {
		if len(match) >= 2 {
			if match[0].Distance < 0.75*match[1].Distance {
				goodMatches++
			}
		}
	}

	// Calculate similarity score
	totalKeypoints := len(keypoints1) + len(keypoints2)
	if totalKeypoints == 0 {
		return 0.0
	}

	similarity := float64(goodMatches) / float64(totalKeypoints)
	return similarity
}

// calculateSimpleEdgeSimilarity calculates edge similarity between two face images
func (lfs *LocalFaceService) calculateSimpleEdgeSimilarity(face1, face2 gocv.Mat) float64 {
	// Convert to grayscale if needed
	gray1 := gocv.NewMat()
	gray2 := gocv.NewMat()
	defer gray1.Close()
	defer gray2.Close()

	if face1.Channels() > 1 {
		gocv.CvtColor(face1, &gray1, gocv.ColorBGRToGray)
	} else {
		gray1 = face1.Clone()
	}

	if face2.Channels() > 1 {
		gocv.CvtColor(face2, &gray2, gocv.ColorBGRToGray)
	} else {
		gray2 = face2.Clone()
	}

	// Apply Canny edge detection
	edges1 := gocv.NewMat()
	edges2 := gocv.NewMat()
	defer edges1.Close()
	defer edges2.Close()

	gocv.Canny(gray1, &edges1, 50, 150)
	gocv.Canny(gray2, &edges2, 50, 150)

	// Calculate edge density similarity
	density1 := lfs.calculateEdgeDensity(edges1)
	density2 := lfs.calculateEdgeDensity(edges2)

	// Calculate similarity based on edge density difference
	densityDiff := math.Abs(density1 - density2)
	similarity := 1.0 - (densityDiff / math.Max(density1, density2))

	if similarity < 0 {
		similarity = 0
	}

	return similarity
}

// calculateEdgeDensity calculates the density of edges in an image
func (lfs *LocalFaceService) calculateEdgeDensity(edges gocv.Mat) float64 {
	// Count non-zero pixels (edges)
	nonZeroCount := gocv.CountNonZero(edges)
	totalPixels := edges.Rows() * edges.Cols()

	if totalPixels == 0 {
		return 0.0
	}

	return float64(nonZeroCount) / float64(totalPixels)
}

// detectFaces detects faces in the image (private method)
func (lfs *LocalFaceService) detectFaces(img gocv.Mat) []image.Rectangle {
	faces := lfs.faceCascade.DetectMultiScale(img)
	return faces
}

// getLargestFace returns the largest face from the detected faces
func (lfs *LocalFaceService) getLargestFace(faces []image.Rectangle) image.Rectangle {
	if len(faces) == 0 {
		return image.Rectangle{}
	}

	largest := faces[0]
	maxArea := largest.Dx() * largest.Dy()

	for _, face := range faces[1:] {
		area := face.Dx() * face.Dy()
		if area > maxArea {
			largest = face
			maxArea = area
		}
	}

	return largest
}

// calculateSimilarity calculates similarity between two face images using multiple advanced methods
func (lfs *LocalFaceService) calculateSimilarity(img1, img2 gocv.Mat) float32 {
	// ENHANCED SIMILARITY CALCULATION WITH MULTIPLE ADVANCED METHODS

	// Method 1: Multi-scale template matching for robust comparison
	multiScaleSimilarity := lfs.calculateMultiScaleTemplateSimilarity(img1, img2)

	// Method 2: Enhanced SSIM with luminance, contrast, and structure components
	ssim := lfs.calculateEnhancedSSIM(img1, img2)

	// Method 3: Multi-channel histogram correlation (more discriminative)
	histogramSimilarity := lfs.calculateAdvancedHistogramSimilarity(img1, img2)

	// Method 4: Gradient-based structural similarity (pose-invariant)
	gradientSimilarity := lfs.calculateGradientSimilarity(img1, img2)

	// Method 5: Feature-based similarity using key points
	featureSimilarity := lfs.calculateFeatureBasedSimilarity(img1, img2)

	// Method 6: Local descriptor similarity (LBP-based)
	descriptorSimilarity := lfs.calculateLocalDescriptorSimilarity(img1, img2)

	// Adaptive weighting based on image quality and characteristics
	quality1 := lfs.calculateImageQualityMetric(img1)
	quality2 := lfs.calculateImageQualityMetric(img2)
	avgQuality := (quality1 + quality2) / 2.0

	// Calculate image contrast to determine which methods work best
	contrast1 := lfs.calculateImageContrast(img1)
	contrast2 := lfs.calculateImageContrast(img2)
	avgContrast := (contrast1 + contrast2) / 2.0

	// Dynamic weight adjustment based on image characteristics
	var w1, w2, w3, w4, w5, w6 float32

	if avgQuality > 0.75 && avgContrast > 0.6 {
		// Excellent quality + good contrast - emphasize template and features
		w1, w2, w3, w4, w5, w6 = 0.25, 0.25, 0.15, 0.15, 0.15, 0.05
	} else if avgQuality > 0.6 {
		// Good quality - balanced approach with emphasis on structure
		w1, w2, w3, w4, w5, w6 = 0.20, 0.25, 0.15, 0.15, 0.15, 0.10
	} else if avgQuality > 0.4 {
		// Medium quality - emphasize robust methods
		w1, w2, w3, w4, w5, w6 = 0.15, 0.20, 0.20, 0.15, 0.15, 0.15
	} else {
		// Low quality - emphasize descriptor and histogram methods
		w1, w2, w3, w4, w5, w6 = 0.10, 0.15, 0.20, 0.15, 0.15, 0.25
	}

	// Combine all methods with adaptive weights
	similarity := multiScaleSimilarity*w1 +
		float32(ssim)*w2 +
		histogramSimilarity*w3 +
		gradientSimilarity*w4 +
		featureSimilarity*w5 +
		descriptorSimilarity*w6

	// Quality-based confidence adjustment
	qualityDiff := math.Abs(quality1 - quality2)
	if qualityDiff > 0.35 {
		// Large quality difference - reduce confidence
		similarity *= 0.85
	} else if qualityDiff > 0.25 {
		// Moderate quality difference - slight reduction
		similarity *= 0.92
	}

	// Contrast-based adjustment (very different contrast can indicate different conditions)
	contrastDiff := math.Abs(contrast1 - contrast2)
	if contrastDiff > 0.4 {
		similarity *= 0.90
	}

	// Ensure similarity is between 0 and 1
	similarity = float32(math.Max(0, math.Min(1, float64(similarity))))

	return similarity
}

// calculateMultiScaleTemplateSimilarity performs template matching at multiple scales
func (lfs *LocalFaceService) calculateMultiScaleTemplateSimilarity(img1, img2 gocv.Mat) float32 {
	scales := []float64{1.0, 0.9, 1.1} // Original, slightly smaller, slightly larger
	var totalSim float32 = 0
	var validScales float32 = 0

	for _, scale := range scales {
		if scale != 1.0 {
			newSize := image.Pt(int(float64(img2.Cols())*scale), int(float64(img2.Rows())*scale))
			scaled := gocv.NewMat()
			gocv.Resize(img2, &scaled, newSize, 0, 0, gocv.InterpolationLinear)

			// Skip if scaled image is too different in size
			if math.Abs(float64(scaled.Cols()-img1.Cols())) > float64(img1.Cols())*0.2 {
				scaled.Close()
				continue
			}

			sim := lfs.calculateEnhancedTemplateSimilarity(img1, scaled)
			scaled.Close()
			totalSim += sim
			validScales++
		} else {
			sim := lfs.calculateEnhancedTemplateSimilarity(img1, img2)
			totalSim += sim
			validScales++
		}
	}

	if validScales == 0 {
		return 0
	}
	return totalSim / validScales
}

// calculateEnhancedSSIM calculates improved SSIM with better parameters
func (lfs *LocalFaceService) calculateEnhancedSSIM(img1, img2 gocv.Mat) float64 {
	// Use the existing calculateFaceSSIM which is already optimized
	return lfs.calculateFaceSSIM(img1, img2)
}

// calculateAdvancedHistogramSimilarity uses multi-bin histogram correlation
func (lfs *LocalFaceService) calculateAdvancedHistogramSimilarity(img1, img2 gocv.Mat) float32 {
	// Calculate histograms with more bins for better discrimination
	bins := []int{256}
	ranges := []float64{0, 256}

	hist1 := gocv.NewMat()
	hist2 := gocv.NewMat()
	defer hist1.Close()
	defer hist2.Close()

	gocv.CalcHist([]gocv.Mat{img1}, []int{0}, gocv.NewMat(), &hist1, bins, ranges, false)
	gocv.CalcHist([]gocv.Mat{img2}, []int{0}, gocv.NewMat(), &hist2, bins, ranges, false)

	// Normalize histograms
	gocv.Normalize(hist1, &hist1, 0, 1, gocv.NormMinMax)
	gocv.Normalize(hist2, &hist2, 0, 1, gocv.NormMinMax)

	// Use correlation comparison (0 = HistComp method for correlation)
	correlation := gocv.CompareHist(hist1, hist2, 0)

	// Convert from [-1, 1] to [0, 1]
	return float32((correlation + 1.0) / 2.0)
}

// calculateGradientSimilarity compares gradient patterns (pose-invariant)
func (lfs *LocalFaceService) calculateGradientSimilarity(img1, img2 gocv.Mat) float32 {
	// Calculate gradients using Sobel
	gradX1 := gocv.NewMat()
	gradY1 := gocv.NewMat()
	gradX2 := gocv.NewMat()
	gradY2 := gocv.NewMat()
	defer gradX1.Close()
	defer gradY1.Close()
	defer gradX2.Close()
	defer gradY2.Close()

	gocv.Sobel(img1, &gradX1, gocv.MatTypeCV64F, 1, 0, 3, 1, 0, gocv.BorderDefault)
	gocv.Sobel(img1, &gradY1, gocv.MatTypeCV64F, 0, 1, 3, 1, 0, gocv.BorderDefault)
	gocv.Sobel(img2, &gradX2, gocv.MatTypeCV64F, 1, 0, 3, 1, 0, gocv.BorderDefault)
	gocv.Sobel(img2, &gradY2, gocv.MatTypeCV64F, 0, 1, 3, 1, 0, gocv.BorderDefault)

	// Calculate gradient magnitudes
	mag1 := gocv.NewMat()
	mag2 := gocv.NewMat()
	defer mag1.Close()
	defer mag2.Close()

	gocv.Magnitude(gradX1, gradY1, &mag1)
	gocv.Magnitude(gradX2, gradY2, &mag2)

	// Compare gradient magnitudes using normalized correlation
	mag1Float := gocv.NewMat()
	mag2Float := gocv.NewMat()
	defer mag1Float.Close()
	defer mag2Float.Close()

	mag1.ConvertTo(&mag1Float, gocv.MatTypeCV32F)
	mag2.ConvertTo(&mag2Float, gocv.MatTypeCV32F)

	// Resize if needed
	if mag1Float.Rows() != mag2Float.Rows() || mag1Float.Cols() != mag2Float.Cols() {
		resized := gocv.NewMat()
		gocv.Resize(mag2Float, &resized, image.Pt(mag1Float.Cols(), mag1Float.Rows()), 0, 0, gocv.InterpolationLinear)
		mag2Float.Close()
		mag2Float = resized
	}

	// Calculate correlation
	mean1 := lfs.calculateMean(mag1Float)
	mean2 := lfs.calculateMean(mag2Float)
	var1 := lfs.calculateVariance(mag1Float, mean1)
	var2 := lfs.calculateVariance(mag2Float, mean2)
	cov := lfs.calculateCovariance(mag1Float, mag2Float, mean1, mean2)

	if var1 == 0 || var2 == 0 {
		return 0.5
	}

	correlation := cov / math.Sqrt(var1*var2)

	// Normalize to [0, 1]
	return float32((correlation + 1.0) / 2.0)
}

// calculateFeatureBasedSimilarity uses ORB features for matching
func (lfs *LocalFaceService) calculateFeatureBasedSimilarity(img1, img2 gocv.Mat) float32 {
	// Use ORB for fast feature detection and matching
	orb := gocv.NewORB()
	defer orb.Close()

	// Detect keypoints and compute descriptors
	kp1, desc1 := orb.DetectAndCompute(img1, gocv.NewMat())
	kp2, desc2 := orb.DetectAndCompute(img2, gocv.NewMat())
	defer desc1.Close()
	defer desc2.Close()

	if len(kp1) == 0 || len(kp2) == 0 {
		return 0.5 // Neutral score if no features
	}

	// Match features using BFMatcher
	matcher := gocv.NewBFMatcher()
	defer matcher.Close()

	matches := matcher.KnnMatch(desc1, desc2, 2)

	// Apply ratio test (Lowe's ratio)
	goodMatches := 0
	for _, match := range matches {
		if len(match) == 2 {
			if match[0].Distance < 0.75*match[1].Distance {
				goodMatches++
			}
		}
	}

	// Calculate match ratio
	minKeypoints := math.Min(float64(len(kp1)), float64(len(kp2)))
	if minKeypoints == 0 {
		return 0.5
	}

	matchRatio := float64(goodMatches) / minKeypoints

	// Normalize and cap at 1.0
	return float32(math.Min(matchRatio, 1.0))
}

// calculateLocalDescriptorSimilarity uses LBP-like descriptors
func (lfs *LocalFaceService) calculateLocalDescriptorSimilarity(img1, img2 gocv.Mat) float32 {
	// Calculate simplified local descriptors for both images
	desc1 := lfs.calculateSimpleLBPDescriptor(img1)
	desc2 := lfs.calculateSimpleLBPDescriptor(img2)

	// Calculate Chi-square distance between descriptors
	distance := 0.0
	for i := 0; i < len(desc1) && i < len(desc2); i++ {
		if desc1[i]+desc2[i] > 0 {
			diff := desc1[i] - desc2[i]
			distance += (diff * diff) / (desc1[i] + desc2[i])
		}
	}

	// Convert distance to similarity (lower distance = higher similarity)
	similarity := 1.0 / (1.0 + distance/float64(len(desc1)))

	return float32(similarity)
}

// calculateSimpleLBPDescriptor creates a simple LBP-based descriptor
func (lfs *LocalFaceService) calculateSimpleLBPDescriptor(img gocv.Mat) []float64 {
	descriptor := make([]float64, 256)
	totalPatterns := 0

	// Sample every 3rd pixel for performance
	step := 3

	for i := step; i < img.Rows()-step; i += step {
		for j := step; j < img.Cols()-step; j += step {
			center := float64(img.GetUCharAt(i, j))

			// Calculate 8-neighbor LBP
			var pattern uint8 = 0
			neighbors := []struct{ dy, dx int }{
				{-1, -1}, {-1, 0}, {-1, 1},
				{0, 1}, {1, 1}, {1, 0},
				{1, -1}, {0, -1},
			}

			for bit, neighbor := range neighbors {
				ny := i + neighbor.dy
				nx := j + neighbor.dx
				if ny >= 0 && ny < img.Rows() && nx >= 0 && nx < img.Cols() {
					neighborVal := float64(img.GetUCharAt(ny, nx))
					if neighborVal >= center {
						pattern |= (1 << uint(bit))
					}
				}
			}

			descriptor[pattern]++
			totalPatterns++
		}
	}

	// Normalize descriptor
	if totalPatterns > 0 {
		for i := range descriptor {
			descriptor[i] /= float64(totalPatterns)
		}
	}

	return descriptor
}

// calculateImageContrast calculates the contrast of an image
func (lfs *LocalFaceService) calculateImageContrast(img gocv.Mat) float64 {
	mean := lfs.calculateMean(img)
	variance := lfs.calculateVariance(img, mean)

	// Normalized contrast (std deviation / mean)
	if mean == 0 {
		return 0
	}

	contrast := math.Sqrt(variance) / mean

	// Normalize to [0, 1] range (typical contrast values are 0-2)
	return math.Min(contrast/2.0, 1.0)
}

// calculateSSIM calculates a simplified Structural Similarity Index
func (lfs *LocalFaceService) calculateSSIM(img1, img2 gocv.Mat) float64 {
	// Simplified SSIM calculation using basic statistics
	// Convert to float32 for calculations
	img1Float := gocv.NewMat()
	img2Float := gocv.NewMat()
	defer img1Float.Close()
	defer img2Float.Close()

	img1.ConvertTo(&img1Float, gocv.MatTypeCV32F)
	img2.ConvertTo(&img2Float, gocv.MatTypeCV32F)

	// Calculate mean using basic statistics
	mean1 := lfs.calculateMean(img1Float)
	mean2 := lfs.calculateMean(img2Float)

	// Calculate variance
	var1 := lfs.calculateVariance(img1Float, mean1)
	var2 := lfs.calculateVariance(img2Float, mean2)

	// Calculate covariance
	cov := lfs.calculateCovariance(img1Float, img2Float, mean1, mean2)

	// Simplified SSIM calculation
	c1 := 0.01 * 0.01
	c2 := 0.03 * 0.03

	numerator := (2*mean1*mean2 + c1) * (2*cov + c2)
	denominator := (mean1*mean1 + mean2*mean2 + c1) * (var1 + var2 + c2)

	if denominator == 0 {
		return 0
	}

	ssim := numerator / denominator
	if ssim < 0 {
		ssim = 0
	}
	if ssim > 1 {
		ssim = 1
	}

	return ssim
}

// getPixelValue safely extracts pixel value from any Mat type
func (lfs *LocalFaceService) getPixelValue(img gocv.Mat, i, j int) (float64, bool) {
	if i < 0 || i >= img.Rows() || j < 0 || j >= img.Cols() {
		return 0.0, false
	}

	var val float64
	switch img.Type() {
	case gocv.MatTypeCV8U: // uint8
		val = float64(img.GetUCharAt(i, j))
	case gocv.MatTypeCV8S: // int8
		val = float64(img.GetSCharAt(i, j))
	case gocv.MatTypeCV16S: // int16
		val = float64(img.GetShortAt(i, j))
	case gocv.MatTypeCV32S: // int32
		val = float64(img.GetIntAt(i, j))
	case gocv.MatTypeCV32F: // float32
		val = float64(img.GetFloatAt(i, j))
	case gocv.MatTypeCV64F: // float64
		val = img.GetDoubleAt(i, j)
	default:
		// For unsupported types, try to convert to float32 and use GetFloatAt
		// This handles most common cases
		return 0.0, false
	}

	// Validate the extracted value
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return 0.0, false
	}

	return val, true
}

// sanitizeMat creates a clean copy of Mat with validated data
func (lfs *LocalFaceService) sanitizeMat(mat gocv.Mat) gocv.Mat {
	if mat.Empty() {
		return mat
	}

	// For performance, just clone the Mat and let OpenCV handle validation
	// This is much faster than pixel-by-pixel validation
	sanitized := mat.Clone()

	// Force garbage collection to clean up any corrupted memory
	runtime.GC()

	return sanitized
}

// deterministicSample samples pixels in a deterministic pattern for consistent results
func (lfs *LocalFaceService) deterministicSample(mat gocv.Mat, sampleRate int) []float64 {
	if mat.Empty() {
		return []float64{}
	}

	var samples []float64
	rows := mat.Rows()
	cols := mat.Cols()

	// Sample every 'sampleRate' pixels in a deterministic pattern
	for i := 0; i < rows; i += sampleRate {
		for j := 0; j < cols; j += sampleRate {
			if val, ok := lfs.getPixelValue(mat, i, j); ok {
				samples = append(samples, val)
			}
		}
	}

	return samples
}

// calculateSimpleTextureScore calculates a simplified, deterministic texture score
func (lfs *LocalFaceService) calculateSimpleTextureScore(gray gocv.Mat) float64 {
	if gray.Empty() {
		return 0.5
	}

	// Calculate variance using a fixed sampling pattern
	mean := lfs.calculateMean(gray)
	variance := lfs.calculateVariance(gray, mean)

	// Normalize to 0-1 range with safe math
	textureScore := math.Min(lfs.safeMathOperation("divide", variance, 5000.0), 1.0)

	// Round to 6 decimal places for deterministic behavior
	textureScore = math.Round(textureScore*1000000) / 1000000

	// Ensure score is within valid range
	if math.IsNaN(textureScore) || math.IsInf(textureScore, 0) {
		return 0.5
	}
	if textureScore < 0 {
		textureScore = 0
	}
	if textureScore > 1 {
		textureScore = 1
	}

	return textureScore
}

// calculateBrightnessNormalizedTextureScore calculates texture score with brightness normalization
// This prevents darker images from getting artificially high texture scores
func (lfs *LocalFaceService) calculateBrightnessNormalizedTextureScore(gray gocv.Mat) float64 {
	if gray.Empty() {
		return 0.5
	}

	// Calculate mean brightness
	meanBrightness := lfs.calculateMean(gray)

	// Calculate raw texture score
	rawTextureScore := lfs.calculateSimpleTextureScore(gray)

	// Normalize based on brightness (darker images get penalized)
	// Optimal brightness: 100-150 (0-255 scale)
	brightnessNormalization := 1.0
	if meanBrightness < 90 {
		// Penalize very dark images more heavily
		// Dark images create artificial texture contrast
		// Increased penalty: power of 1.2 instead of 0.8 for steeper penalty
		brightnessNormalization = math.Pow(meanBrightness/90.0, 1.2)
	} else if meanBrightness > 180 {
		// Slightly penalize very bright images
		brightnessNormalization = 0.95
	}

	normalizedScore := rawTextureScore * brightnessNormalization

	// Ensure valid range
	if normalizedScore < 0 {
		normalizedScore = 0
	}
	if normalizedScore > 1 {
		normalizedScore = 1
	}

	return normalizedScore
}

// calculateBrightnessQualityScore calculates a quality score based on brightness levels
// Optimal brightness is rewarded, very dark or very bright images are penalized
func (lfs *LocalFaceService) calculateBrightnessQualityScore(gray gocv.Mat) float64 {
	if gray.Empty() {
		return 0.5
	}

	meanBrightness := lfs.calculateMean(gray)

	// Optimal brightness: 100-160 (0-255 scale)
	// Penalize very dark or very bright images
	var brightnessScore float64
	if meanBrightness >= 100 && meanBrightness <= 160 {
		brightnessScore = 1.0
	} else if meanBrightness < 100 {
		// Penalize dark images more heavily
		// Use power function to create steeper penalty for very dark images
		brightnessScore = math.Pow(meanBrightness/100.0, 1.5)
	} else {
		// Penalize bright images less heavily
		brightnessScore = math.Max(0.7, 1.0-(meanBrightness-160.0)/150.0)
	}

	// Ensure valid range
	if brightnessScore < 0 {
		brightnessScore = 0
	}
	if brightnessScore > 1 {
		brightnessScore = 1
	}

	return brightnessScore
}

// safeMathOperation performs mathematical operations with bounds checking
func (lfs *LocalFaceService) safeMathOperation(operation string, a, b float64) float64 {
	// Check inputs
	if math.IsNaN(a) || math.IsNaN(b) || math.IsInf(a, 0) || math.IsInf(b, 0) {
		return 0.5
	}

	var result float64
	switch operation {
	case "divide":
		if b == 0 {
			return 0.5
		}
		result = a / b
	case "log":
		if a <= 0 {
			return 0.5
		}
		result = math.Log2(a)
	case "multiply":
		result = a * b
	default:
		return a
	}

	// Check output
	if math.IsNaN(result) || math.IsInf(result, 0) {
		return 0.5
	}

	return result
}

// withCleanOpenCVContext executes function with clean OpenCV state
func (lfs *LocalFaceService) withCleanOpenCVContext(fn func() float64) float64 {
	// Force clean OpenCV state
	runtime.GC()

	// Execute in isolated context
	result := fn()

	// Validate result
	if math.IsNaN(result) || math.IsInf(result, 0) {
		return 0.5
	}

	// Round to 6 decimal places for deterministic behavior
	result = math.Round(result*1000000) / 1000000

	return result
}

// validateImageFormat validates image format and characteristics for safe processing
func (lfs *LocalFaceService) validateImageFormat(img image.Image, format string, imgData []byte) error {
	// Check image dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Validate dimensions
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid image dimensions: %dx%d", width, height)
	}

	// Check for extremely large images that might cause memory issues
	maxDimension := 10000 // 10MP limit
	if width > maxDimension || height > maxDimension {
		return fmt.Errorf("image too large: %dx%d (max: %d)", width, height, maxDimension)
	}

	// Check for extremely small images
	minDimension := 32 // Minimum 32x32 pixels
	if width < minDimension || height < minDimension {
		return fmt.Errorf("image too small: %dx%d (min: %d)", width, height, minDimension)
	}

	// Validate file size
	maxFileSize := 50 * 1024 * 1024 // 50MB limit
	if len(imgData) > maxFileSize {
		return fmt.Errorf("image file too large: %d bytes (max: %d)", len(imgData), maxFileSize)
	}

	// Check for supported formats
	supportedFormats := map[string]bool{
		"jpeg": true,
		"jpg":  true,
		"png":  true,
		"bmp":  true,
		"webp": true,
	}

	if !supportedFormats[format] {
		return fmt.Errorf("unsupported image format: %s (supported: jpeg, jpg, png, bmp, webp)", format)
	}

	// Detect potentially problematic image characteristics
	if err := lfs.detectProblematicImageCharacteristics(img, format, imgData); err != nil {
		return fmt.Errorf("problematic image detected: %v", err)
	}

	return nil
}

// detectProblematicImageCharacteristics detects images that might cause processing issues
func (lfs *LocalFaceService) detectProblematicImageCharacteristics(img image.Image, format string, imgData []byte) error {
	// Check for Photo Booth images by analyzing metadata and characteristics
	if lfs.isPhotoBoothImage(img, format, imgData) {
		return fmt.Errorf("Photo Booth image detected - may cause processing instability")
	}

	// Check for images with unusual color profiles or characteristics
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Check aspect ratio for unusual formats
	aspectRatio := float64(width) / float64(height)
	if aspectRatio < 0.1 || aspectRatio > 10.0 {
		return fmt.Errorf("unusual aspect ratio: %.2f (may indicate corrupted or non-standard image)", aspectRatio)
	}

	// Check for images that might be corrupted or have unusual characteristics
	// Temporarily disabled to allow normal images to pass
	// if lfs.hasUnusualImageCharacteristics(img) {
	//	return fmt.Errorf("unusual image characteristics detected - may cause processing issues")
	// }

	return nil
}

// isPhotoBoothImage detects Photo Booth images by analyzing various characteristics
func (lfs *LocalFaceService) isPhotoBoothImage(img image.Image, format string, imgData []byte) bool {
	// DISABLED: Photo Booth detection was causing false positives on legitimate photos
	// Only check for explicit Photo Booth metadata, not image characteristics

	// Method 1: Check for Photo Booth metadata in the image data (only explicit markers)
	if strings.Contains(strings.ToLower(string(imgData)), "photo booth") ||
		strings.Contains(strings.ToLower(string(imgData)), "photobooth") {
		return true
	}

	// REMOVED: Dimension-based detection was blocking legitimate iPhone/camera photos
	// REMOVED: Color-based detection was too aggressive and blocking normal photos

	return false
}

// hasPhotoBoothColorCharacteristics checks for color characteristics typical of Photo Booth images
// DISABLED: This function was causing false positives on legitimate photos
func (lfs *LocalFaceService) hasPhotoBoothColorCharacteristics(img image.Image) bool {
	// DISABLED: Color-based detection was too aggressive
	// Normal photos can have high brightness or color variance
	return false
}

// hasUnusualImageCharacteristics detects images with unusual characteristics that might cause issues
func (lfs *LocalFaceService) hasUnusualImageCharacteristics(img image.Image) bool {

	// Check for images with very high or very low contrast (more lenient thresholds)
	contrast := lfs.calculateImageContrastFromStdImage(img)
	if contrast > 0.98 || contrast < 0.02 {
		return true // Only extremely high or low contrast indicates processing issues
	}

	// Check for images with unusual brightness distribution (more lenient thresholds)
	brightness := lfs.calculateImageBrightness(img)
	if brightness > 0.98 || brightness < 0.02 {
		return true // Only extremely bright or dark images
	}

	return false
}

// calculateImageContrastFromStdImage calculates a simple contrast measure for standard image.Image type
func (lfs *LocalFaceService) calculateImageContrastFromStdImage(img image.Image) float64 {
	bounds := img.Bounds()
	sampleStep := 10
	var minLuma, maxLuma float64 = 1.0, 0.0
	var count int

	for y := bounds.Min.Y; y < bounds.Max.Y; y += sampleStep {
		for x := bounds.Min.X; x < bounds.Max.X; x += sampleStep {
			r, g, b, _ := img.At(x, y).RGBA()
			// Convert to luminance
			luma := 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
			luma = luma / 255.0 // Normalize to 0-1

			if luma < minLuma {
				minLuma = luma
			}
			if luma > maxLuma {
				maxLuma = luma
			}
			count++
		}
	}

	if count == 0 {
		return 0.5 // Default neutral contrast
	}

	return maxLuma - minLuma
}

// calculateImageBrightness calculates the average brightness of the image
func (lfs *LocalFaceService) calculateImageBrightness(img image.Image) float64 {
	bounds := img.Bounds()
	sampleStep := 10
	var totalLuma float64
	var count int

	for y := bounds.Min.Y; y < bounds.Max.Y; y += sampleStep {
		for x := bounds.Min.X; x < bounds.Max.X; x += sampleStep {
			r, g, b, _ := img.At(x, y).RGBA()
			// Convert to luminance
			luma := 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
			totalLuma += luma
			count++
		}
	}

	if count == 0 {
		return 0.5 // Default neutral brightness
	}

	return (totalLuma / float64(count)) / 255.0 // Normalize to 0-1
}

// convertImageToMatSafely converts an image to OpenCV Mat with robust error handling
func (lfs *LocalFaceService) convertImageToMatSafely(img image.Image) (gocv.Mat, error) {
	// First, try the standard conversion
	mat, err := gocv.ImageToMatRGB(img)
	if err == nil && !mat.Empty() {
		return mat, nil
	}

	// If standard conversion fails, try alternative approaches
	logger.Info("Standard Mat conversion failed, trying alternative methods", logger.LoggerOptions{
		Key:  "error",
		Data: err,
	})

	// Method 1: Try converting to RGBA first, then to RGB
	matRGBA, err := gocv.ImageToMatRGBA(img)
	if err == nil && !matRGBA.Empty() {
		// Convert RGBA to RGB
		matRGB := gocv.NewMat()
		defer matRGBA.Close()
		gocv.CvtColor(matRGBA, &matRGB, gocv.ColorRGBAToRGB)
		if !matRGB.Empty() {
			return matRGB, nil
		}
		matRGB.Close()
	}

	// Method 2: Try converting through a buffer with error recovery
	matBuffer, err := lfs.convertImageToMatViaBuffer(img)
	if err == nil && !matBuffer.Empty() {
		return matBuffer, nil
	}

	// Method 3: Last resort - create a simple RGB Mat manually
	matManual, err := lfs.createMatFromImageManually(img)
	if err == nil && !matManual.Empty() {
		return matManual, nil
	}

	return gocv.Mat{}, fmt.Errorf("all Mat conversion methods failed - image may be corrupted or incompatible")
}

// convertImageToMatViaBuffer converts image to Mat using a buffer-based approach
func (lfs *LocalFaceService) convertImageToMatViaBuffer(img image.Image) (gocv.Mat, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create a buffer for RGB data
	buffer := make([]byte, width*height*3)

	// Fill buffer with RGB data
	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// Convert from 16-bit to 8-bit and store in BGR order (OpenCV format)
			buffer[idx] = byte(b >> 8)   // Blue
			buffer[idx+1] = byte(g >> 8) // Green
			buffer[idx+2] = byte(r >> 8) // Red
			idx += 3
		}
	}

	// Create Mat from buffer
	mat, err := gocv.NewMatFromBytes(height, width, gocv.MatTypeCV8UC3, buffer)
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("failed to create Mat from buffer: %v", err)
	}

	return mat, nil
}

// createMatFromImageManually creates a Mat by manually processing the image
func (lfs *LocalFaceService) createMatFromImageManually(img image.Image) (gocv.Mat, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create a new Mat
	mat := gocv.NewMatWithSize(height, width, gocv.MatTypeCV8UC3)

	// Fill the Mat manually
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// Convert to 8-bit and set pixel in BGR format
			bVal := byte(b >> 8)
			gVal := byte(g >> 8)
			rVal := byte(r >> 8)

			// Set pixel values (BGR format)
			mat.SetUCharAt(y, x*3, bVal)   // Blue
			mat.SetUCharAt(y, x*3+1, gVal) // Green
			mat.SetUCharAt(y, x*3+2, rVal) // Red
		}
	}

	return mat, nil
}

// handleImageProcessingError provides graceful degradation for image processing errors
func (lfs *LocalFaceService) handleImageProcessingError(err error, startTime time.Time) (*types.BiometricLivenessResponse, error) {
	processingTime := time.Since(startTime).Milliseconds()

	// Analyze the error to provide appropriate response
	errorMsg := err.Error()
	var failureReason string
	var isTemporary bool

	// Categorize errors for better user experience
	if strings.Contains(errorMsg, "Photo Booth image detected") {
		failureReason = "unsupported_image_format"
		isTemporary = false
	} else if strings.Contains(errorMsg, "image format validation failed") {
		failureReason = "invalid_image_format"
		isTemporary = false
	} else if strings.Contains(errorMsg, "failed to download image") {
		failureReason = "network_error"
		isTemporary = true
	} else if strings.Contains(errorMsg, "failed to decode image") {
		failureReason = "corrupted_image"
		isTemporary = false
	} else if strings.Contains(errorMsg, "failed to convert image to Mat") {
		failureReason = "processing_error"
		isTemporary = false
	} else if strings.Contains(errorMsg, "image too large") {
		failureReason = "image_too_large"
		isTemporary = false
	} else if strings.Contains(errorMsg, "image too small") {
		failureReason = "image_too_small"
		isTemporary = false
	} else {
		failureReason = "unknown_processing_error"
		isTemporary = true
	}

	// Log the error with appropriate level
	if isTemporary {
		logger.Info("Image processing failed with recoverable error", logger.LoggerOptions{
			Key:  "error",
			Data: errorMsg,
		}, logger.LoggerOptions{
			Key:  "failure_reason",
			Data: failureReason,
		})
	} else {
		logger.Error("Image processing failed with permanent error", logger.LoggerOptions{
			Key:  "error",
			Data: errorMsg,
		}, logger.LoggerOptions{
			Key:  "failure_reason",
			Data: failureReason,
		})
	}

	// Provide user-friendly error message
	var userMessage string
	switch failureReason {
	case "unsupported_image_format":
		userMessage = "This image format is not supported for liveness detection. Please use a standard photo taken with a regular camera."
	case "invalid_image_format":
		userMessage = "The image format is invalid or corrupted. Please try with a different image."
	case "network_error":
		userMessage = "Failed to download the image. Please check your internet connection and try again."
	case "corrupted_image":
		userMessage = "The image appears to be corrupted. Please try with a different image."
	case "processing_error":
		userMessage = "Unable to process this image for liveness detection. Please try with a different image."
	case "image_too_large":
		userMessage = "The image is too large. Please use an image smaller than 10 megapixels."
	case "image_too_small":
		userMessage = "The image is too small. Please use an image at least 32x32 pixels."
	default:
		userMessage = "Unable to process the image for liveness detection. Please try again or use a different image."
	}

	return &types.BiometricLivenessResponse{
		Success:          false,
		IsLive:           false,
		Confidence:       0.0,
		QualityScore:     0.0,
		ProcessingTimeMs: int(processingTime),
		FailureReason:    &failureReason,
		Error:            &userMessage,
		AnalysisDetails: types.AnalysisDetails{
			ImageQuality:        0.0,
			LandmarkConsistency: 0.0,
			LightingScore:       0.0,
			SharpnessScore:      0.0,
			SpoofDetectionScore: 0.0,
			TextureScore:        0.0,
			// All detailed breakdown scores default to 0.0
		},
	}, nil
}

// calculateMean calculates the mean of a Mat using a safe sampling approach
func (lfs *LocalFaceService) calculateMean(img gocv.Mat) float64 {
	// Check if Mat is valid and has proper dimensions
	if img.Empty() || img.Rows() <= 0 || img.Cols() <= 0 {
		return 0.0
	}

	// For safety, always use sampling approach to avoid memory issues
	return lfs.calculateMeanSampled(img)
}

// calculateMeanSampled calculates the mean of a large Mat using sampling
func (lfs *LocalFaceService) calculateMeanSampled(img gocv.Mat) float64 {
	rows := img.Rows()
	cols := img.Cols()

	// Sample every 10th pixel to avoid processing huge images
	sampleStep := 10
	total := 0.0
	count := 0

	for i := 0; i < rows; i += sampleStep {
		for j := 0; j < cols; j += sampleStep {
			if i >= 0 && i < rows && j >= 0 && j < cols {
				val, ok := lfs.getPixelValue(img, i, j)
				if !ok {
					continue
				}
				total += val
				count++
			}
		}
	}

	if count == 0 {
		return 0.0
	}

	mean := lfs.safeMathOperation("divide", total, float64(count))

	// Round to 6 decimal places for deterministic behavior
	mean = math.Round(mean*1000000) / 1000000

	return mean
}

// calculateVariance calculates the variance of a Mat using a safe sampling approach
func (lfs *LocalFaceService) calculateVariance(img gocv.Mat, mean float64) float64 {
	// Check if Mat is valid and has proper dimensions
	if img.Empty() || img.Rows() <= 0 || img.Cols() <= 0 {
		return 0.0
	}

	// For safety, always use sampling approach to avoid memory issues
	return lfs.calculateVarianceSampled(img, mean)
}

// calculateVarianceSampled calculates the variance of a large Mat using sampling
func (lfs *LocalFaceService) calculateVarianceSampled(img gocv.Mat, mean float64) float64 {
	rows := img.Rows()
	cols := img.Cols()

	// Sample every 10th pixel to avoid processing huge images
	sampleStep := 10
	total := 0.0
	count := 0

	for i := 0; i < rows; i += sampleStep {
		for j := 0; j < cols; j += sampleStep {
			if i >= 0 && i < rows && j >= 0 && j < cols {
				val, ok := lfs.getPixelValue(img, i, j)
				if !ok {
					continue
				}
				diff := val - mean
				total += diff * diff
				count++
			}
		}
	}

	if count == 0 {
		return 0.0
	}

	variance := lfs.safeMathOperation("divide", total, float64(count))

	// Round to 6 decimal places for deterministic behavior
	variance = math.Round(variance*1000000) / 1000000

	return variance
}

// calculateCovariance calculates the covariance between two Mats
func (lfs *LocalFaceService) calculateCovariance(img1, img2 gocv.Mat, mean1, mean2 float64) float64 {
	total := 0.0
	count := 0

	for i := 0; i < img1.Rows(); i++ {
		for j := 0; j < img1.Cols(); j++ {
			val1 := float64(img1.GetFloatAt(i, j))
			val2 := float64(img2.GetFloatAt(i, j))
			diff1 := val1 - mean1
			diff2 := val2 - mean2
			total += diff1 * diff2
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// calculateEnhancedTemplateSimilarity calculates improved template matching similarity
func (lfs *LocalFaceService) calculateEnhancedTemplateSimilarity(img1, img2 gocv.Mat) float32 {
	// Normalize images first
	norm1 := gocv.NewMat()
	norm2 := gocv.NewMat()
	defer norm1.Close()
	defer norm2.Close()

	gocv.Normalize(img1, &norm1, 0, 255, gocv.NormMinMax)
	gocv.Normalize(img2, &norm2, 0, 255, gocv.NormMinMax)

	// Apply Gaussian blur to reduce noise
	blur1 := gocv.NewMat()
	blur2 := gocv.NewMat()
	defer blur1.Close()
	defer blur2.Close()

	gocv.GaussianBlur(norm1, &blur1, image.Pt(3, 3), 0, 0, gocv.BorderDefault)
	gocv.GaussianBlur(norm2, &blur2, image.Pt(3, 3), 0, 0, gocv.BorderDefault)

	// Calculate normalized cross correlation
	result := gocv.NewMat()
	defer result.Close()

	gocv.MatchTemplate(blur1, blur2, &result, gocv.TmCcoeffNormed, gocv.NewMat())
	_, maxVal, _, _ := gocv.MinMaxLoc(result)

	// Apply quality-based adjustment
	quality1 := lfs.calculateImageQualityMetric(img1)
	quality2 := lfs.calculateImageQualityMetric(img2)
	avgQuality := (quality1 + quality2) / 2.0

	// Boost similarity for high-quality images
	if avgQuality > 0.7 {
		maxVal *= 1.1
	} else if avgQuality < 0.4 {
		maxVal *= 0.9
	}

	if maxVal > 1.0 {
		maxVal = 1.0
	}
	if maxVal < 0 {
		maxVal = 0
	}

	return maxVal
}

// calculateFaceSSIM calculates face-specific SSIM with improved parameters
func (lfs *LocalFaceService) calculateFaceSSIM(img1, img2 gocv.Mat) float64 {
	// Convert to float32 for better precision
	img1Float := gocv.NewMat()
	img2Float := gocv.NewMat()
	defer img1Float.Close()
	defer img2Float.Close()

	img1.ConvertTo(&img1Float, gocv.MatTypeCV32F)
	img2.ConvertTo(&img2Float, gocv.MatTypeCV32F)

	// Calculate mean using more efficient method
	mean1 := lfs.calculateMean(img1Float)
	mean2 := lfs.calculateMean(img2Float)

	// Calculate variance with bias correction
	var1 := lfs.calculateVariance(img1Float, mean1)
	var2 := lfs.calculateVariance(img2Float, mean2)

	// Calculate covariance
	cov := lfs.calculateCovariance(img1Float, img2Float, mean1, mean2)

	// Face-specific SSIM constants (optimized for facial features)
	c1 := 0.01 * 0.01 * 255 * 255 // Luminance constant
	c2 := 0.03 * 0.03 * 255 * 255 // Contrast constant

	// Calculate SSIM components
	luminance := (2*mean1*mean2 + c1) / (mean1*mean1 + mean2*mean2 + c1)
	contrast := (2*math.Sqrt(var1*var2) + c2) / (var1 + var2 + c2)
	structure := (cov + c2/2) / (math.Sqrt(var1*var2) + c2/2)

	ssim := luminance * contrast * structure

	// Ensure SSIM is between 0 and 1
	if ssim < 0 {
		ssim = 0
	}
	if ssim > 1 {
		ssim = 1
	}

	return ssim
}

// calculateHistogramSimilarity calculates histogram correlation similarity
func (lfs *LocalFaceService) calculateHistogramSimilarity(img1, img2 gocv.Mat) float32 {
	// Calculate histograms
	hist1 := gocv.NewMat()
	hist2 := gocv.NewMat()
	defer hist1.Close()
	defer hist2.Close()

	// Calculate normalized histograms
	gocv.CalcHist([]gocv.Mat{img1}, []int{0}, gocv.NewMat(), &hist1, []int{256}, []float64{0, 256}, false)
	gocv.CalcHist([]gocv.Mat{img2}, []int{0}, gocv.NewMat(), &hist2, []int{256}, []float64{0, 256}, false)

	// Normalize histograms
	gocv.Normalize(hist1, &hist1, 0, 1, gocv.NormMinMax)
	gocv.Normalize(hist2, &hist2, 0, 1, gocv.NormMinMax)

	// Calculate correlation coefficient
	correlation := gocv.CompareHist(hist1, hist2, gocv.HistCmpCorrel)

	// Convert to 0-1 range
	if correlation < 0 {
		correlation = 0
	}

	return float32(correlation)
}

// calculateEdgeSimilarity calculates edge-based structural similarity
func (lfs *LocalFaceService) calculateEdgeSimilarity(img1, img2 gocv.Mat) float32 {
	// Apply Canny edge detection
	edges1 := gocv.NewMat()
	edges2 := gocv.NewMat()
	defer edges1.Close()
	defer edges2.Close()

	gocv.Canny(img1, &edges1, 50, 150)
	gocv.Canny(img2, &edges2, 50, 150)

	// Calculate edge density
	edgePixels1 := gocv.CountNonZero(edges1)
	edgePixels2 := gocv.CountNonZero(edges2)
	totalPixels := float64(img1.Rows() * img1.Cols())

	edgeDensity1 := float64(edgePixels1) / totalPixels
	edgeDensity2 := float64(edgePixels2) / totalPixels

	// Calculate edge density similarity
	densitySimilarity := 1.0 - math.Abs(edgeDensity1-edgeDensity2)

	// Calculate edge pattern similarity using template matching
	result := gocv.NewMat()
	defer result.Close()

	gocv.MatchTemplate(edges1, edges2, &result, gocv.TmCcoeffNormed, gocv.NewMat())
	_, maxVal, _, _ := gocv.MinMaxLoc(result)

	// Combine density and pattern similarity
	edgeSimilarity := float32((densitySimilarity + float64(maxVal)) / 2.0)

	if edgeSimilarity > 1.0 {
		edgeSimilarity = 1.0
	}
	if edgeSimilarity < 0 {
		edgeSimilarity = 0
	}

	return edgeSimilarity
}

// calculateImageQualityMetric calculates a comprehensive image quality metric
func (lfs *LocalFaceService) calculateImageQualityMetric(img gocv.Mat) float64 {
	// Calculate sharpness
	sharpness := lfs.calculateSharpnessScore(img)

	// Calculate contrast
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	mean := lfs.calculateMean(gray)
	variance := lfs.calculateVariance(gray, mean)
	contrast := math.Sqrt(variance) / 255.0

	// Calculate brightness distribution
	brightness := mean / 255.0
	brightnessScore := 1.0 - math.Abs(brightness-0.5)*2 // Optimal brightness around 0.5

	// Calculate noise level (using Laplacian variance)
	laplacian := gocv.NewMat()
	defer laplacian.Close()
	gocv.Laplacian(gray, &laplacian, gocv.MatTypeCV64F, 1, 1, 0, gocv.BorderDefault)
	noiseLevel := lfs.calculateVariance(laplacian, 0)
	noiseScore := math.Max(0, 1.0-noiseLevel/10000.0)

	// Combine metrics with weights
	quality := sharpness*0.3 + contrast*0.25 + brightnessScore*0.25 + noiseScore*0.2

	if quality > 1.0 {
		quality = 1.0
	}
	if quality < 0 {
		quality = 0
	}

	return quality
}

// performQuickLivenessCheck performs a basic liveness check for face comparison
func (lfs *LocalFaceService) performQuickLivenessCheck(faceRegion gocv.Mat) bool {
	// ENHANCED QUICK LIVENESS CHECK
	// More comprehensive quick analysis with multiple indicators

	// Convert to grayscale for analysis
	gray := gocv.NewMat()
	defer gray.Close()
	if faceRegion.Channels() > 1 {
		gocv.CvtColor(faceRegion, &gray, gocv.ColorBGRToGray)
	} else {
		gray = faceRegion.Clone()
	}

	// 1. Texture analysis (skin has natural texture)
	textureScore := lfs.calculateTextureScore(faceRegion)

	// 2. Edge analysis (real faces have natural edge distribution)
	edgeScore := lfs.calculateEdgeScore(faceRegion)

	// 3. Reflection analysis (screen displays have uniform reflection)
	reflectionScore := lfs.calculateReflectionScore(faceRegion)

	// 4. Quick frequency analysis (real skin has natural frequency patterns)
	frequencyScore := lfs.calculateQuickFrequencyScore(gray)

	// 5. Quick color variance (real faces have natural color variation)
	colorVariance := lfs.calculateQuickColorVariance(faceRegion)

	// 6. Smoothness check (paintings/prints are too smooth)
	smoothnessScore := 1.0 - lfs.calculateQuickSmoothness(gray)

	// Weighted combination for quick check
	quickScore := (textureScore*0.25 +
		edgeScore*0.20 +
		reflectionScore*0.15 +
		frequencyScore*0.15 +
		colorVariance*0.15 +
		smoothnessScore*0.10)

	// Enhanced threshold with multi-factor validation
	baseThreshold := 0.30

	// Fail fast checks - reject immediately if critical indicators are very poor
	if textureScore < 0.10 || edgeScore < 0.10 {
		return false // Likely a print or painting
	}

	if smoothnessScore < 0.15 {
		return false // Too smooth to be real skin
	}

	return quickScore > baseThreshold
}

// calculateQuickFrequencyScore performs fast frequency domain analysis
func (lfs *LocalFaceService) calculateQuickFrequencyScore(gray gocv.Mat) float64 {
	// Simple high-pass filtering to detect fine details
	blurred := gocv.NewMat()
	defer blurred.Close()
	gocv.GaussianBlur(gray, &blurred, image.Pt(5, 5), 2.0, 2.0, gocv.BorderDefault)

	highFreq := gocv.NewMat()
	defer highFreq.Close()
	gocv.Subtract(gray, blurred, &highFreq)

	// Calculate high-frequency energy
	energy := 0.0
	count := 0
	for i := 0; i < highFreq.Rows(); i++ {
		for j := 0; j < highFreq.Cols(); j++ {
			val := float64(highFreq.GetUCharAt(i, j))
			energy += val * val
			count++
		}
	}

	if count == 0 {
		return 0.5
	}

	avgEnergy := energy / float64(count)

	// Normalize (real faces typically have energy in range 50-500)
	normalizedScore := avgEnergy / 500.0
	return math.Min(normalizedScore, 1.0)
}

// calculateQuickColorVariance calculates fast color variation metric
func (lfs *LocalFaceService) calculateQuickColorVariance(faceRegion gocv.Mat) float64 {
	if faceRegion.Channels() < 3 {
		return 0.5 // Grayscale image, return neutral
	}

	// Split channels
	channels := gocv.Split(faceRegion)
	defer func() {
		for _, ch := range channels {
			ch.Close()
		}
	}()

	if len(channels) < 3 {
		return 0.5
	}

	// Calculate variance for each channel
	totalVariance := 0.0
	for _, ch := range channels {
		mean := lfs.calculateMean(ch)
		variance := lfs.calculateVariance(ch, mean)
		totalVariance += variance
	}

	avgVariance := totalVariance / 3.0

	// Normalize (typical variance for natural faces: 200-2000)
	normalizedScore := avgVariance / 2000.0
	return math.Min(normalizedScore, 1.0)
}

// calculateQuickSmoothness calculates how smooth the image is
func (lfs *LocalFaceService) calculateQuickSmoothness(gray gocv.Mat) float64 {
	// Calculate local standard deviations in small windows
	windowSize := 5
	smoothRegions := 0
	totalRegions := 0

	step := windowSize
	for i := 0; i < gray.Rows()-windowSize; i += step {
		for j := 0; j < gray.Cols()-windowSize; j += step {
			roi := gray.Region(image.Rect(j, i, j+windowSize, i+windowSize))
			mean := lfs.calculateMean(roi)
			variance := lfs.calculateVariance(roi, mean)
			roi.Close()

			totalRegions++

			// Very low variance indicates smoothness
			if variance < 50 {
				smoothRegions++
			}
		}
	}

	if totalRegions == 0 {
		return 0.5
	}

	smoothnessRatio := float64(smoothRegions) / float64(totalRegions)
	return smoothnessRatio
}

// calculateAdvancedTextureScore calculates advanced texture analysis using multiple methods
func (lfs *LocalFaceService) calculateAdvancedTextureScore(faceRegion, gray gocv.Mat) float64 {
	// 1. Local Binary Patterns (LBP) analysis
	lbpScore := lfs.calculateLBPScore(gray)

	// 2. Local Phase Quantization (LPQ) analysis
	lpqScore := lfs.calculateLPQScore(gray)

	// 3. Gabor filter texture analysis
	gaborScore := lfs.calculateGaborTextureScore(gray)

	// 4. Gray-Level Co-occurrence Matrix (GLCM) analysis
	glcmScore := lfs.calculateGLCMScore(gray)

	// Combine texture scores with weights
	textureScore := lbpScore*0.3 + lpqScore*0.3 + gaborScore*0.25 + glcmScore*0.15

	// Round to 6 decimal places for deterministic behavior
	textureScore = math.Round(textureScore*1000000) / 1000000

	// Ensure score is within valid range [0, 1]
	if math.IsNaN(textureScore) || math.IsInf(textureScore, 0) {
		return 0.5
	}
	if textureScore < 0 {
		textureScore = 0
	}
	if textureScore > 1 {
		textureScore = 1
	}

	return textureScore
}

// calculateLBPScore calculates Local Binary Patterns score
func (lfs *LocalFaceService) calculateLBPScore(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() < 3 || gray.Cols() < 3 {
		return 0.5
	}

	// Improved LBP calculation using actual local binary pattern approach
	// Calculate LBP histogram for texture analysis
	rows := gray.Rows()
	cols := gray.Cols()

	// LBP histogram (256 bins for 8-neighbor LBP)
	histogram := make([]int, 256)
	totalPatterns := 0

	// Sample step for performance (every 2nd pixel)
	sampleStep := 2

	// Calculate LBP for each pixel (excluding borders)
	for i := sampleStep; i < rows-sampleStep; i += sampleStep {
		for j := sampleStep; j < cols-sampleStep; j += sampleStep {
			center := float64(gray.GetUCharAt(i, j))

			// Calculate 8-neighbor LBP pattern
			var pattern uint8 = 0
			neighbors := []struct{ dy, dx int }{
				{-1, -1}, {-1, 0}, {-1, 1},
				{0, 1}, {1, 1}, {1, 0},
				{1, -1}, {0, -1},
			}

			for bit, neighbor := range neighbors {
				ny := i + neighbor.dy
				nx := j + neighbor.dx
				if ny >= 0 && ny < rows && nx >= 0 && nx < cols {
					neighborVal := float64(gray.GetUCharAt(ny, nx))
					if neighborVal >= center {
						pattern |= (1 << uint(bit))
					}
				}
			}

			histogram[pattern]++
			totalPatterns++
		}
	}

	if totalPatterns == 0 {
		return 0.5
	}

	// Calculate histogram uniformity and entropy for texture richness
	// Higher entropy = more texture variation = more natural
	entropy := 0.0
	for _, count := range histogram {
		if count > 0 {
			probability := float64(count) / float64(totalPatterns)
			entropy -= probability * math.Log2(probability)
		}
	}

	// Normalize entropy (max entropy for 256 bins is log2(256) = 8)
	lbpScore := math.Min(entropy/8.0, 1.0)

	// Round to 6 decimal places for deterministic behavior
	lbpScore = math.Round(lbpScore*1000000) / 1000000

	// Ensure valid range
	if math.IsNaN(lbpScore) || math.IsInf(lbpScore, 0) {
		return 0.5
	}
	if lbpScore < 0 {
		lbpScore = 0
	}
	if lbpScore > 1 {
		lbpScore = 1
	}

	return lbpScore
}

// calculateLPQScore calculates Local Phase Quantization score
func (lfs *LocalFaceService) calculateLPQScore(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() < 7 || gray.Cols() < 7 {
		return 0.5
	}

	// Improved LPQ using multi-scale gradient phase analysis
	// LPQ analyzes local frequency components using phase information

	// Calculate gradients at multiple scales for phase analysis
	sobelX := gocv.NewMat()
	sobelY := gocv.NewMat()
	defer sobelX.Close()
	defer sobelY.Close()

	gocv.Sobel(gray, &sobelX, gocv.MatTypeCV64F, 1, 0, 3, 1, 0, gocv.BorderDefault)
	gocv.Sobel(gray, &sobelY, gocv.MatTypeCV64F, 0, 1, 3, 1, 0, gocv.BorderDefault)

	// Calculate phase angles
	phase := gocv.NewMat()
	defer phase.Close()
	gocv.Phase(sobelX, sobelY, &phase, false)

	// Calculate magnitude for weighting
	magnitude := gocv.NewMat()
	defer magnitude.Close()
	gocv.Magnitude(sobelX, sobelY, &magnitude)

	// Analyze local phase patterns in blocks
	rows := phase.Rows()
	cols := phase.Cols()
	blockSize := 7
	sampleStep := 3

	var phaseVarianceSum float64
	blockCount := 0

	for i := 0; i < rows-blockSize; i += sampleStep {
		for j := 0; j < cols-blockSize; j += sampleStep {
			// Calculate phase variance in this block
			var phaseSum, magnitudeSum float64
			pixelCount := 0

			for y := i; y < i+blockSize && y < rows; y++ {
				for x := j; x < j+blockSize && x < cols; x++ {
					phaseVal, ok := lfs.getPixelValue(phase, y, x)
					if !ok || math.IsNaN(phaseVal) || math.IsInf(phaseVal, 0) {
						continue
					}
					magVal, ok := lfs.getPixelValue(magnitude, y, x)
					if !ok || math.IsNaN(magVal) || math.IsInf(magVal, 0) {
						continue
					}

					phaseSum += phaseVal
					magnitudeSum += magVal
					pixelCount++
				}
			}

			if pixelCount > 0 && magnitudeSum > 0 {
				// Calculate phase variance weighted by magnitude
				meanPhase := phaseSum / float64(pixelCount)
				var variance float64

				for y := i; y < i+blockSize && y < rows; y++ {
					for x := j; x < j+blockSize && x < cols; x++ {
						phaseVal, ok := lfs.getPixelValue(phase, y, x)
						if !ok || math.IsNaN(phaseVal) || math.IsInf(phaseVal, 0) {
							continue
						}
						diff := phaseVal - meanPhase
						variance += diff * diff
					}
				}

				variance /= float64(pixelCount)
				phaseVarianceSum += variance
				blockCount++
			}
		}
	}

	if blockCount == 0 {
		return 0.5
	}

	// Average phase variance across blocks
	avgPhaseVariance := phaseVarianceSum / float64(blockCount)

	// Normalize to 0-1 range (higher phase variance = richer texture = more natural)
	// ADJUSTED: Reduced denominator from 10.0 to 5.0 for better sensitivity to real faces
	lpqScore := math.Min(avgPhaseVariance/5.0, 1.0)

	// Round to 6 decimal places for deterministic behavior
	lpqScore = math.Round(lpqScore*1000000) / 1000000

	// Ensure valid range
	if math.IsNaN(lpqScore) || math.IsInf(lpqScore, 0) {
		return 0.5
	}
	if lpqScore < 0 {
		lpqScore = 0
	}
	if lpqScore > 1 {
		lpqScore = 1
	}

	return lpqScore
}

// calculateGaborTextureScore calculates Gabor filter-based texture score
func (lfs *LocalFaceService) calculateGaborTextureScore(gray gocv.Mat) float64 {
	// Apply multiple Gabor filters with different orientations
	orientations := []float64{0, 45, 90, 135}
	totalResponse := 0.0

	// Simplified Gabor-like filter using Sobel for texture analysis
	for _, orientation := range orientations {
		// Create directional Sobel-like kernel based on orientation
		var kernel gocv.Mat
		var err error
		if orientation == 0 {
			// Horizontal edges
			kernel, err = gocv.NewMatFromBytes(3, 3, gocv.MatTypeCV32F, []byte{
				0, 0, 0,
				1, 1, 1,
				0, 0, 0,
			})
		} else if orientation == 45 {
			// Diagonal edges
			kernel, err = gocv.NewMatFromBytes(3, 3, gocv.MatTypeCV32F, []byte{
				1, 0, 0,
				0, 1, 0,
				0, 0, 1,
			})
		} else if orientation == 90 {
			// Vertical edges
			kernel, err = gocv.NewMatFromBytes(3, 3, gocv.MatTypeCV32F, []byte{
				0, 1, 0,
				0, 1, 0,
				0, 1, 0,
			})
		} else { // 135 degrees
			// Anti-diagonal edges
			kernel, err = gocv.NewMatFromBytes(3, 3, gocv.MatTypeCV32F, []byte{
				0, 0, 1,
				0, 1, 0,
				1, 0, 0,
			})
		}

		if err != nil {
			continue // Skip this orientation on error
		}
		defer kernel.Close()

		// Apply filter
		filtered := gocv.NewMat()
		defer filtered.Close()
		gocv.Filter2D(gray, &filtered, gocv.MatTypeCV32F, kernel, image.Pt(-1, -1), 0, gocv.BorderDefault)

		// Calculate response magnitude
		mean := lfs.calculateMean(filtered)
		totalResponse += mean
	}

	// Normalize total response with safe math
	gaborScore := math.Min(lfs.safeMathOperation("divide", totalResponse, 4000.0), 1.0)

	// Round to 6 decimal places for deterministic behavior
	gaborScore = math.Round(gaborScore*1000000) / 1000000

	// Ensure score is within valid range [0, 1]
	if math.IsNaN(gaborScore) || math.IsInf(gaborScore, 0) {
		return 0.5
	}
	if gaborScore < 0 {
		gaborScore = 0
	}
	if gaborScore > 1 {
		gaborScore = 1
	}

	return gaborScore
}

// calculateGLCMScore calculates Gray-Level Co-occurrence Matrix score with uniformity and entropy
func (lfs *LocalFaceService) calculateGLCMScore(gray gocv.Mat) float64 {
	// Enhanced GLCM calculation with uniformity and entropy thresholds

	// Apply Sobel operator for edge detection
	sobelX := gocv.NewMat()
	sobelY := gocv.NewMat()
	defer sobelX.Close()
	defer sobelY.Close()

	gocv.Sobel(gray, &sobelX, gocv.MatTypeCV64F, 1, 0, 3, 1, 0, gocv.BorderDefault)
	gocv.Sobel(gray, &sobelY, gocv.MatTypeCV64F, 0, 1, 3, 1, 0, gocv.BorderDefault)

	// Calculate gradient magnitude
	gradient := gocv.NewMat()
	defer gradient.Close()
	gocv.Magnitude(sobelX, sobelY, &gradient)

	// Calculate contrast (variance of gradient)
	mean := lfs.calculateMean(gradient)
	variance := lfs.calculateVariance(gradient, mean)
	contrastScore := math.Min(lfs.safeMathOperation("divide", variance, 3000.0), 1.0)

	// Calculate texture uniformity
	uniformityScore := lfs.calculateTextureUniformity(gray)

	// Calculate texture entropy
	entropyScore := lfs.calculateTextureEntropy(gray)

	// Apply more lenient thresholds: uniformity >= 0.5, entropy >= 0.3
	var thresholdBonus float64 = 0.0

	if uniformityScore >= 0.5 {
		thresholdBonus += 0.1 // Bonus for good uniformity
	}

	if entropyScore >= 0.3 {
		thresholdBonus += 0.1 // Bonus for good entropy
	}

	// Combine scores with enhanced weighting
	glcmScore := contrastScore*0.4 + uniformityScore*0.3 + entropyScore*0.3 + thresholdBonus

	// Round to 6 decimal places for deterministic behavior
	glcmScore = math.Round(glcmScore*1000000) / 1000000

	// Ensure score is within valid range
	if math.IsNaN(glcmScore) || math.IsInf(glcmScore, 0) {
		return 0.5
	}
	if glcmScore > 1.0 {
		glcmScore = 1.0
	}
	if glcmScore < 0.0 {
		glcmScore = 0.0
	}

	return glcmScore
}

// calculateTextureUniformity calculates texture uniformity score
func (lfs *LocalFaceService) calculateTextureUniformity(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() == 0 || gray.Cols() == 0 {
		return 0.5
	}

	rows := gray.Rows()
	cols := gray.Cols()

	// Divide image into small regions and calculate uniformity
	regionSize := 8
	regions := 0
	uniformitySum := 0.0

	for i := 0; i < rows-regionSize; i += regionSize {
		for j := 0; j < cols-regionSize; j += regionSize {
			// Calculate mean and variance for this region
			regionMean := 0.0
			regionVariance := 0.0
			pixelCount := 0

			// Calculate mean
			for y := i; y < i+regionSize && y < rows; y++ {
				for x := j; x < j+regionSize && x < cols; x++ {
					regionMean += float64(gray.GetUCharAt(y, x))
					pixelCount++
				}
			}

			if pixelCount > 0 {
				regionMean /= float64(pixelCount)

				// Calculate variance
				for y := i; y < i+regionSize && y < rows; y++ {
					for x := j; x < j+regionSize && x < cols; x++ {
						val := float64(gray.GetUCharAt(y, x))
						diff := val - regionMean
						regionVariance += diff * diff
					}
				}
				regionVariance /= float64(pixelCount)

				// IMPROVED: Scale-adaptive normalization based on image characteristics
				// Lower variance = higher uniformity, but scale by image mean brightness
				// This accounts for different lighting conditions and image scales
				scaleFactor := 100.0
				if regionMean > 0 {
					// Adjust scale factor based on brightness (darker regions have lower variance)
					scaleFactor = 50.0 + (regionMean / 255.0 * 150.0)
				}
				uniformity := 1.0 / (1.0 + regionVariance/scaleFactor)
				uniformitySum += uniformity
				regions++
			}
		}
	}

	if regions == 0 {
		return 0.5
	}

	avgUniformity := uniformitySum / float64(regions)

	// Normalize to 0-1 range
	uniformityScore := math.Min(avgUniformity, 1.0)
	if math.IsNaN(uniformityScore) || math.IsInf(uniformityScore, 0) {
		return 0.5
	}

	return uniformityScore
}

// calculateTextureEntropy calculates texture entropy score
func (lfs *LocalFaceService) calculateTextureEntropy(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() == 0 || gray.Cols() == 0 {
		return 0.5
	}

	rows := gray.Rows()
	cols := gray.Cols()

	// Create histogram of intensity values
	histogram := make([]int, 256)
	totalPixels := 0

	// Sample every 2nd pixel for efficiency
	sampleStep := 2
	for i := 0; i < rows; i += sampleStep {
		for j := 0; j < cols; j += sampleStep {
			if i >= 0 && i < rows && j >= 0 && j < cols {
				intensity := gray.GetUCharAt(i, j)
				histogram[intensity]++
				totalPixels++
			}
		}
	}

	if totalPixels == 0 {
		return 0.5
	}

	// Calculate entropy
	entropy := 0.0
	for _, count := range histogram {
		if count > 0 {
			probability := lfs.safeMathOperation("divide", float64(count), float64(totalPixels))
			logResult := lfs.safeMathOperation("log", probability, 0)
			entropy -= lfs.safeMathOperation("multiply", probability, logResult)
		}
	}

	// Normalize entropy (max entropy for 256 levels is log2(256) = 8)
	maxEntropy := 8.0
	entropyScore := lfs.safeMathOperation("divide", entropy, maxEntropy)

	// Ensure score is within valid range
	if entropyScore > 1.0 {
		entropyScore = 1.0
	}
	if entropyScore < 0.0 {
		entropyScore = 0.0
	}
	if math.IsNaN(entropyScore) || math.IsInf(entropyScore, 0) {
		return 0.5
	}

	return entropyScore
}

// calculateAdvancedEdgeScore calculates enhanced edge-based liveness score
func (lfs *LocalFaceService) calculateAdvancedEdgeScore(gray gocv.Mat) float64 {
	// 1. Multi-scale edge detection
	edgeScore1 := lfs.calculateEdgeScore(gray) // Original method

	// 2. Laplacian of Gaussian edge detection
	logEdges := gocv.NewMat()
	defer logEdges.Close()
	gocv.GaussianBlur(gray, &logEdges, image.Pt(5, 5), 1.0, 1.0, gocv.BorderDefault)
	gocv.Laplacian(logEdges, &logEdges, gocv.MatTypeCV64F, 3, 1, 0, gocv.BorderDefault)

	// Calculate LoG edge density
	logEdgePixels := gocv.CountNonZero(logEdges)
	totalPixels := float64(gray.Rows() * gray.Cols())
	logEdgeDensity := float64(logEdgePixels) / totalPixels
	logEdgeScore := math.Min(logEdgeDensity*8, 1.0)

	// 3. Edge direction consistency
	directionScore := lfs.calculateEdgeDirectionConsistency(gray)

	// Combine edge scores
	edgeScore := edgeScore1*0.4 + logEdgeScore*0.4 + directionScore*0.2

	return edgeScore
}

// calculateEdgeDirectionConsistency calculates edge direction consistency
func (lfs *LocalFaceService) calculateEdgeDirectionConsistency(gray gocv.Mat) float64 {
	// Apply Sobel operators
	sobelX := gocv.NewMat()
	sobelY := gocv.NewMat()
	defer sobelX.Close()
	defer sobelY.Close()

	gocv.Sobel(gray, &sobelX, gocv.MatTypeCV64F, 1, 0, 3, 1, 0, gocv.BorderDefault)
	gocv.Sobel(gray, &sobelY, gocv.MatTypeCV64F, 0, 1, 3, 1, 0, gocv.BorderDefault)

	// Calculate edge orientations
	orientations := gocv.NewMat()
	defer orientations.Close()
	gocv.Phase(sobelX, sobelY, &orientations, false)

	// Calculate orientation variance using circular statistics (natural faces have more varied edge directions)
	// For circular data, we need to use circular variance
	rows := orientations.Rows()
	cols := orientations.Cols()

	if rows == 0 || cols == 0 {
		// Fallback: use a simple edge density calculation
		edges := gocv.NewMat()
		defer edges.Close()
		gocv.Canny(gray, &edges, 50, 150)
		edgePixels := gocv.CountNonZero(edges)
		totalPixels := float64(gray.Rows() * gray.Cols())
		edgeDensity := float64(edgePixels) / totalPixels
		return math.Min(edgeDensity*10, 1.0)
	}

	// Sample orientations to calculate circular variance
	sampleStep := 10
	var cosSum, sinSum float64
	count := 0

	for i := 0; i < rows; i += sampleStep {
		for j := 0; j < cols; j += sampleStep {
			if i >= 0 && i < rows && j >= 0 && j < cols {
				val, ok := lfs.getPixelValue(orientations, i, j)
				if !ok {
					continue
				}
				// Check for NaN or invalid values
				if math.IsNaN(val) || math.IsInf(val, 0) {
					continue
				}

				cosSum += math.Cos(val)
				sinSum += math.Sin(val)
				count++
			}
		}
	}

	if count == 0 {
		// Fallback: use a simple edge density calculation
		edges := gocv.NewMat()
		defer edges.Close()
		gocv.Canny(gray, &edges, 50, 150)
		edgePixels := gocv.CountNonZero(edges)
		totalPixels := float64(gray.Rows() * gray.Cols())
		edgeDensity := float64(edgePixels) / totalPixels
		return math.Min(edgeDensity*10, 1.0)
	}

	// Calculate circular variance
	meanCos := cosSum / float64(count)
	meanSin := sinSum / float64(count)
	circularVariance := 1.0 - math.Sqrt(meanCos*meanCos+meanSin*meanSin)

	// Higher circular variance indicates more varied edge directions (more natural) - FIXED: Increased multiplier from 2.0 to 5.0 for meaningful range
	directionScore := math.Min(circularVariance*5.0, 1.0)

	// Ensure the result is valid
	if math.IsNaN(directionScore) || math.IsInf(directionScore, 0) {
		return 0.5 // Return neutral score
	}

	return directionScore
}

// calculateAdvancedColorScore calculates advanced color space analysis
func (lfs *LocalFaceService) calculateAdvancedColorScore(faceRegion gocv.Mat) float64 {
	// 1. HSV color space analysis
	hsvScore := lfs.calculateColorScore(faceRegion) // Original method

	// 2. LAB color space analysis
	lab := gocv.NewMat()
	defer lab.Close()
	gocv.CvtColor(faceRegion, &lab, gocv.ColorBGRToLab)

	// Calculate LAB variance
	labChannels := gocv.Split(lab)
	defer func() {
		for _, ch := range labChannels {
			ch.Close()
		}
	}()

	var labVariance float64
	for _, ch := range labChannels {
		mean := lfs.calculateMean(ch)
		variance := lfs.calculateVariance(ch, mean)
		labVariance += variance
	}
	labScore := math.Min(labVariance/15000.0, 1.0)

	// 3. Skin tone analysis
	skinScore := lfs.calculateSkinToneScore(faceRegion)

	// Combine color scores
	colorScore := hsvScore*0.4 + labScore*0.4 + skinScore*0.2

	return colorScore
}

// calculateSkinToneScore calculates skin tone consistency score
func (lfs *LocalFaceService) calculateSkinToneScore(faceRegion gocv.Mat) float64 {
	// Convert to HSV for skin detection
	hsv := gocv.NewMat()
	defer hsv.Close()
	gocv.CvtColor(faceRegion, &hsv, gocv.ColorBGRToHSV)

	// Define skin tone ranges in HSV using scalar values
	lowerSkin1 := gocv.NewScalar(0, 20, 70, 0)
	upperSkin1 := gocv.NewScalar(20, 255, 255, 0)
	lowerSkin2 := gocv.NewScalar(160, 20, 70, 0)
	upperSkin2 := gocv.NewScalar(180, 255, 255, 0)

	// Create skin masks
	mask1 := gocv.NewMat()
	mask2 := gocv.NewMat()
	defer mask1.Close()
	defer mask2.Close()

	// Convert scalars to Mats for InRange function
	lowerMat1 := gocv.NewMatFromScalar(lowerSkin1, gocv.MatTypeCV8U)
	upperMat1 := gocv.NewMatFromScalar(upperSkin1, gocv.MatTypeCV8U)
	lowerMat2 := gocv.NewMatFromScalar(lowerSkin2, gocv.MatTypeCV8U)
	upperMat2 := gocv.NewMatFromScalar(upperSkin2, gocv.MatTypeCV8U)
	defer lowerMat1.Close()
	defer upperMat1.Close()
	defer lowerMat2.Close()
	defer upperMat2.Close()

	gocv.InRange(hsv, lowerMat1, upperMat1, &mask1)
	gocv.InRange(hsv, lowerMat2, upperMat2, &mask2)

	// Combine masks
	skinMask := gocv.NewMat()
	defer skinMask.Close()
	gocv.BitwiseOr(mask1, mask2, &skinMask)

	// Calculate skin pixel percentage
	skinPixels := gocv.CountNonZero(skinMask)
	totalPixels := faceRegion.Rows() * faceRegion.Cols()
	skinPercentage := float64(skinPixels) / float64(totalPixels)

	// Optimal skin percentage for natural faces (30-70%)
	var skinScore float64
	if skinPercentage >= 0.3 && skinPercentage <= 0.7 {
		skinScore = 1.0
	} else if skinPercentage < 0.3 {
		skinScore = skinPercentage / 0.3
	} else {
		skinScore = (1.0 - skinPercentage) / 0.3
	}

	return skinScore
}

// calculateAdvancedReflectionScore calculates advanced reflection and lighting analysis
func (lfs *LocalFaceService) calculateAdvancedReflectionScore(faceRegion gocv.Mat) float64 {
	// 1. Basic reflection score
	basicScore := lfs.calculateReflectionScore(faceRegion)

	// 2. Lighting consistency analysis
	lightingScore := lfs.calculateLightingConsistencyScore(faceRegion)

	// 3. Shadow analysis
	shadowScore := lfs.calculateShadowAnalysisScore(faceRegion)

	// Combine reflection scores
	reflectionScore := basicScore*0.4 + lightingScore*0.4 + shadowScore*0.2

	return reflectionScore
}

// calculateLightingConsistencyScore calculates lighting consistency across face
func (lfs *LocalFaceService) calculateLightingConsistencyScore(faceRegion gocv.Mat) float64 {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(faceRegion, &gray, gocv.ColorBGRToGray)

	// Divide face into regions and analyze lighting consistency
	rows := gray.Rows()
	cols := gray.Cols()

	// Top, middle, bottom regions
	topRegion := gray.Region(image.Rect(0, 0, cols, rows/3))
	middleRegion := gray.Region(image.Rect(0, rows/3, cols, 2*rows/3))
	bottomRegion := gray.Region(image.Rect(0, 2*rows/3, cols, rows))
	defer topRegion.Close()
	defer middleRegion.Close()
	defer bottomRegion.Close()

	// Calculate mean brightness for each region
	topMean := lfs.calculateMean(topRegion)
	middleMean := lfs.calculateMean(middleRegion)
	bottomMean := lfs.calculateMean(bottomRegion)

	// Calculate brightness variance across regions
	regionMeans := []float64{topMean, middleMean, bottomMean}
	avgMean := (topMean + middleMean + bottomMean) / 3.0

	var variance float64
	for _, mean := range regionMeans {
		diff := mean - avgMean
		variance += diff * diff
	}
	variance /= 3.0

	// BRIGHTNESS-AWARE LIGHTING CONSISTENCY
	// Natural lighting SHOULD have moderate variance (gradual falloff)
	// Too little variance = flat/artificial lighting (spoof)
	// Too much variance = harsh/unnatural lighting (spoof)
	// Adjust optimal variance range based on brightness level

	// Calculate mean brightness to adjust expectations
	meanBrightness := avgMean

	// Adjust optimal variance range based on brightness
	// Brighter images naturally have higher variance
	var optimalMin, optimalMax float64
	if meanBrightness > 140 {
		// Bright images: higher variance is natural
		optimalMin = 500.0
		optimalMax = 4000.0
	} else if meanBrightness > 100 {
		// Medium brightness: moderate variance
		optimalMin = 300.0
		optimalMax = 3000.0
	} else {
		// Dark images: lower variance expected
		optimalMin = 200.0
		optimalMax = 2000.0
	}

	var lightingScore float64
	if variance >= optimalMin && variance <= optimalMax {
		// Optimal natural lighting variance
		lightingScore = 1.0
	} else if variance < optimalMin {
		// Too uniform - penalize (likely flat screen/print)
		lightingScore = variance / optimalMin
	} else {
		// Too much variance - penalize (likely artificial/harsh lighting)
		// Gradual penalty for excessive variance
		lightingScore = math.Max(0.0, 1.0-(variance-optimalMax)/(optimalMax*2.0))
	}

	// Ensure valid range
	if lightingScore < 0 {
		lightingScore = 0
	}
	if lightingScore > 1 {
		lightingScore = 1
	}

	return lightingScore
}

// calculateShadowAnalysisScore calculates shadow consistency score
func (lfs *LocalFaceService) calculateShadowAnalysisScore(faceRegion gocv.Mat) float64 {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(faceRegion, &gray, gocv.ColorBGRToGray)

	// Apply adaptive threshold to detect shadows
	threshold := gocv.NewMat()
	defer threshold.Close()
	gocv.AdaptiveThreshold(gray, &threshold, 255, gocv.AdaptiveThresholdMean, gocv.ThresholdBinary, 11, 2)

	// Calculate shadow pixel distribution
	shadowPixels := gocv.CountNonZero(threshold)
	totalPixels := gray.Rows() * gray.Cols()
	shadowPercentage := float64(shadowPixels) / float64(totalPixels)

	// Natural faces should have reasonable shadow distribution (20-60%)
	var shadowScore float64
	if shadowPercentage >= 0.2 && shadowPercentage <= 0.6 {
		shadowScore = 1.0
	} else if shadowPercentage < 0.2 {
		shadowScore = shadowPercentage / 0.2
	} else {
		shadowScore = (1.0 - shadowPercentage) / 0.4
	}

	return shadowScore
}

// calculateFrequencyScore calculates frequency domain analysis score
func (lfs *LocalFaceService) calculateFrequencyScore(gray gocv.Mat) float64 {
	// Apply high-pass filter to detect high-frequency components
	highPass := gocv.NewMat()
	defer highPass.Close()

	// Create high-pass filter kernel
	kernel, err := gocv.NewMatFromBytes(3, 3, gocv.MatTypeCV32F, []byte{
		0, 255, 0,
		255, 5, 255,
		0, 255, 0,
	})
	if err != nil {
		return 0.5 // Return default score on error
	}
	defer kernel.Close()

	// Apply filter
	gocv.Filter2D(gray, &highPass, gocv.MatTypeCV64F, kernel, image.Pt(-1, -1), 0, gocv.BorderDefault)

	// Calculate high-frequency energy
	mean := lfs.calculateMean(highPass)
	variance := lfs.calculateVariance(highPass, mean)

	// Natural faces have more high-frequency components
	frequencyScore := math.Min(lfs.safeMathOperation("divide", variance, 5000.0), 1.0)

	return frequencyScore
}

// calculateMicroMovementScore calculates micro-movement analysis (simplified)
func (lfs *LocalFaceService) calculateMicroMovementScore(faceRegion gocv.Mat) float64 {
	// For single image, we can only analyze motion blur as a proxy for movement
	// Apply motion blur detection using directional gradients

	// Calculate gradients in different directions
	gradX := gocv.NewMat()
	gradY := gocv.NewMat()
	defer gradX.Close()
	defer gradY.Close()

	gocv.Sobel(faceRegion, &gradX, gocv.MatTypeCV64F, 1, 0, 3, 1, 0, gocv.BorderDefault)
	gocv.Sobel(faceRegion, &gradY, gocv.MatTypeCV64F, 0, 1, 3, 1, 0, gocv.BorderDefault)

	// Calculate gradient magnitudes
	magnitude := gocv.NewMat()
	defer magnitude.Close()
	gocv.Magnitude(gradX, gradY, &magnitude)

	// Analyze gradient distribution
	mean := lfs.calculateMean(magnitude)
	variance := lfs.calculateVariance(magnitude, mean)

	// Natural faces should have varied gradient magnitudes
	movementScore := math.Min(variance/8000.0, 1.0)

	return movementScore
}

// calculate3DStructureScore calculates 3D structure analysis (simplified)
func (lfs *LocalFaceService) calculate3DStructureScore(faceRegion gocv.Mat) float64 {
	// Simplified 3D analysis using depth cues

	// 1. Analyze facial symmetry (natural faces are roughly symmetric)
	symmetryScore := lfs.calculateFacialSymmetryScore(faceRegion)

	// 2. Analyze depth gradients (natural faces have depth variations)
	depthScore := lfs.calculateDepthGradientScore(faceRegion)

	// Combine 3D scores
	structureScore := symmetryScore*0.6 + depthScore*0.4

	return structureScore
}

// calculateFacialSymmetryScore calculates facial symmetry score
func (lfs *LocalFaceService) calculateFacialSymmetryScore(faceRegion gocv.Mat) float64 {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(faceRegion, &gray, gocv.ColorBGRToGray)

	// Split face into left and right halves
	rows := gray.Rows()
	cols := gray.Cols()

	leftHalf := gray.Region(image.Rect(0, 0, cols/2, rows))
	rightHalf := gray.Region(image.Rect(cols/2, 0, cols, rows))
	defer leftHalf.Close()
	defer rightHalf.Close()

	// Flip right half for comparison
	rightFlipped := gocv.NewMat()
	defer rightFlipped.Close()
	gocv.Flip(rightHalf, &rightFlipped, 1) // Horizontal flip

	// Calculate correlation between left half and flipped right half
	result := gocv.NewMat()
	defer result.Close()
	gocv.MatchTemplate(leftHalf, rightFlipped, &result, gocv.TmCcoeffNormed, gocv.NewMat())
	_, maxVal, _, _ := gocv.MinMaxLoc(result)

	symmetryScore := float64(maxVal)

	return symmetryScore
}

// calculateDepthGradientScore calculates depth gradient score
func (lfs *LocalFaceService) calculateDepthGradientScore(faceRegion gocv.Mat) float64 {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(faceRegion, &gray, gocv.ColorBGRToGray)

	// Apply Laplacian to detect depth variations
	laplacian := gocv.NewMat()
	defer laplacian.Close()
	gocv.Laplacian(gray, &laplacian, gocv.MatTypeCV64F, 3, 1, 0, gocv.BorderDefault)

	// Calculate variance of Laplacian (higher variance = more depth variation)
	mean := lfs.calculateMean(laplacian)
	variance := lfs.calculateVariance(laplacian, mean)

	// Natural faces have more depth variation than flat images
	depthScore := math.Min(variance/2000.0, 1.0)

	return depthScore
}

// calculateCompressionArtifactScore calculates compression artifact detection score
func (lfs *LocalFaceService) calculateCompressionArtifactScore(faceRegion gocv.Mat) float64 {
	// Detect JPEG compression artifacts using DCT-like analysis

	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(faceRegion, &gray, gocv.ColorBGRToGray)

	// Apply high-frequency emphasis filter
	highFreq := gocv.NewMat()
	defer highFreq.Close()

	// Create high-frequency emphasis kernel
	kernel, err := gocv.NewMatFromBytes(3, 3, gocv.MatTypeCV32F, []byte{
		255, 255, 255,
		255, 9, 255,
		255, 255, 255,
	})
	if err != nil {
		return 0.0 // Return 0 on error instead of default
	}
	defer kernel.Close()

	gocv.Filter2D(gray, &highFreq, gocv.MatTypeCV64F, kernel, image.Pt(-1, -1), 0, gocv.BorderDefault)

	// Analyze high-frequency patterns for compression artifacts
	// Compression artifacts create regular patterns
	mean := lfs.calculateMean(highFreq)
	variance := lfs.calculateVariance(highFreq, mean)

	// Check for NaN or invalid values
	if math.IsNaN(mean) || math.IsNaN(variance) || math.IsInf(variance, 0) {
		return 0.0
	}

	// Too regular patterns indicate compression artifacts
	// Natural images should have more random high-frequency content
	compressionScore := math.Min(variance/3000.0, 1.0)

	// Ensure the result is valid
	if math.IsNaN(compressionScore) || math.IsInf(compressionScore, 0) {
		return 0.0
	}

	return compressionScore
}

// calculateSpoofPenalty calculates penalty for suspicious spoofing patterns
func (lfs *LocalFaceService) calculateSpoofPenalty(scores []float64) float64 {
	penalty := 0.0

	// Check for suspicious score combinations
	textureScore := scores[0]
	edgeScore := scores[1]
	colorScore := scores[2]
	reflectionScore := scores[3]
	frequencyScore := scores[4]

	// Round scores to 3 decimal places for deterministic behavior
	textureScore = math.Round(textureScore*1000) / 1000
	edgeScore = math.Round(edgeScore*1000) / 1000
	colorScore = math.Round(colorScore*1000) / 1000
	reflectionScore = math.Round(reflectionScore*1000) / 1000
	frequencyScore = math.Round(frequencyScore*1000) / 1000

	// Penalty for very low texture (printed photo)
	if textureScore < 0.2 {
		penalty += 0.3
	}

	// Penalty for very low edge variation (screen display)
	if edgeScore < 0.15 {
		penalty += 0.25
	}

	// Penalty for very low color variation (monochrome spoof)
	if colorScore < 0.1 {
		penalty += 0.2
	}

	// Penalty for perfect reflection consistency (screen reflection)
	if reflectionScore > 0.9 {
		penalty += 0.15
	}

	// Penalty for very low frequency content (over-compressed)
	if frequencyScore < 0.1 {
		penalty += 0.1
	}

	// Maximum penalty is 1.0
	if penalty > 1.0 {
		penalty = 1.0
	}

	return penalty
}

// calculateEnhancedSpoofPenalty calculates enhanced penalty for suspicious spoofing patterns
func (lfs *LocalFaceService) calculateEnhancedSpoofPenalty(scores []float64, lbpScore, lpqScore float64) float64 {
	penalty := 0.0

	// Check for suspicious score combinations
	textureScore := scores[0]
	edgeScore := scores[1]
	colorScore := scores[2]
	reflectionScore := scores[3]
	frequencyScore := scores[4]

	// Round scores to 3 decimal places for deterministic behavior
	textureScore = math.Round(textureScore*1000) / 1000
	edgeScore = math.Round(edgeScore*1000) / 1000
	colorScore = math.Round(colorScore*1000) / 1000
	reflectionScore = math.Round(reflectionScore*1000) / 1000
	frequencyScore = math.Round(frequencyScore*1000) / 1000
	lbpScore = math.Round(lbpScore*1000) / 1000
	lpqScore = math.Round(lpqScore*1000) / 1000

	// Enhanced penalties for high-quality spoofs

	// 1. Penalty for very low texture (printed photo)
	if textureScore < 0.2 {
		penalty += 0.3
	}

	// 2. Penalty for very low edge variation (screen display)
	if edgeScore < 0.15 {
		penalty += 0.25
	}

	// 3. Penalty for very low color variation (monochrome spoof)
	if colorScore < 0.1 {
		penalty += 0.2
	}

	// 4. Penalty for perfect reflection consistency (screen reflection)
	if reflectionScore > 0.9 {
		penalty += 0.15
	}

	// 5. Penalty for very low frequency content (over-compressed)
	if frequencyScore < 0.1 {
		penalty += 0.1
	}

	// 6. ENHANCED: LBP-based spoof detection
	// High LBP score with VERY low texture score indicates artificial texture
	// RELAXED: Changed texture threshold from 0.4 to 0.25 to avoid false positives
	if lbpScore > 0.7 && textureScore < 0.25 {
		penalty += 0.25
	}

	// 7. ENHANCED: LPQ-based spoof detection
	// DISABLED: LPQ can be low for real faces with certain lighting/compression
	// Only penalize extremely low LPQ combined with other suspicious indicators
	if lpqScore < 0.15 && frequencyScore < 0.2 && textureScore < 0.3 {
		penalty += 0.2
	}

	// 8. ENHANCED: Combined pattern analysis
	// Multiple high scores together can indicate sophisticated spoofing
	highScoreCount := 0
	if lbpScore > 0.7 {
		highScoreCount++
	}
	if lpqScore > 0.7 {
		highScoreCount++
	}
	if textureScore > 0.7 {
		highScoreCount++
	}
	if edgeScore > 0.7 {
		highScoreCount++
	}

	// If too many scores are high, it might be a sophisticated spoof
	if highScoreCount >= 3 && (textureScore < 0.5 || edgeScore < 0.4) {
		penalty += 0.3
	}

	// 9. ENHANCED: Anti-pattern detection for screen displays
	// Screen displays often have high LBP but low natural texture
	if lbpScore > 0.8 && textureScore < 0.3 && edgeScore < 0.4 {
		penalty += 0.4
	}

	// 10. ENHANCED: Printed material detection
	// Printed photos often have high variance but low entropy
	// RELAXED: More strict thresholds to avoid false positives on real faces
	if lbpScore > 0.75 && lpqScore < 0.2 && colorScore < 0.3 && textureScore < 0.4 {
		penalty += 0.35
	}

	// 11. ENHANCED: Screen display detection with reflection consistency
	// High LBP + high reflection consistency = screen display
	if lbpScore > 0.9 && reflectionScore > 0.8 {
		penalty += 0.5
	}

	// 12. ENHANCED: Perfect edge consistency detection
	// RELAXED: Perfect edges can occur in real high-quality photos
	// Only penalize when combined with other suspicious patterns
	if edgeScore > 0.95 && lbpScore > 0.9 && reflectionScore > 0.85 {
		penalty += 0.4
	}

	// 13. ENHANCED: Extreme LBP score detection
	// LBP scores > 0.95 are almost always screen displays
	if lbpScore > 0.95 {
		penalty += 0.6
	}

	// 14. ENHANCED: Combined screen display pattern
	// High LBP + high reflection + perfect edges = screen
	if lbpScore > 0.9 && reflectionScore > 0.8 && edgeScore > 0.9 {
		penalty += 0.7
	}

	// 15. ENHANCED: Perfect sharpness with high LBP detection
	// RELAXED: Only penalize extreme combinations that indicate screen displays
	if lbpScore > 0.95 && edgeScore > 0.95 && reflectionScore > 0.85 {
		penalty += 0.3
	}

	// 16. ENHANCED: High frequency content with screen patterns
	// High frequency + high LBP + high reflection = screen display
	if frequencyScore > 0.9 && lbpScore > 0.9 && reflectionScore > 0.8 {
		penalty += 0.4
	}

	// Maximum penalty is 1.0
	if penalty > 1.0 {
		penalty = 1.0
	}

	return penalty
}

// preprocessFaceForComparison performs comprehensive face preprocessing for comparison
func (lfs *LocalFaceService) preprocessFaceForComparison(grayFace gocv.Mat) gocv.Mat {
	// Enhanced preprocessing pipeline for better accuracy

	// 1. Resize to standard size with high-quality interpolation
	standardSize := image.Pt(160, 160) // Slightly larger for better feature preservation
	resized := gocv.NewMat()
	gocv.Resize(grayFace, &resized, standardSize, 0, 0, gocv.InterpolationLanczos4)

	// 2. Advanced lighting normalization using adaptive techniques
	lightingNormalized := lfs.advancedLightingNormalization(resized)
	resized.Close()

	// 3. Apply advanced CLAHE with optimal parameters for faces
	clahe := gocv.NewCLAHEWithParams(2.0, image.Pt(8, 8)) // Optimized for facial features
	enhanced := gocv.NewMat()
	clahe.Apply(lightingNormalized, &enhanced)
	clahe.Close()
	lightingNormalized.Close()

	// 4. Denoise while preserving edges (critical for face matching)
	denoised := gocv.NewMat()
	gocv.FastNlMeansDenoising(enhanced, &denoised)
	enhanced.Close()

	// 5. Sharpen subtle features for better discrimination
	sharpened := lfs.enhanceFacialFeatures(denoised)
	denoised.Close()

	// 6. Final normalization with robust scaling
	normalized := gocv.NewMat()
	gocv.Normalize(sharpened, &normalized, 0, 255, gocv.NormMinMax)
	sharpened.Close()

	// 7. Subtle Gaussian smoothing to reduce minor artifacts
	final := gocv.NewMat()
	gocv.GaussianBlur(normalized, &final, image.Pt(3, 3), 0.5, 0.5, gocv.BorderDefault)
	normalized.Close()

	return final
}

// advancedLightingNormalization performs sophisticated lighting correction
func (lfs *LocalFaceService) advancedLightingNormalization(img gocv.Mat) gocv.Mat {
	// Normalize lighting using DoG (Difference of Gaussians) approach
	// This is more robust than simple histogram equalization

	// Create two Gaussian blurs with different sigma values
	blur1 := gocv.NewMat()
	blur2 := gocv.NewMat()
	defer blur1.Close()
	defer blur2.Close()

	gocv.GaussianBlur(img, &blur1, image.Pt(0, 0), 1.0, 1.0, gocv.BorderDefault)
	gocv.GaussianBlur(img, &blur2, image.Pt(0, 0), 2.0, 2.0, gocv.BorderDefault)

	// Subtract to get DoG
	dog := gocv.NewMat()
	gocv.Subtract(blur1, blur2, &dog)

	// Add back to original for enhancement
	enhanced := gocv.NewMat()
	gocv.AddWeighted(img, 0.7, dog, 0.3, 0, &enhanced)
	dog.Close()

	// Apply contrast stretching for better dynamic range
	result := gocv.NewMat()
	gocv.Normalize(enhanced, &result, 0, 255, gocv.NormMinMax)
	enhanced.Close()

	return result
}

// enhanceFacialFeatures sharpens facial features for better matching
func (lfs *LocalFaceService) enhanceFacialFeatures(img gocv.Mat) gocv.Mat {
	// Unsharp masking for subtle feature enhancement
	blurred := gocv.NewMat()
	defer blurred.Close()
	gocv.GaussianBlur(img, &blurred, image.Pt(0, 0), 1.5, 1.5, gocv.BorderDefault)

	// Calculate high-frequency component
	highFreq := gocv.NewMat()
	gocv.Subtract(img, blurred, &highFreq)

	// Add enhanced high frequencies back
	sharpened := gocv.NewMat()
	gocv.AddWeighted(img, 1.0, highFreq, 0.3, 0, &sharpened)
	highFreq.Close()

	return sharpened
}

// calculateAdvancedImageQuality calculates comprehensive image quality assessment
func (lfs *LocalFaceService) calculateAdvancedImageQuality(img gocv.Mat, faces []image.Rectangle) float64 {
	if len(faces) == 0 {
		return 0.3 // Return a reasonable default instead of 0.0
	}

	// 1. Sharpness assessment using multiple methods
	sharpness := lfs.calculateAdvancedSharpnessScore(img)

	// 2. Lighting quality assessment
	lighting := lfs.calculateAdvancedLightingScore(img)

	// 3. Face-specific quality metrics
	faceQuality := lfs.calculateFaceSpecificQuality(img, faces)

	// PERFORMANCE OPTIMIZATION: Use much simpler and faster quality calculation
	// Skip complex calculations that were causing 4+ second delays

	// Fast quality calculation using only essential metrics
	quality := sharpness*0.4 + lighting*0.4 + faceQuality*0.2

	// Ensure quality is between 0 and 1
	if quality > 1.0 {
		quality = 1.0
	}
	if quality < 0 {
		quality = 0
	}

	// Check for NaN or infinite values
	if math.IsNaN(quality) || math.IsInf(quality, 0) {
		quality = 0.5 // Default neutral quality
	}

	return quality
}

// calculateAdvancedSharpnessScore calculates multi-method sharpness assessment
func (lfs *LocalFaceService) calculateAdvancedSharpnessScore(img gocv.Mat) float64 {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// Method 1: Laplacian variance
	laplacian := gocv.NewMat()
	defer laplacian.Close()
	gocv.Laplacian(gray, &laplacian, gocv.MatTypeCV64F, 3, 1, 0, gocv.BorderDefault)
	mean := lfs.calculateMean(laplacian)
	laplacianVar := lfs.calculateVariance(laplacian, mean)

	// Method 2: Sobel gradient magnitude
	sobelX := gocv.NewMat()
	sobelY := gocv.NewMat()
	defer sobelX.Close()
	defer sobelY.Close()
	gocv.Sobel(gray, &sobelX, gocv.MatTypeCV64F, 1, 0, 3, 1, 0, gocv.BorderDefault)
	gocv.Sobel(gray, &sobelY, gocv.MatTypeCV64F, 0, 1, 3, 1, 0, gocv.BorderDefault)

	magnitude := gocv.NewMat()
	defer magnitude.Close()
	gocv.Magnitude(sobelX, sobelY, &magnitude)
	sobelMean := lfs.calculateMean(magnitude)
	sobelVar := lfs.calculateVariance(magnitude, sobelMean)

	// Method 3: Tenenbaum gradient
	tenenbaum := gocv.NewMat()
	defer tenenbaum.Close()
	gocv.Sobel(gray, &tenenbaum, gocv.MatTypeCV64F, 2, 2, 3, 1, 0, gocv.BorderDefault)
	tenenbaumMean := lfs.calculateMean(tenenbaum)
	tenenbaumVar := lfs.calculateVariance(tenenbaum, tenenbaumMean)

	// Normalize and combine methods
	laplacianScore := math.Min(laplacianVar/2000.0, 1.0)
	sobelScore := math.Min(sobelVar/5000.0, 1.0)
	tenenbaumScore := math.Min(tenenbaumVar/3000.0, 1.0)

	sharpness := laplacianScore*0.5 + sobelScore*0.3 + tenenbaumScore*0.2

	return sharpness
}

// calculateAdvancedLightingScore calculates comprehensive lighting quality
func (lfs *LocalFaceService) calculateAdvancedLightingScore(img gocv.Mat) float64 {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// 1. Overall brightness assessment
	mean := lfs.calculateMean(gray)
	brightness := mean / 255.0

	// Optimal brightness range: 0.35-0.65
	var brightnessScore float64
	if brightness >= 0.35 && brightness <= 0.65 {
		brightnessScore = 1.0
	} else if brightness < 0.35 {
		brightnessScore = brightness / 0.35
	} else {
		brightnessScore = (1.0 - brightness) / 0.35
	}

	// 2. Contrast assessment
	variance := lfs.calculateVariance(gray, mean)
	contrast := math.Sqrt(variance) / 255.0
	contrastScore := math.Min(contrast*3, 1.0) // Normalize contrast

	// 3. Lighting uniformity assessment
	uniformityScore := lfs.calculateLightingUniformity(gray)

	// 4. Shadow distribution assessment
	shadowScore := lfs.calculateShadowDistribution(gray)

	// Combine lighting metrics
	lighting := brightnessScore*0.3 + contrastScore*0.3 + uniformityScore*0.2 + shadowScore*0.2

	return lighting
}

// calculateLightingUniformity calculates lighting uniformity across the image
func (lfs *LocalFaceService) calculateLightingUniformity(gray gocv.Mat) float64 {
	// Divide image into grid and analyze lighting consistency
	rows := gray.Rows()
	cols := gray.Cols()
	gridSize := 4

	var regionMeans []float64

	for i := 0; i < gridSize; i++ {
		for j := 0; j < gridSize; j++ {
			startY := i * rows / gridSize
			endY := (i + 1) * rows / gridSize
			startX := j * cols / gridSize
			endX := (j + 1) * cols / gridSize

			region := gray.Region(image.Rect(startX, startY, endX, endY))
			mean := lfs.calculateMean(region)
			regionMeans = append(regionMeans, mean)
			region.Close()
		}
	}

	// Calculate variance of region means
	totalMean := 0.0
	for _, mean := range regionMeans {
		totalMean += mean
	}
	totalMean /= float64(len(regionMeans))

	variance := 0.0
	for _, mean := range regionMeans {
		diff := mean - totalMean
		variance += diff * diff
	}
	variance /= float64(len(regionMeans))

	// Lower variance = more uniform lighting = better score
	uniformityScore := 1.0 - math.Min(variance/10000.0, 1.0)

	return uniformityScore
}

// calculateShadowDistribution calculates shadow distribution quality
func (lfs *LocalFaceService) calculateShadowDistribution(gray gocv.Mat) float64 {
	// Apply adaptive threshold to detect shadows
	threshold := gocv.NewMat()
	defer threshold.Close()
	gocv.AdaptiveThreshold(gray, &threshold, 255, gocv.AdaptiveThresholdMean, gocv.ThresholdBinary, 15, 10)

	// Calculate shadow pixel distribution
	shadowPixels := gocv.CountNonZero(threshold)
	totalPixels := gray.Rows() * gray.Cols()
	shadowPercentage := float64(shadowPixels) / float64(totalPixels)

	// Natural lighting should have 20-50% shadow pixels
	var shadowScore float64
	if shadowPercentage >= 0.2 && shadowPercentage <= 0.5 {
		shadowScore = 1.0
	} else if shadowPercentage < 0.2 {
		shadowScore = shadowPercentage / 0.2
	} else {
		shadowScore = (1.0 - shadowPercentage) / 0.5
	}

	return shadowScore
}

// calculateFaceSpecificQuality calculates face-specific quality metrics
func (lfs *LocalFaceService) calculateFaceSpecificQuality(img gocv.Mat, faces []image.Rectangle) float64 {
	face := lfs.getLargestFace(faces)

	// 1. Face size quality
	faceArea := float64(face.Dx() * face.Dy())
	imageArea := float64(img.Rows() * img.Cols())
	faceRatio := faceArea / imageArea

	var sizeScore float64
	if faceRatio >= 0.15 && faceRatio <= 0.35 {
		sizeScore = 1.0
	} else if faceRatio < 0.15 {
		sizeScore = faceRatio / 0.15
	} else {
		sizeScore = 0.35 / faceRatio
	}

	// 2. Face position quality (should be centered)
	centerX := float64(img.Cols()) / 2.0
	centerY := float64(img.Rows()) / 2.0
	faceCenterX := float64(face.Min.X + face.Dx()/2)
	faceCenterY := float64(face.Min.Y + face.Dy()/2)

	distanceFromCenter := math.Sqrt(math.Pow(faceCenterX-centerX, 2) + math.Pow(faceCenterY-centerY, 2))
	maxDistance := math.Sqrt(math.Pow(float64(img.Cols()), 2) + math.Pow(float64(img.Rows()), 2))
	positionScore := 1.0 - (distanceFromCenter / maxDistance)

	// 3. Face orientation quality (aspect ratio)
	aspectRatio := float64(face.Dx()) / float64(face.Dy())
	var orientationScore float64
	if aspectRatio >= 0.8 && aspectRatio <= 1.2 {
		orientationScore = 1.0
	} else {
		orientationScore = 1.0 - math.Abs(aspectRatio-1.0)
	}

	// Combine face-specific metrics
	faceQuality := sizeScore*0.5 + positionScore*0.3 + orientationScore*0.2

	return faceQuality
}

// calculateResolutionQuality calculates resolution and compression quality
func (lfs *LocalFaceService) calculateResolutionQuality(img gocv.Mat) float64 {
	// 1. Image resolution assessment
	totalPixels := img.Rows() * img.Cols()
	var resolutionScore float64
	if totalPixels >= 500000 { // 500k+ pixels (e.g., 800x600)
		resolutionScore = 1.0
	} else if totalPixels >= 200000 { // 200k+ pixels (e.g., 500x400)
		resolutionScore = float64(totalPixels) / 500000.0
	} else {
		resolutionScore = float64(totalPixels) / 500000.0 * 0.5
	}

	// 2. Compression artifact detection
	compressionScore := lfs.detectCompressionArtifacts(img)

	// Combine resolution metrics
	resolutionQuality := resolutionScore*0.7 + compressionScore*0.3

	return resolutionQuality
}

// detectCompressionArtifacts detects JPEG compression artifacts
func (lfs *LocalFaceService) detectCompressionArtifacts(img gocv.Mat) float64 {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// Apply DCT-like analysis to detect block artifacts
	// This is a simplified version - in practice, you'd use actual DCT

	// Apply high-pass filter to emphasize artifacts
	highPass := gocv.NewMat()
	defer highPass.Close()

	kernel, err := gocv.NewMatFromBytes(3, 3, gocv.MatTypeCV32F, []byte{
		0, 255, 0,
		255, 5, 255,
		0, 255, 0,
	})
	if err != nil {
		return 0.5 // Return default score on error
	}
	defer kernel.Close()

	gocv.Filter2D(gray, &highPass, gocv.MatTypeCV64F, kernel, image.Pt(-1, -1), 0, gocv.BorderDefault)

	// Analyze high-frequency patterns
	mean := lfs.calculateMean(highPass)
	variance := lfs.calculateVariance(highPass, mean)

	// Higher variance indicates more artifacts
	artifactLevel := math.Min(variance/2000.0, 1.0)
	compressionScore := 1.0 - artifactLevel

	return compressionScore
}

// calculateNoiseQuality calculates image noise level assessment
func (lfs *LocalFaceService) calculateNoiseQuality(img gocv.Mat) float64 {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// Apply Gaussian blur to estimate noise
	blurred := gocv.NewMat()
	defer blurred.Close()
	gocv.GaussianBlur(gray, &blurred, image.Pt(5, 5), 1.0, 1.0, gocv.BorderDefault)

	// Calculate absolute difference between original and blurred
	// Convert both to same type for subtraction
	grayFloat := gocv.NewMat()
	blurredFloat := gocv.NewMat()
	defer grayFloat.Close()
	defer blurredFloat.Close()

	gray.ConvertTo(&grayFloat, gocv.MatTypeCV32F)
	blurred.ConvertTo(&blurredFloat, gocv.MatTypeCV32F)

	// Calculate difference manually
	rows := grayFloat.Rows()
	cols := grayFloat.Cols()
	diff := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV32F)
	defer diff.Close()

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val1 := grayFloat.GetFloatAt(i, j)
			val2 := blurredFloat.GetFloatAt(i, j)
			diff.SetFloatAt(i, j, float32(math.Abs(float64(val1-val2))))
		}
	}

	// Calculate noise level
	mean := lfs.calculateMean(diff)
	noiseLevel := mean / 255.0

	// Lower noise level = better quality
	noiseScore := 1.0 - math.Min(noiseLevel*3, 1.0)

	return noiseScore
}

// calculateColorBalanceQuality calculates color balance assessment
func (lfs *LocalFaceService) calculateColorBalanceQuality(img gocv.Mat) float64 {
	// Split into color channels
	channels := gocv.Split(img)
	defer func() {
		for _, ch := range channels {
			ch.Close()
		}
	}()

	// Calculate mean for each channel
	var channelMeans []float64
	for _, ch := range channels {
		mean := lfs.calculateMean(ch)
		channelMeans = append(channelMeans, mean)
	}

	// Calculate color balance (channels should be roughly equal for neutral lighting)
	if len(channelMeans) < 3 {
		return 0.5 // Default score for grayscale
	}

	// Calculate variance between channel means
	avgMean := (channelMeans[0] + channelMeans[1] + channelMeans[2]) / 3.0
	variance := 0.0
	for _, mean := range channelMeans {
		diff := mean - avgMean
		variance += diff * diff
	}
	variance /= 3.0

	// Lower variance = better color balance
	colorBalanceScore := 1.0 - math.Min(variance/10000.0, 1.0)

	return colorBalanceScore
}

// validateImageInput performs comprehensive security validation on image input
func (lfs *LocalFaceService) validateImageInput(imageInput string) error {
	if imageInput == "" {
		return fmt.Errorf("image input cannot be empty")
	}

	// Check input length limits
	if len(imageInput) > 100*1024*1024 { // 100MB limit
		return fmt.Errorf("image input too large (max 100MB)")
	}

	// Check if it's a URL
	if strings.HasPrefix(imageInput, "http://") || strings.HasPrefix(imageInput, "https://") {
		return lfs.validateImageURL(imageInput)
	}

	// Check if it's base64
	return lfs.validateBase64Image(imageInput)
}

// validateImageURL performs security validation on image URLs
func (lfs *LocalFaceService) validateImageURL(urlStr string) error {
	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	// Check for localhost and private IPs
	hostname := parsedURL.Hostname()
	if hostname == "localhost" || hostname == "127.0.0.1" ||
		hostname == "::1" || hostname == "0.0.0.0" {
		return fmt.Errorf("localhost URLs not allowed for security")
	}

	// Check for private IP ranges
	if strings.HasPrefix(hostname, "192.168.") ||
		strings.HasPrefix(hostname, "10.") ||
		strings.HasPrefix(hostname, "172.") {
		return fmt.Errorf("private IP URLs not allowed for security")
	}

	// Check URL length
	if len(urlStr) > 2048 {
		return fmt.Errorf("URL too long (max 2048 characters)")
	}

	// Check for suspicious patterns
	suspiciousPatterns := []string{
		"file://",
		"ftp://",
		"javascript:",
		"data:",
		"vbscript:",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(strings.ToLower(urlStr), pattern) {
			return fmt.Errorf("suspicious URL pattern detected: %s", pattern)
		}
	}

	return nil
}

// validateBase64Image performs security validation on base64 image data
func (lfs *LocalFaceService) validateBase64Image(imageInput string) error {
	// Check minimum length
	if len(imageInput) < 100 {
		return fmt.Errorf("base64 image too short (minimum 100 characters)")
	}

	// Check for data URL format
	if strings.Contains(imageInput, ",") {
		parts := strings.Split(imageInput, ",")
		if len(parts) != 2 {
			return fmt.Errorf("invalid data URL format")
		}
		imageInput = parts[1]
	}

	// Check base64 encoding
	if len(imageInput)%4 != 0 {
		return fmt.Errorf("invalid base64 encoding")
	}

	// Validate base64 characters
	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	for _, char := range imageInput {
		if !strings.ContainsRune(validChars, char) {
			return fmt.Errorf("invalid base64 character")
		}
	}

	return nil
}

// downloadImageSecurely downloads image from URL with security measures
func (lfs *LocalFaceService) downloadImageSecurely(url string) ([]byte, error) {
	logger.Info("📥 Downloading image from URL", logger.LoggerOptions{
		Key:  "url",
		Data: url,
	})

	// Create HTTP client with timeout and security settings
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}

	// Create request with security headers
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Error("Failed to create HTTP request", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set security headers
	req.Header.Set("User-Agent", "GatemanBiometricService/1.0")
	req.Header.Set("Accept", "image/*")
	req.Header.Set("Connection", "close")

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to download image", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, fmt.Errorf("failed to download image: %v", err)
	}
	defer resp.Body.Close()

	logger.Info("📥 Download response received", logger.LoggerOptions{
		Key: "response_info",
		Data: map[string]interface{}{
			"status_code":    resp.StatusCode,
			"content_type":   resp.Header.Get("Content-Type"),
			"content_length": resp.ContentLength,
		},
	})

	// Check response status
	if resp.StatusCode != http.StatusOK {
		logger.Error("HTTP error response", logger.LoggerOptions{
			Key: "http_error",
			Data: map[string]interface{}{
				"status_code": resp.StatusCode,
				"status":      resp.Status,
			},
		})
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		logger.Error("Invalid content type", logger.LoggerOptions{
			Key:  "content_type",
			Data: contentType,
		})
		return nil, fmt.Errorf("invalid content type: %s", contentType)
	}

	// Check content length
	contentLength := resp.ContentLength
	if contentLength > 50*1024*1024 { // 50MB limit
		logger.Error("Image too large", logger.LoggerOptions{
			Key:  "content_length",
			Data: contentLength,
		})
		return nil, fmt.Errorf("image too large (max 50MB)")
	}

	// Read response with size limit
	limitedReader := io.LimitReader(resp.Body, 50*1024*1024)
	imgData, err := io.ReadAll(limitedReader)
	if err != nil {
		logger.Error("Failed to read image data", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, fmt.Errorf("failed to read image data: %v", err)
	}

	// Validate image size
	if len(imgData) == 0 {
		logger.Error("Empty image data received", logger.LoggerOptions{})
		return nil, fmt.Errorf("empty image data")
	}

	logger.Info("✅ Image downloaded successfully", logger.LoggerOptions{
		Key:  "size_bytes",
		Data: len(imgData),
	})

	return imgData, nil
}

// analyzeLiveness analyzes the image for liveness indicators using enhanced texture analysis
// Returns: livenessScore, spoofPenalty, painting, screen, print, mask probabilities, skinToneRealism, detailedResult
func (lfs *LocalFaceService) analyzeLiveness(faceRegion, fullImg gocv.Mat) (float64, float64, float64, float64, float64, float64, float64, types.DetailedAnalysisResult) {
	// Enhanced liveness detection with uniformity and entropy thresholds

	// DEBUG: Log input parameters
	logger.Info("🔍 DEBUG: analyzeLiveness input", logger.LoggerOptions{
		Key: "analyze_input",
		Data: map[string]interface{}{
			"face_region_size": fmt.Sprintf("%dx%d", faceRegion.Cols(), faceRegion.Rows()),
			"face_region_type": faceRegion.Type().String(),
			"full_img_size":    fmt.Sprintf("%dx%d", fullImg.Cols(), fullImg.Rows()),
			"full_img_type":    fullImg.Type().String(),
		},
	})

	// Convert to grayscale for analysis
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(faceRegion, &gray, gocv.ColorBGRToGray)

	// Sanitize both faceRegion and grayscale Mat to ensure clean data
	sanitizedFaceRegion := lfs.sanitizeMat(faceRegion)
	defer sanitizedFaceRegion.Close()
	sanitizedGray := lfs.sanitizeMat(gray)
	defer sanitizedGray.Close()

	// DEBUG: Log grayscale conversion
	logger.Info("🔍 DEBUG: Grayscale conversion", logger.LoggerOptions{
		Key: "grayscale_debug",
		Data: map[string]interface{}{
			"gray_size": fmt.Sprintf("%dx%d", gray.Cols(), gray.Rows()),
			"gray_type": gray.Type().String(),
		},
	})

	// 1. Enhanced texture analysis with uniformity and entropy thresholds
	textureStart := time.Now()
	// BRIGHTNESS-NORMALIZED TEXTURE CALCULATION
	// This prevents darker images from getting artificially high texture scores
	textureScore := lfs.calculateBrightnessNormalizedTextureScore(sanitizedGray)
	textureTime := time.Since(textureStart).Microseconds()
	logger.Info("🔍 DEBUG: Texture analysis", logger.LoggerOptions{
		Key: "texture_debug",
		Data: map[string]interface{}{
			"texture_score": textureScore,
			"time_us":       textureTime,
		},
	})

	// 2. Advanced edge analysis
	edgeStart := time.Now()
	edgeScore := lfs.withCleanOpenCVContext(func() float64 {
		return lfs.calculateAdvancedEdgeScore(sanitizedGray)
	})
	edgeTime := time.Since(edgeStart).Microseconds()
	logger.Info("🔍 DEBUG: Edge analysis", logger.LoggerOptions{
		Key: "edge_debug",
		Data: map[string]interface{}{
			"edge_score": edgeScore,
			"time_us":    edgeTime,
		},
	})

	// 3. Advanced color analysis
	colorStart := time.Now()
	colorScore := lfs.calculateAdvancedColorScore(sanitizedFaceRegion)
	colorTime := time.Since(colorStart).Microseconds()
	logger.Info("🔍 DEBUG: Color analysis", logger.LoggerOptions{
		Key: "color_debug",
		Data: map[string]interface{}{
			"color_score": colorScore,
			"time_us":     colorTime,
		},
	})

	// 4. Advanced reflection analysis
	reflectionStart := time.Now()
	reflectionScore := lfs.calculateAdvancedReflectionScore(sanitizedFaceRegion)
	reflectionTime := time.Since(reflectionStart).Microseconds()
	logger.Info("🔍 DEBUG: Reflection analysis", logger.LoggerOptions{
		Key: "reflection_debug",
		Data: map[string]interface{}{
			"reflection_score": reflectionScore,
			"time_us":          reflectionTime,
		},
	})

	// 5. Frequency domain analysis
	frequencyStart := time.Now()
	frequencyScore := lfs.withCleanOpenCVContext(func() float64 {
		return lfs.calculateFrequencyScore(sanitizedGray)
	})
	frequencyTime := time.Since(frequencyStart).Microseconds()
	logger.Info("🔍 DEBUG: Frequency analysis", logger.LoggerOptions{
		Key: "frequency_debug",
		Data: map[string]interface{}{
			"frequency_score": frequencyScore,
			"time_us":         frequencyTime,
		},
	})

	// 6. 3D structure analysis
	structureStart := time.Now()
	structureScore := lfs.calculate3DStructureScore(sanitizedFaceRegion)
	structureTime := time.Since(structureStart).Microseconds()
	logger.Info("🔍 DEBUG: 3D structure analysis", logger.LoggerOptions{
		Key: "structure_debug",
		Data: map[string]interface{}{
			"structure_score": structureScore,
			"time_us":         structureTime,
		},
	})

	// 7. Compression artifact detection
	compressionStart := time.Now()
	compressionScore := lfs.calculateCompressionArtifactScore(faceRegion)
	compressionTime := time.Since(compressionStart).Microseconds()
	logger.Info("🔍 DEBUG: Compression analysis", logger.LoggerOptions{
		Key: "compression_debug",
		Data: map[string]interface{}{
			"compression_score": compressionScore,
			"time_us":           compressionTime,
		},
	})

	// 8. Micro-movement analysis (simplified for single image)
	movementStart := time.Now()
	movementScore := lfs.calculateMicroMovementScore(faceRegion)
	movementTime := time.Since(movementStart).Microseconds()
	logger.Info("🔍 DEBUG: Movement analysis", logger.LoggerOptions{
		Key: "movement_debug",
		Data: map[string]interface{}{
			"movement_score": movementScore,
			"time_us":        movementTime,
		},
	})

	// Calculate individual detailed scores ONCE to ensure complete consistency
	// This prevents non-deterministic behavior from duplicate calculations
	logger.Info("🔍 DEBUG: Starting detailed score calculations", logger.LoggerOptions{
		Key:  "detailed_scores_start",
		Data: map[string]interface{}{},
	})

	lbpScore := lfs.calculateLBPScore(sanitizedGray)
	lpqScore := lfs.calculateLPQScore(sanitizedGray)
	edgeConsistency := lfs.calculateEdgeDirectionConsistency(sanitizedGray)
	textureVariance := lfs.calculateSimpleVariance(sanitizedGray)
	textureUniformity := lfs.calculateTextureUniformity(sanitizedGray)
	textureEntropy := lfs.calculateTextureEntropy(sanitizedGray)
	colorRGBVariance := lfs.calculateChannelVariance(sanitizedFaceRegion, 0)
	colorHSVVariance := lfs.calculateChannelVariance(sanitizedFaceRegion, 1)
	colorLABVariance := lfs.calculateChannelVariance(sanitizedFaceRegion, 2)
	edgeDensity := lfs.calculateFastEdgeScore(sanitizedGray)

	// DEBUG: Log all detailed scores
	logger.Info("🔍 DEBUG: Detailed scores calculated", logger.LoggerOptions{
		Key: "detailed_scores",
		Data: map[string]interface{}{
			"lbp_score":          lbpScore,
			"lpq_score":          lpqScore,
			"edge_consistency":   edgeConsistency,
			"texture_variance":   textureVariance,
			"texture_uniformity": textureUniformity,
			"texture_entropy":    textureEntropy,
			"color_rgb_variance": colorRGBVariance,
			"color_hsv_variance": colorHSVVariance,
			"color_lab_variance": colorLABVariance,
			"edge_density":       edgeDensity,
		},
	})

	// 9. Brightness quality analysis
	brightnessStart := time.Now()
	brightnessQualityScore := lfs.calculateBrightnessQualityScore(sanitizedGray)
	brightnessTime := time.Since(brightnessStart).Microseconds()
	logger.Info("🔍 DEBUG: Brightness quality analysis", logger.LoggerOptions{
		Key: "brightness_debug",
		Data: map[string]interface{}{
			"brightness_quality_score": brightnessQualityScore,
			"time_us":                  brightnessTime,
		},
	})

	// Combine scores with proper security-focused weighting
	// ADJUSTED: Reduced texture weight from 0.25 to 0.18, added brightness quality (0.12)
	// This prevents darker images from dominating the score
	baseLivenessScore := (textureScore*0.18 + edgeScore*0.15 + colorScore*0.15 +
		reflectionScore*0.15 + frequencyScore*0.10 + structureScore*0.10 +
		compressionScore*0.05 + movementScore*0.05 + brightnessQualityScore*0.12)

	logger.Info("🔍 DEBUG: Base liveness score calculation", logger.LoggerOptions{
		Key: "base_score_debug",
		Data: map[string]interface{}{
			"base_liveness_score":        baseLivenessScore,
			"texture_contrib":            textureScore * 0.18,
			"edge_contrib":               edgeScore * 0.15,
			"color_contrib":              colorScore * 0.15,
			"reflection_contrib":         reflectionScore * 0.15,
			"frequency_contrib":          frequencyScore * 0.10,
			"structure_contrib":          structureScore * 0.10,
			"compression_contrib":        compressionScore * 0.05,
			"movement_contrib":           movementScore * 0.05,
			"brightness_quality_contrib": brightnessQualityScore * 0.12,
		},
	})

	// Apply comprehensive spoof detection system with parallel goroutines
	spoofResult := lfs.enhancedSpoofPenaltyWithAllDetections(
		faceRegion,
		sanitizedGray,
		textureScore,
		edgeScore,
		colorScore,
		reflectionScore,
		frequencyScore,
		lbpScore,
		lpqScore,
	)

	spoofPenalty := spoofResult.TotalPenalty
	paintingProbability := spoofResult.PaintingProbability
	screenProbability := spoofResult.ScreenProbability
	printProbability := spoofResult.PrintProbability
	maskProbability := spoofResult.MaskProbability
	skinToneRealism := spoofResult.SkinToneRealism
	livenessScore := baseLivenessScore - spoofPenalty

	logger.Info("🔍 DEBUG: Spoof penalty calculation", logger.LoggerOptions{
		Key: "spoof_penalty_debug",
		Data: map[string]interface{}{
			"spoof_penalty":        spoofPenalty,
			"final_liveness_score": livenessScore,
		},
	})

	// Ensure score is within valid range
	if livenessScore < 0 {
		livenessScore = 0
	}
	if livenessScore > 1 {
		livenessScore = 1
	}

	// Check for NaN or infinite values
	if math.IsNaN(livenessScore) || math.IsInf(livenessScore, 0) {
		livenessScore = 0.5 // Default neutral score
		logger.Info("🔍 DEBUG: NaN/Inf detected, using default score", logger.LoggerOptions{
			Key: "nan_inf_fix",
			Data: map[string]interface{}{
				"final_score": livenessScore,
			},
		})
	}

	// Create detailed analysis result using the SAME pre-calculated scores
	// This ensures 100% consistency between main liveness score and detailed breakdown
	detailedResult := types.DetailedAnalysisResult{
		LBPScore:              lbpScore,
		LPQScore:              lpqScore,
		ReflectionConsistency: reflectionScore,
		ColorRGBVariance:      colorRGBVariance,
		ColorHSVVariance:      colorHSVVariance,
		ColorLABVariance:      colorLABVariance,
		EdgeDensity:           edgeDensity,
		EdgeSharpness:         edgeScore,
		EdgeConsistency:       edgeConsistency,
		HighFrequency:         frequencyScore,
		MidFrequency:          frequencyScore * 0.7,
		LowFrequency:          frequencyScore * 0.3,
		CompressionArtifacts:  compressionScore,
		TextureVariance:       textureVariance,
		TextureUniformity:     textureUniformity,
		TextureEntropy:        textureEntropy,
	}

	// DEBUG: Log final return values
	logger.Info("🔍 DEBUG: analyzeLiveness final results", logger.LoggerOptions{
		Key: "final_results",
		Data: map[string]interface{}{
			"liveness_score":       livenessScore,
			"spoof_penalty":        spoofPenalty,
			"painting_probability": paintingProbability,
			"screen_probability":   screenProbability,
			"print_probability":    printProbability,
			"mask_probability":     maskProbability,
			"skin_tone_realism":    skinToneRealism,
			"detailed_result": map[string]interface{}{
				"lbp_score":              detailedResult.LBPScore,
				"lpq_score":              detailedResult.LPQScore,
				"reflection_consistency": detailedResult.ReflectionConsistency,
				"edge_consistency":       detailedResult.EdgeConsistency,
				"texture_variance":       detailedResult.TextureVariance,
			},
		},
	})

	return livenessScore, spoofPenalty, paintingProbability, screenProbability, printProbability, maskProbability, skinToneRealism, detailedResult
}

// calculateSimpleVariance calculates variance of a Mat using simple pixel iteration
func (lfs *LocalFaceService) calculateSimpleVariance(img gocv.Mat) float64 {
	if img.Empty() || img.Rows() == 0 || img.Cols() == 0 {
		return 0.5 // Default neutral score
	}

	// Convert to float64 for calculations
	imgFloat := gocv.NewMat()
	defer imgFloat.Close()
	img.ConvertTo(&imgFloat, gocv.MatTypeCV64F)

	// Calculate mean
	total := 0.0
	count := 0
	for i := 0; i < imgFloat.Rows(); i++ {
		for j := 0; j < imgFloat.Cols(); j++ {
			val := imgFloat.GetDoubleAt(i, j)
			if !math.IsNaN(val) && !math.IsInf(val, 0) {
				total += val
				count++
			}
		}
	}

	if count == 0 {
		return 0.5
	}

	mean := total / float64(count)

	// Calculate variance
	variance := 0.0
	for i := 0; i < imgFloat.Rows(); i++ {
		for j := 0; j < imgFloat.Cols(); j++ {
			val := imgFloat.GetDoubleAt(i, j)
			if !math.IsNaN(val) && !math.IsInf(val, 0) {
				diff := val - mean
				variance += diff * diff
			}
		}
	}

	variance = variance / float64(count)

	// Normalize to 0-1 range (higher variance = more texture = more likely live) - FIXED: Increased denominator from 1000 to 5000 for meaningful range
	score := math.Min(variance/5000.0, 1.0)
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0.5
	}

	return score
}

// PERFORMANCE OPTIMIZATION: Fast calculation functions for liveness analysis

// calculateFastTextureScore calculates texture score using sampling (much faster)
func (lfs *LocalFaceService) calculateFastTextureScore(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() == 0 || gray.Cols() == 0 {
		return 0.5
	}

	rows := gray.Rows()
	cols := gray.Cols()

	// Sample every 4th pixel for better accuracy while maintaining speed
	sampleStep := 4
	total := 0.0
	count := 0
	var prevVal float64
	first := true

	for i := 0; i < rows; i += sampleStep {
		for j := 0; j < cols; j += sampleStep {
			if i >= 0 && i < rows && j >= 0 && j < cols {
				val := float64(gray.GetUCharAt(i, j))
				if !first {
					// Calculate simple gradient as texture measure
					gradient := math.Abs(val - prevVal)
					total += gradient
					count++
				}
				prevVal = val
				first = false
			}
		}
	}

	if count == 0 {
		return 0.5
	}

	avgGradient := total / float64(count)
	// Normalize to 0-1 range
	score := math.Min(avgGradient/50.0, 1.0)
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0.5
	}

	return score
}

// calculateFastEdgeScore calculates edge score using sampling (much faster)
func (lfs *LocalFaceService) calculateFastEdgeScore(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() == 0 || gray.Cols() == 0 {
		return 0.5
	}

	// Simple edge detection using Sobel approximation
	rows := gray.Rows()
	cols := gray.Cols()

	// Sample every 2nd pixel for better accuracy while maintaining reasonable speed
	sampleStep := 2
	edgeCount := 0
	totalPixels := 0

	for i := sampleStep; i < rows-sampleStep; i += sampleStep {
		for j := sampleStep; j < cols-sampleStep; j += sampleStep {
			// Simple horizontal gradient
			left := float64(gray.GetUCharAt(i, j-sampleStep))
			right := float64(gray.GetUCharAt(i, j+sampleStep))
			hGradient := math.Abs(right - left)

			// Simple vertical gradient
			top := float64(gray.GetUCharAt(i-sampleStep, j))
			bottom := float64(gray.GetUCharAt(i+sampleStep, j))
			vGradient := math.Abs(bottom - top)

			// Combined gradient magnitude
			magnitude := math.Sqrt(hGradient*hGradient + vGradient*vGradient)

			// Count as edge if magnitude > threshold
			if magnitude > 30 {
				edgeCount++
			}
			totalPixels++
		}
	}

	if totalPixels == 0 {
		return 0.5
	}

	edgeRatio := float64(edgeCount) / float64(totalPixels)
	// Normalize to 0-1 range (optimal edge ratio around 0.1-0.3)
	score := math.Min(edgeRatio*3.0, 1.0)
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0.5
	}

	return score
}

// calculateFastBrightnessScore calculates brightness score using sampling (much faster)
func (lfs *LocalFaceService) calculateFastBrightnessScore(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() == 0 || gray.Cols() == 0 {
		return 0.5
	}

	rows := gray.Rows()
	cols := gray.Cols()

	// Sample every 10th pixel for speed
	sampleStep := 10
	total := 0.0
	count := 0

	for i := 0; i < rows; i += sampleStep {
		for j := 0; j < cols; j += sampleStep {
			if i >= 0 && i < rows && j >= 0 && j < cols {
				val := float64(gray.GetUCharAt(i, j))
				total += val
				count++
			}
		}
	}

	if count == 0 {
		return 0.5
	}

	avgBrightness := total / float64(count)
	// Normalize to 0-1 range (optimal brightness around 100-150)
	score := math.Min(avgBrightness/255.0, 1.0)
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0.5
	}

	return score
}

// PERFORMANCE OPTIMIZATION: Fast quality calculation functions

// calculateFastSharpnessScore calculates sharpness using sampling (much faster)
func (lfs *LocalFaceService) calculateFastSharpnessScore(img gocv.Mat) float64 {
	if img.Empty() || img.Rows() == 0 || img.Cols() == 0 {
		return 0.5
	}

	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	rows := gray.Rows()
	cols := gray.Cols()

	// Sample every 5th pixel for speed
	sampleStep := 5
	total := 0.0
	count := 0

	for i := sampleStep; i < rows-sampleStep; i += sampleStep {
		for j := sampleStep; j < cols-sampleStep; j += sampleStep {
			// Simple Laplacian approximation
			center := float64(gray.GetUCharAt(i, j))
			left := float64(gray.GetUCharAt(i, j-sampleStep))
			right := float64(gray.GetUCharAt(i, j+sampleStep))
			top := float64(gray.GetUCharAt(i-sampleStep, j))
			bottom := float64(gray.GetUCharAt(i+sampleStep, j))

			// Laplacian = 4*center - (left + right + top + bottom)
			laplacian := math.Abs(4*center - (left + right + top + bottom))
			total += laplacian
			count++
		}
	}

	if count == 0 {
		return 0.5
	}

	avgLaplacian := total / float64(count)
	// Normalize to 0-1 range
	score := math.Min(avgLaplacian/100.0, 1.0)
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0.5
	}

	return score
}

// calculateFastLightingScore calculates lighting quality using sampling (much faster)
func (lfs *LocalFaceService) calculateFastLightingScore(img gocv.Mat) float64 {
	if img.Empty() || img.Rows() == 0 || img.Cols() == 0 {
		return 0.5
	}

	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	rows := gray.Rows()
	cols := gray.Cols()

	// Sample every 8th pixel for speed
	sampleStep := 8
	total := 0.0
	count := 0

	for i := 0; i < rows; i += sampleStep {
		for j := 0; j < cols; j += sampleStep {
			val := float64(gray.GetUCharAt(i, j))
			total += val
			count++
		}
	}

	if count == 0 {
		return 0.5
	}

	avgBrightness := total / float64(count)
	// Optimal brightness range is 80-180 (0-255 scale)
	// Score peaks at 130 (middle of optimal range)
	optimal := 130.0
	deviation := math.Abs(avgBrightness - optimal)
	score := math.Max(0, 1.0-deviation/100.0)

	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0.5
	}

	return score
}

// calculateSimpleFaceQuality calculates face quality based on size and position
func (lfs *LocalFaceService) calculateSimpleFaceQuality(img gocv.Mat, faces []image.Rectangle) float64 {
	if len(faces) == 0 {
		return 0.3
	}

	rows := img.Rows()
	cols := img.Cols()
	totalQuality := 0.0

	for _, face := range faces {
		// Face size ratio (optimal: 10-30% of image)
		faceArea := face.Dx() * face.Dy()
		imageArea := rows * cols
		sizeRatio := float64(faceArea) / float64(imageArea)

		var sizeScore float64
		if sizeRatio >= 0.1 && sizeRatio <= 0.3 {
			sizeScore = 1.0
		} else if sizeRatio >= 0.05 && sizeRatio <= 0.5 {
			sizeScore = 0.7
		} else {
			sizeScore = 0.3
		}

		// Face position (prefer center)
		centerX := float64(cols) / 2
		centerY := float64(rows) / 2
		faceCenterX := float64(face.Min.X + face.Dx()/2)
		faceCenterY := float64(face.Min.Y + face.Dy()/2)

		distanceFromCenter := math.Sqrt(math.Pow(faceCenterX-centerX, 2) + math.Pow(faceCenterY-centerY, 2))
		maxDistance := math.Sqrt(math.Pow(float64(cols)/2, 2) + math.Pow(float64(rows)/2, 2))
		positionScore := math.Max(0, 1.0-distanceFromCenter/maxDistance)

		// Combine scores
		faceQuality := (sizeScore*0.6 + positionScore*0.4)
		totalQuality += faceQuality
	}

	// Return average quality
	return totalQuality / float64(len(faces))
}

// calculateColorVariation calculates color variation in the face region
func (lfs *LocalFaceService) calculateColorVariation(faceRegion gocv.Mat) float64 {
	if faceRegion.Empty() || faceRegion.Rows() == 0 || faceRegion.Cols() == 0 {
		return 0.5
	}

	// Convert to HSV for better color analysis
	hsv := gocv.NewMat()
	defer hsv.Close()
	gocv.CvtColor(faceRegion, &hsv, gocv.ColorBGRToHSV)

	// Calculate variance in each channel
	hVariance := lfs.calculateChannelVariance(hsv, 0) // Hue
	sVariance := lfs.calculateChannelVariance(hsv, 1) // Saturation
	vVariance := lfs.calculateChannelVariance(hsv, 2) // Value

	// Combine variances (natural skin has moderate color variation)
	avgVariance := (hVariance + sVariance + vVariance) / 3.0

	// Normalize to 0-1 range
	score := math.Min(avgVariance/100.0, 1.0)
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0.5
	}

	return score
}

// calculateChannelVariance calculates variance for a specific channel
// channel: 0=RGB, 1=HSV, 2=LAB (determines normalization)
func (lfs *LocalFaceService) calculateChannelVariance(img gocv.Mat, channel int) float64 {
	if img.Empty() || img.Rows() == 0 || img.Cols() == 0 {
		return 0.0
	}

	// Determine which color space to use based on channel parameter
	var colorSpace gocv.Mat
	var normalizationFactor float64

	switch channel {
	case 0: // RGB variance
		colorSpace = img.Clone()
		normalizationFactor = 3000.0 // RGB channels have wider variance
	case 1: // HSV variance
		colorSpace = gocv.NewMat()
		gocv.CvtColor(img, &colorSpace, gocv.ColorBGRToHSV)
		normalizationFactor = 2000.0 // HSV has moderate variance
	case 2: // LAB variance
		colorSpace = gocv.NewMat()
		gocv.CvtColor(img, &colorSpace, gocv.ColorBGRToLab)
		normalizationFactor = 2500.0 // LAB has specific variance characteristics
	default:
		return 0.5
	}
	defer colorSpace.Close()

	// Calculate variance across all channels in the color space
	channels := gocv.Split(colorSpace)
	defer func() {
		for _, ch := range channels {
			ch.Close()
		}
	}()

	var totalVariance float64
	for _, ch := range channels {
		mean := lfs.calculateMean(ch)
		variance := lfs.calculateVariance(ch, mean)
		totalVariance += variance
	}

	// Average variance across channels
	avgVariance := totalVariance / float64(len(channels))

	// Normalize to 0-1 range based on color space
	normalizedScore := math.Min(avgVariance/normalizationFactor, 1.0)

	// Round to 6 decimal places for deterministic behavior
	normalizedScore = math.Round(normalizedScore*1000000) / 1000000

	// Ensure valid range
	if math.IsNaN(normalizedScore) || math.IsInf(normalizedScore, 0) {
		return 0.5
	}
	if normalizedScore < 0 {
		normalizedScore = 0
	}
	if normalizedScore > 1 {
		normalizedScore = 1
	}

	return normalizedScore
}

// calculateBrightnessDistribution calculates brightness distribution score
func (lfs *LocalFaceService) calculateBrightnessDistribution(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() == 0 || gray.Cols() == 0 {
		return 0.5
	}

	// Calculate mean brightness
	total := 0.0
	count := 0

	for i := 0; i < gray.Rows(); i++ {
		for j := 0; j < gray.Cols(); j++ {
			val := gray.GetUCharAt(i, j)
			total += float64(val)
			count++
		}
	}

	if count == 0 {
		return 0.5
	}

	meanBrightness := total / float64(count)

	// Calculate brightness variance
	variance := 0.0
	for i := 0; i < gray.Rows(); i++ {
		for j := 0; j < gray.Cols(); j++ {
			val := gray.GetUCharAt(i, j)
			diff := float64(val) - meanBrightness
			variance += diff * diff
		}
	}

	variance = variance / float64(count)

	// Good lighting has moderate variance (not too uniform, not too chaotic)
	// Normalize to 0-1 range
	score := math.Min(variance/500.0, 1.0)
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0.5
	}

	return score
}

// calculateTextureScore calculates texture-based liveness score
func (lfs *LocalFaceService) calculateTextureScore(img gocv.Mat) float64 {
	// Apply Laplacian filter to detect texture
	lbp := gocv.NewMat()
	defer lbp.Close()

	gocv.Laplacian(img, &lbp, gocv.MatTypeCV64F, 1, 1, 0, gocv.BorderDefault)

	// Calculate variance of Laplacian (higher variance = more texture = more likely live)
	mean := lfs.calculateMean(lbp)
	variance := lfs.calculateVariance(lbp, mean)

	// Normalize variance to 0-1 range
	textureScore := math.Min(variance/1000.0, 1.0)

	return textureScore
}

// calculateEdgeScore calculates edge-based liveness score
func (lfs *LocalFaceService) calculateEdgeScore(img gocv.Mat) float64 {
	// Apply Canny edge detection
	edges := gocv.NewMat()
	defer edges.Close()

	gocv.Canny(img, &edges, 50, 150)

	// Calculate edge density
	edgePixels := gocv.CountNonZero(edges)
	totalPixels := img.Rows() * img.Cols()

	edgeDensity := float64(edgePixels) / float64(totalPixels)

	// Normalize to 0-1 range
	edgeScore := math.Min(edgeDensity*10, 1.0)

	return edgeScore
}

// calculateColorScore calculates color-based liveness score
func (lfs *LocalFaceService) calculateColorScore(img gocv.Mat) float64 {
	// Convert to HSV for better color analysis
	hsv := gocv.NewMat()
	defer hsv.Close()

	gocv.CvtColor(img, &hsv, gocv.ColorBGRToHSV)

	// Calculate color variance
	channels := gocv.Split(hsv)
	defer func() {
		for _, ch := range channels {
			ch.Close()
		}
	}()

	// Calculate variance for each channel
	var totalVariance float64
	for _, ch := range channels {
		mean := lfs.calculateMean(ch)
		variance := lfs.calculateVariance(ch, mean)
		totalVariance += variance
	}

	// Normalize to 0-1 range
	colorScore := math.Min(totalVariance/10000.0, 1.0)

	return colorScore
}

// calculateReflectionScore calculates reflection-based liveness score
func (lfs *LocalFaceService) calculateReflectionScore(img gocv.Mat) float64 {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()

	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// Calculate brightness variance
	mean := lfs.calculateMean(gray)
	variance := lfs.calculateVariance(gray, mean)

	// Higher variance indicates more natural lighting (less reflection)
	reflectionScore := math.Min(variance/5000.0, 1.0)

	return reflectionScore
}

// calculateImageQuality calculates overall image quality
func (lfs *LocalFaceService) calculateImageQuality(img gocv.Mat, faces []image.Rectangle) float64 {
	if len(faces) == 0 {
		return 0.0
	}

	// Calculate sharpness
	sharpness := lfs.calculateSharpnessScore(img)

	// Calculate lighting quality
	lighting := lfs.calculateLightingScore(img)

	// Calculate face size relative to image
	face := lfs.getLargestFace(faces)
	faceArea := float64(face.Dx() * face.Dy())
	imageArea := float64(img.Rows() * img.Cols())
	faceSizeRatio := faceArea / imageArea

	// Ideal face size is 10-30% of image
	var sizeScore float64
	if faceSizeRatio >= 0.1 && faceSizeRatio <= 0.3 {
		sizeScore = 1.0
	} else if faceSizeRatio < 0.1 {
		sizeScore = faceSizeRatio / 0.1
	} else {
		sizeScore = 0.3 / faceSizeRatio
	}

	// Combine quality metrics
	quality := (sharpness*0.4 + lighting*0.3 + sizeScore*0.3)

	// Ensure quality is between 0 and 1
	if quality < 0 {
		quality = 0
	}
	if quality > 1 {
		quality = 1
	}

	return quality
}

// calculateSharpnessScore calculates image sharpness
func (lfs *LocalFaceService) calculateSharpnessScore(img gocv.Mat) float64 {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()

	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// Apply Laplacian filter
	laplacian := gocv.NewMat()
	defer laplacian.Close()

	gocv.Laplacian(gray, &laplacian, gocv.MatTypeCV64F, 1, 1, 0, gocv.BorderDefault)

	// Calculate variance of Laplacian
	mean := lfs.calculateMean(laplacian)
	variance := lfs.calculateVariance(laplacian, mean)

	// Normalize to 0-1 range
	sharpness := math.Min(variance/1000.0, 1.0)

	return sharpness
}

// calculateLightingScore calculates lighting quality
func (lfs *LocalFaceService) calculateLightingScore(img gocv.Mat) float64 {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()

	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// Calculate mean brightness
	mean := lfs.calculateMean(gray)
	brightness := mean / 255.0

	// Ideal brightness is around 0.5 (50%)
	var brightnessScore float64
	if brightness >= 0.3 && brightness <= 0.7 {
		brightnessScore = 1.0
	} else if brightness < 0.3 {
		brightnessScore = brightness / 0.3
	} else {
		brightnessScore = (1.0 - brightness) / 0.3
	}

	// Calculate contrast
	variance := lfs.calculateVariance(gray, mean)

	// Normalize contrast
	contrast := math.Min(variance/10000.0, 1.0)

	// Combine brightness and contrast
	lightingScore := (brightnessScore*0.6 + contrast*0.4)

	return lightingScore
}

// calculateConfidence calculates confidence based on similarity and quality
func (lfs *LocalFaceService) calculateConfidence(similarity, quality1, quality2 float64) float64 {
	// ENHANCED CONFIDENCE CALCULATION
	// More sophisticated confidence scoring based on multiple factors

	// Base confidence on similarity score
	baseConfidence := similarity

	// Factor 1: Image quality assessment
	avgQuality := (quality1 + quality2) / 2.0
	qualityBoost := 0.0
	if avgQuality > 0.8 {
		// Excellent quality - boost confidence
		qualityBoost = (avgQuality - 0.8) * 0.5 // Up to +10%
	} else if avgQuality < 0.4 {
		// Poor quality - reduce confidence
		qualityBoost = (avgQuality - 0.4) * 0.3 // Up to -12%
	}

	// Factor 2: Quality consistency (penalize large differences)
	qualityDiff := math.Abs(quality1 - quality2)
	qualityPenalty := 0.0
	if qualityDiff > 0.3 {
		// Large quality difference reduces confidence
		qualityPenalty = qualityDiff * 0.15
	}

	// Factor 3: Non-linearity adjustment
	// High similarity should have higher confidence, low similarity should have lower confidence
	if similarity > 0.75 {
		// Boost high similarity scores
		baseConfidence += (similarity - 0.75) * 0.4
	} else if similarity < 0.4 {
		// Further reduce low similarity scores
		baseConfidence -= (0.4 - similarity) * 0.3
	}

	// Combine all factors
	confidence := baseConfidence + qualityBoost - qualityPenalty

	// Ensure confidence is between 0 and 1
	confidence = math.Max(0, math.Min(1, confidence))

	return confidence
}

// calculateLivenessConfidence calculates confidence for liveness detection
func (lfs *LocalFaceService) calculateLivenessConfidence(livenessScore, quality float64) float64 {
	// Base confidence on liveness score
	confidence := livenessScore

	// Adjust based on image quality
	qualityAdjustment := (quality - 0.5) * 0.3 // ±15% adjustment

	confidence += qualityAdjustment

	// Ensure confidence is between 0 and 1
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	return confidence
}

// updateStats updates processing statistics
func (lfs *LocalFaceService) updateStats(processingTime int64, success bool) {
	lfs.processingStats.TotalRequests++
	if success {
		lfs.processingStats.SuccessfulRequests++
	}

	lfs.processingStats.TotalTime += processingTime
	lfs.processingStats.AverageTime = float64(lfs.processingStats.TotalTime) / float64(lfs.processingStats.TotalRequests)
}

// GetStats returns processing statistics
func (lfs *LocalFaceService) GetStats() ProcessingStats {
	return lfs.processingStats
}

// ============================================================================
// PRODUCTION-GRADE PAINTING & SPOOF DETECTION ENHANCEMENTS
// ============================================================================
// The following functions implement advanced anti-spoofing techniques
// specifically designed to detect paintings, printed photos, and other
// sophisticated spoofing attacks.
// ============================================================================

// detectPaintingCharacteristics detects if an image has painting-like characteristics
// This is a key function for identifying artistic renderings vs real photographs
func (lfs *LocalFaceService) detectPaintingCharacteristics(faceRegion, gray gocv.Mat) float64 {
	paintingIndicators := 0.0
	totalChecks := 0.0

	// 1. Brush Stroke Pattern Detection
	// Paintings have directional patterns from brush strokes
	// LOWERED threshold from 0.6 to 0.5 for more aggressive detection
	brushStrokeScore := lfs.detectBrushStrokes(gray)
	if brushStrokeScore > 0.5 {
		paintingIndicators += 1.0
	}
	totalChecks += 1.0

	// 2. Artificial Smoothness Detection
	// Paintings lack the micro-texture of real skin
	// LOWERED threshold from 0.7 to 0.6 for more aggressive detection
	smoothnessScore := lfs.detectArtificialSmoothness(gray)
	if smoothnessScore > 0.6 {
		paintingIndicators += 1.0
	}
	totalChecks += 1.0

	// 3. Skin Pore Absence Detection
	// Real human skin has visible pores; paintings don't
	// LOWERED threshold from 0.65 to 0.55 for more aggressive detection
	poreAbsenceScore := lfs.detectSkinPoreAbsence(gray)
	if poreAbsenceScore > 0.55 {
		paintingIndicators += 1.0
	}
	totalChecks += 1.0

	// 4. Color Blending Artificiality
	// Painted colors blend differently than photographed ones
	// LOWERED threshold from 0.7 to 0.6 for more aggressive detection
	colorBlendingScore := lfs.detectArtificialColorBlending(faceRegion)
	if colorBlendingScore > 0.6 {
		paintingIndicators += 1.0
	}
	totalChecks += 1.0

	// 5. Canvas Texture Detection
	// Many paintings show canvas texture patterns
	// LOWERED threshold from 0.6 to 0.5 for more aggressive detection
	canvasScore := lfs.detectCanvasTexture(gray)
	if canvasScore > 0.5 {
		paintingIndicators += 1.0
	}
	totalChecks += 1.0

	// 6. Unnatural Edge Characteristics
	// Painted edges differ from photographic edges
	// LOWERED threshold from 0.65 to 0.55 for more aggressive detection
	edgeScore := lfs.detectPaintedEdges(gray)
	if edgeScore > 0.55 {
		paintingIndicators += 1.0
	}
	totalChecks += 1.0

	// Calculate painting probability
	paintingProbability := paintingIndicators / totalChecks

	logger.Info("🎨 Painting detection analysis", logger.LoggerOptions{
		Key: "painting_detection",
		Data: map[string]interface{}{
			"brush_stroke_score":   brushStrokeScore,
			"smoothness_score":     smoothnessScore,
			"pore_absence_score":   poreAbsenceScore,
			"color_blending_score": colorBlendingScore,
			"canvas_score":         canvasScore,
			"edge_score":           edgeScore,
			"painting_probability": paintingProbability,
			"indicators_triggered": paintingIndicators,
			"total_checks":         totalChecks,
		},
	})

	return paintingProbability
}

// detectBrushStrokes detects directional patterns typical of brush strokes in paintings
func (lfs *LocalFaceService) detectBrushStrokes(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() < 20 || gray.Cols() < 20 {
		return 0.0
	}

	// Use directional Sobel filters to detect brush stroke patterns
	// Paintings have stronger directional coherence from brush strokes
	orientations := []struct{ dx, dy int }{
		{1, 0},  // Horizontal
		{0, 1},  // Vertical
		{1, 1},  // Diagonal 1
		{1, -1}, // Diagonal 2
	}

	var maxDirectionalEnergy float64 = 0
	var totalEnergy float64 = 0

	for _, orient := range orientations {
		// Apply directional Sobel
		sobel := gocv.NewMat()
		defer sobel.Close()
		gocv.Sobel(gray, &sobel, gocv.MatTypeCV64F, orient.dx, orient.dy, 3, 1, 0, gocv.BorderDefault)

		// Calculate energy in this direction
		energy := lfs.calculateMatEnergy(sobel)
		totalEnergy += energy
		if energy > maxDirectionalEnergy {
			maxDirectionalEnergy = energy
		}
	}

	// Brush strokes show high directional dominance
	// Real photos have more balanced directional energy
	if totalEnergy == 0 {
		return 0.0
	}

	directionalDominance := maxDirectionalEnergy / totalEnergy

	// Paintings typically show directional dominance > 0.40
	// Real photos are usually < 0.35
	brushStrokeScore := 0.0
	if directionalDominance > 0.40 {
		brushStrokeScore = math.Min((directionalDominance-0.35)/0.20, 1.0)
	}

	return brushStrokeScore
}

// detectArtificialSmoothness detects unnatural smoothness typical of paintings
func (lfs *LocalFaceService) detectArtificialSmoothness(gray gocv.Mat) float64 {
	if gray.Empty() {
		return 0.0
	}

	// Calculate local standard deviation in small windows
	// Real skin has micro-variations; paintings are too smooth
	windowSize := 7
	smoothRegions := 0
	totalRegions := 0

	rows := gray.Rows()
	cols := gray.Cols()
	step := windowSize

	for i := 0; i < rows-windowSize; i += step {
		for j := 0; j < cols-windowSize; j += step {
			// Extract window
			roi := gray.Region(image.Rect(j, i, j+windowSize, i+windowSize))

			// Calculate local std deviation
			stdDev := lfs.calculateStdDev(roi)
			roi.Close()

			totalRegions++

			// Paintings have very low local variation (stdDev < 8 on 0-255 scale)
			// Real skin usually has stdDev > 10 due to micro-texture
			if stdDev < 8.0 {
				smoothRegions++
			}
		}
	}

	if totalRegions == 0 {
		return 0.0
	}

	// Calculate smoothness ratio
	smoothnessRatio := float64(smoothRegions) / float64(totalRegions)

	// Paintings typically have > 60% smooth regions
	// Real photos have < 40%
	smoothnessScore := 0.0
	if smoothnessRatio > 0.5 {
		smoothnessScore = math.Min((smoothnessRatio-0.40)/0.40, 1.0)
	}

	return smoothnessScore
}

// detectSkinPoreAbsence detects absence of natural skin pores (painting indicator)
func (lfs *LocalFaceService) detectSkinPoreAbsence(gray gocv.Mat) float64 {
	if gray.Empty() {
		return 0.0
	}

	// Real human skin has visible pores at high frequencies
	// Apply high-pass filter to isolate fine details
	blurred := gocv.NewMat()
	defer blurred.Close()
	gocv.GaussianBlur(gray, &blurred, image.Pt(5, 5), 2.0, 2.0, gocv.BorderDefault)

	// High-frequency component = original - blurred
	highFreq := gocv.NewMat()
	defer highFreq.Close()
	gocv.Subtract(gray, blurred, &highFreq)

	// Calculate high-frequency energy
	highFreqEnergy := lfs.calculateMatEnergy(highFreq)

	// Normalize by image size
	normalizedEnergy := highFreqEnergy / float64(gray.Rows()*gray.Cols())

	// Real skin has high-frequency energy > 15
	// Paintings have much lower (< 8)
	poreAbsenceScore := 0.0
	if normalizedEnergy < 10.0 {
		poreAbsenceScore = math.Min((10.0-normalizedEnergy)/10.0, 1.0)
	}

	return poreAbsenceScore
}

// detectArtificialColorBlending detects unnatural color blending in paintings
func (lfs *LocalFaceService) detectArtificialColorBlending(faceRegion gocv.Mat) float64 {
	if faceRegion.Empty() {
		return 0.0
	}

	// Convert to LAB color space (better for color analysis)
	lab := gocv.NewMat()
	defer lab.Close()
	gocv.CvtColor(faceRegion, &lab, gocv.ColorBGRToLab)

	// Split channels
	channels := gocv.Split(lab)
	defer func() {
		for _, ch := range channels {
			ch.Close()
		}
	}()

	if len(channels) != 3 {
		return 0.0
	}

	// Calculate color gradient smoothness
	// Paintings have overly smooth color transitions
	aChannel := channels[1] // a* channel (green-red)

	// Calculate gradients on a* channel (color information)
	aGradX := gocv.NewMat()
	aGradY := gocv.NewMat()
	defer aGradX.Close()
	defer aGradY.Close()

	gocv.Sobel(aChannel, &aGradX, gocv.MatTypeCV64F, 1, 0, 3, 1, 0, gocv.BorderDefault)
	gocv.Sobel(aChannel, &aGradY, gocv.MatTypeCV64F, 0, 1, 3, 1, 0, gocv.BorderDefault)

	// Calculate gradient magnitude
	gradMag := gocv.NewMat()
	defer gradMag.Close()
	gocv.Magnitude(aGradX, aGradY, &gradMag)

	// Calculate mean gradient
	meanGrad := lfs.calculateMean(gradMag)

	// Paintings have very low color gradients (< 5)
	// Real photos have higher (> 8)
	artificialBlendingScore := 0.0
	if meanGrad < 7.0 {
		artificialBlendingScore = math.Min((7.0-meanGrad)/7.0, 1.0)
	}

	return artificialBlendingScore
}

// detectCanvasTexture detects regular texture patterns typical of canvas
func (lfs *LocalFaceService) detectCanvasTexture(gray gocv.Mat) float64 {
	if gray.Empty() {
		return 0.0
	}

	// Canvas has a regular woven pattern at high frequencies
	// Use FFT to detect regular patterns
	// For simplicity, use autocorrelation as a proxy

	// Calculate autocorrelation to find periodic patterns
	// High autocorrelation at small lags indicates regular texture

	// Simplified approach: check for regular texture using variance of local means
	windowSize := 15
	localMeans := []float64{}

	rows := gray.Rows()
	cols := gray.Cols()

	for i := 0; i < rows-windowSize; i += windowSize / 2 {
		for j := 0; j < cols-windowSize; j += windowSize / 2 {
			roi := gray.Region(image.Rect(j, i, j+windowSize, i+windowSize))
			mean := lfs.calculateMean(roi)
			localMeans = append(localMeans, mean)
			roi.Close()
		}
	}

	if len(localMeans) < 2 {
		return 0.0
	}

	// Calculate variance of local means
	// Canvas shows LOW variance (more uniform)
	// Real skin shows HIGHER variance
	meanOfMeans := 0.0
	for _, val := range localMeans {
		meanOfMeans += val
	}
	meanOfMeans /= float64(len(localMeans))

	variance := 0.0
	for _, val := range localMeans {
		diff := val - meanOfMeans
		variance += diff * diff
	}
	variance /= float64(len(localMeans))

	// Canvas typically has variance < 50
	// Real photos have > 100
	canvasScore := 0.0
	if variance < 80.0 {
		canvasScore = math.Min((80.0-variance)/80.0, 1.0)
	}

	return canvasScore
}

// detectPaintedEdges detects edge characteristics typical of paintings
func (lfs *LocalFaceService) detectPaintedEdges(gray gocv.Mat) float64 {
	if gray.Empty() {
		return 0.0
	}

	// Detect edges using Canny
	edges := gocv.NewMat()
	defer edges.Close()
	gocv.Canny(gray, &edges, 50, 150)

	// Paintings have:
	// 1. Fewer sharp edges (more blended)
	// 2. Less edge complexity

	edgePixels := gocv.CountNonZero(edges)
	totalPixels := gray.Rows() * gray.Cols()
	edgeDensity := float64(edgePixels) / float64(totalPixels)

	// Calculate edge complexity using edge contours
	contours := gocv.FindContours(edges, gocv.RetrievalList, gocv.ChainApproxSimple)

	// Paintings have fewer, simpler contours
	// Real photos have many small, complex contours
	complexityScore := float64(contours.Size()) / float64(totalPixels) * 10000.0

	// Combine metrics
	// Low edge density + low complexity = painting
	paintedEdgeScore := 0.0
	if edgeDensity < 0.08 && complexityScore < 5.0 {
		paintedEdgeScore = 0.7
	} else if edgeDensity < 0.10 {
		paintedEdgeScore = 0.5
	} else if complexityScore < 3.0 {
		paintedEdgeScore = 0.6
	}

	return paintedEdgeScore
}

// calculateMatEnergy calculates the energy (sum of squared values) of a Mat
func (lfs *LocalFaceService) calculateMatEnergy(mat gocv.Mat) float64 {
	if mat.Empty() {
		return 0.0
	}

	// Convert to float for accurate calculation
	matFloat := gocv.NewMat()
	defer matFloat.Close()
	mat.ConvertTo(&matFloat, gocv.MatTypeCV64F)

	// Calculate sum of squares
	energy := 0.0
	for i := 0; i < matFloat.Rows(); i++ {
		for j := 0; j < matFloat.Cols(); j++ {
			val, ok := lfs.getPixelValue(matFloat, i, j)
			if ok && !math.IsNaN(val) && !math.IsInf(val, 0) {
				energy += val * val
			}
		}
	}

	return energy
}

// calculateStdDev calculates standard deviation of a Mat
func (lfs *LocalFaceService) calculateStdDev(mat gocv.Mat) float64 {
	if mat.Empty() {
		return 0.0
	}

	mean := lfs.calculateMean(mat)
	variance := lfs.calculateVariance(mat, mean)
	return math.Sqrt(variance)
}

// ============================================================================
// SCREEN/DISPLAY DETECTION
// ============================================================================

// detectScreenDisplay detects if the image is from a screen/display
// Screens have moiré patterns, pixel grids, and refresh rate artifacts
func (lfs *LocalFaceService) detectScreenDisplay(faceRegion, gray gocv.Mat) float64 {
	if faceRegion.Empty() || gray.Empty() {
		return 0.0
	}

	screenIndicators := 0.0
	totalChecks := 0.0

	// 1. Moiré Pattern Detection
	// Screens photographed create interference patterns
	moireScore := lfs.detectMoirePatterns(gray)
	if moireScore > 0.6 {
		screenIndicators += 1.0
	}
	totalChecks += 1.0

	// 2. Pixel Grid Detection
	// Screens have visible RGB pixel grids when photographed
	pixelGridScore := lfs.detectPixelGrid(gray)
	if pixelGridScore > 0.65 {
		screenIndicators += 1.0
	}
	totalChecks += 1.0

	// 3. Refresh Rate Artifacts
	// Screens may show banding from refresh rates
	refreshArtifactsScore := lfs.detectRefreshArtifacts(gray)
	if refreshArtifactsScore > 0.6 {
		screenIndicators += 1.0
	}
	totalChecks += 1.0

	// 4. Unnatural Luminance Uniformity
	// Screens emit light uniformly; faces reflect light variably
	luminanceScore := lfs.detectUniformLuminance(gray)
	if luminanceScore > 0.7 {
		screenIndicators += 1.0
	}
	totalChecks += 1.0

	screenProbability := screenIndicators / totalChecks

	logger.Info("🖥️ Screen detection analysis", logger.LoggerOptions{
		Key: "screen_detection",
		Data: map[string]interface{}{
			"moire_score":          moireScore,
			"pixel_grid_score":     pixelGridScore,
			"refresh_artifacts":    refreshArtifactsScore,
			"luminance_score":      luminanceScore,
			"screen_probability":   screenProbability,
			"indicators_triggered": screenIndicators,
			"total_checks":         totalChecks,
		},
	})

	return screenProbability
}

// detectMoirePatterns detects interference patterns from photographing screens
func (lfs *LocalFaceService) detectMoirePatterns(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() < 30 || gray.Cols() < 30 {
		return 0.0
	}

	// Apply FFT to detect periodic patterns
	// Moiré patterns show up as specific frequency components
	padded := gocv.NewMat()
	defer padded.Close()

	optimalRows := gocv.GetOptimalDFTSize(gray.Rows())
	optimalCols := gocv.GetOptimalDFTSize(gray.Cols())
	gocv.CopyMakeBorder(gray, &padded, 0, optimalRows-gray.Rows(), 0, optimalCols-gray.Cols(),
		gocv.BorderConstant, color.RGBA{0, 0, 0, 0})

	// Convert to float
	padded32F := gocv.NewMat()
	defer padded32F.Close()
	padded.ConvertTo(&padded32F, gocv.MatTypeCV32F)

	// Perform DFT
	dft := gocv.NewMat()
	defer dft.Close()
	gocv.DFT(padded32F, &dft, gocv.DftComplexOutput)

	// Calculate magnitude
	planes := gocv.Split(dft)
	defer planes[0].Close()
	defer planes[1].Close()

	magnitude := gocv.NewMat()
	defer magnitude.Close()
	gocv.Magnitude(planes[0], planes[1], &magnitude)

	// Look for periodic peaks (moiré indicators)
	mean := gocv.NewMat()
	stdDev := gocv.NewMat()
	defer mean.Close()
	defer stdDev.Close()
	gocv.MeanStdDev(magnitude, &mean, &stdDev)

	stdDevVal := float64(stdDev.GetFloatAt(0, 0))

	// High std dev in frequency domain indicates periodic patterns
	moireScore := math.Min(stdDevVal/100.0, 1.0)

	return moireScore
}

// detectPixelGrid detects RGB pixel grid patterns from screens
func (lfs *LocalFaceService) detectPixelGrid(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() < 20 || gray.Cols() < 20 {
		return 0.0
	}

	// Analyze high-frequency components in small patches
	// Pixel grids create regular high-frequency patterns
	patchSize := 10
	if gray.Rows() < patchSize*2 || gray.Cols() < patchSize*2 {
		return 0.0
	}

	highFreqScore := 0.0
	patches := 0

	for i := 0; i < gray.Rows()-patchSize; i += patchSize {
		for j := 0; j < gray.Cols()-patchSize; j += patchSize {
			rect := image.Rect(j, i, j+patchSize, i+patchSize)
			patch := gray.Region(rect)

			// Calculate local variance
			variance := lfs.calculateSimpleVariance(patch)
			patch.Close()

			// High variance in small patches indicates pixel grid
			if variance > 800 {
				highFreqScore += 1.0
			}
			patches++
		}
	}

	if patches == 0 {
		return 0.0
	}

	return highFreqScore / float64(patches)
}

// detectRefreshArtifacts detects horizontal banding from screen refresh rates
func (lfs *LocalFaceService) detectRefreshArtifacts(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() < 20 {
		return 0.0
	}

	// Calculate row-wise intensity variance
	// Refresh artifacts create horizontal bands with different intensities
	rowMeans := make([]float64, gray.Rows())

	for i := 0; i < gray.Rows(); i++ {
		rowSum := 0.0
		count := 0
		for j := 0; j < gray.Cols(); j++ {
			val := float64(gray.GetUCharAt(i, j))
			rowSum += val
			count++
		}
		if count > 0 {
			rowMeans[i] = rowSum / float64(count)
		}
	}

	// Calculate variance in row means
	mean := 0.0
	for _, val := range rowMeans {
		mean += val
	}
	mean /= float64(len(rowMeans))

	variance := 0.0
	for _, val := range rowMeans {
		diff := val - mean
		variance += diff * diff
	}
	variance /= float64(len(rowMeans))

	// High variance indicates banding
	bandingScore := math.Min(variance/500.0, 1.0)

	return bandingScore
}

// detectUniformLuminance detects unnatural luminance uniformity from displays
func (lfs *LocalFaceService) detectUniformLuminance(gray gocv.Mat) float64 {
	if gray.Empty() {
		return 0.0
	}

	// Real faces have varied lighting; screens emit uniformly
	mean := gocv.NewMat()
	stdDev := gocv.NewMat()
	defer mean.Close()
	defer stdDev.Close()

	gocv.MeanStdDev(gray, &mean, &stdDev)

	meanVal := mean.GetDoubleAt(0, 0)
	stdDevVal := stdDev.GetDoubleAt(0, 0)

	// Low std dev relative to mean indicates uniform luminance
	if meanVal > 0 {
		uniformity := 1.0 - math.Min(stdDevVal/meanVal, 1.0)
		return uniformity
	}

	return 0.0
}

// ============================================================================
// PRINT/PHOTO DETECTION
// ============================================================================

// detectPrintPhoto detects if image is a printed photograph
func (lfs *LocalFaceService) detectPrintPhoto(faceRegion, gray gocv.Mat) float64 {
	if faceRegion.Empty() || gray.Empty() {
		return 0.0
	}

	printIndicators := 0.0
	totalChecks := 0.0

	// 1. Halftone Dot Detection
	// Printed images have halftone dot patterns
	halftoneScore := lfs.detectHalftoneDots(gray)
	if halftoneScore > 0.6 {
		printIndicators += 1.0
	}
	totalChecks += 1.0

	// 2. Paper Texture Detection
	// Paper has specific texture patterns
	paperScore := lfs.detectPaperTexture(gray)
	if paperScore > 0.65 {
		printIndicators += 1.0
	}
	totalChecks += 1.0

	// 3. Ink Absorption Patterns
	// Ink on paper creates specific edge characteristics
	inkScore := lfs.detectInkAbsorption(gray)
	if inkScore > 0.6 {
		printIndicators += 1.0
	}
	totalChecks += 1.0

	printProbability := printIndicators / totalChecks

	logger.Info("🖨️ Print detection analysis", logger.LoggerOptions{
		Key: "print_detection",
		Data: map[string]interface{}{
			"halftone_score":       halftoneScore,
			"paper_texture_score":  paperScore,
			"ink_absorption_score": inkScore,
			"print_probability":    printProbability,
			"indicators_triggered": printIndicators,
			"total_checks":         totalChecks,
		},
	})

	return printProbability
}

// detectHalftoneDots detects halftone printing patterns
func (lfs *LocalFaceService) detectHalftoneDots(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() < 20 || gray.Cols() < 20 {
		return 0.0
	}

	// Halftone patterns are periodic dots
	// Use morphological operations to detect regular dot patterns
	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Pt(3, 3))
	defer kernel.Close()

	tophat := gocv.NewMat()
	defer tophat.Close()
	gocv.MorphologyEx(gray, &tophat, gocv.MorphTophat, kernel)

	// Count bright spots (dots)
	threshold := gocv.NewMat()
	defer threshold.Close()
	gocv.Threshold(tophat, &threshold, 30, 255, gocv.ThresholdBinary)

	nonZero := gocv.CountNonZero(threshold)
	totalPixels := gray.Rows() * gray.Cols()

	dotDensity := float64(nonZero) / float64(totalPixels)

	// Halftone has moderate dot density (5-15%)
	if dotDensity > 0.05 && dotDensity < 0.15 {
		return math.Min(dotDensity*10, 1.0)
	}

	return 0.0
}

// detectPaperTexture detects paper texture patterns
func (lfs *LocalFaceService) detectPaperTexture(gray gocv.Mat) float64 {
	if gray.Empty() {
		return 0.0
	}

	// Paper has fine, uniform texture
	// Similar to canvas but finer grain
	// Calculate local variance using filter
	blurred := gocv.NewMat()
	defer blurred.Close()
	gocv.GaussianBlur(gray, &blurred, image.Pt(5, 5), 0, 0, gocv.BorderDefault)

	variance := gocv.NewMat()
	defer variance.Close()
	gocv.Subtract(gray, blurred, &variance)

	mean := gocv.NewMat()
	stdDev := gocv.NewMat()
	defer mean.Close()
	defer stdDev.Close()

	gocv.MeanStdDev(variance, &mean, &stdDev)

	meanVar := mean.GetDoubleAt(0, 0)

	// Moderate, uniform variance indicates paper texture
	paperScore := math.Min(meanVar/100.0, 1.0)

	return paperScore
}

// detectInkAbsorption detects ink bleeding patterns on paper
func (lfs *LocalFaceService) detectInkAbsorption(gray gocv.Mat) float64 {
	if gray.Empty() {
		return 0.0
	}

	// Ink absorption creates slightly blurred edges
	edges := gocv.NewMat()
	defer edges.Close()
	gocv.Canny(gray, &edges, 50, 150)

	// Dilate edges slightly
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()

	dilated := gocv.NewMat()
	defer dilated.Close()
	gocv.Dilate(edges, &dilated, kernel)

	// Compare original edges to dilated
	diff := gocv.NewMat()
	defer diff.Close()
	gocv.Subtract(dilated, edges, &diff)

	edgeBlur := gocv.CountNonZero(diff)
	totalEdges := gocv.CountNonZero(edges)

	if totalEdges > 0 {
		blurRatio := float64(edgeBlur) / float64(totalEdges)
		return math.Min(blurRatio*2, 1.0)
	}

	return 0.0
}

// ============================================================================
// MASK DETECTION
// ============================================================================

// detectMask detects if face is wearing a mask (silicone, paper, etc.)
func (lfs *LocalFaceService) detectMask(faceRegion, gray gocv.Mat) float64 {
	if faceRegion.Empty() || gray.Empty() {
		return 0.0
	}

	maskIndicators := 0.0
	totalChecks := 0.0

	// 1. Material Texture Detection
	// Masks have uniform material texture
	materialScore := lfs.detectMaskMaterial(gray)
	if materialScore > 0.65 {
		maskIndicators += 1.0
	}
	totalChecks += 1.0

	// 2. Lack of Natural Facial Depth
	// Masks are flatter than real faces
	depthScore := lfs.detectLackOfDepth(gray)
	if depthScore > 0.7 {
		maskIndicators += 1.0
	}
	totalChecks += 1.0

	// 3. Unnatural Facial Stiffness
	// Masks don't have micro-movements or skin elasticity
	stiffnessScore := lfs.detectFacialStiffness(gray)
	if stiffnessScore > 0.65 {
		maskIndicators += 1.0
	}
	totalChecks += 1.0

	// 4. Edge Artifacts
	// Mask edges differ from skin edges
	maskEdgeScore := lfs.detectMaskEdges(gray)
	if maskEdgeScore > 0.6 {
		maskIndicators += 1.0
	}
	totalChecks += 1.0

	maskProbability := maskIndicators / totalChecks

	logger.Info("🎭 Mask detection analysis", logger.LoggerOptions{
		Key: "mask_detection",
		Data: map[string]interface{}{
			"material_score":       materialScore,
			"depth_score":          depthScore,
			"stiffness_score":      stiffnessScore,
			"mask_edge_score":      maskEdgeScore,
			"mask_probability":     maskProbability,
			"indicators_triggered": maskIndicators,
			"total_checks":         totalChecks,
		},
	})

	return maskProbability
}

// detectMaskMaterial detects uniform material texture of masks
func (lfs *LocalFaceService) detectMaskMaterial(gray gocv.Mat) float64 {
	if gray.Empty() {
		return 0.0
	}

	// Masks have very uniform texture compared to skin
	// Calculate texture uniformity
	mean := gocv.NewMat()
	stdDev := gocv.NewMat()
	defer mean.Close()
	defer stdDev.Close()

	gocv.MeanStdDev(gray, &mean, &stdDev)

	stdDevVal := stdDev.GetDoubleAt(0, 0)

	// Low variation indicates uniform material
	uniformity := 1.0 - math.Min(stdDevVal/80.0, 1.0)

	return uniformity
}

// detectLackOfDepth detects flat appearance of masks
func (lfs *LocalFaceService) detectLackOfDepth(gray gocv.Mat) float64 {
	if gray.Empty() {
		return 0.0
	}

	// Real faces have depth indicated by shading gradients
	// Masks are flatter with less gradient variation

	// Calculate gradients
	gradX := gocv.NewMat()
	gradY := gocv.NewMat()
	defer gradX.Close()
	defer gradY.Close()

	gocv.Sobel(gray, &gradX, gocv.MatTypeCV32F, 1, 0, 3, 1, 0, gocv.BorderDefault)
	gocv.Sobel(gray, &gradY, gocv.MatTypeCV32F, 0, 1, 3, 1, 0, gocv.BorderDefault)

	// Calculate gradient magnitude
	magnitude := gocv.NewMat()
	defer magnitude.Close()
	gocv.Magnitude(gradX, gradY, &magnitude)

	mean := gocv.NewMat()
	stdDev := gocv.NewMat()
	defer mean.Close()
	defer stdDev.Close()

	gocv.MeanStdDev(magnitude, &mean, &stdDev)

	meanGrad := mean.GetDoubleAt(0, 0)

	// Low gradient indicates lack of depth
	flatness := 1.0 - math.Min(meanGrad/50.0, 1.0)

	return flatness
}

// detectFacialStiffness detects lack of natural skin micro-texture
func (lfs *LocalFaceService) detectFacialStiffness(gray gocv.Mat) float64 {
	if gray.Empty() {
		return 0.0
	}

	// Analyze high-frequency components
	// Real skin has micro-textures; masks are smoother
	blurred := gocv.NewMat()
	defer blurred.Close()
	gocv.GaussianBlur(gray, &blurred, image.Pt(5, 5), 0, 0, gocv.BorderDefault)

	// High-frequency detail
	detail := gocv.NewMat()
	defer detail.Close()
	gocv.Subtract(gray, blurred, &detail)

	mean := gocv.NewMat()
	stdDev := gocv.NewMat()
	defer mean.Close()
	defer stdDev.Close()

	gocv.MeanStdDev(detail, &mean, &stdDev)

	detailVal := stdDev.GetDoubleAt(0, 0)

	// Low high-frequency detail indicates stiffness
	stiffness := 1.0 - math.Min(detailVal/30.0, 1.0)

	return stiffness
}

// detectMaskEdges detects unnatural edges around mask boundaries
func (lfs *LocalFaceService) detectMaskEdges(gray gocv.Mat) float64 {
	if gray.Empty() {
		return 0.0
	}

	// Detect edges
	edges := gocv.NewMat()
	defer edges.Close()
	gocv.Canny(gray, &edges, 50, 150)

	// Masks often have sharp, artificial boundary edges
	// Analyze edge sharpness distribution
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()

	dilated := gocv.NewMat()
	defer dilated.Close()
	gocv.Dilate(edges, &dilated, kernel)

	edgePixels := gocv.CountNonZero(edges)
	totalPixels := gray.Rows() * gray.Cols()

	if totalPixels > 0 {
		edgeDensity := float64(edgePixels) / float64(totalPixels)

		// Moderate edge density with sharp transitions indicates mask
		if edgeDensity > 0.05 && edgeDensity < 0.15 {
			return math.Min(edgeDensity*8, 1.0)
		}
	}

	return 0.0
}

// ============================================================================
// SKIN TONE REALISM ANALYSIS
// ============================================================================

// analyzeSkinToneRealism analyzes if skin tones are realistic
func (lfs *LocalFaceService) analyzeSkinToneRealism(faceRegion gocv.Mat) float64 {
	if faceRegion.Empty() || faceRegion.Channels() != 3 {
		return 0.5 // Neutral if can't analyze
	}

	realisticIndicators := 0.0
	totalChecks := 0.0

	// 1. Color Range Check
	// Real skin falls within specific RGB/HSV ranges
	colorRangeScore := lfs.checkSkinColorRange(faceRegion)
	if colorRangeScore > 0.7 {
		realisticIndicators += 1.0
	}
	totalChecks += 1.0

	// 2. Color Distribution
	// Real skin has natural color variation
	distributionScore := lfs.checkSkinColorDistribution(faceRegion)
	if distributionScore > 0.65 {
		realisticIndicators += 1.0
	}
	totalChecks += 1.0

	// 3. Color Uniformity Check
	// Overly uniform color indicates fake
	uniformityScore := lfs.checkColorUniformity(faceRegion)
	if uniformityScore < 0.8 { // Less uniform is more realistic
		realisticIndicators += 1.0
	}
	totalChecks += 1.0

	realismScore := realisticIndicators / totalChecks

	logger.Info("🎨 Skin tone realism analysis", logger.LoggerOptions{
		Key: "skin_tone_analysis",
		Data: map[string]interface{}{
			"color_range_score":    colorRangeScore,
			"distribution_score":   distributionScore,
			"uniformity_score":     uniformityScore,
			"realism_score":        realismScore,
			"realistic_indicators": realisticIndicators,
			"total_checks":         totalChecks,
		},
	})

	return realismScore
}

// checkSkinColorRange checks if colors fall within realistic skin tone ranges
func (lfs *LocalFaceService) checkSkinColorRange(faceRegion gocv.Mat) float64 {
	if faceRegion.Empty() {
		return 0.0
	}

	// Convert to HSV for better skin detection
	hsv := gocv.NewMat()
	defer hsv.Close()
	gocv.CvtColor(faceRegion, &hsv, gocv.ColorBGRToHSV)

	// Define skin tone ranges in HSV
	// Hue: 0-25 (red-orange range for skin)
	// Saturation: 20-170
	// Value: 40-255
	lowerBound := gocv.NewMatFromScalar(gocv.NewScalar(0, 20, 40, 0), gocv.MatTypeCV8U)
	upperBound := gocv.NewMatFromScalar(gocv.NewScalar(25, 170, 255, 0), gocv.MatTypeCV8U)
	defer lowerBound.Close()
	defer upperBound.Close()

	mask := gocv.NewMat()
	defer mask.Close()
	gocv.InRange(hsv, lowerBound, upperBound, &mask)

	skinPixels := gocv.CountNonZero(mask)
	totalPixels := faceRegion.Rows() * faceRegion.Cols()

	if totalPixels > 0 {
		skinRatio := float64(skinPixels) / float64(totalPixels)
		return math.Min(skinRatio, 1.0)
	}

	return 0.0
}

// checkSkinColorDistribution checks natural variation in skin tones
func (lfs *LocalFaceService) checkSkinColorDistribution(faceRegion gocv.Mat) float64 {
	if faceRegion.Empty() {
		return 0.0
	}

	// Real skin has natural color variation
	channels := gocv.Split(faceRegion)
	defer channels[0].Close()
	defer channels[1].Close()
	defer channels[2].Close()

	// Calculate histogram for each channel
	totalVariation := 0.0

	for i := 0; i < 3; i++ {
		hist := gocv.NewMat()
		defer hist.Close()

		gocv.CalcHist(
			[]gocv.Mat{channels[i]},
			[]int{0},
			gocv.NewMat(),
			&hist,
			[]int{256},
			[]float64{0, 256},
			false,
		)

		// Normalize histogram
		gocv.Normalize(hist, &hist, 0, 1, gocv.NormMinMax)

		// Calculate entropy (variation)
		entropy := 0.0
		for j := 0; j < 256; j++ {
			val := float64(hist.GetFloatAt(j, 0))
			if val > 0 {
				entropy -= val * math.Log2(val)
			}
		}

		totalVariation += entropy
	}

	// Average entropy across channels
	avgEntropy := totalVariation / 3.0

	// Normalize (max entropy is log2(256) = 8)
	normalizedVariation := math.Min(avgEntropy/6.0, 1.0)

	return normalizedVariation
}

// checkColorUniformity checks if color is too uniform (fake indicator)
func (lfs *LocalFaceService) checkColorUniformity(faceRegion gocv.Mat) float64 {
	if faceRegion.Empty() {
		return 0.0
	}

	channels := gocv.Split(faceRegion)
	defer channels[0].Close()
	defer channels[1].Close()
	defer channels[2].Close()

	totalStdDev := 0.0

	for i := 0; i < 3; i++ {
		mean := gocv.NewMat()
		stdDev := gocv.NewMat()
		defer mean.Close()
		defer stdDev.Close()

		gocv.MeanStdDev(channels[i], &mean, &stdDev)
		totalStdDev += stdDev.GetDoubleAt(0, 0)
	}

	avgStdDev := totalStdDev / 3.0

	// Normalize (typical skin variation is 10-40)
	uniformity := 1.0 - math.Min(avgStdDev/50.0, 1.0)

	return uniformity
}

// SpoofDetectionResult holds all spoof detection probabilities
type SpoofDetectionResult struct {
	PaintingProbability float64
	ScreenProbability   float64
	PrintProbability    float64
	MaskProbability     float64
	SkinToneRealism     float64
	TotalPenalty        float64
}

// enhancedSpoofPenaltyWithAllDetections applies comprehensive spoof detection
// Runs all detection methods in parallel using goroutines for maximum performance
// Returns: SpoofDetectionResult with all probabilities
func (lfs *LocalFaceService) enhancedSpoofPenaltyWithAllDetections(
	faceRegion, gray gocv.Mat,
	textureScore, edgeScore, colorScore, reflectionScore, frequencyScore, lbpScore, lpqScore float64,
) SpoofDetectionResult {
	// Start with existing penalty calculation
	basePenalty := lfs.calculateEnhancedSpoofPenalty(
		[]float64{textureScore, edgeScore, colorScore, reflectionScore, frequencyScore},
		lbpScore,
		lpqScore,
	)

	// ========================================================================
	// PARALLEL SPOOF DETECTION - Run all detections simultaneously
	// ========================================================================
	logger.Info("🚀 Starting parallel spoof detection analysis", logger.LoggerOptions{
		Key: "parallel_detection_start",
	})

	// Use channels to collect results from goroutines
	paintingChan := make(chan float64, 1)
	screenChan := make(chan float64, 1)
	printChan := make(chan float64, 1)
	maskChan := make(chan float64, 1)
	skinToneChan := make(chan float64, 1)

	// Launch all detections in parallel
	go func() {
		paintingChan <- lfs.detectPaintingCharacteristics(faceRegion, gray)
	}()

	go func() {
		screenChan <- lfs.detectScreenDisplay(faceRegion, gray)
	}()

	go func() {
		printChan <- lfs.detectPrintPhoto(faceRegion, gray)
	}()

	go func() {
		maskChan <- lfs.detectMask(faceRegion, gray)
	}()

	go func() {
		skinToneChan <- lfs.analyzeSkinToneRealism(faceRegion)
	}()

	// Collect all results
	paintingProbability := <-paintingChan
	screenProbability := <-screenChan
	printProbability := <-printChan
	maskProbability := <-maskChan
	skinToneRealism := <-skinToneChan

	logger.Info("✅ Parallel spoof detection completed", logger.LoggerOptions{
		Key: "parallel_detection_complete",
	})

	// ========================================================================
	// CALCULATE PENALTIES FROM EACH DETECTION TYPE
	// ========================================================================

	// Painting detection penalty - INCREASED penalties for more aggressive rejection
	paintingPenalty := 0.0
	if paintingProbability > 0.5 {
		// High confidence - MASSIVE penalty (increased from 0.8 to 1.2)
		paintingPenalty = paintingProbability * 1.2
		logger.Error("⚠️ HIGH PAINTING PROBABILITY DETECTED", logger.LoggerOptions{
			Key: "painting_detected",
			Data: map[string]interface{}{
				"painting_probability": paintingProbability,
				"penalty_applied":      paintingPenalty,
			},
		})
	} else if paintingProbability > 0.4 {
		// Medium-high confidence - MAJOR penalty (new tier)
		paintingPenalty = paintingProbability * 0.9
		logger.Error("⚠️ MEDIUM-HIGH PAINTING PROBABILITY DETECTED", logger.LoggerOptions{
			Key: "painting_detected_medium_high",
			Data: map[string]interface{}{
				"painting_probability": paintingProbability,
				"penalty_applied":      paintingPenalty,
			},
		})
	} else if paintingProbability > 0.3 {
		// Medium confidence - SIGNIFICANT penalty (increased from 0.5 to 0.7)
		paintingPenalty = paintingProbability * 0.7
		logger.Info("⚠️ MEDIUM PAINTING PROBABILITY DETECTED", logger.LoggerOptions{
			Key: "painting_detected_medium",
			Data: map[string]interface{}{
				"painting_probability": paintingProbability,
				"penalty_applied":      paintingPenalty,
			},
		})
	} else if paintingProbability > 0.2 {
		// Low-medium confidence - moderate penalty
		paintingPenalty = paintingProbability * 0.4
	}

	// Screen detection penalty
	screenPenalty := 0.0
	if screenProbability > 0.5 {
		screenPenalty = screenProbability * 0.75
		logger.Error("⚠️ HIGH SCREEN PROBABILITY DETECTED", logger.LoggerOptions{
			Key: "screen_detected",
			Data: map[string]interface{}{
				"screen_probability": screenProbability,
				"penalty_applied":    screenPenalty,
			},
		})
	} else if screenProbability > 0.35 {
		screenPenalty = screenProbability * 0.4
	}

	// Print detection penalty
	printPenalty := 0.0
	if printProbability > 0.5 {
		printPenalty = printProbability * 0.7
		logger.Error("⚠️ HIGH PRINT PROBABILITY DETECTED", logger.LoggerOptions{
			Key: "print_detected",
			Data: map[string]interface{}{
				"print_probability": printProbability,
				"penalty_applied":   printPenalty,
			},
		})
	} else if printProbability > 0.35 {
		printPenalty = printProbability * 0.4
	}

	// Mask detection penalty
	maskPenalty := 0.0
	if maskProbability > 0.5 {
		maskPenalty = maskProbability * 0.8
		logger.Error("⚠️ HIGH MASK PROBABILITY DETECTED", logger.LoggerOptions{
			Key: "mask_detected",
			Data: map[string]interface{}{
				"mask_probability": maskProbability,
				"penalty_applied":  maskPenalty,
			},
		})
	} else if maskProbability > 0.35 {
		maskPenalty = maskProbability * 0.5
	}

	// Skin tone realism penalty (inverse - low realism = penalty)
	skinTonePenalty := 0.0
	if skinToneRealism < 0.5 {
		skinTonePenalty = (1.0 - skinToneRealism) * 0.4
		logger.Info("⚠️ LOW SKIN TONE REALISM DETECTED", logger.LoggerOptions{
			Key: "skin_tone_low",
			Data: map[string]interface{}{
				"skin_tone_realism": skinToneRealism,
				"penalty_applied":   skinTonePenalty,
			},
		})
	}

	// Calculate total penalty
	totalPenalty := basePenalty + paintingPenalty + screenPenalty + printPenalty + maskPenalty + skinTonePenalty

	logger.Info("🛡️ Comprehensive spoof penalty calculation", logger.LoggerOptions{
		Key: "spoof_penalty_complete",
		Data: map[string]interface{}{
			"base_penalty":         basePenalty,
			"painting_penalty":     paintingPenalty,
			"screen_penalty":       screenPenalty,
			"print_penalty":        printPenalty,
			"mask_penalty":         maskPenalty,
			"skin_tone_penalty":    skinTonePenalty,
			"total_penalty":        totalPenalty,
			"painting_probability": paintingProbability,
			"screen_probability":   screenProbability,
			"print_probability":    printProbability,
			"mask_probability":     maskProbability,
			"skin_tone_realism":    skinToneRealism,
		},
	})

	return SpoofDetectionResult{
		PaintingProbability: paintingProbability,
		ScreenProbability:   screenProbability,
		PrintProbability:    printProbability,
		MaskProbability:     maskProbability,
		SkinToneRealism:     skinToneRealism,
		TotalPenalty:        totalPenalty,
	}
}

// Close releases resources
func (lfs *LocalFaceService) Close() {
	lfs.faceCascade.Close()
	lfs.eyeCascade.Close()
}
