package facematch

import (
	"errors"
	"fmt"
	"math"

	"gocv.io/x/gocv"
)

// FeatureNormalizer handles enhanced feature vector normalization with validation and consistency checks
type FeatureNormalizer struct {
	normalizationMethod NormalizationMethod
	validationEnabled   bool
	consistencyChecks   bool
}

// NormalizationResult contains the result of feature normalization
type NormalizationResult struct {
	NormalizedFeatures gocv.Mat          `json:"-"`
	ValidationPassed   bool              `json:"validation_passed"`
	ConsistencyScore   float64           `json:"consistency_score"`
	NormalizationInfo  NormalizationInfo `json:"normalization_info"`
	Error              error             `json:"error,omitempty"`
}

// NormalizationInfo contains information about the normalization process
type NormalizationInfo struct {
	Method          string  `json:"method"`
	OriginalNorm    float64 `json:"original_norm"`
	NormalizedNorm  float64 `json:"normalized_norm"`
	FeatureCount    int     `json:"feature_count"`
	ZeroFeatures    int     `json:"zero_features"`
	NormalizationOk bool    `json:"normalization_ok"`
}

// Feature validation constants
const (
	MIN_FEATURE_NORM         = 1e-8 // Minimum norm to avoid division by zero
	MAX_FEATURE_NORM         = 1e6  // Maximum norm to detect outliers
	EXPECTED_NORMALIZED_NORM = 1.0  // Expected norm after L2 normalization
	NORM_TOLERANCE           = 1e-6 // Tolerance for norm validation
	MIN_NON_ZERO_FEATURES    = 10   // Minimum number of non-zero features
	MAX_ZERO_FEATURE_RATIO   = 0.9  // Maximum ratio of zero features
	CONSISTENCY_THRESHOLD    = 0.8  // Minimum consistency score
)

// NewFeatureNormalizer creates a new feature normalizer with specified method
func NewFeatureNormalizer(method NormalizationMethod, enableValidation, enableConsistencyChecks bool) *FeatureNormalizer {
	return &FeatureNormalizer{
		normalizationMethod: method,
		validationEnabled:   enableValidation,
		consistencyChecks:   enableConsistencyChecks,
	}
}

// NormalizeFeatures performs enhanced L2 normalization with validation and consistency checks
func (fn *FeatureNormalizer) NormalizeFeatures(features gocv.Mat) NormalizationResult {
	if features.Empty() {
		return NormalizationResult{
			NormalizedFeatures: gocv.NewMat(), // Initialize empty Mat
			Error:              errors.New("input features are empty"),
		}
	}

	// Validate input features
	if fn.validationEnabled {
		if err := fn.validateInputFeatures(features); err != nil {
			return NormalizationResult{
				NormalizedFeatures: gocv.NewMat(), // Initialize empty Mat
				Error:              fmt.Errorf("input validation failed: %v", err),
			}
		}
	}

	// Calculate original norm and feature statistics
	originalNorm := fn.calculateNorm(features)
	featureCount := features.Total()
	zeroFeatures := fn.countZeroFeatures(features)

	info := NormalizationInfo{
		Method:       fn.getMethodName(),
		OriginalNorm: originalNorm,
		FeatureCount: int(featureCount),
		ZeroFeatures: zeroFeatures,
	}

	// Check if normalization is possible
	if originalNorm < MIN_FEATURE_NORM {
		return NormalizationResult{
			NormalizedFeatures: gocv.NewMat(), // Initialize empty Mat
			ValidationPassed:   false,
			NormalizationInfo:  info,
			Error:              errors.New("feature vector has zero or near-zero norm, cannot normalize"),
		}
	}

	// Perform normalization based on method
	var normalized gocv.Mat
	var err error

	switch fn.normalizationMethod {
	case StandardNormalization:
		normalized, err = fn.applyStandardL2Normalization(features)
	case ZScoreNormalization:
		normalized, err = fn.applyZScoreNormalization(features)
	case AdaptiveNormalization:
		normalized, err = fn.applyAdaptiveNormalization(features)
	default:
		normalized, err = fn.applyImprovedL2Normalization(features)
	}

	if err != nil {
		return NormalizationResult{
			NormalizedFeatures: gocv.NewMat(), // Initialize empty Mat
			ValidationPassed:   false,
			NormalizationInfo:  info,
			Error:              fmt.Errorf("normalization failed: %v", err),
		}
	}

	// Calculate normalized norm
	normalizedNorm := fn.calculateNorm(normalized)
	info.NormalizedNorm = normalizedNorm
	info.NormalizationOk = math.Abs(normalizedNorm-EXPECTED_NORMALIZED_NORM) < NORM_TOLERANCE

	// Validate normalized features
	validationPassed := true
	if fn.validationEnabled {
		if err := fn.validateNormalizedFeatures(normalized, normalizedNorm); err != nil {
			validationPassed = false
		}
	}

	// Calculate consistency score
	consistencyScore := 1.0
	if fn.consistencyChecks {
		consistencyScore = fn.calculateConsistencyScore(features, normalized, info)
	}

	return NormalizationResult{
		NormalizedFeatures: normalized,
		ValidationPassed:   validationPassed,
		ConsistencyScore:   consistencyScore,
		NormalizationInfo:  info,
	}
}

// applyImprovedL2Normalization applies enhanced L2 normalization with stability improvements
func (fn *FeatureNormalizer) applyImprovedL2Normalization(features gocv.Mat) (gocv.Mat, error) {
	// Convert to float64 for higher precision
	floatFeatures := gocv.NewMat()
	defer floatFeatures.Close()
	features.ConvertTo(&floatFeatures, gocv.MatTypeCV64F)

	// Calculate L2 norm with numerical stability
	norm := fn.calculateNorm(floatFeatures)

	if norm < MIN_FEATURE_NORM {
		return gocv.Mat{}, errors.New("feature vector norm too small for stable normalization")
	}

	// Normalize by dividing by the norm
	normalized := gocv.NewMat()
	floatFeatures.DivideFloat(float32(norm))
	floatFeatures.CopyTo(&normalized)

	// Convert back to original type if needed
	if features.Type() != gocv.MatTypeCV64F {
		result := gocv.NewMat()
		normalized.ConvertTo(&result, features.Type())
		normalized.Close()
		return result, nil
	}

	return normalized, nil
}

// applyStandardL2Normalization applies standard L2 normalization
func (fn *FeatureNormalizer) applyStandardL2Normalization(features gocv.Mat) (gocv.Mat, error) {
	normalized := gocv.NewMat()

	// Use OpenCV's normalize function
	gocv.Normalize(features, &normalized, 1.0, 0.0, gocv.NormL2)

	return normalized, nil
}

// applyZScoreNormalization applies Z-score normalization (mean=0, std=1)
func (fn *FeatureNormalizer) applyZScoreNormalization(features gocv.Mat) (gocv.Mat, error) {
	// Convert to float for calculations
	floatFeatures := gocv.NewMat()
	defer floatFeatures.Close()
	features.ConvertTo(&floatFeatures, gocv.MatTypeCV32F)

	// Calculate mean and standard deviation
	meanMat := gocv.NewMat()
	stddevMat := gocv.NewMat()
	defer meanMat.Close()
	defer stddevMat.Close()

	gocv.MeanStdDev(floatFeatures, &meanMat, &stddevMat)

	mean := meanMat.GetDoubleAt(0, 0)
	stddev := stddevMat.GetDoubleAt(0, 0)

	if stddev < MIN_FEATURE_NORM {
		return gocv.Mat{}, errors.New("feature vector has zero variance, cannot apply Z-score normalization")
	}

	// Apply Z-score normalization: (x - mean) / stddev
	normalized := gocv.NewMat()
	floatFeatures.SubtractFloat(float32(mean))
	floatFeatures.DivideFloat(float32(stddev))
	floatFeatures.CopyTo(&normalized)

	return normalized, nil
}

// applyAdaptiveNormalization applies adaptive normalization based on feature distribution
func (fn *FeatureNormalizer) applyAdaptiveNormalization(features gocv.Mat) (gocv.Mat, error) {
	// Convert to float for calculations
	floatFeatures := gocv.NewMat()
	defer floatFeatures.Close()
	features.ConvertTo(&floatFeatures, gocv.MatTypeCV32F)

	// Calculate statistics
	meanMat := gocv.NewMat()
	stddevMat := gocv.NewMat()
	defer meanMat.Close()
	defer stddevMat.Close()

	gocv.MeanStdDev(floatFeatures, &meanMat, &stddevMat)

	mean := meanMat.GetDoubleAt(0, 0)
	stddev := stddevMat.GetDoubleAt(0, 0)

	// Choose normalization method based on distribution characteristics
	if stddev > mean*0.5 {
		// High variance: use Z-score normalization
		return fn.applyZScoreNormalization(features)
	} else {
		// Low variance: use L2 normalization
		return fn.applyImprovedL2Normalization(features)
	}
}

// validateInputFeatures validates input feature vector
func (fn *FeatureNormalizer) validateInputFeatures(features gocv.Mat) error {
	if features.Empty() {
		return errors.New("feature vector is empty")
	}

	if features.Rows() != 1 && features.Cols() != 1 {
		return errors.New("feature vector must be 1-dimensional")
	}

	// Check for invalid values (NaN, Inf)
	if err := fn.checkForInvalidValues(features); err != nil {
		return err
	}

	// Check norm bounds
	norm := fn.calculateNorm(features)
	if norm < MIN_FEATURE_NORM {
		return errors.New("feature vector norm is too small")
	}
	if norm > MAX_FEATURE_NORM {
		return errors.New("feature vector norm is too large (possible outlier)")
	}

	// Check zero feature ratio
	zeroCount := fn.countZeroFeatures(features)
	totalFeatures := features.Total()
	zeroRatio := float64(zeroCount) / float64(totalFeatures)

	if zeroRatio > MAX_ZERO_FEATURE_RATIO {
		return fmt.Errorf("too many zero features (%.2f%% > %.2f%%)", zeroRatio*100, MAX_ZERO_FEATURE_RATIO*100)
	}

	nonZeroFeatures := int(totalFeatures) - zeroCount
	if nonZeroFeatures < MIN_NON_ZERO_FEATURES {
		return fmt.Errorf("insufficient non-zero features (%d < %d)", nonZeroFeatures, MIN_NON_ZERO_FEATURES)
	}

	return nil
}

// validateNormalizedFeatures validates the normalized feature vector
func (fn *FeatureNormalizer) validateNormalizedFeatures(normalized gocv.Mat, norm float64) error {
	if normalized.Empty() {
		return errors.New("normalized feature vector is empty")
	}

	// Check for invalid values
	if err := fn.checkForInvalidValues(normalized); err != nil {
		return fmt.Errorf("normalized features contain invalid values: %v", err)
	}

	// Check if norm is close to expected value (1.0 for L2 normalization)
	if fn.normalizationMethod == StandardNormalization || fn.normalizationMethod == AdaptiveNormalization {
		if math.Abs(norm-EXPECTED_NORMALIZED_NORM) > NORM_TOLERANCE {
			return fmt.Errorf("normalized vector norm (%.6f) is not close to expected value (%.6f)", norm, EXPECTED_NORMALIZED_NORM)
		}
	}

	return nil
}

// checkForInvalidValues checks for NaN and Inf values in the feature vector
func (fn *FeatureNormalizer) checkForInvalidValues(features gocv.Mat) error {
	// Convert to float64 for checking
	floatFeatures := gocv.NewMat()
	defer floatFeatures.Close()
	features.ConvertTo(&floatFeatures, gocv.MatTypeCV64F)

	rows := floatFeatures.Rows()
	cols := floatFeatures.Cols()

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val := floatFeatures.GetDoubleAt(i, j)
			if math.IsNaN(val) {
				return fmt.Errorf("NaN value found at position (%d, %d)", i, j)
			}
			if math.IsInf(val, 0) {
				return fmt.Errorf("Inf value found at position (%d, %d)", i, j)
			}
		}
	}

	return nil
}

// calculateNorm calculates the L2 norm of a feature vector
func (fn *FeatureNormalizer) calculateNorm(features gocv.Mat) float64 {
	if features.Empty() {
		return 0.0
	}

	// Use OpenCV's norm function for accuracy
	return gocv.Norm(features, gocv.NormL2)
}

// countZeroFeatures counts the number of zero or near-zero features
func (fn *FeatureNormalizer) countZeroFeatures(features gocv.Mat) int {
	if features.Empty() {
		return 0
	}

	// Convert to float64 for precision
	floatFeatures := gocv.NewMat()
	defer floatFeatures.Close()
	features.ConvertTo(&floatFeatures, gocv.MatTypeCV64F)

	zeroCount := 0
	rows := floatFeatures.Rows()
	cols := floatFeatures.Cols()

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val := floatFeatures.GetDoubleAt(i, j)
			if math.Abs(val) < MIN_FEATURE_NORM {
				zeroCount++
			}
		}
	}

	return zeroCount
}

// calculateConsistencyScore calculates a consistency score for the normalization
func (fn *FeatureNormalizer) calculateConsistencyScore(original, normalized gocv.Mat, info NormalizationInfo) float64 {
	score := 1.0

	// Penalize if normalization didn't achieve expected norm
	if !info.NormalizationOk {
		score -= 0.3
	}

	// Penalize high zero feature ratio
	zeroRatio := float64(info.ZeroFeatures) / float64(info.FeatureCount)
	if zeroRatio > 0.5 {
		score -= (zeroRatio - 0.5) * 0.4
	}

	// Penalize if original norm was too extreme
	if info.OriginalNorm < MIN_FEATURE_NORM*10 {
		score -= 0.2
	}
	if info.OriginalNorm > MAX_FEATURE_NORM/10 {
		score -= 0.2
	}

	// Ensure score is in valid range
	return math.Max(0.0, math.Min(1.0, score))
}

// getMethodName returns the string representation of the normalization method
func (fn *FeatureNormalizer) getMethodName() string {
	switch fn.normalizationMethod {
	case StandardNormalization:
		return "standard_l2"
	case ZScoreNormalization:
		return "z_score"
	case AdaptiveNormalization:
		return "adaptive"
	case HistogramEqualization:
		return "histogram_equalization"
	default:
		return "improved_l2"
	}
}

// CompareNormalizedFeatures compares two normalized feature vectors for consistency
func (fn *FeatureNormalizer) CompareNormalizedFeatures(features1, features2 gocv.Mat) (float64, error) {
	if features1.Empty() || features2.Empty() {
		return 0.0, errors.New("one or both feature vectors are empty")
	}

	if features1.Total() != features2.Total() {
		return 0.0, errors.New("feature vectors have different dimensions")
	}

	// Calculate cosine similarity between normalized features
	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	// Convert to float64 for precision
	float1 := gocv.NewMat()
	float2 := gocv.NewMat()
	defer float1.Close()
	defer float2.Close()

	features1.ConvertTo(&float1, gocv.MatTypeCV64F)
	features2.ConvertTo(&float2, gocv.MatTypeCV64F)

	rows := float1.Rows()
	cols := float1.Cols()

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val1 := float1.GetDoubleAt(i, j)
			val2 := float2.GetDoubleAt(i, j)

			dotProduct += val1 * val2
			norm1 += val1 * val1
			norm2 += val2 * val2
		}
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0, errors.New("one or both feature vectors have zero norm")
	}

	similarity := dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
	return similarity, nil
}

// SetValidationEnabled enables or disables input validation
func (fn *FeatureNormalizer) SetValidationEnabled(enabled bool) {
	fn.validationEnabled = enabled
}

// SetConsistencyChecksEnabled enables or disables consistency checks
func (fn *FeatureNormalizer) SetConsistencyChecksEnabled(enabled bool) {
	fn.consistencyChecks = enabled
}

// GetNormalizationMethod returns the current normalization method
func (fn *FeatureNormalizer) GetNormalizationMethod() NormalizationMethod {
	return fn.normalizationMethod
}

// SetNormalizationMethod updates the normalization method
func (fn *FeatureNormalizer) SetNormalizationMethod(method NormalizationMethod) {
	fn.normalizationMethod = method
}

// IsConsistentNormalization checks if the normalization result is consistent
func (fn *FeatureNormalizer) IsConsistentNormalization(result NormalizationResult) bool {
	return result.ValidationPassed &&
		result.ConsistencyScore >= CONSISTENCY_THRESHOLD &&
		result.NormalizationInfo.NormalizationOk
}
