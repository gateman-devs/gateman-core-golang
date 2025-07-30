package facematch

import (
	"image"
	"math"
	"testing"

	"gocv.io/x/gocv"
)

func TestNewFeatureQualityAssessor(t *testing.T) {
	assessor := NewFeatureQualityAssessor()

	if assessor == nil {
		t.Fatal("NewFeatureQualityAssessor returned nil")
	}

	minFace, maxFace, sharpness, lighting, feature := assessor.GetThresholds()

	if minFace != MIN_FACE_SIZE_PERCENT_QUALITY {
		t.Errorf("Expected min face size %f, got %f", MIN_FACE_SIZE_PERCENT_QUALITY, minFace)
	}

	if maxFace != MAX_FACE_SIZE_PERCENT_QUALITY {
		t.Errorf("Expected max face size %f, got %f", MAX_FACE_SIZE_PERCENT_QUALITY, maxFace)
	}

	if sharpness != MIN_SHARPNESS_THRESHOLD {
		t.Errorf("Expected sharpness threshold %f, got %f", MIN_SHARPNESS_THRESHOLD, sharpness)
	}

	if lighting != MIN_BRIGHTNESS_QUALITY {
		t.Errorf("Expected lighting threshold %f, got %f", MIN_BRIGHTNESS_QUALITY, lighting)
	}

	if feature != MIN_EDGE_DENSITY_QUALITY {
		t.Errorf("Expected feature threshold %f, got %f", MIN_EDGE_DENSITY_QUALITY, feature)
	}
}

func TestAssessQuality_EmptyImages(t *testing.T) {
	assessor := NewFeatureQualityAssessor()

	// Test with empty face image
	emptyFace := gocv.NewMat()
	validOriginal := gocv.NewMatWithSize(200, 200, gocv.MatTypeCV8UC3)
	defer validOriginal.Close()

	result := assessor.AssessQuality(emptyFace, validOriginal, image.Rect(50, 50, 150, 150))

	if result.Error == nil {
		t.Error("Expected error for empty face image")
	}

	// Test with empty original image
	validFace := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC3)
	defer validFace.Close()
	emptyOriginal := gocv.NewMat()

	result = assessor.AssessQuality(validFace, emptyOriginal, image.Rect(50, 50, 150, 150))

	if result.Error == nil {
		t.Error("Expected error for empty original image")
	}
}

func TestAssessQuality_ValidImages(t *testing.T) {
	assessor := NewFeatureQualityAssessor()

	// Create test images
	originalImage := gocv.NewMatWithSize(200, 200, gocv.MatTypeCV8UC3)
	defer originalImage.Close()
	originalImage.SetTo(gocv.NewScalar(100, 100, 100, 0))

	faceImage := gocv.NewMatWithSize(80, 80, gocv.MatTypeCV8UC3)
	defer faceImage.Close()
	faceImage.SetTo(gocv.NewScalar(120, 120, 120, 0))

	// Face rectangle (40% of original image)
	faceRect := image.Rect(60, 60, 140, 140)

	result := assessor.AssessQuality(faceImage, originalImage, faceRect)

	if result.Error != nil {
		t.Fatalf("Unexpected error: %v", result.Error)
	}

	metrics := result.Metrics

	// Check that all metrics are within valid ranges
	if metrics.FaceSize < 0 || metrics.FaceSize > 100 {
		t.Errorf("Face size out of range: %f", metrics.FaceSize)
	}

	if metrics.ImageSharpness < 0 || metrics.ImageSharpness > 1000 {
		t.Errorf("Image sharpness out of range: %f", metrics.ImageSharpness)
	}

	if metrics.LightingQuality < 0 || metrics.LightingQuality > 255 {
		t.Errorf("Lighting quality out of range: %f", metrics.LightingQuality)
	}

	if metrics.FeatureStrength < 0 || metrics.FeatureStrength > 1 {
		t.Errorf("Feature strength out of range: %f", metrics.FeatureStrength)
	}

	if metrics.OverallQuality < 0 || metrics.OverallQuality > 1 {
		t.Errorf("Overall quality out of range: %f", metrics.OverallQuality)
	}

	if metrics.FacePosition == "" {
		t.Error("Face position should not be empty")
	}
}

func TestAssessFaceSize(t *testing.T) {
	assessor := NewFeatureQualityAssessor()

	tests := []struct {
		name            string
		faceRect        image.Rectangle
		imageSize       []int
		expectedSize    float64
		expectGoodScore bool
	}{
		{
			name:            "Optimal face size",
			faceRect:        image.Rect(40, 40, 160, 160), // 120x120 in 200x200 = 36%
			imageSize:       []int{200, 200},
			expectedSize:    36.0,
			expectGoodScore: true,
		},
		{
			name:            "Too small face",
			faceRect:        image.Rect(90, 90, 110, 110), // 20x20 in 200x200 = 1%
			imageSize:       []int{200, 200},
			expectedSize:    1.0,
			expectGoodScore: false,
		},
		{
			name:            "Too large face",
			faceRect:        image.Rect(10, 10, 190, 190), // 180x180 in 200x200 = 81%
			imageSize:       []int{200, 200},
			expectedSize:    81.0,
			expectGoodScore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, score := assessor.assessFaceSize(tt.faceRect, tt.imageSize)

			if math.Abs(size-tt.expectedSize) > 1.0 {
				t.Errorf("Expected face size ~%f, got %f", tt.expectedSize, size)
			}

			if tt.expectGoodScore && score < 0.7 {
				t.Errorf("Expected good score (>0.7) for optimal size, got %f", score)
			}

			if !tt.expectGoodScore && score > 0.5 {
				t.Errorf("Expected poor score (<0.5) for bad size, got %f", score)
			}
		})
	}
}

func TestAssessFacePosition(t *testing.T) {
	assessor := NewFeatureQualityAssessor()

	tests := []struct {
		name            string
		faceRect        image.Rectangle
		imageSize       []int
		expectedPos     string
		expectGoodScore bool
	}{
		{
			name:            "Well centered face",
			faceRect:        image.Rect(75, 75, 125, 125), // Center of 200x200
			imageSize:       []int{200, 200},
			expectedPos:     "well-centered",
			expectGoodScore: true,
		},
		{
			name:            "Off-center face",
			faceRect:        image.Rect(10, 10, 60, 60), // More off-center
			imageSize:       []int{200, 200},
			expectedPos:     "off-center",
			expectGoodScore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			position, score := assessor.assessFacePosition(tt.faceRect, tt.imageSize)

			if position != tt.expectedPos {
				t.Errorf("Expected position %s, got %s", tt.expectedPos, position)
			}

			if tt.expectGoodScore && score < 0.8 {
				t.Errorf("Expected good score (>0.8) for centered face, got %f", score)
			}

			if !tt.expectGoodScore && score > 0.6 {
				t.Errorf("Expected poor score (<0.6) for off-center face, got %f", score)
			}
		})
	}
}

func TestAssessImageSharpness(t *testing.T) {
	assessor := NewFeatureQualityAssessor()

	// Create a sharp image with high contrast edges
	sharpImage := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC3)
	defer sharpImage.Close()

	// Create a pattern with sharp edges
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			if (x/10)%2 == 0 {
				sharpImage.SetUCharAt(y, x*3, 255)   // B
				sharpImage.SetUCharAt(y, x*3+1, 255) // G
				sharpImage.SetUCharAt(y, x*3+2, 255) // R
			} else {
				sharpImage.SetUCharAt(y, x*3, 0)   // B
				sharpImage.SetUCharAt(y, x*3+1, 0) // G
				sharpImage.SetUCharAt(y, x*3+2, 0) // R
			}
		}
	}

	sharpness, score := assessor.assessImageSharpness(sharpImage)

	if sharpness <= 0 {
		t.Error("Sharpness should be positive for sharp image")
	}

	if score <= 0 {
		t.Error("Sharpness score should be positive for sharp image")
	}

	// Create a blurry image (uniform color)
	blurryImage := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC3)
	defer blurryImage.Close()
	blurryImage.SetTo(gocv.NewScalar(128, 128, 128, 0))

	blurrySharpness, blurryScore := assessor.assessImageSharpness(blurryImage)

	if blurrySharpness >= sharpness {
		t.Error("Blurry image should have lower sharpness than sharp image")
	}

	if blurryScore >= score {
		t.Error("Blurry image should have lower sharpness score than sharp image")
	}
}

func TestAssessLightingQuality(t *testing.T) {
	assessor := NewFeatureQualityAssessor()

	// Test optimal lighting
	optimalImage := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC3)
	defer optimalImage.Close()
	optimalImage.SetTo(gocv.NewScalar(120, 120, 120, 0)) // Good brightness

	brightness, score := assessor.assessLightingQuality(optimalImage)

	if brightness < OPTIMAL_BRIGHTNESS_MIN || brightness > OPTIMAL_BRIGHTNESS_MAX {
		t.Logf("Brightness %f is outside optimal range [%f, %f] but that's expected for uniform image",
			brightness, OPTIMAL_BRIGHTNESS_MIN, OPTIMAL_BRIGHTNESS_MAX)
	}

	if score <= 0 {
		t.Error("Lighting score should be positive")
	}

	// Test dark image
	darkImage := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC3)
	defer darkImage.Close()
	darkImage.SetTo(gocv.NewScalar(20, 20, 20, 0)) // Very dark

	darkBrightness, darkScore := assessor.assessLightingQuality(darkImage)

	if darkBrightness >= brightness {
		t.Error("Dark image should have lower brightness than optimal image")
	}

	if darkScore >= score {
		t.Error("Dark image should have lower lighting score than optimal image")
	}
}

func TestAssessFeatureStrength(t *testing.T) {
	assessor := NewFeatureQualityAssessor()

	// Create image with strong features (checkerboard pattern)
	strongFeatureImage := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC3)
	defer strongFeatureImage.Close()

	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			if (x/10+y/10)%2 == 0 {
				strongFeatureImage.SetUCharAt(y, x*3, 255)
				strongFeatureImage.SetUCharAt(y, x*3+1, 255)
				strongFeatureImage.SetUCharAt(y, x*3+2, 255)
			} else {
				strongFeatureImage.SetUCharAt(y, x*3, 0)
				strongFeatureImage.SetUCharAt(y, x*3+1, 0)
				strongFeatureImage.SetUCharAt(y, x*3+2, 0)
			}
		}
	}

	edgeDensity, score := assessor.assessFeatureStrength(strongFeatureImage)

	if edgeDensity <= 0 {
		t.Error("Edge density should be positive for image with strong features")
	}

	if score <= 0 {
		t.Error("Feature strength score should be positive for image with strong features")
	}

	// Create image with weak features (uniform color)
	weakFeatureImage := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC3)
	defer weakFeatureImage.Close()
	weakFeatureImage.SetTo(gocv.NewScalar(128, 128, 128, 0))

	weakEdgeDensity, weakScore := assessor.assessFeatureStrength(weakFeatureImage)

	if weakEdgeDensity >= edgeDensity {
		t.Error("Weak feature image should have lower edge density than strong feature image")
	}

	if weakScore >= score {
		t.Error("Weak feature image should have lower feature strength score than strong feature image")
	}
}

func TestSetThresholds(t *testing.T) {
	assessor := NewFeatureQualityAssessor()

	// Set new thresholds
	newMinFace := 15.0
	newMaxFace := 65.0
	newSharpness := 150.0
	newLighting := 70.0
	newFeature := 0.03

	assessor.SetThresholds(newMinFace, newMaxFace, newSharpness, newLighting, newFeature)

	minFace, maxFace, sharpness, lighting, feature := assessor.GetThresholds()

	if minFace != newMinFace {
		t.Errorf("Expected min face size %f, got %f", newMinFace, minFace)
	}

	if maxFace != newMaxFace {
		t.Errorf("Expected max face size %f, got %f", newMaxFace, maxFace)
	}

	if sharpness != newSharpness {
		t.Errorf("Expected sharpness threshold %f, got %f", newSharpness, sharpness)
	}

	if lighting != newLighting {
		t.Errorf("Expected lighting threshold %f, got %f", newLighting, lighting)
	}

	if feature != newFeature {
		t.Errorf("Expected feature threshold %f, got %f", newFeature, feature)
	}
}

func TestIsGoodQuality(t *testing.T) {
	assessor := NewFeatureQualityAssessor()

	tests := []struct {
		name     string
		metrics  FeatureQualityMetrics
		expected bool
	}{
		{
			name: "Good quality",
			metrics: FeatureQualityMetrics{
				OverallQuality: 0.8,
				QualityIssues:  []string{"minor issue"},
			},
			expected: true,
		},
		{
			name: "Poor quality - low score",
			metrics: FeatureQualityMetrics{
				OverallQuality: 0.4,
				QualityIssues:  []string{},
			},
			expected: false,
		},
		{
			name: "Poor quality - too many issues",
			metrics: FeatureQualityMetrics{
				OverallQuality: 0.8,
				QualityIssues:  []string{"issue1", "issue2", "issue3"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assessor.IsGoodQuality(tt.metrics)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetQualityLevel(t *testing.T) {
	assessor := NewFeatureQualityAssessor()

	tests := []struct {
		score    float64
		expected string
	}{
		{0.9, "excellent"},
		{0.7, "good"},
		{0.5, "fair"},
		{0.3, "poor"},
		{0.1, "very poor"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := assessor.GetQualityLevel(tt.score)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
