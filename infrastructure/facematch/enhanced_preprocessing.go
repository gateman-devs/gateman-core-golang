package facematch

import (
	"errors"
	"fmt"
	"image"
	"math"

	"gocv.io/x/gocv"
)

// ImagePreprocessor handles consistent face preprocessing for enhanced feature extraction
type ImagePreprocessor struct {
	targetSize      image.Point
	normalizeMethod NormalizationMethod
}

// NormalizationMethod defines different normalization approaches
type NormalizationMethod int

const (
	StandardNormalization NormalizationMethod = iota
	HistogramEqualization
	AdaptiveNormalization
	ZScoreNormalization
)

// PreprocessingResult contains the result of face preprocessing
type PreprocessingResult struct {
	ProcessedFace gocv.Mat
	Quality       PreprocessingQuality
	Error         error
}

// PreprocessingQuality contains quality metrics from preprocessing
type PreprocessingQuality struct {
	OriginalSize    image.Point `json:"original_size"`
	ProcessedSize   image.Point `json:"processed_size"`
	ScaleFactor     float64     `json:"scale_factor"`
	AlignmentScore  float64     `json:"alignment_score"`
	NormalizationOk bool        `json:"normalization_ok"`
	PreprocessingOk bool        `json:"preprocessing_ok"`
}

// NewImagePreprocessor creates a new image preprocessor with specified target size
func NewImagePreprocessor(targetSize image.Point, method NormalizationMethod) *ImagePreprocessor {
	if targetSize.X <= 0 || targetSize.Y <= 0 {
		targetSize = image.Pt(112, 112) // Default ArcFace input size
	}

	return &ImagePreprocessor{
		targetSize:      targetSize,
		normalizeMethod: method,
	}
}

// PreprocessFace performs comprehensive face preprocessing with quality assessment
func (ip *ImagePreprocessor) PreprocessFace(face gocv.Mat) PreprocessingResult {
	if face.Empty() {
		return PreprocessingResult{
			Error: errors.New("input face image is empty"),
		}
	}

	originalSize := face.Size()
	if len(originalSize) < 2 {
		return PreprocessingResult{
			Error: errors.New("invalid face image dimensions"),
		}
	}

	quality := PreprocessingQuality{
		OriginalSize:    image.Pt(originalSize[1], originalSize[0]), // OpenCV uses height, width
		ProcessedSize:   ip.targetSize,
		PreprocessingOk: true,
	}

	// Step 1: Face alignment (basic geometric normalization)
	alignedFace, alignmentScore, err := ip.alignFace(face)
	if err != nil {
		return PreprocessingResult{
			Quality: quality,
			Error:   fmt.Errorf("face alignment failed: %v", err),
		}
	}
	defer alignedFace.Close()

	quality.AlignmentScore = alignmentScore

	// Step 2: Resize to target size with high-quality interpolation
	resized := gocv.NewMat()
	defer resized.Close()

	// Calculate scale factor
	origWidth, origHeight := originalSize[1], originalSize[0]
	quality.ScaleFactor = math.Min(
		float64(ip.targetSize.X)/float64(origWidth),
		float64(ip.targetSize.Y)/float64(origHeight),
	)

	gocv.Resize(alignedFace, &resized, ip.targetSize, 0, 0, gocv.InterpolationCubic)

	// Step 3: Apply normalization method
	normalized, normOk, err := ip.applyNormalization(resized)
	if err != nil {
		return PreprocessingResult{
			Quality: quality,
			Error:   fmt.Errorf("normalization failed: %v", err),
		}
	}

	quality.NormalizationOk = normOk

	return PreprocessingResult{
		ProcessedFace: normalized,
		Quality:       quality,
	}
}

// alignFace performs basic face alignment using geometric transformations
func (ip *ImagePreprocessor) alignFace(face gocv.Mat) (gocv.Mat, float64, error) {
	// For now, implement basic centering and rotation correction
	// In a full implementation, this would use facial landmarks for precise alignment

	// Convert to grayscale for analysis
	gray := gocv.NewMat()
	defer gray.Close()

	if face.Channels() == 3 {
		gocv.CvtColor(face, &gray, gocv.ColorBGRToGray)
	} else {
		face.CopyTo(&gray)
	}

	// Basic alignment score based on face symmetry
	alignmentScore := ip.calculateAlignmentScore(gray)

	// For basic implementation, return the original face
	// In production, this would apply rotation and translation corrections
	aligned := face.Clone()

	return aligned, alignmentScore, nil
}

// calculateAlignmentScore calculates a basic alignment score based on face symmetry
func (ip *ImagePreprocessor) calculateAlignmentScore(grayFace gocv.Mat) float64 {
	if grayFace.Empty() {
		return 0.0
	}

	size := grayFace.Size()
	if len(size) < 2 {
		return 0.0
	}

	height, width := size[0], size[1]
	centerX := width / 2

	// Calculate left-right symmetry by comparing pixel intensities
	var symmetryScore float64
	var totalPixels int

	for y := 0; y < height; y++ {
		for x := 0; x < centerX; x++ {
			leftPixel := grayFace.GetUCharAt(y, x)
			rightPixel := grayFace.GetUCharAt(y, width-1-x)

			// Calculate absolute difference
			diff := math.Abs(float64(leftPixel) - float64(rightPixel))
			symmetryScore += (255.0 - diff) / 255.0 // Normalize to 0-1
			totalPixels++
		}
	}

	if totalPixels == 0 {
		return 0.0
	}

	return symmetryScore / float64(totalPixels)
}

// applyNormalization applies the specified normalization method
func (ip *ImagePreprocessor) applyNormalization(face gocv.Mat) (gocv.Mat, bool, error) {
	if face.Empty() {
		return gocv.Mat{}, false, errors.New("input face is empty")
	}

	switch ip.normalizeMethod {
	case StandardNormalization:
		return ip.applyStandardNormalization(face)
	case HistogramEqualization:
		return ip.applyHistogramEqualization(face)
	case AdaptiveNormalization:
		return ip.applyAdaptiveNormalization(face)
	case ZScoreNormalization:
		return ip.applyZScoreNormalization(face)
	default:
		return ip.applyStandardNormalization(face)
	}
}

// applyStandardNormalization applies standard [0,1] normalization
func (ip *ImagePreprocessor) applyStandardNormalization(face gocv.Mat) (gocv.Mat, bool, error) {
	normalized := gocv.NewMat()
	face.ConvertTo(&normalized, gocv.MatTypeCV32F)
	normalized.DivideFloat(255.0)

	return normalized, true, nil
}

// applyHistogramEqualization applies histogram equalization for better contrast
func (ip *ImagePreprocessor) applyHistogramEqualization(face gocv.Mat) (gocv.Mat, bool, error) {
	// Convert to grayscale if needed
	var gray gocv.Mat
	if face.Channels() == 3 {
		gray = gocv.NewMat()
		gocv.CvtColor(face, &gray, gocv.ColorBGRToGray)
		defer gray.Close()
	} else {
		gray = face.Clone()
		defer gray.Close()
	}

	// Apply histogram equalization
	equalized := gocv.NewMat()
	defer equalized.Close()
	gocv.EqualizeHist(gray, &equalized)

	// Convert back to color if original was color
	var result gocv.Mat
	if face.Channels() == 3 {
		result = gocv.NewMat()
		gocv.CvtColor(equalized, &result, gocv.ColorGrayToBGR)
	} else {
		result = equalized.Clone()
	}

	// Normalize to [0,1]
	normalized := gocv.NewMat()
	result.ConvertTo(&normalized, gocv.MatTypeCV32F)
	normalized.DivideFloat(255.0)

	if !result.Empty() {
		result.Close()
	}

	return normalized, true, nil
}

// applyAdaptiveNormalization applies adaptive histogram equalization (CLAHE)
func (ip *ImagePreprocessor) applyAdaptiveNormalization(face gocv.Mat) (gocv.Mat, bool, error) {
	// Convert to LAB color space for better adaptive processing
	lab := gocv.NewMat()
	defer lab.Close()

	if face.Channels() == 3 {
		gocv.CvtColor(face, &lab, gocv.ColorBGRToLab)
	} else {
		// For grayscale, convert to BGR first then to LAB
		bgr := gocv.NewMat()
		defer bgr.Close()
		gocv.CvtColor(face, &bgr, gocv.ColorGrayToBGR)
		gocv.CvtColor(bgr, &lab, gocv.ColorBGRToLab)
	}

	// Split LAB channels
	channels := gocv.Split(lab)
	defer func() {
		for _, ch := range channels {
			ch.Close()
		}
	}()

	if len(channels) < 3 {
		return gocv.Mat{}, false, errors.New("failed to split LAB channels")
	}

	// Apply CLAHE to L channel
	clahe := gocv.NewCLAHEWithParams(2.0, image.Pt(8, 8))
	defer clahe.Close()

	enhanced := gocv.NewMat()
	defer enhanced.Close()
	clahe.Apply(channels[0], &enhanced)

	// Replace L channel with enhanced version
	channels[0].Close()
	channels[0] = enhanced.Clone()

	// Merge channels back
	merged := gocv.NewMat()
	defer merged.Close()
	gocv.Merge(channels, &merged)

	// Convert back to BGR
	result := gocv.NewMat()
	defer result.Close()
	gocv.CvtColor(merged, &result, gocv.ColorLabToBGR)

	// Normalize to [0,1]
	normalized := gocv.NewMat()
	result.ConvertTo(&normalized, gocv.MatTypeCV32F)
	normalized.DivideFloat(255.0)

	return normalized, true, nil
}

// applyZScoreNormalization applies Z-score normalization (mean=0, std=1)
func (ip *ImagePreprocessor) applyZScoreNormalization(face gocv.Mat) (gocv.Mat, bool, error) {
	// Convert to float32
	floatFace := gocv.NewMat()
	defer floatFace.Close()
	face.ConvertTo(&floatFace, gocv.MatTypeCV32F)

	// Calculate mean and standard deviation
	meanMat := gocv.NewMat()
	stddevMat := gocv.NewMat()
	defer meanMat.Close()
	defer stddevMat.Close()

	gocv.MeanStdDev(floatFace, &meanMat, &stddevMat)

	// Get scalar values
	mean := meanMat.GetDoubleAt(0, 0)
	stddev := stddevMat.GetDoubleAt(0, 0)

	if stddev == 0 {
		return gocv.Mat{}, false, errors.New("standard deviation is zero, cannot apply Z-score normalization")
	}

	// Apply Z-score normalization: (x - mean) / stddev
	normalized := gocv.NewMat()
	floatFace.SubtractFloat(float32(mean))
	floatFace.DivideFloat(float32(stddev))
	floatFace.CopyTo(&normalized)

	return normalized, true, nil
}

// StandardizeImage applies consistent preprocessing for face comparison
func (ip *ImagePreprocessor) StandardizeImage(face gocv.Mat) (gocv.Mat, error) {
	result := ip.PreprocessFace(face)
	if result.Error != nil {
		return gocv.Mat{}, result.Error
	}

	return result.ProcessedFace, nil
}

// GetTargetSize returns the target size for preprocessing
func (ip *ImagePreprocessor) GetTargetSize() image.Point {
	return ip.targetSize
}

// SetTargetSize updates the target size for preprocessing
func (ip *ImagePreprocessor) SetTargetSize(size image.Point) {
	if size.X > 0 && size.Y > 0 {
		ip.targetSize = size
	}
}

// GetNormalizationMethod returns the current normalization method
func (ip *ImagePreprocessor) GetNormalizationMethod() NormalizationMethod {
	return ip.normalizeMethod
}

// SetNormalizationMethod updates the normalization method
func (ip *ImagePreprocessor) SetNormalizationMethod(method NormalizationMethod) {
	ip.normalizeMethod = method
}
