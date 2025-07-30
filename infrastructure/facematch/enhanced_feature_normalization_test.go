package facematch

import (
	"fmt"
	"math"
	"testing"

	"gocv.io/x/gocv"
)

func TestNewFeatureNormalizer(t *testing.T) {
	tests := []struct {
		name                    string
		method                  NormalizationMethod
		enableValidation        bool
		enableConsistencyChecks bool
	}{
		{
			name:                    "Standard normalization with validation",
			method:                  StandardNormalization,
			enableValidation:        true,
			enableConsistencyChecks: true,
		},
		{
			name:                    "Z-score normalization without validation",
			method:                  ZScoreNormalization,
			enableValidation:        false,
			enableConsistencyChecks: false,
		},
		{
			name:                    "Adaptive normalization",
			method:                  AdaptiveNormalization,
			enableValidation:        true,
			enableConsistencyChecks: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalizer := NewFeatureNormalizer(tt.method, tt.enableValidation, tt.enableConsistencyChecks)

			if normalizer == nil {
				t.Fatal("NewFeatureNormalizer returned nil")
			}

			if normalizer.GetNormalizationMethod() != tt.method {
				t.Errorf("Expected method %v, got %v", tt.method, normalizer.GetNormalizationMethod())
			}
		})
	}
}

func TestNormalizeFeatures_EmptyInput(t *testing.T) {
	normalizer := NewFeatureNormalizer(StandardNormalization, true, true)

	emptyFeatures := gocv.NewMat()
	result := normalizer.NormalizeFeatures(emptyFeatures)

	if result.Error == nil {
		t.Error("Expected error for empty input features")
	}
}

func TestNormalizeFeatures_ValidInput(t *testing.T) {
	normalizer := NewFeatureNormalizer(StandardNormalization, true, true)

	// Create a valid feature vector with enough non-zero features
	features := gocv.NewMatWithSize(1, 15, gocv.MatTypeCV32F)
	defer features.Close()

	// Set test values with enough non-zero features
	for i := 0; i < 15; i++ {
		if i < 12 { // 12 non-zero features out of 15
			features.SetFloatAt(0, i, float32(i+1))
		} else {
			features.SetFloatAt(0, i, 0.0)
		}
	}

	result := normalizer.NormalizeFeatures(features)
	defer func() {
		if !result.NormalizedFeatures.Empty() {
			result.NormalizedFeatures.Close()
		}
	}()

	if result.Error != nil {
		t.Fatalf("Unexpected error: %v", result.Error)
	}

	if result.NormalizedFeatures.Empty() {
		t.Error("Normalized features should not be empty")
	}

	// Check that the normalized vector has norm close to 1.0
	norm := gocv.Norm(result.NormalizedFeatures, gocv.NormL2)
	if math.Abs(norm-1.0) > NORM_TOLERANCE {
		t.Errorf("Expected normalized norm ~1.0, got %f", norm)
	}

	if !result.ValidationPassed {
		t.Error("Validation should have passed for valid input")
	}

	if result.ConsistencyScore < 0.5 {
		t.Errorf("Expected reasonable consistency score, got %f", result.ConsistencyScore)
	}
}

func TestNormalizeFeatures_ZeroNorm(t *testing.T) {
	normalizer := NewFeatureNormalizer(StandardNormalization, true, true)

	// Create a zero vector
	features := gocv.NewMatWithSize(1, 5, gocv.MatTypeCV32F)
	defer features.Close()
	features.SetTo(gocv.NewScalar(0, 0, 0, 0))

	result := normalizer.NormalizeFeatures(features)

	if result.Error == nil {
		t.Error("Expected error for zero norm vector")
	}

	if !result.NormalizedFeatures.Empty() {
		result.NormalizedFeatures.Close()
	}
}

func TestApplyImprovedL2Normalization(t *testing.T) {
	normalizer := NewFeatureNormalizer(StandardNormalization, false, false)

	// Create test feature vector [3, 4, 0, 5, 12] (norm = 13)
	features := gocv.NewMatWithSize(1, 5, gocv.MatTypeCV32F)
	defer features.Close()

	features.SetFloatAt(0, 0, 3.0)
	features.SetFloatAt(0, 1, 4.0)
	features.SetFloatAt(0, 2, 0.0)
	features.SetFloatAt(0, 3, 5.0)
	features.SetFloatAt(0, 4, 12.0)

	normalized, err := normalizer.applyImprovedL2Normalization(features)
	defer func() {
		if !normalized.Empty() {
			normalized.Close()
		}
	}()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check norm is 1.0
	norm := gocv.Norm(normalized, gocv.NormL2)
	if math.Abs(norm-1.0) > NORM_TOLERANCE {
		t.Errorf("Expected norm 1.0, got %f", norm)
	}

	// Check individual values are correctly normalized
	// Calculate the actual norm: sqrt(3^2 + 4^2 + 0^2 + 5^2 + 12^2) = sqrt(9 + 16 + 0 + 25 + 144) = sqrt(194)
	actualNorm := math.Sqrt(9 + 16 + 0 + 25 + 144)
	expectedValues := []float64{3.0 / actualNorm, 4.0 / actualNorm, 0.0, 5.0 / actualNorm, 12.0 / actualNorm}
	for i, expected := range expectedValues {
		actual := normalized.GetFloatAt(0, i)
		if math.Abs(float64(actual)-expected) > 1e-5 {
			t.Errorf("Expected value[%d] = %f, got %f", i, expected, actual)
		}
	}
}

func TestApplyZScoreNormalization(t *testing.T) {
	normalizer := NewFeatureNormalizer(ZScoreNormalization, false, false)

	// Create test feature vector with known mean and std
	features := gocv.NewMatWithSize(1, 5, gocv.MatTypeCV32F)
	defer features.Close()

	features.SetFloatAt(0, 0, 1.0)
	features.SetFloatAt(0, 1, 2.0)
	features.SetFloatAt(0, 2, 3.0)
	features.SetFloatAt(0, 3, 4.0)
	features.SetFloatAt(0, 4, 5.0)

	normalized, err := normalizer.applyZScoreNormalization(features)
	defer func() {
		if !normalized.Empty() {
			normalized.Close()
		}
	}()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that mean is approximately 0
	meanMat := gocv.NewMat()
	stddevMat := gocv.NewMat()
	defer meanMat.Close()
	defer stddevMat.Close()

	gocv.MeanStdDev(normalized, &meanMat, &stddevMat)
	mean := meanMat.GetDoubleAt(0, 0)
	stddev := stddevMat.GetDoubleAt(0, 0)

	if math.Abs(mean) > 1e-6 {
		t.Errorf("Expected mean ~0, got %f", mean)
	}

	if math.Abs(stddev-1.0) > 1e-6 {
		t.Errorf("Expected stddev ~1, got %f", stddev)
	}
}

func TestValidateInputFeatures(t *testing.T) {
	normalizer := NewFeatureNormalizer(StandardNormalization, true, true)

	tests := []struct {
		name          string
		setupFeatures func() gocv.Mat
		expectError   bool
	}{
		{
			name: "Valid features",
			setupFeatures: func() gocv.Mat {
				features := gocv.NewMatWithSize(1, 12, gocv.MatTypeCV32F)
				for i := 0; i < 12; i++ {
					features.SetFloatAt(0, i, float32(i+1))
				}
				return features
			},
			expectError: false,
		},
		{
			name: "Empty features",
			setupFeatures: func() gocv.Mat {
				return gocv.NewMat()
			},
			expectError: true,
		},
		{
			name: "Too many zero features",
			setupFeatures: func() gocv.Mat {
				features := gocv.NewMatWithSize(1, 10, gocv.MatTypeCV32F)
				features.SetTo(gocv.NewScalar(0, 0, 0, 0))
				features.SetFloatAt(0, 0, 1.0) // Only one non-zero
				return features
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := tt.setupFeatures()
			defer func() {
				if !features.Empty() {
					features.Close()
				}
			}()

			err := normalizer.validateInputFeatures(features)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestCalculateNorm(t *testing.T) {
	normalizer := NewFeatureNormalizer(StandardNormalization, false, false)

	// Test with known vector [3, 4] (norm = 5)
	features := gocv.NewMatWithSize(1, 2, gocv.MatTypeCV32F)
	defer features.Close()

	features.SetFloatAt(0, 0, 3.0)
	features.SetFloatAt(0, 1, 4.0)

	norm := normalizer.calculateNorm(features)
	expected := 5.0

	if math.Abs(norm-expected) > 1e-6 {
		t.Errorf("Expected norm %f, got %f", expected, norm)
	}
}

func TestCountZeroFeatures(t *testing.T) {
	normalizer := NewFeatureNormalizer(StandardNormalization, false, false)

	// Create vector with 2 zero features out of 5
	features := gocv.NewMatWithSize(1, 5, gocv.MatTypeCV32F)
	defer features.Close()

	features.SetFloatAt(0, 0, 1.0)
	features.SetFloatAt(0, 1, 0.0)
	features.SetFloatAt(0, 2, 2.0)
	features.SetFloatAt(0, 3, 0.0)
	features.SetFloatAt(0, 4, 3.0)

	zeroCount := normalizer.countZeroFeatures(features)
	expected := 2

	if zeroCount != expected {
		t.Errorf("Expected %d zero features, got %d", expected, zeroCount)
	}
}

func TestCompareNormalizedFeatures(t *testing.T) {
	normalizer := NewFeatureNormalizer(StandardNormalization, false, false)

	// Create two identical normalized vectors
	features1 := gocv.NewMatWithSize(1, 3, gocv.MatTypeCV32F)
	defer features1.Close()
	features1.SetFloatAt(0, 0, 0.6)
	features1.SetFloatAt(0, 1, 0.8)
	features1.SetFloatAt(0, 2, 0.0)

	features2 := gocv.NewMatWithSize(1, 3, gocv.MatTypeCV32F)
	defer features2.Close()
	features2.SetFloatAt(0, 0, 0.6)
	features2.SetFloatAt(0, 1, 0.8)
	features2.SetFloatAt(0, 2, 0.0)

	similarity, err := normalizer.CompareNormalizedFeatures(features1, features2)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if math.Abs(similarity-1.0) > 1e-6 {
		t.Errorf("Expected similarity 1.0 for identical vectors, got %f", similarity)
	}

	// Test with orthogonal vectors
	features3 := gocv.NewMatWithSize(1, 3, gocv.MatTypeCV32F)
	defer features3.Close()
	features3.SetFloatAt(0, 0, 0.0)
	features3.SetFloatAt(0, 1, 0.0)
	features3.SetFloatAt(0, 2, 1.0)

	similarity2, err := normalizer.CompareNormalizedFeatures(features1, features3)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if math.Abs(similarity2) > 1e-6 {
		t.Errorf("Expected similarity ~0.0 for orthogonal vectors, got %f", similarity2)
	}
}

func TestNormalizationMethods(t *testing.T) {
	methods := []NormalizationMethod{
		StandardNormalization,
		ZScoreNormalization,
		AdaptiveNormalization,
	}

	// Create test feature vector
	features := gocv.NewMatWithSize(1, 5, gocv.MatTypeCV32F)
	defer features.Close()
	features.SetFloatAt(0, 0, 1.0)
	features.SetFloatAt(0, 1, 2.0)
	features.SetFloatAt(0, 2, 3.0)
	features.SetFloatAt(0, 3, 4.0)
	features.SetFloatAt(0, 4, 5.0)

	for _, method := range methods {
		t.Run(fmt.Sprintf("Method_%d", method), func(t *testing.T) {
			normalizer := NewFeatureNormalizer(method, false, false)
			result := normalizer.NormalizeFeatures(features)

			defer func() {
				if !result.NormalizedFeatures.Empty() {
					result.NormalizedFeatures.Close()
				}
			}()

			if result.Error != nil {
				t.Errorf("Method %d failed: %v", method, result.Error)
			}

			if result.NormalizedFeatures.Empty() {
				t.Errorf("Method %d produced empty result", method)
			}
		})
	}
}

func TestSettersAndGetters(t *testing.T) {
	normalizer := NewFeatureNormalizer(StandardNormalization, true, true)

	// Test method setter/getter
	newMethod := ZScoreNormalization
	normalizer.SetNormalizationMethod(newMethod)
	if normalizer.GetNormalizationMethod() != newMethod {
		t.Error("SetNormalizationMethod/GetNormalizationMethod failed")
	}

	// Test validation setter
	normalizer.SetValidationEnabled(false)
	// We can't directly test this, but we can test that it doesn't crash

	// Test consistency checks setter
	normalizer.SetConsistencyChecksEnabled(false)
	// We can't directly test this, but we can test that it doesn't crash
}

func TestIsConsistentNormalization(t *testing.T) {
	normalizer := NewFeatureNormalizer(StandardNormalization, true, true)

	tests := []struct {
		name     string
		result   NormalizationResult
		expected bool
	}{
		{
			name: "Consistent normalization",
			result: NormalizationResult{
				ValidationPassed: true,
				ConsistencyScore: 0.9,
				NormalizationInfo: NormalizationInfo{
					NormalizationOk: true,
				},
			},
			expected: true,
		},
		{
			name: "Inconsistent - validation failed",
			result: NormalizationResult{
				ValidationPassed: false,
				ConsistencyScore: 0.9,
				NormalizationInfo: NormalizationInfo{
					NormalizationOk: true,
				},
			},
			expected: false,
		},
		{
			name: "Inconsistent - low consistency score",
			result: NormalizationResult{
				ValidationPassed: true,
				ConsistencyScore: 0.5,
				NormalizationInfo: NormalizationInfo{
					NormalizationOk: true,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.IsConsistentNormalization(tt.result)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Benchmark tests
func BenchmarkNormalizeFeatures(b *testing.B) {
	normalizer := NewFeatureNormalizer(StandardNormalization, false, false)
	features := gocv.NewMatWithSize(1, 512, gocv.MatTypeCV32F) // Typical ArcFace feature size
	defer features.Close()

	// Fill with random-like values
	for i := 0; i < 512; i++ {
		features.SetFloatAt(0, i, float32(i%100)/100.0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := normalizer.NormalizeFeatures(features)
		if !result.NormalizedFeatures.Empty() {
			result.NormalizedFeatures.Close()
		}
	}
}
