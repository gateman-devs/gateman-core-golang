package facematch

import (
	"image"
	"testing"

	"gocv.io/x/gocv"
)

func TestNewImagePreprocessor(t *testing.T) {
	tests := []struct {
		name           string
		targetSize     image.Point
		method         NormalizationMethod
		expectedSize   image.Point
		expectedMethod NormalizationMethod
	}{
		{
			name:           "Valid target size",
			targetSize:     image.Pt(224, 224),
			method:         StandardNormalization,
			expectedSize:   image.Pt(224, 224),
			expectedMethod: StandardNormalization,
		},
		{
			name:           "Invalid target size - defaults to 112x112",
			targetSize:     image.Pt(0, 0),
			method:         HistogramEqualization,
			expectedSize:   image.Pt(112, 112),
			expectedMethod: HistogramEqualization,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewImagePreprocessor(tt.targetSize, tt.method)

			if processor.GetTargetSize() != tt.expectedSize {
				t.Errorf("Expected target size %v, got %v", tt.expectedSize, processor.GetTargetSize())
			}

			if processor.GetNormalizationMethod() != tt.expectedMethod {
				t.Errorf("Expected normalization method %v, got %v", tt.expectedMethod, processor.GetNormalizationMethod())
			}
		})
	}
}

func TestPreprocessFace_EmptyImage(t *testing.T) {
	processor := NewImagePreprocessor(image.Pt(112, 112), StandardNormalization)

	// Test with empty image
	emptyFace := gocv.NewMat()
	result := processor.PreprocessFace(emptyFace)

	if result.Error == nil {
		t.Error("Expected error for empty image but got none")
	}
}

func TestPreprocessFace_ValidImage(t *testing.T) {
	processor := NewImagePreprocessor(image.Pt(112, 112), StandardNormalization)

	// Create a valid color image
	face := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC3)
	face.SetTo(gocv.NewScalar(128, 128, 128, 0))
	defer face.Close()

	result := processor.PreprocessFace(face)
	defer func() {
		if !result.ProcessedFace.Empty() {
			result.ProcessedFace.Close()
		}
	}()

	if result.Error != nil {
		t.Errorf("Unexpected error: %v", result.Error)
	}

	if result.ProcessedFace.Empty() {
		t.Error("Processed face should not be empty")
	} else {
		size := result.ProcessedFace.Size()
		if len(size) < 2 || size[0] != 112 || size[1] != 112 {
			t.Errorf("Expected size 112x112, got %v", size)
		}
	}
}

func TestStandardizeImage(t *testing.T) {
	processor := NewImagePreprocessor(image.Pt(112, 112), StandardNormalization)

	// Create a test face image
	face := gocv.NewMatWithSize(80, 80, gocv.MatTypeCV8UC3)
	defer face.Close()
	face.SetTo(gocv.NewScalar(100, 150, 200, 0))

	standardized, err := processor.StandardizeImage(face)
	defer func() {
		if !standardized.Empty() {
			standardized.Close()
		}
	}()

	if err != nil {
		t.Fatalf("StandardizeImage failed: %v", err)
	}

	if standardized.Empty() {
		t.Error("Standardized image is empty")
		return
	}

	// Check dimensions
	size := standardized.Size()
	if len(size) < 2 || size[0] != 112 || size[1] != 112 {
		t.Errorf("Standardized image has incorrect size: %v", size)
	}
}
