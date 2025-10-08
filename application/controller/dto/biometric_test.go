package dto

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestValidateEnhancedFaceComparisonRequest(t *testing.T) {
	tests := []struct {
		name    string
		request *EnhancedFaceComparisonRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil request",
			request: nil,
			wantErr: true,
			errMsg:  "request cannot be nil",
		},
		{
			name: "empty reference image",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: "",
				TestImage:      "valid_base64_image_data_here",
			},
			wantErr: true,
			errMsg:  "reference_image cannot be empty",
		},
		{
			name: "empty test image",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: strings.Repeat("abcd", 25), // Valid base64
				TestImage:      "",
			},
			wantErr: true,
			errMsg:  "test_image cannot be empty",
		},
		{
			name: "invalid threshold - negative",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: strings.Repeat("abcd", 25), // Valid base64
				TestImage:      strings.Repeat("efgh", 25), // Valid base64
				Threshold:      -0.1,
			},
			wantErr: true,
			errMsg:  "threshold must be between 0.0 and 1.0",
		},
		{
			name: "invalid threshold - greater than 1",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: strings.Repeat("abcd", 25), // Valid base64
				TestImage:      strings.Repeat("efgh", 25), // Valid base64
				Threshold:      1.1,
			},
			wantErr: true,
			errMsg:  "threshold must be between 0.0 and 1.0",
		},
		{
			name: "valid request with base64 images",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage:  strings.Repeat("a", 200), // Valid base64-like string
				TestImage:       strings.Repeat("b", 200), // Valid base64-like string
				Threshold:       0.7,
				RequestID:       "test-123",
				RequireLiveness: true,
			},
			wantErr: false,
		},
		{
			name: "valid request with URL images",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage:  "https://example.com/image1.jpg",
				TestImage:       "https://example.com/image2.jpg",
				Threshold:       0.8,
				RequestID:       "test-456",
				RequireLiveness: false,
			},
			wantErr: false,
		},
		{
			name: "valid request with data URL",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: "data:image/jpeg;base64," + strings.Repeat("a", 200),
				TestImage:      "data:image/jpeg;base64," + strings.Repeat("b", 200),
			},
			wantErr: false,
		},
		{
			name: "localhost URL not allowed",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: "http://localhost:8080/image.jpg",
				TestImage:      strings.Repeat("abcd", 25), // Valid base64
			},
			wantErr: true,
			errMsg:  "localhost URLs not allowed for security reasons",
		},
		{
			name: "127.0.0.1 URL not allowed",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: "http://127.0.0.1/image.jpg",
				TestImage:      strings.Repeat("abcd", 25), // Valid base64
			},
			wantErr: true,
			errMsg:  "localhost URLs not allowed for security reasons",
		},
		{
			name: "image too small",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: "abc",                      // Too small
				TestImage:      strings.Repeat("abcd", 25), // Valid base64
			},
			wantErr: true,
			errMsg:  "too small (minimum 100 characters)",
		},
		{
			name: "invalid base64 encoding",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: strings.Repeat("a", 101),   // Invalid base64 (not multiple of 4)
				TestImage:      strings.Repeat("abcd", 25), // Valid base64
			},
			wantErr: true,
			errMsg:  "invalid base64 encoding",
		},
		{
			name: "URL too long",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: "https://example.com/" + strings.Repeat("a", 2050), // Too long
				TestImage:      strings.Repeat("abcd", 25),                         // Valid base64
			},
			wantErr: true,
			errMsg:  "URL too long (max 2048 characters)",
		},
		{
			name: "invalid data URL format",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: "data:image/jpeg;base64,part1,part2,part3", // Too many parts
				TestImage:      strings.Repeat("abcd", 25),                 // Valid base64
			},
			wantErr: true,
			errMsg:  "invalid data URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnhancedFaceComparisonRequest(tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateEnhancedFaceComparisonRequest() expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateEnhancedFaceComparisonRequest() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateEnhancedFaceComparisonRequest() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateImageInput(t *testing.T) {
	tests := []struct {
		name      string
		image     string
		fieldName string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "empty image",
			image:     "",
			fieldName: "test_field",
			wantErr:   true,
			errMsg:    "test_field cannot be empty",
		},
		{
			name:      "valid HTTP URL",
			image:     "http://example.com/image.jpg",
			fieldName: "test_field",
			wantErr:   false,
		},
		{
			name:      "valid HTTPS URL",
			image:     "https://example.com/image.jpg",
			fieldName: "test_field",
			wantErr:   false,
		},
		{
			name:      "valid base64",
			image:     strings.Repeat("abcd", 25), // 100 chars, multiple of 4
			fieldName: "test_field",
			wantErr:   false,
		},
		{
			name:      "valid data URL",
			image:     "data:image/jpeg;base64," + strings.Repeat("abcd", 25),
			fieldName: "test_field",
			wantErr:   false,
		},
		{
			name:      "localhost URL rejected",
			image:     "http://localhost/image.jpg",
			fieldName: "test_field",
			wantErr:   true,
			errMsg:    "localhost URLs not allowed for security reasons",
		},
		{
			name:      "127.0.0.1 URL rejected",
			image:     "http://127.0.0.1/image.jpg",
			fieldName: "test_field",
			wantErr:   true,
			errMsg:    "localhost URLs not allowed for security reasons",
		},
		{
			name:      "0.0.0.0 URL rejected",
			image:     "http://0.0.0.0/image.jpg",
			fieldName: "test_field",
			wantErr:   true,
			errMsg:    "localhost URLs not allowed for security reasons",
		},
		{
			name:      "URL too long",
			image:     "https://example.com/" + strings.Repeat("a", 2050),
			fieldName: "test_field",
			wantErr:   true,
			errMsg:    "URL too long (max 2048 characters)",
		},
		{
			name:      "base64 too small",
			image:     "abc",
			fieldName: "test_field",
			wantErr:   true,
			errMsg:    "too small (minimum 100 characters)",
		},
		{
			name:      "invalid base64 length",
			image:     strings.Repeat("a", 101), // Not multiple of 4
			fieldName: "test_field",
			wantErr:   true,
			errMsg:    "invalid base64 encoding",
		},
		{
			name:      "invalid data URL format - no comma",
			image:     strings.Repeat("abcd", 30), // Valid base64 length, no data URL prefix
			fieldName: "test_field",
			wantErr:   false, // This should be treated as regular base64
		},
		{
			name:      "invalid data URL format - multiple commas",
			image:     "data:image/jpeg;base64,part1,part2",
			fieldName: "test_field",
			wantErr:   true,
			errMsg:    "invalid data URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateImageInput(tt.image, tt.fieldName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateImageInput() expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateImageInput() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateImageInput() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestEnhancedFaceComparisonRequestFields tests specific fields of the enhanced request
func TestEnhancedFaceComparisonRequestFields(t *testing.T) {
	tests := []struct {
		name    string
		request *EnhancedFaceComparisonRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request with RequireLiveness true",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage:  strings.Repeat("abcd", 25),
				TestImage:       strings.Repeat("efgh", 25),
				RequireLiveness: true,
			},
			wantErr: false,
		},
		{
			name: "valid request with RequireLiveness false",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage:  strings.Repeat("abcd", 25),
				TestImage:       strings.Repeat("efgh", 25),
				RequireLiveness: false,
			},
			wantErr: false,
		},
		{
			name: "valid request with empty RequestID",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: strings.Repeat("abcd", 25),
				TestImage:      strings.Repeat("efgh", 25),
				RequestID:      "",
			},
			wantErr: false,
		},
		{
			name: "valid request with long RequestID",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: strings.Repeat("abcd", 25),
				TestImage:      strings.Repeat("efgh", 25),
				RequestID:      strings.Repeat("x", 100),
			},
			wantErr: false,
		},
		{
			name: "valid request with zero threshold",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: strings.Repeat("abcd", 25),
				TestImage:      strings.Repeat("efgh", 25),
				Threshold:      0.0,
			},
			wantErr: false,
		},
		{
			name: "valid request with threshold 1.0",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: strings.Repeat("abcd", 25),
				TestImage:      strings.Repeat("efgh", 25),
				Threshold:      1.0,
			},
			wantErr: false,
		},
		{
			name: "valid minimal request",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage: strings.Repeat("abcd", 25),
				TestImage:      strings.Repeat("efgh", 25),
			},
			wantErr: false,
		},
		{
			name: "valid complete request",
			request: &EnhancedFaceComparisonRequest{
				ReferenceImage:  "https://example.com/ref.jpg",
				TestImage:       "https://example.com/test.jpg",
				Threshold:       0.85,
				RequestID:       "enhanced-test-123",
				RequireLiveness: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnhancedFaceComparisonRequest(tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateEnhancedFaceComparisonRequest() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateEnhancedFaceComparisonRequest() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateEnhancedFaceComparisonRequest() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestEnhancedFaceComparisonResponse tests the enhanced response structure
func TestEnhancedFaceComparisonResponse(t *testing.T) {
	t.Run("NewEnhancedFaceComparisonResponse", func(t *testing.T) {
		requestID := "test-123"
		response := NewEnhancedFaceComparisonResponse(requestID)

		if response.RequestID != requestID {
			t.Errorf("Expected RequestID %s, got %s", requestID, response.RequestID)
		}

		if response.Timestamp.IsZero() {
			t.Error("Expected Timestamp to be set")
		}

		if !response.IsSuccessful() {
			t.Error("Expected new response to be successful")
		}
	})

	t.Run("SetError", func(t *testing.T) {
		response := NewEnhancedFaceComparisonResponse("test")
		testError := fmt.Errorf("test error")

		response.SetError(testError)

		if response.Error != "test error" {
			t.Errorf("Expected error 'test error', got '%s'", response.Error)
		}

		if response.IsSuccessful() {
			t.Error("Expected response to not be successful after setting error")
		}
	})

	t.Run("SetComparisonResult", func(t *testing.T) {
		response := NewEnhancedFaceComparisonResponse("test")

		response.SetComparisonResult(true, 0.95, 0.92, 150)

		if !response.IsMatch {
			t.Error("Expected IsMatch to be true")
		}
		if response.Similarity != 0.95 {
			t.Errorf("Expected Similarity 0.95, got %f", response.Similarity)
		}
		if response.Confidence != 0.92 {
			t.Errorf("Expected Confidence 0.92, got %f", response.Confidence)
		}
		if response.ProcessingTime != 150 {
			t.Errorf("Expected ProcessingTime 150, got %d", response.ProcessingTime)
		}
	})

	t.Run("SetLivenessResults", func(t *testing.T) {
		response := NewEnhancedFaceComparisonResponse("test")

		refLiveness := NewLivenessResultDTO(true, 0.1, 0.95, []string{})
		testLiveness := NewLivenessResultDTO(true, 0.2, 0.90, []string{})

		response.SetLivenessResults(refLiveness, testLiveness, 200)

		if response.ReferenceLiveness != refLiveness {
			t.Error("Expected ReferenceLiveness to be set")
		}
		if response.TestLiveness != testLiveness {
			t.Error("Expected TestLiveness to be set")
		}
		if response.LivenessProcessTime != 200 {
			t.Errorf("Expected LivenessProcessTime 200, got %d", response.LivenessProcessTime)
		}
	})

	t.Run("AddProcessingStep", func(t *testing.T) {
		response := NewEnhancedFaceComparisonResponse("test")

		response.AddProcessingStep("face_detection", 50, true, "Detected 1 face")
		response.AddProcessingStep("feature_extraction", 100, true, "Extracted 512 features")

		if response.ComparisonMetadata == nil {
			t.Fatal("Expected ComparisonMetadata to be created")
		}

		steps := response.ComparisonMetadata.ProcessingSteps
		if len(steps) != 2 {
			t.Errorf("Expected 2 processing steps, got %d", len(steps))
		}

		if steps[0].Step != "face_detection" {
			t.Errorf("Expected first step 'face_detection', got '%s'", steps[0].Step)
		}
		if steps[0].Duration != 50 {
			t.Errorf("Expected first step duration 50, got %d", steps[0].Duration)
		}
		if !steps[0].Success {
			t.Error("Expected first step to be successful")
		}

		if steps[1].Step != "feature_extraction" {
			t.Errorf("Expected second step 'feature_extraction', got '%s'", steps[1].Step)
		}
	})

	t.Run("GetTotalProcessingTime", func(t *testing.T) {
		response := NewEnhancedFaceComparisonResponse("test")
		response.ProcessingTime = 150
		response.LivenessProcessTime = 200

		total := response.GetTotalProcessingTime()
		expected := int64(350)

		if total != expected {
			t.Errorf("Expected total processing time %d, got %d", expected, total)
		}
	})
}

// TestLivenessResultDTO tests the liveness result DTO
func TestLivenessResultDTO(t *testing.T) {
	t.Run("NewLivenessResultDTO", func(t *testing.T) {
		spoofReasons := []string{"low_quality", "reflection_detected"}
		result := NewLivenessResultDTO(false, 0.8, 0.75, spoofReasons)

		if result.IsLive {
			t.Error("Expected IsLive to be false")
		}
		if result.SpoofScore != 0.8 {
			t.Errorf("Expected SpoofScore 0.8, got %f", result.SpoofScore)
		}
		if result.Confidence != 0.75 {
			t.Errorf("Expected Confidence 0.75, got %f", result.Confidence)
		}
		if len(result.SpoofReasons) != 2 {
			t.Errorf("Expected 2 spoof reasons, got %d", len(result.SpoofReasons))
		}
		if result.SpoofReasons[0] != "low_quality" {
			t.Errorf("Expected first reason 'low_quality', got '%s'", result.SpoofReasons[0])
		}
	})
}

// TestFeatureQualityMetricsDTO tests the feature quality metrics DTO
func TestFeatureQualityMetricsDTO(t *testing.T) {
	t.Run("NewFeatureQualityMetricsDTO", func(t *testing.T) {
		metrics := NewFeatureQualityMetricsDTO(25.5, "center", 0.85, 0.90, 0.88)

		if metrics.FaceSize != 25.5 {
			t.Errorf("Expected FaceSize 25.5, got %f", metrics.FaceSize)
		}
		if metrics.FacePosition != "center" {
			t.Errorf("Expected FacePosition 'center', got '%s'", metrics.FacePosition)
		}
		if metrics.ImageSharpness != 0.85 {
			t.Errorf("Expected ImageSharpness 0.85, got %f", metrics.ImageSharpness)
		}
		if metrics.LightingQuality != 0.90 {
			t.Errorf("Expected LightingQuality 0.90, got %f", metrics.LightingQuality)
		}
		if metrics.FeatureStrength != 0.88 {
			t.Errorf("Expected FeatureStrength 0.88, got %f", metrics.FeatureStrength)
		}
	})
}

// TestEnhancedComparisonMetadataDTO tests the enhanced comparison metadata DTO
func TestEnhancedComparisonMetadataDTO(t *testing.T) {
	t.Run("NewEnhancedComparisonMetadataDTO", func(t *testing.T) {
		metadata := NewEnhancedComparisonMetadataDTO("cosine", 0.8, 0.05, 0.92, "high")

		if metadata.SimilarityMethod != "cosine" {
			t.Errorf("Expected SimilarityMethod 'cosine', got '%s'", metadata.SimilarityMethod)
		}
		if metadata.ThresholdUsed != 0.8 {
			t.Errorf("Expected ThresholdUsed 0.8, got %f", metadata.ThresholdUsed)
		}
		if metadata.QualityAdjustment != 0.05 {
			t.Errorf("Expected QualityAdjustment 0.05, got %f", metadata.QualityAdjustment)
		}
		if metadata.FeatureStrength != 0.92 {
			t.Errorf("Expected FeatureStrength 0.92, got %f", metadata.FeatureStrength)
		}
		if metadata.ConfidenceLevel != "high" {
			t.Errorf("Expected ConfidenceLevel 'high', got '%s'", metadata.ConfidenceLevel)
		}
		if metadata.ProcessingSteps == nil {
			t.Error("Expected ProcessingSteps to be initialized")
		}
		if len(metadata.ProcessingSteps) != 0 {
			t.Errorf("Expected empty ProcessingSteps, got %d items", len(metadata.ProcessingSteps))
		}
	})
}

// TestResponseSerialization tests JSON serialization of the response
func TestResponseSerialization(t *testing.T) {
	t.Run("CompleteResponseSerialization", func(t *testing.T) {
		response := NewEnhancedFaceComparisonResponse("test-123")
		response.SetComparisonResult(true, 0.95, 0.92, 150)

		refLiveness := NewLivenessResultDTO(true, 0.1, 0.95, []string{})
		testLiveness := NewLivenessResultDTO(true, 0.2, 0.90, []string{"slight_blur"})
		response.SetLivenessResults(refLiveness, testLiveness, 200)

		quality := NewFeatureQualityMetricsDTO(25.5, "center", 0.85, 0.90, 0.88)
		response.SetFeatureQuality(quality)

		metadata := NewEnhancedComparisonMetadataDTO("cosine", 0.8, 0.05, 0.92, "high")
		response.SetComparisonMetadata(metadata)

		response.AddProcessingStep("face_detection", 50, true, "Detected 1 face")

		// Test JSON serialization
		jsonData, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal response: %v", err)
		}

		// Test JSON deserialization
		var deserializedResponse EnhancedFaceComparisonResponse
		err = json.Unmarshal(jsonData, &deserializedResponse)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Verify key fields
		if deserializedResponse.RequestID != "test-123" {
			t.Errorf("Expected RequestID 'test-123', got '%s'", deserializedResponse.RequestID)
		}
		if !deserializedResponse.IsMatch {
			t.Error("Expected IsMatch to be true")
		}
		if deserializedResponse.Similarity != 0.95 {
			t.Errorf("Expected Similarity 0.95, got %f", deserializedResponse.Similarity)
		}
		if deserializedResponse.ReferenceLiveness == nil {
			t.Error("Expected ReferenceLiveness to be present")
		}
		if deserializedResponse.TestLiveness == nil {
			t.Error("Expected TestLiveness to be present")
		}
		if deserializedResponse.FeatureQuality == nil {
			t.Error("Expected FeatureQuality to be present")
		}
		if deserializedResponse.ComparisonMetadata == nil {
			t.Error("Expected ComparisonMetadata to be present")
		}
	})

	t.Run("ErrorResponseSerialization", func(t *testing.T) {
		response := NewEnhancedFaceComparisonResponse("error-test")
		response.SetError(fmt.Errorf("face detection failed"))

		jsonData, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal error response: %v", err)
		}

		var deserializedResponse EnhancedFaceComparisonResponse
		err = json.Unmarshal(jsonData, &deserializedResponse)
		if err != nil {
			t.Fatalf("Failed to unmarshal error response: %v", err)
		}

		if deserializedResponse.Error != "face detection failed" {
			t.Errorf("Expected error 'face detection failed', got '%s'", deserializedResponse.Error)
		}
		if deserializedResponse.IsSuccessful() {
			t.Error("Expected error response to not be successful")
		}
	})
}
