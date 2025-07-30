package routev1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID("test")
	id2 := generateRequestID("test")

	// Should be different
	assert.NotEqual(t, id1, id2)

	// Should start with prefix
	assert.Contains(t, id1, "test_")
	assert.Contains(t, id2, "test_")
}

func TestGetConfidenceLevel(t *testing.T) {
	tests := []struct {
		similarity float64
		expected   string
	}{
		{0.95, "high"},
		{0.85, "medium"},
		{0.65, "low"},
		{0.3, "very_low"},
	}

	for _, test := range tests {
		result := getConfidenceLevel(test.similarity)
		assert.Equal(t, test.expected, result, "Similarity %.2f should return %s", test.similarity, test.expected)
	}
}

func TestValidateProductionImageInput(t *testing.T) {
	tests := []struct {
		name        string
		image       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Empty image",
			image:       "",
			expectError: true,
			errorMsg:    "image cannot be empty",
		},
		{
			name:        "Valid HTTPS URL",
			image:       "https://example.com/image.jpg",
			expectError: false,
		},
		{
			name:        "Localhost URL blocked",
			image:       "http://localhost:8080/image.jpg",
			expectError: true,
			errorMsg:    "localhost URLs not allowed for security reasons",
		},
		{
			name:        "Valid base64 with data URL",
			image:       "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD//2Q==",
			expectError: false,
		},
		{
			name:        "Valid base64 without prefix",
			image:       "/9j/4AAQSkZJRgABAQAAAQABAAD//2Q==",
			expectError: false,
		},
		{
			name:        "Invalid base64 format",
			image:       "invalid-base64-data",
			expectError: true,
			errorMsg:    "invalid base64 encoding",
		},
		{
			name:        "Image too small",
			image:       "dGVzdA==", // "test" in base64, too small
			expectError: true,
			errorMsg:    "image too small",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateProductionImageInput(test.image)

			if test.expectError {
				assert.Error(t, err)
				if test.errorMsg != "" {
					assert.Contains(t, err.Error(), test.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test that the processing functions exist and can be called
func TestProcessingFunctionsExist(t *testing.T) {
	// These tests just verify the functions exist and have the right signatures
	// Actual functionality testing would require mocking the facematch service

	t.Run("processProductionLivenessDetection exists", func(t *testing.T) {
		// This test just ensures the function exists and can be referenced
		assert.NotNil(t, processProductionLivenessDetection)
	})

	t.Run("processProductionFaceComparison exists", func(t *testing.T) {
		assert.NotNil(t, processProductionFaceComparison)
	})

	t.Run("processProductionImageQuality exists", func(t *testing.T) {
		assert.NotNil(t, processProductionImageQuality)
	})

	t.Run("processProductionBatchLiveness exists", func(t *testing.T) {
		assert.NotNil(t, processProductionBatchLiveness)
	})

	t.Run("processProductionSystemHealth exists", func(t *testing.T) {
		assert.NotNil(t, processProductionSystemHealth)
	})

	t.Run("processProductionSystemMetrics exists", func(t *testing.T) {
		assert.NotNil(t, processProductionSystemMetrics)
	})

	t.Run("processProductionSystemConfig exists", func(t *testing.T) {
		assert.NotNil(t, processProductionSystemConfig)
	})
}

// Test request/response structures
func TestRequestResponseStructures(t *testing.T) {
	t.Run("ProductionLivenessRequest structure", func(t *testing.T) {
		req := ProductionLivenessRequest{
			Image:     "test-image",
			RequestID: "test-123",
			Verbose:   true,
		}

		assert.Equal(t, "test-image", req.Image)
		assert.Equal(t, "test-123", req.RequestID)
		assert.True(t, req.Verbose)
	})

	t.Run("ProductionFaceComparisonRequest structure", func(t *testing.T) {
		req := ProductionFaceComparisonRequest{
			ReferenceImage: "ref-image",
			TestImage:      "test-image",
			Threshold:      0.8,
			RequestID:      "test-456",
		}

		assert.Equal(t, "ref-image", req.ReferenceImage)
		assert.Equal(t, "test-image", req.TestImage)
		assert.Equal(t, 0.8, req.Threshold)
		assert.Equal(t, "test-456", req.RequestID)
	})

	t.Run("ProductionImageQualityRequest structure", func(t *testing.T) {
		req := ProductionImageQualityRequest{
			Image:     "quality-image",
			RequestID: "test-789",
		}

		assert.Equal(t, "quality-image", req.Image)
		assert.Equal(t, "test-789", req.RequestID)
	})

	t.Run("ProductionBatchLivenessRequest structure", func(t *testing.T) {
		req := ProductionBatchLivenessRequest{
			Images:    []string{"image1", "image2"},
			RequestID: "batch-123",
			Verbose:   false,
		}

		assert.Equal(t, []string{"image1", "image2"}, req.Images)
		assert.Equal(t, "batch-123", req.RequestID)
		assert.False(t, req.Verbose)
	})
}
