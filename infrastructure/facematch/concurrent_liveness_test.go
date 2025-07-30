package facematch

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// MockFaceMatcher for testing concurrent liveness detection
type MockFaceMatcher struct {
	mu               sync.RWMutex
	shouldFail       bool
	shouldTimeout    bool
	processingDelay  time.Duration
	livenessResult   bool
	spoofScore       float64
	confidence       float64
	callCount        int
	lastRequestID    string
	simulateError    string
	simulatePanic    bool
	processedImages  []string
	customDetectFunc func(string, string, bool) AdvancedAntiSpoofResult
}

// NewMockFaceMatcher creates a new mock face matcher for testing
func NewMockFaceMatcher() *MockFaceMatcher {
	return &MockFaceMatcher{
		livenessResult: true,
		spoofScore:     0.1,
		confidence:     0.9,
		callCount:      0,
	}
}

// DetectAdvancedAntiSpoof mocks the advanced anti-spoof detection
func (m *MockFaceMatcher) DetectAdvancedAntiSpoof(input string, requestID string, verboseMode bool) AdvancedAntiSpoofResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++
	m.lastRequestID = requestID
	m.processedImages = append(m.processedImages, input)

	// Use custom function if provided
	if m.customDetectFunc != nil {
		return m.customDetectFunc(input, requestID, verboseMode)
	}

	// Simulate panic if configured
	if m.simulatePanic {
		panic("simulated panic in liveness detection")
	}

	// Simulate processing delay
	if m.processingDelay > 0 {
		time.Sleep(m.processingDelay)
	}

	// Simulate timeout by sleeping longer than expected
	if m.shouldTimeout {
		time.Sleep(15 * time.Second) // Longer than typical timeout
	}

	// Simulate error
	if m.simulateError != "" {
		return AdvancedAntiSpoofResult{
			Error: m.simulateError,
		}
	}

	// Simulate failure based on configuration
	if m.shouldFail {
		return AdvancedAntiSpoofResult{
			IsReal:       false,
			SpoofScore:   0.8,
			Confidence:   0.9,
			HasFace:      true,
			ProcessTime:  int64(m.processingDelay.Milliseconds()),
			SpoofReasons: []string{"mock failure reason"},
		}
	}

	// Return successful result
	return AdvancedAntiSpoofResult{
		IsReal:      m.livenessResult,
		SpoofScore:  m.spoofScore,
		Confidence:  m.confidence,
		HasFace:     true,
		ProcessTime: int64(m.processingDelay.Milliseconds()),
	}
}

// SetCustomDetectFunc sets a custom detection function
func (m *MockFaceMatcher) SetCustomDetectFunc(fn func(string, string, bool) AdvancedAntiSpoofResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.customDetectFunc = fn
}

// SetShouldFail configures the mock to simulate failure
func (m *MockFaceMatcher) SetShouldFail(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
}

// SetProcessingDelay configures the mock processing delay
func (m *MockFaceMatcher) SetProcessingDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processingDelay = delay
}

// SetShouldTimeout configures the mock to simulate timeout
func (m *MockFaceMatcher) SetShouldTimeout(timeout bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldTimeout = timeout
}

// SetSimulateError configures the mock to simulate specific errors
func (m *MockFaceMatcher) SetSimulateError(errorMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateError = errorMsg
}

// SetSimulatePanic configures the mock to simulate panic
func (m *MockFaceMatcher) SetSimulatePanic(panic bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulatePanic = panic
}

// GetCallCount returns the number of times DetectAdvancedAntiSpoof was called
func (m *MockFaceMatcher) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetProcessedImages returns the list of processed images
func (m *MockFaceMatcher) GetProcessedImages() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string{}, m.processedImages...)
}

// Reset resets the mock state
func (m *MockFaceMatcher) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
	m.processedImages = []string{}
	m.shouldFail = false
	m.shouldTimeout = false
	m.simulateError = ""
	m.simulatePanic = false
	m.processingDelay = 0
	m.customDetectFunc = nil
}

// Test basic concurrent liveness detection functionality
func TestConcurrentLivenessProcessor_DetectBothImages_Success(t *testing.T) {
	mockFaceMatcher := NewMockFaceMatcher()
	processor := NewConcurrentLivenessProcessor(mockFaceMatcher, 5*time.Second)

	referenceImage := "base64_reference_image_data"
	testImage := "base64_test_image_data"
	requestID := "test_request_123"

	result := processor.DetectBothImages(referenceImage, testImage, requestID)

	// Verify successful processing
	if !result.Success {
		t.Errorf("Expected successful processing, got failure with errors: %v", result.Errors)
	}

	// Verify both images were processed
	if result.ReferenceResult.Error != nil {
		t.Errorf("Reference image processing failed: %v", result.ReferenceResult.Error)
	}

	if result.TestResult.Error != nil {
		t.Errorf("Test image processing failed: %v", result.TestResult.Error)
	}

	// Verify liveness results
	if !result.ReferenceResult.Result.IsReal {
		t.Error("Reference image should be detected as real")
	}

	if !result.TestResult.Result.IsReal {
		t.Error("Test image should be detected as real")
	}

	// Verify concurrent processing (both images should be processed)
	if mockFaceMatcher.GetCallCount() != 2 {
		t.Errorf("Expected 2 calls to face matcher, got %d", mockFaceMatcher.GetCallCount())
	}

	// Verify processing time is reasonable (should be less than sequential processing)
	if result.TotalTime > 3*time.Second {
		t.Errorf("Concurrent processing took too long: %v", result.TotalTime)
	}

	// Verify image IDs are correctly set
	expectedRefID := fmt.Sprintf("%s_ref", requestID)
	expectedTestID := fmt.Sprintf("%s_test", requestID)

	if result.ReferenceResult.ImageID != expectedRefID {
		t.Errorf("Expected reference image ID %s, got %s", expectedRefID, result.ReferenceResult.ImageID)
	}

	if result.TestResult.ImageID != expectedTestID {
		t.Errorf("Expected test image ID %s, got %s", expectedTestID, result.TestResult.ImageID)
	}
}

// Test concurrent processing with one image failing liveness detection
func TestConcurrentLivenessProcessor_DetectBothImages_OneImageFailsLiveness(t *testing.T) {
	mockFaceMatcher := NewMockFaceMatcher()
	processor := NewConcurrentLivenessProcessor(mockFaceMatcher, 5*time.Second)

	// Configure mock to fail liveness for the first call (reference image)
	callCount := 0
	mockFaceMatcher.SetCustomDetectFunc(func(input string, requestID string, verboseMode bool) AdvancedAntiSpoofResult {
		callCount++
		if callCount == 1 { // First call (reference image)
			return AdvancedAntiSpoofResult{
				IsReal:       false,
				SpoofScore:   0.8,
				Confidence:   0.9,
				HasFace:      true,
				ProcessTime:  100,
				SpoofReasons: []string{"detected as spoof"},
			}
		}
		// Second call (test image) - return success
		return AdvancedAntiSpoofResult{
			IsReal:      true,
			SpoofScore:  0.1,
			Confidence:  0.9,
			HasFace:     true,
			ProcessTime: 100,
		}
	})

	referenceImage := "base64_reference_image_data"
	testImage := "base64_test_image_data"
	requestID := "test_request_123"

	result := processor.DetectBothImages(referenceImage, testImage, requestID)

	// Verify processing failed due to liveness failure
	if result.Success {
		t.Error("Expected processing to fail due to liveness failure")
	}

	// Verify error message contains liveness failure
	if len(result.Errors) == 0 {
		t.Error("Expected error messages for liveness failure")
	}

	foundLivenessError := false
	for _, err := range result.Errors {
		if strings.Contains(err, "failed liveness detection") {
			foundLivenessError = true
			break
		}
	}

	if !foundLivenessError {
		t.Errorf("Expected liveness failure error, got errors: %v", result.Errors)
	}

	// Verify both images were still processed
	if result.ReferenceResult.Error != nil {
		t.Errorf("Reference image processing should not have error: %v", result.ReferenceResult.Error)
	}

	if result.TestResult.Error != nil {
		t.Errorf("Test image processing should not have error: %v", result.TestResult.Error)
	}
}

// Test timeout handling
func TestConcurrentLivenessProcessor_DetectBothImages_Timeout(t *testing.T) {
	mockFaceMatcher := NewMockFaceMatcher()
	mockFaceMatcher.SetProcessingDelay(3 * time.Second)                         // Longer than timeout
	processor := NewConcurrentLivenessProcessor(mockFaceMatcher, 1*time.Second) // Short timeout

	referenceImage := "base64_reference_image_data"
	testImage := "base64_test_image_data"
	requestID := "test_request_timeout"

	start := time.Now()
	result := processor.DetectBothImages(referenceImage, testImage, requestID)
	elapsed := time.Since(start)

	// Verify processing failed due to timeout
	if result.Success {
		t.Error("Expected processing to fail due to timeout")
	}

	// Verify timeout occurred within reasonable time (should be close to timeout duration)
	if elapsed > 2*time.Second {
		t.Errorf("Timeout took too long: %v", elapsed)
	}

	// Verify error messages contain timeout information
	if len(result.Errors) == 0 {
		t.Error("Expected error messages for timeout")
	}

	foundTimeoutError := false
	for _, err := range result.Errors {
		if strings.Contains(err, "timeout") {
			foundTimeoutError = true
			break
		}
	}

	if !foundTimeoutError {
		t.Errorf("Expected timeout error, got errors: %v", result.Errors)
	}
}

// Test error handling for invalid inputs
func TestConcurrentLivenessProcessor_DetectBothImages_InvalidInputs(t *testing.T) {
	mockFaceMatcher := NewMockFaceMatcher()
	processor := NewConcurrentLivenessProcessor(mockFaceMatcher, 5*time.Second)

	testCases := []struct {
		name           string
		referenceImage string
		testImage      string
		expectedErrors int
	}{
		{
			name:           "Both images empty",
			referenceImage: "",
			testImage:      "",
			expectedErrors: 2,
		},
		{
			name:           "Reference image empty",
			referenceImage: "",
			testImage:      "valid_test_image",
			expectedErrors: 1,
		},
		{
			name:           "Test image empty",
			referenceImage: "valid_reference_image",
			testImage:      "",
			expectedErrors: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := processor.DetectBothImages(tc.referenceImage, tc.testImage, "test_request")

			if result.Success {
				t.Error("Expected processing to fail with invalid inputs")
			}

			if len(result.Errors) != tc.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %v", tc.expectedErrors, len(result.Errors), result.Errors)
			}
		})
	}
}

// Test face matcher error handling
func TestConcurrentLivenessProcessor_DetectBothImages_FaceMatcherErrors(t *testing.T) {
	mockFaceMatcher := NewMockFaceMatcher()
	mockFaceMatcher.SetSimulateError("face detection failed")
	processor := NewConcurrentLivenessProcessor(mockFaceMatcher, 5*time.Second)

	referenceImage := "base64_reference_image_data"
	testImage := "base64_test_image_data"
	requestID := "test_request_error"

	result := processor.DetectBothImages(referenceImage, testImage, requestID)

	// Verify processing failed due to face matcher errors
	if result.Success {
		t.Error("Expected processing to fail due to face matcher errors")
	}

	// Verify both results have errors
	if result.ReferenceResult.Error == nil {
		t.Error("Expected reference result to have error")
	}

	if result.TestResult.Error == nil {
		t.Error("Expected test result to have error")
	}

	// Verify error messages
	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

// Test panic recovery
func TestConcurrentLivenessProcessor_DetectBothImages_PanicRecovery(t *testing.T) {
	mockFaceMatcher := NewMockFaceMatcher()
	mockFaceMatcher.SetSimulatePanic(true)
	processor := NewConcurrentLivenessProcessor(mockFaceMatcher, 5*time.Second)

	referenceImage := "base64_reference_image_data"
	testImage := "base64_test_image_data"
	requestID := "test_request_panic"

	result := processor.DetectBothImages(referenceImage, testImage, requestID)

	// Verify processing failed due to panic
	if result.Success {
		t.Error("Expected processing to fail due to panic")
	}

	// Verify both results have errors
	if result.ReferenceResult.Error == nil {
		t.Error("Expected reference result to have error from panic")
	}

	if result.TestResult.Error == nil {
		t.Error("Expected test result to have error from panic")
	}

	// Verify panic was recovered and converted to error
	foundPanicError := false
	for _, err := range result.Errors {
		if strings.Contains(err, "panic") {
			foundPanicError = true
			break
		}
	}

	if !foundPanicError {
		t.Errorf("Expected panic error, got errors: %v", result.Errors)
	}
}

// Test single image detection with timeout
func TestConcurrentLivenessProcessor_DetectSingleImageWithTimeout(t *testing.T) {
	mockFaceMatcher := NewMockFaceMatcher()
	processor := NewConcurrentLivenessProcessor(mockFaceMatcher, 5*time.Second)

	image := "base64_image_data"
	imageID := "single_image_test"
	timeout := 2 * time.Second

	result := processor.DetectSingleImageWithTimeout(image, imageID, timeout)

	// Verify successful processing
	if result.Error != nil {
		t.Errorf("Single image detection failed: %v", result.Error)
	}

	// Verify result properties
	if result.ImageID != imageID {
		t.Errorf("Expected image ID %s, got %s", imageID, result.ImageID)
	}

	if !result.Result.IsReal {
		t.Error("Expected image to be detected as real")
	}

	// Verify processing time is reasonable
	if result.ProcessTime > timeout {
		t.Errorf("Processing time %v exceeded timeout %v", result.ProcessTime, timeout)
	}
}

// Test processor configuration methods
func TestConcurrentLivenessProcessor_Configuration(t *testing.T) {
	mockFaceMatcher := NewMockFaceMatcher()
	processor := NewConcurrentLivenessProcessor(mockFaceMatcher, 5*time.Second)

	// Test initial timeout
	if processor.GetTimeout() != 5*time.Second {
		t.Errorf("Expected initial timeout 5s, got %v", processor.GetTimeout())
	}

	// Test timeout update
	newTimeout := 10 * time.Second
	processor.SetTimeout(newTimeout)

	if processor.GetTimeout() != newTimeout {
		t.Errorf("Expected updated timeout %v, got %v", newTimeout, processor.GetTimeout())
	}

	// Test health check
	if !processor.IsHealthy() {
		t.Error("Processor should be healthy with valid face matcher")
	}

	// Test stats
	stats := processor.GetStats()
	if stats["timeout_seconds"] != newTimeout.Seconds() {
		t.Errorf("Expected timeout in stats %v, got %v", newTimeout.Seconds(), stats["timeout_seconds"])
	}

	if stats["face_matcher_ready"] != true {
		t.Error("Face matcher should be ready in stats")
	}

	if stats["processor_type"] != "concurrent_liveness" {
		t.Errorf("Expected processor type 'concurrent_liveness', got %v", stats["processor_type"])
	}
}

// Test processor with nil face matcher
func TestConcurrentLivenessProcessor_NilFaceMatcher(t *testing.T) {
	processor := NewConcurrentLivenessProcessor(nil, 5*time.Second)

	// Test health check with nil face matcher
	if processor.IsHealthy() {
		t.Error("Processor should not be healthy with nil face matcher")
	}

	// Test detection with nil face matcher
	result := processor.DetectBothImages("image1", "image2", "test_request")

	if result.Success {
		t.Error("Expected processing to fail with nil face matcher")
	}

	// Verify both results have face matcher errors
	if result.ReferenceResult.Error == nil {
		t.Error("Expected reference result to have face matcher error")
	}

	if result.TestResult.Error == nil {
		t.Error("Expected test result to have face matcher error")
	}
}

// Test default timeout configuration
func TestConcurrentLivenessProcessor_DefaultTimeout(t *testing.T) {
	mockFaceMatcher := NewMockFaceMatcher()

	// Test with zero timeout (should use default)
	processor1 := NewConcurrentLivenessProcessor(mockFaceMatcher, 0)
	if processor1.GetTimeout() != 10*time.Second {
		t.Errorf("Expected default timeout 10s, got %v", processor1.GetTimeout())
	}

	// Test with negative timeout (should use default)
	processor2 := NewConcurrentLivenessProcessor(mockFaceMatcher, -5*time.Second)
	if processor2.GetTimeout() != 10*time.Second {
		t.Errorf("Expected default timeout 10s, got %v", processor2.GetTimeout())
	}
}

// Benchmark concurrent vs sequential processing
func BenchmarkConcurrentLivenessProcessor_ConcurrentVsSequential(b *testing.B) {
	mockFaceMatcher := NewMockFaceMatcher()
	mockFaceMatcher.SetProcessingDelay(100 * time.Millisecond) // Simulate realistic processing time
	processor := NewConcurrentLivenessProcessor(mockFaceMatcher, 10*time.Second)

	referenceImage := "base64_reference_image_data"
	testImage := "base64_test_image_data"

	b.Run("Concurrent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			mockFaceMatcher.Reset()
			requestID := fmt.Sprintf("bench_concurrent_%d", i)
			processor.DetectBothImages(referenceImage, testImage, requestID)
		}
	})

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			mockFaceMatcher.Reset()
			requestID := fmt.Sprintf("bench_sequential_%d", i)
			// Simulate sequential processing
			processor.DetectSingleImageWithTimeout(referenceImage, fmt.Sprintf("%s_ref", requestID), 5*time.Second)
			processor.DetectSingleImageWithTimeout(testImage, fmt.Sprintf("%s_test", requestID), 5*time.Second)
		}
	})
}

// Test concurrent processing with various image combinations
func TestConcurrentLivenessProcessor_VariousImageCombinations(t *testing.T) {
	mockFaceMatcher := NewMockFaceMatcher()
	processor := NewConcurrentLivenessProcessor(mockFaceMatcher, 5*time.Second)

	testCases := []struct {
		name           string
		referenceImage string
		testImage      string
		setupMock      func(*MockFaceMatcher)
		expectSuccess  bool
		expectErrors   int
	}{
		{
			name:           "Both images valid and live",
			referenceImage: "valid_reference_image",
			testImage:      "valid_test_image",
			setupMock:      func(m *MockFaceMatcher) { /* default setup */ },
			expectSuccess:  true,
			expectErrors:   0,
		},
		{
			name:           "Reference image spoof, test image live",
			referenceImage: "spoof_reference_image",
			testImage:      "valid_test_image",
			setupMock: func(m *MockFaceMatcher) {
				m.SetCustomDetectFunc(func(input string, requestID string, verboseMode bool) AdvancedAntiSpoofResult {
					if strings.Contains(input, "spoof") {
						return AdvancedAntiSpoofResult{
							IsReal:       false,
							SpoofScore:   0.8,
							Confidence:   0.9,
							HasFace:      true,
							ProcessTime:  50,
							SpoofReasons: []string{"detected as spoof"},
						}
					}
					return AdvancedAntiSpoofResult{
						IsReal:      true,
						SpoofScore:  0.1,
						Confidence:  0.9,
						HasFace:     true,
						ProcessTime: 50,
					}
				})
			},
			expectSuccess: false,
			expectErrors:  1,
		},
		{
			name:           "Both images spoof",
			referenceImage: "spoof_reference_image",
			testImage:      "spoof_test_image",
			setupMock: func(m *MockFaceMatcher) {
				m.SetCustomDetectFunc(func(input string, requestID string, verboseMode bool) AdvancedAntiSpoofResult {
					return AdvancedAntiSpoofResult{
						IsReal:       false,
						SpoofScore:   0.8,
						Confidence:   0.9,
						HasFace:      true,
						ProcessTime:  50,
						SpoofReasons: []string{"detected as spoof"},
					}
				})
			},
			expectSuccess: false,
			expectErrors:  2,
		},
		{
			name:           "Large base64 images",
			referenceImage: strings.Repeat("a", 1000), // Large base64 string
			testImage:      strings.Repeat("b", 1000), // Large base64 string
			setupMock:      func(m *MockFaceMatcher) { /* default setup */ },
			expectSuccess:  true,
			expectErrors:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockFaceMatcher.Reset()
			tc.setupMock(mockFaceMatcher)

			result := processor.DetectBothImages(tc.referenceImage, tc.testImage, fmt.Sprintf("test_%s", tc.name))

			if result.Success != tc.expectSuccess {
				t.Errorf("Expected success %v, got %v", tc.expectSuccess, result.Success)
			}

			if len(result.Errors) != tc.expectErrors {
				t.Errorf("Expected %d errors, got %d: %v", tc.expectErrors, len(result.Errors), result.Errors)
			}

			// Verify both images were processed (even if they failed)
			if result.ReferenceResult.ImageID == "" {
				t.Error("Reference result should have image ID")
			}

			if result.TestResult.ImageID == "" {
				t.Error("Test result should have image ID")
			}
		})
	}
}
