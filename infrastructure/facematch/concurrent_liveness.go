package facematch

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LivenessDetector interface for liveness detection operations
type LivenessDetector interface {
	DetectAdvancedAntiSpoof(input string, requestID string, verboseMode bool) AdvancedAntiSpoofResult
}

// ConcurrentLivenessProcessor handles parallel liveness detection for multiple images
type ConcurrentLivenessProcessor struct {
	faceMatcher LivenessDetector
	timeout     time.Duration
	mu          sync.RWMutex
}

// LivenessDetectionResult represents the result of a single liveness detection operation
type LivenessDetectionResult struct {
	ImageID     string                  `json:"image_id"`
	Result      AdvancedAntiSpoofResult `json:"result"`
	Error       error                   `json:"error,omitempty"`
	ProcessTime time.Duration           `json:"process_time"`
}

// ConcurrentLivenessResult represents the aggregated result of concurrent liveness detection
type ConcurrentLivenessResult struct {
	ReferenceResult LivenessDetectionResult `json:"reference_result"`
	TestResult      LivenessDetectionResult `json:"test_result"`
	TotalTime       time.Duration           `json:"total_time"`
	Success         bool                    `json:"success"`
	Errors          []string                `json:"errors,omitempty"`
}

// NewConcurrentLivenessProcessor creates a new concurrent liveness processor
func NewConcurrentLivenessProcessor(faceMatcher LivenessDetector, timeout time.Duration) *ConcurrentLivenessProcessor {
	if timeout <= 0 {
		timeout = 10 * time.Second // Default timeout from requirements
	}

	return &ConcurrentLivenessProcessor{
		faceMatcher: faceMatcher,
		timeout:     timeout,
	}
}

// DetectBothImages performs liveness detection on both images simultaneously using goroutines
func (clp *ConcurrentLivenessProcessor) DetectBothImages(referenceImage, testImage, requestID string) ConcurrentLivenessResult {
	startTime := time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), clp.timeout)
	defer cancel()

	// Create channels for results
	referenceChan := make(chan LivenessDetectionResult, 1)
	testChan := make(chan LivenessDetectionResult, 1)

	// Start goroutines for parallel processing
	var wg sync.WaitGroup
	wg.Add(2)

	// Process reference image
	go func() {
		defer wg.Done()
		result := clp.detectSingleImage(ctx, referenceImage, fmt.Sprintf("%s_ref", requestID))
		select {
		case referenceChan <- result:
		case <-ctx.Done():
			// Context cancelled, send timeout result
			referenceChan <- LivenessDetectionResult{
				ImageID:     fmt.Sprintf("%s_ref", requestID),
				Error:       fmt.Errorf("reference image liveness detection timeout"),
				ProcessTime: time.Since(startTime),
			}
		}
	}()

	// Process test image
	go func() {
		defer wg.Done()
		result := clp.detectSingleImage(ctx, testImage, fmt.Sprintf("%s_test", requestID))
		select {
		case testChan <- result:
		case <-ctx.Done():
			// Context cancelled, send timeout result
			testChan <- LivenessDetectionResult{
				ImageID:     fmt.Sprintf("%s_test", requestID),
				Error:       fmt.Errorf("test image liveness detection timeout"),
				ProcessTime: time.Since(startTime),
			}
		}
	}()

	// Wait for both goroutines to complete or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	var referenceResult, testResult LivenessDetectionResult

	select {
	case <-done:
		// Both operations completed
		referenceResult = <-referenceChan
		testResult = <-testChan
	case <-ctx.Done():
		// Timeout occurred
		select {
		case referenceResult = <-referenceChan:
		default:
			referenceResult = LivenessDetectionResult{
				ImageID:     fmt.Sprintf("%s_ref", requestID),
				Error:       fmt.Errorf("reference image liveness detection timeout"),
				ProcessTime: clp.timeout,
			}
		}

		select {
		case testResult = <-testChan:
		default:
			testResult = LivenessDetectionResult{
				ImageID:     fmt.Sprintf("%s_test", requestID),
				Error:       fmt.Errorf("test image liveness detection timeout"),
				ProcessTime: clp.timeout,
			}
		}
	}

	// Aggregate results
	totalTime := time.Since(startTime)
	var errors []string
	success := true

	if referenceResult.Error != nil {
		errors = append(errors, fmt.Sprintf("Reference image: %v", referenceResult.Error))
		success = false
	}

	if testResult.Error != nil {
		errors = append(errors, fmt.Sprintf("Test image: %v", testResult.Error))
		success = false
	}

	// Check if both images passed liveness detection
	if success {
		if !referenceResult.Result.IsReal {
			errors = append(errors, "Reference image failed liveness detection")
			success = false
		}
		if !testResult.Result.IsReal {
			errors = append(errors, "Test image failed liveness detection")
			success = false
		}
	}

	return ConcurrentLivenessResult{
		ReferenceResult: referenceResult,
		TestResult:      testResult,
		TotalTime:       totalTime,
		Success:         success,
		Errors:          errors,
	}
}

// detectSingleImage performs liveness detection on a single image with context support
func (clp *ConcurrentLivenessProcessor) detectSingleImage(ctx context.Context, image, imageID string) LivenessDetectionResult {
	startTime := time.Now()

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return LivenessDetectionResult{
			ImageID:     imageID,
			Error:       ctx.Err(),
			ProcessTime: time.Since(startTime),
		}
	default:
	}

	// Validate input
	if image == "" {
		return LivenessDetectionResult{
			ImageID:     imageID,
			Error:       fmt.Errorf("empty image input"),
			ProcessTime: time.Since(startTime),
		}
	}

	// Validate face matcher
	clp.mu.RLock()
	faceMatcher := clp.faceMatcher
	clp.mu.RUnlock()

	if faceMatcher == nil {
		return LivenessDetectionResult{
			ImageID:     imageID,
			Error:       fmt.Errorf("face matcher not initialized"),
			ProcessTime: time.Since(startTime),
		}
	}

	// Create a channel to receive the result
	resultChan := make(chan AdvancedAntiSpoofResult, 1)
	errorChan := make(chan error, 1)

	// Run liveness detection in a separate goroutine to support cancellation
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorChan <- fmt.Errorf("liveness detection panic: %v", r)
			}
		}()

		result := faceMatcher.DetectAdvancedAntiSpoof(image, imageID, false)
		if result.Error != "" {
			errorChan <- fmt.Errorf(result.Error)
		} else {
			resultChan <- result
		}
	}()

	// Wait for result or context cancellation
	select {
	case result := <-resultChan:
		return LivenessDetectionResult{
			ImageID:     imageID,
			Result:      result,
			ProcessTime: time.Since(startTime),
		}
	case err := <-errorChan:
		return LivenessDetectionResult{
			ImageID:     imageID,
			Error:       err,
			ProcessTime: time.Since(startTime),
		}
	case <-ctx.Done():
		return LivenessDetectionResult{
			ImageID:     imageID,
			Error:       ctx.Err(),
			ProcessTime: time.Since(startTime),
		}
	}
}

// DetectSingleImageWithTimeout performs liveness detection on a single image with timeout
func (clp *ConcurrentLivenessProcessor) DetectSingleImageWithTimeout(image, imageID string, timeout time.Duration) LivenessDetectionResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return clp.detectSingleImage(ctx, image, imageID)
}

// SetTimeout updates the timeout duration for concurrent operations
func (clp *ConcurrentLivenessProcessor) SetTimeout(timeout time.Duration) {
	clp.mu.Lock()
	defer clp.mu.Unlock()
	clp.timeout = timeout
}

// GetTimeout returns the current timeout duration
func (clp *ConcurrentLivenessProcessor) GetTimeout() time.Duration {
	clp.mu.RLock()
	defer clp.mu.RUnlock()
	return clp.timeout
}

// IsHealthy checks if the processor is ready to handle requests
func (clp *ConcurrentLivenessProcessor) IsHealthy() bool {
	clp.mu.RLock()
	defer clp.mu.RUnlock()
	return clp.faceMatcher != nil
}

// GetStats returns basic statistics about the processor
func (clp *ConcurrentLivenessProcessor) GetStats() map[string]interface{} {
	clp.mu.RLock()
	defer clp.mu.RUnlock()

	return map[string]interface{}{
		"timeout_seconds":    clp.timeout.Seconds(),
		"face_matcher_ready": clp.faceMatcher != nil,
		"processor_type":     "concurrent_liveness",
	}
}
