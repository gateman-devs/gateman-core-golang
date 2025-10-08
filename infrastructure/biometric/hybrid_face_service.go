package biometric

import (
	"fmt"
	"image"
	"sync"
	"time"

	"gateman.io/infrastructure/logger"
	"gocv.io/x/gocv"
)

// HybridFaceService combines Haar cascade and MobileNet for optimal face detection
type HybridFaceService struct {
	haarService      *LocalFaceService
	mobileNetService *MobileNetFaceService
	useMobileNet     bool
	fallbackMode     bool
	preferMobileNet  bool
	mutex            sync.RWMutex
	processingStats  ProcessingStats
}

// HybridConfig holds configuration for hybrid service
type HybridConfig struct {
	UseMobileNet    bool
	FallbackMode    bool
	PreferMobileNet bool
	MobileNetConfig MobileNetConfig
}

// HybridDetectionResult holds detection results from hybrid service
type HybridDetectionResult struct {
	Faces          []image.Rectangle
	Confidences    []float32
	ProcessingTime time.Duration
	Method         string
	FallbackUsed   bool
}

// NewHybridFaceService creates a new hybrid face detection service
func NewHybridFaceService(config HybridConfig) *HybridFaceService {
	service := &HybridFaceService{
		haarService:     NewLocalFaceService(),
		useMobileNet:    config.UseMobileNet,
		fallbackMode:    config.FallbackMode,
		preferMobileNet: config.PreferMobileNet,
		processingStats: ProcessingStats{},
	}

	// Initialize MobileNet if enabled
	if config.UseMobileNet {
		service.mobileNetService = NewMobileNetFaceService(config.MobileNetConfig)
	}

	logger.Info("Hybrid face service initialized", logger.LoggerOptions{
		Key: "hybrid_config",
		Data: map[string]interface{}{
			"use_mobilenet":    config.UseMobileNet,
			"fallback_mode":    config.FallbackMode,
			"prefer_mobilenet": config.PreferMobileNet,
		},
	})

	return service
}

// DetectFaces performs face detection using the hybrid approach
func (hfs *HybridFaceService) DetectFaces(img gocv.Mat) (*HybridDetectionResult, error) {
	startTime := time.Now()

	// Try MobileNet first if enabled and preferred
	if hfs.useMobileNet && hfs.preferMobileNet && hfs.mobileNetService != nil {
		result, err := hfs.tryMobileNetDetection(img)
		if err == nil && len(result.Faces) > 0 {
			result.ProcessingTime = time.Since(startTime)
			hfs.updateStats(result.ProcessingTime, true)
			return result, nil
		}

		// If MobileNet fails and fallback is enabled, try Haar
		if hfs.fallbackMode {
			logger.Info("MobileNet failed, falling back to Haar cascade")
			return hfs.tryHaarDetection(img, startTime, true)
		}
	}

	// Try Haar cascade first if MobileNet is not preferred or not available
	if !hfs.preferMobileNet || !hfs.useMobileNet {
		result, err := hfs.tryHaarDetection(img, startTime, false)
		if err == nil && len(result.Faces) > 0 {
			return result, nil
		}

		// If Haar fails and MobileNet is available, try MobileNet
		if hfs.useMobileNet && hfs.mobileNetService != nil {
			logger.Info("Haar cascade failed, trying MobileNet")
			mobileResult, err := hfs.tryMobileNetDetection(img)
			if err == nil && len(mobileResult.Faces) > 0 {
				mobileResult.ProcessingTime = time.Since(startTime)
				mobileResult.FallbackUsed = true
				hfs.updateStats(mobileResult.ProcessingTime, true)
				return mobileResult, nil
			}
		}
	}

	// If both methods fail, return error
	return nil, fmt.Errorf("no faces detected using any method")
}

// tryMobileNetDetection attempts face detection using MobileNet
func (hfs *HybridFaceService) tryMobileNetDetection(img gocv.Mat) (*HybridDetectionResult, error) {
	if hfs.mobileNetService == nil {
		return nil, fmt.Errorf("MobileNet service not available")
	}

	result, err := hfs.mobileNetService.DetectFaces(img)
	if err != nil {
		return nil, err
	}

	return &HybridDetectionResult{
		Faces:          result.Faces,
		Confidences:    result.Confidences,
		ProcessingTime: result.ProcessingTime,
		Method:         "MobileNet-SSD",
		FallbackUsed:   false,
	}, nil
}

// tryHaarDetection attempts face detection using Haar cascade
func (hfs *HybridFaceService) tryHaarDetection(img gocv.Mat, startTime time.Time, isFallback bool) (*HybridDetectionResult, error) {
	faces := hfs.haarService.enhancedFaceDetection(img)
	processingTime := time.Since(startTime)

	// Create confidences array (Haar doesn't provide confidence scores)
	confidences := make([]float32, len(faces))
	for i := range confidences {
		confidences[i] = 0.8 // Default confidence for Haar cascade
	}

	result := &HybridDetectionResult{
		Faces:          faces,
		Confidences:    confidences,
		ProcessingTime: processingTime,
		Method:         "Haar-Cascade",
		FallbackUsed:   isFallback,
	}

	hfs.updateStats(processingTime, len(faces) > 0)

	logger.Info("Haar cascade face detection completed", logger.LoggerOptions{
		Key: "haar_detection",
		Data: map[string]interface{}{
			"faces_detected":     len(faces),
			"processing_time_ms": processingTime.Milliseconds(),
			"is_fallback":        isFallback,
		},
	})

	return result, nil
}

// updateStats updates processing statistics
func (hfs *HybridFaceService) updateStats(processingTime time.Duration, success bool) {
	hfs.mutex.Lock()
	defer hfs.mutex.Unlock()

	hfs.processingStats.TotalRequests++
	if success {
		hfs.processingStats.SuccessfulRequests++
	}
	hfs.processingStats.TotalTime += processingTime.Milliseconds()
	hfs.processingStats.AverageTime = float64(hfs.processingStats.TotalTime) / float64(hfs.processingStats.TotalRequests)
}

// GetStats returns processing statistics
func (hfs *HybridFaceService) GetStats() ProcessingStats {
	hfs.mutex.RLock()
	defer hfs.mutex.RUnlock()
	return hfs.processingStats
}

// IsHealthy checks if the service is healthy
func (hfs *HybridFaceService) IsHealthy() bool {
	haarHealthy := hfs.haarService != nil && hfs.haarService.modelsLoaded
	mobileNetHealthy := true

	if hfs.useMobileNet && hfs.mobileNetService != nil {
		mobileNetHealthy = hfs.mobileNetService.IsHealthy()
	}

	return haarHealthy && mobileNetHealthy
}

// Close releases resources
func (hfs *HybridFaceService) Close() {
	if hfs.haarService != nil {
		hfs.haarService.Close()
	}
	if hfs.mobileNetService != nil {
		hfs.mobileNetService.Close()
	}
}

// GetDefaultHybridConfig returns default hybrid configuration
func GetDefaultHybridConfig() HybridConfig {
	return HybridConfig{
		UseMobileNet:    true,
		FallbackMode:    true,
		PreferMobileNet: true,
		MobileNetConfig: GetDefaultMobileNetConfig(),
	}
}

// SetMobileNetPreference changes the preference for MobileNet vs Haar
func (hfs *HybridFaceService) SetMobileNetPreference(prefer bool) {
	hfs.mutex.Lock()
	defer hfs.mutex.Unlock()
	hfs.preferMobileNet = prefer
}

// SetFallbackMode enables or disables fallback mode
func (hfs *HybridFaceService) SetFallbackMode(enabled bool) {
	hfs.mutex.Lock()
	defer hfs.mutex.Unlock()
	hfs.fallbackMode = enabled
}
