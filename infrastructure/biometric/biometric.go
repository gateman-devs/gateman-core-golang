package biometric

import (
	"fmt"
	"os"
	"sync"

	"gateman.io/infrastructure/logger"
	"gocv.io/x/gocv"
)

const (
	DEFAULT_YUNET_MODEL_PATH        = "./models/yunet.onnx"
	DEFAULT_ARCFACE_MODEL_PATH      = "./models/arcface.onnx"
	DEFAULT_ANTISPOOFING_MODEL_PATH = "./models/silent_face_anti_spoofing.onnx"

	FACE_COMPARISON_THRESHOLD = 0.7
	LIVENESS_THRESHOLD        = 0.8
)

type BiometricSystem struct {
	faceDetector    *FaceDetector
	faceComparator  *FaceComparator
	livenessChecker *LivenessChecker
	initialized     bool
	mu              sync.RWMutex
}

type FaceDetectionResult struct {
	Success     bool         `json:"success"`
	FaceFound   bool         `json:"face_found"`
	FaceRegion  FaceBox      `json:"face_region,omitempty"`
	Confidence  float64      `json:"confidence"`
	ProcessTime int64        `json:"process_time_ms"`
	Metadata    FaceMetadata `json:"metadata"`
	Error       string       `json:"error,omitempty"`
}

type FaceComparisonResult struct {
	Success     bool               `json:"success"`
	Match       bool               `json:"match"`
	Similarity  float64            `json:"similarity"`
	Threshold   float64            `json:"threshold"`
	ProcessTime int64              `json:"process_time_ms"`
	Metadata    ComparisonMetadata `json:"metadata"`
	Error       string             `json:"error,omitempty"`
}

type LivenessCheckResult struct {
	Success       bool             `json:"success"`
	IsLive        bool             `json:"is_live"`
	LivenessScore float64          `json:"liveness_score"`
	Threshold     float64          `json:"threshold"`
	ProcessTime   int64            `json:"process_time_ms"`
	Metadata      LivenessMetadata `json:"metadata"`
	Error         string           `json:"error,omitempty"`
}

type BiometricVerificationResult struct {
	Success          bool                 `json:"success"`
	LivenessCheck    LivenessCheckResult  `json:"liveness_check"`
	FaceComparison   FaceComparisonResult `json:"face_comparison"`
	OverallMatch     bool                 `json:"overall_match"`
	TotalProcessTime int64                `json:"total_process_time_ms"`
	Recommendations  []string             `json:"recommendations,omitempty"`
	Error            string               `json:"error,omitempty"`
}

type FaceBox struct {
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Score  float64 `json:"score"`
}

type FaceMetadata struct {
	ImageSize   string   `json:"image_size"`
	FaceSize    string   `json:"face_size"`
	Quality     string   `json:"quality"`
	Lighting    string   `json:"lighting"`
	Orientation string   `json:"orientation"`
	Warnings    []string `json:"warnings,omitempty"`
}

type ComparisonMetadata struct {
	Face1Quality    string   `json:"face1_quality"`
	Face2Quality    string   `json:"face2_quality"`
	MatchConfidence string   `json:"match_confidence"`
	Warnings        []string `json:"warnings,omitempty"`
}

type LivenessMetadata struct {
	ImageQuality      string   `json:"image_quality"`
	AntiSpoofingTests []string `json:"anti_spoofing_tests"`
	Confidence        string   `json:"confidence"`
	Warnings          []string `json:"warnings,omitempty"`
}

var GlobalBiometricSystem *BiometricSystem

func InitializeBiometricSystem() error {
	GlobalBiometricSystem = NewBiometricSystem()

	yunetPath := os.Getenv("YUNET_MODEL_PATH")
	if yunetPath == "" {
		yunetPath = DEFAULT_YUNET_MODEL_PATH
	}

	arcfacePath := os.Getenv("ARCFACE_MODEL_PATH")
	if arcfacePath == "" {
		arcfacePath = DEFAULT_ARCFACE_MODEL_PATH
	}

	antispoofingPath := os.Getenv("ANTISPOOFING_MODEL_PATH")
	if antispoofingPath == "" {
		antispoofingPath = DEFAULT_ANTISPOOFING_MODEL_PATH
	}

	return GlobalBiometricSystem.Initialize(yunetPath, arcfacePath, antispoofingPath)
}

func NewBiometricSystem() *BiometricSystem {
	return &BiometricSystem{
		faceDetector:    NewFaceDetector(),
		faceComparator:  NewFaceComparator(),
		livenessChecker: NewLivenessChecker(),
	}
}

func (bs *BiometricSystem) Initialize(yunetPath, arcfacePath, antispoofingPath string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.initialized {
		return nil
	}

	var initErrors []string

	if err := bs.faceDetector.Initialize(yunetPath); err != nil {
		initErrors = append(initErrors, fmt.Sprintf("Face detector initialization failed: %v", err))
		logger.Error(fmt.Sprintf("Face detector initialization failed: %v", err))
	}

	if err := bs.faceComparator.Initialize(arcfacePath); err != nil {
		initErrors = append(initErrors, fmt.Sprintf("Face comparator initialization failed: %v", err))
		logger.Error(fmt.Sprintf("Face comparator initialization failed: %v", err))
	}

	if err := bs.livenessChecker.Initialize(antispoofingPath); err != nil {
		initErrors = append(initErrors, fmt.Sprintf("Liveness checker initialization failed: %v", err))
		logger.Error(fmt.Sprintf("Liveness checker initialization failed: %v", err))
	}

	if len(initErrors) > 0 {
		return fmt.Errorf("biometric system initialization failed: %v", initErrors)
	}

	bs.initialized = true
	logger.Info("Biometric system initialized successfully")
	return nil
}

func (bs *BiometricSystem) ExtractFace(input string) FaceDetectionResult {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	if !bs.initialized {
		return FaceDetectionResult{
			Success: false,
			Error:   "biometric system not initialized",
		}
	}

	return bs.faceDetector.DetectFace(input)
}

func (bs *BiometricSystem) CompareFaces(input1, input2 string) FaceComparisonResult {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	if !bs.initialized {
		return FaceComparisonResult{
			Success: false,
			Error:   "biometric system not initialized",
		}
	}

	return bs.faceComparator.Compare(input1, input2, FACE_COMPARISON_THRESHOLD)
}

func (bs *BiometricSystem) CheckLiveness(input string) LivenessCheckResult {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	if !bs.initialized {
		return LivenessCheckResult{
			Success: false,
			Error:   "biometric system not initialized",
		}
	}

	return bs.livenessChecker.CheckLiveness(input)
}

func (bs *BiometricSystem) VerifyBiometric(referenceImage, testImage string) BiometricVerificationResult {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	if !bs.initialized {
		return BiometricVerificationResult{
			Success: false,
			Error:   "biometric system not initialized",
		}
	}

	result := BiometricVerificationResult{
		Success:         true,
		Recommendations: []string{},
	}

	livenessStart := getCurrentTimeMs()

	testLiveness := bs.livenessChecker.CheckLiveness(testImage)
	result.LivenessCheck = testLiveness

	if !testLiveness.Success {
		result.Success = false
		result.Error = fmt.Sprintf("Liveness check failed: %s", testLiveness.Error)
		result.TotalProcessTime = getCurrentTimeMs() - livenessStart
		return result
	}

	if !testLiveness.IsLive {
		result.OverallMatch = false
		result.Recommendations = append(result.Recommendations, "Test image failed liveness check - possible spoof attempt detected")
		result.TotalProcessTime = getCurrentTimeMs() - livenessStart
		return result
	}

	comparison := bs.faceComparator.Compare(referenceImage, testImage, FACE_COMPARISON_THRESHOLD)
	result.FaceComparison = comparison

	if !comparison.Success {
		result.Success = false
		result.Error = fmt.Sprintf("Face comparison failed: %s", comparison.Error)
		result.TotalProcessTime = getCurrentTimeMs() - livenessStart
		return result
	}

	result.OverallMatch = testLiveness.IsLive && comparison.Match
	result.TotalProcessTime = getCurrentTimeMs() - livenessStart

	if result.OverallMatch {
		result.Recommendations = append(result.Recommendations, "Biometric verification successful - high confidence match")
	} else {
		if comparison.Similarity > 0.5 {
			result.Recommendations = append(result.Recommendations, "Partial face similarity detected but below threshold")
		} else {
			result.Recommendations = append(result.Recommendations, "Low face similarity - different person likely")
		}
	}

	return result
}

func (bs *BiometricSystem) Close() {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if !bs.initialized {
		return
	}

	if bs.faceDetector != nil {
		bs.faceDetector.Close()
	}
	if bs.faceComparator != nil {
		bs.faceComparator.Close()
	}
	if bs.livenessChecker != nil {
		bs.livenessChecker.Close()
	}

	bs.initialized = false
	logger.Info("Biometric system closed")
}

func getCurrentTimeMs() int64 {
	return int64(gocv.GetTickCount() * 1000 / gocv.GetTickFrequency())
}
