package facematch

import (
	"errors"
	"fmt"
	"image"
	"math"

	"gocv.io/x/gocv"
)

// FeatureQualityMetrics contains comprehensive quality assessment metrics for face features
type FeatureQualityMetrics struct {
	FaceSize        float64  `json:"face_size_percent"`        // Face size as percentage of image
	FacePosition    string   `json:"face_position"`            // Face position description (centered, off-center, etc.)
	ImageSharpness  float64  `json:"image_sharpness"`          // Sharpness score (0-1, higher is sharper)
	LightingQuality float64  `json:"lighting_quality"`         // Lighting quality score (0-1, higher is better)
	FeatureStrength float64  `json:"feature_strength"`         // Feature distinctiveness score (0-1, higher is better)
	OverallQuality  float64  `json:"overall_quality"`          // Combined quality score (0-1, higher is better)
	QualityIssues   []string `json:"quality_issues,omitempty"` // List of detected quality issues
}

// QualityAssessmentResult contains the result of quality assessment
type QualityAssessmentResult struct {
	Metrics FeatureQualityMetrics `json:"metrics"`
	Error   error                 `json:"error,omitempty"`
}

// FeatureQualityAssessor handles quality assessment of face features
type FeatureQualityAssessor struct {
	minFaceSize     float64 // Minimum acceptable face size percentage
	maxFaceSize     float64 // Maximum acceptable face size percentage
	sharpnessThresh float64 // Minimum sharpness threshold
	lightingThresh  float64 // Minimum lighting quality threshold
	featureThresh   float64 // Minimum feature strength threshold
}

// Quality thresholds and constants
const (
	// Face size thresholds (percentage of image)
	MIN_FACE_SIZE_PERCENT_QUALITY = 10.0 // Minimum face size for good quality
	MAX_FACE_SIZE_PERCENT_QUALITY = 70.0 // Maximum face size for good quality
	OPTIMAL_FACE_SIZE_MIN         = 20.0 // Optimal minimum face size
	OPTIMAL_FACE_SIZE_MAX         = 50.0 // Optimal maximum face size

	// Position thresholds
	MAX_CENTER_OFFSET_PERCENT = 25.0 // Maximum offset from center for good positioning

	// Sharpness thresholds
	MIN_SHARPNESS_THRESHOLD  = 100.0 // Minimum Laplacian variance for sharpness
	GOOD_SHARPNESS_THRESHOLD = 200.0 // Good sharpness threshold

	// Lighting thresholds
	MIN_BRIGHTNESS_QUALITY = 60.0  // Minimum brightness for good quality
	MAX_BRIGHTNESS_QUALITY = 200.0 // Maximum brightness for good quality
	OPTIMAL_BRIGHTNESS_MIN = 80.0  // Optimal minimum brightness
	OPTIMAL_BRIGHTNESS_MAX = 180.0 // Optimal maximum brightness

	// Feature strength thresholds
	MIN_EDGE_DENSITY_QUALITY = 0.02 // Minimum edge density for feature strength
	GOOD_EDGE_DENSITY        = 0.05 // Good edge density threshold

	// Overall quality weights
	FACE_SIZE_WEIGHT = 0.2  // Weight for face size in overall quality
	POSITION_WEIGHT  = 0.15 // Weight for face position in overall quality
	SHARPNESS_WEIGHT = 0.25 // Weight for sharpness in overall quality
	LIGHTING_WEIGHT  = 0.2  // Weight for lighting in overall quality
	FEATURE_WEIGHT   = 0.2  // Weight for feature strength in overall quality
)

// NewFeatureQualityAssessor creates a new feature quality assessor with default thresholds
func NewFeatureQualityAssessor() *FeatureQualityAssessor {
	return &FeatureQualityAssessor{
		minFaceSize:     MIN_FACE_SIZE_PERCENT_QUALITY,
		maxFaceSize:     MAX_FACE_SIZE_PERCENT_QUALITY,
		sharpnessThresh: MIN_SHARPNESS_THRESHOLD,
		lightingThresh:  MIN_BRIGHTNESS_QUALITY,
		featureThresh:   MIN_EDGE_DENSITY_QUALITY,
	}
}

// AssessQuality performs comprehensive quality assessment on a face image
func (fqa *FeatureQualityAssessor) AssessQuality(faceImage gocv.Mat, originalImage gocv.Mat, faceRect image.Rectangle) QualityAssessmentResult {
	if faceImage.Empty() {
		return QualityAssessmentResult{
			Error: errors.New("face image is empty"),
		}
	}

	if originalImage.Empty() {
		return QualityAssessmentResult{
			Error: errors.New("original image is empty"),
		}
	}

	metrics := FeatureQualityMetrics{
		QualityIssues: make([]string, 0),
	}

	// 1. Assess face size
	faceSize, faceSizeScore := fqa.assessFaceSize(faceRect, originalImage.Size())
	metrics.FaceSize = faceSize

	// 2. Assess face position
	position, positionScore := fqa.assessFacePosition(faceRect, originalImage.Size())
	metrics.FacePosition = position

	// 3. Assess image sharpness
	sharpness, sharpnessScore := fqa.assessImageSharpness(faceImage)
	metrics.ImageSharpness = sharpness

	// 4. Assess lighting quality
	lighting, lightingScore := fqa.assessLightingQuality(faceImage)
	metrics.LightingQuality = lighting

	// 5. Assess feature strength
	featureStrength, featureScore := fqa.assessFeatureStrength(faceImage)
	metrics.FeatureStrength = featureStrength

	// 6. Calculate overall quality score
	overallScore := fqa.calculateOverallQuality(faceSizeScore, positionScore, sharpnessScore, lightingScore, featureScore)
	metrics.OverallQuality = overallScore

	// 7. Identify quality issues
	metrics.QualityIssues = fqa.identifyQualityIssues(metrics)

	return QualityAssessmentResult{
		Metrics: metrics,
	}
}

// assessFaceSize evaluates the face size relative to the image
func (fqa *FeatureQualityAssessor) assessFaceSize(faceRect image.Rectangle, imageSize []int) (float64, float64) {
	if len(imageSize) < 2 {
		return 0.0, 0.0
	}

	imageHeight, imageWidth := imageSize[0], imageSize[1]
	faceArea := float64(faceRect.Dx() * faceRect.Dy())
	imageArea := float64(imageWidth * imageHeight)

	facePercentage := (faceArea / imageArea) * 100.0

	// Calculate score based on how close to optimal range
	var score float64
	if facePercentage >= OPTIMAL_FACE_SIZE_MIN && facePercentage <= OPTIMAL_FACE_SIZE_MAX {
		score = 1.0 // Perfect size
	} else if facePercentage >= fqa.minFaceSize && facePercentage <= fqa.maxFaceSize {
		// Acceptable but not optimal
		if facePercentage < OPTIMAL_FACE_SIZE_MIN {
			score = (facePercentage - fqa.minFaceSize) / (OPTIMAL_FACE_SIZE_MIN - fqa.minFaceSize) * 0.8
		} else {
			score = (fqa.maxFaceSize - facePercentage) / (fqa.maxFaceSize - OPTIMAL_FACE_SIZE_MAX) * 0.8
		}
	} else {
		score = 0.0 // Poor size
	}

	return facePercentage, score
}

// assessFacePosition evaluates the face position within the image
func (fqa *FeatureQualityAssessor) assessFacePosition(faceRect image.Rectangle, imageSize []int) (string, float64) {
	if len(imageSize) < 2 {
		return "unknown", 0.0
	}

	imageHeight, imageWidth := imageSize[0], imageSize[1]
	imageCenterX := imageWidth / 2
	imageCenterY := imageHeight / 2

	faceCenterX := faceRect.Min.X + faceRect.Dx()/2
	faceCenterY := faceRect.Min.Y + faceRect.Dy()/2

	// Calculate offset from center as percentage
	offsetX := math.Abs(float64(faceCenterX-imageCenterX)) / float64(imageWidth) * 100.0
	offsetY := math.Abs(float64(faceCenterY-imageCenterY)) / float64(imageHeight) * 100.0

	maxOffset := math.Max(offsetX, offsetY)

	var position string
	var score float64

	if maxOffset <= 10.0 {
		position = "well-centered"
		score = 1.0
	} else if maxOffset <= MAX_CENTER_OFFSET_PERCENT {
		position = "slightly off-center"
		score = 1.0 - (maxOffset-10.0)/(MAX_CENTER_OFFSET_PERCENT-10.0)*0.3
	} else if maxOffset <= 40.0 {
		position = "off-center"
		score = 0.7 - (maxOffset-MAX_CENTER_OFFSET_PERCENT)/(40.0-MAX_CENTER_OFFSET_PERCENT)*0.4
	} else {
		position = "poorly positioned"
		score = 0.3 - math.Min((maxOffset-40.0)/20.0*0.3, 0.3)
	}

	return position, math.Max(score, 0.0)
}

// assessImageSharpness evaluates the sharpness of the face image
func (fqa *FeatureQualityAssessor) assessImageSharpness(faceImage gocv.Mat) (float64, float64) {
	// Convert to grayscale if needed
	var gray gocv.Mat
	if faceImage.Channels() == 3 {
		gray = gocv.NewMat()
		defer gray.Close()
		gocv.CvtColor(faceImage, &gray, gocv.ColorBGRToGray)
	} else {
		gray = faceImage.Clone()
		defer gray.Close()
	}

	// Calculate Laplacian variance for sharpness
	laplacian := gocv.NewMat()
	defer laplacian.Close()
	gocv.Laplacian(gray, &laplacian, gocv.MatTypeCV64F, 1, 1, 0, gocv.BorderDefault)

	// Calculate variance
	meanMat := gocv.NewMat()
	stddevMat := gocv.NewMat()
	defer meanMat.Close()
	defer stddevMat.Close()

	gocv.MeanStdDev(laplacian, &meanMat, &stddevMat)
	variance := stddevMat.GetDoubleAt(0, 0) * stddevMat.GetDoubleAt(0, 0)

	// Normalize sharpness score
	var score float64
	if variance >= GOOD_SHARPNESS_THRESHOLD {
		score = 1.0
	} else if variance >= fqa.sharpnessThresh {
		score = (variance-fqa.sharpnessThresh)/(GOOD_SHARPNESS_THRESHOLD-fqa.sharpnessThresh)*0.7 + 0.3
	} else {
		score = variance / fqa.sharpnessThresh * 0.3
	}

	return variance, math.Min(score, 1.0)
}

// assessLightingQuality evaluates the lighting conditions of the face image
func (fqa *FeatureQualityAssessor) assessLightingQuality(faceImage gocv.Mat) (float64, float64) {
	// Convert to grayscale if needed
	var gray gocv.Mat
	if faceImage.Channels() == 3 {
		gray = gocv.NewMat()
		defer gray.Close()
		gocv.CvtColor(faceImage, &gray, gocv.ColorBGRToGray)
	} else {
		gray = faceImage.Clone()
		defer gray.Close()
	}

	// Calculate mean brightness
	meanMat := gocv.NewMat()
	stddevMat := gocv.NewMat()
	defer meanMat.Close()
	defer stddevMat.Close()

	gocv.MeanStdDev(gray, &meanMat, &stddevMat)
	brightness := meanMat.GetDoubleAt(0, 0)
	contrast := stddevMat.GetDoubleAt(0, 0)

	// Assess lighting quality based on brightness and contrast
	var score float64

	// Check if brightness is in optimal range
	if brightness >= OPTIMAL_BRIGHTNESS_MIN && brightness <= OPTIMAL_BRIGHTNESS_MAX {
		score = 1.0
	} else if brightness >= fqa.lightingThresh && brightness <= MAX_BRIGHTNESS_QUALITY {
		if brightness < OPTIMAL_BRIGHTNESS_MIN {
			score = (brightness-fqa.lightingThresh)/(OPTIMAL_BRIGHTNESS_MIN-fqa.lightingThresh)*0.8 + 0.2
		} else {
			score = (MAX_BRIGHTNESS_QUALITY-brightness)/(MAX_BRIGHTNESS_QUALITY-OPTIMAL_BRIGHTNESS_MAX)*0.8 + 0.2
		}
	} else {
		score = 0.2
	}

	// Adjust score based on contrast (good contrast improves lighting quality)
	if contrast > 30.0 {
		score = math.Min(score*1.1, 1.0)
	} else if contrast < 15.0 {
		score = score * 0.9
	}

	return brightness, math.Max(score, 0.0)
}

// assessFeatureStrength evaluates the strength and distinctiveness of facial features
func (fqa *FeatureQualityAssessor) assessFeatureStrength(faceImage gocv.Mat) (float64, float64) {
	// Convert to grayscale if needed
	var gray gocv.Mat
	if faceImage.Channels() == 3 {
		gray = gocv.NewMat()
		defer gray.Close()
		gocv.CvtColor(faceImage, &gray, gocv.ColorBGRToGray)
	} else {
		gray = faceImage.Clone()
		defer gray.Close()
	}

	// Calculate edge density using Canny edge detection
	edges := gocv.NewMat()
	defer edges.Close()
	gocv.Canny(gray, &edges, 50, 150)

	// Count edge pixels
	edgePixels := gocv.CountNonZero(edges)
	totalPixels := gray.Rows() * gray.Cols()
	edgeDensity := float64(edgePixels) / float64(totalPixels)

	// Calculate feature strength score
	var score float64
	if edgeDensity >= GOOD_EDGE_DENSITY {
		score = 1.0
	} else if edgeDensity >= fqa.featureThresh {
		score = (edgeDensity-fqa.featureThresh)/(GOOD_EDGE_DENSITY-fqa.featureThresh)*0.7 + 0.3
	} else {
		score = edgeDensity / fqa.featureThresh * 0.3
	}

	return edgeDensity, math.Min(score, 1.0)
}

// calculateOverallQuality computes the weighted overall quality score
func (fqa *FeatureQualityAssessor) calculateOverallQuality(faceSizeScore, positionScore, sharpnessScore, lightingScore, featureScore float64) float64 {
	overallScore := FACE_SIZE_WEIGHT*faceSizeScore +
		POSITION_WEIGHT*positionScore +
		SHARPNESS_WEIGHT*sharpnessScore +
		LIGHTING_WEIGHT*lightingScore +
		FEATURE_WEIGHT*featureScore

	return math.Min(overallScore, 1.0)
}

// identifyQualityIssues identifies specific quality issues based on the metrics
func (fqa *FeatureQualityAssessor) identifyQualityIssues(metrics FeatureQualityMetrics) []string {
	issues := make([]string, 0)

	// Face size issues
	if metrics.FaceSize < fqa.minFaceSize {
		issues = append(issues, fmt.Sprintf("Face too small (%.1f%% of image)", metrics.FaceSize))
	} else if metrics.FaceSize > fqa.maxFaceSize {
		issues = append(issues, fmt.Sprintf("Face too large (%.1f%% of image)", metrics.FaceSize))
	}

	// Position issues
	if metrics.FacePosition == "off-center" || metrics.FacePosition == "poorly positioned" {
		issues = append(issues, "Face not well-centered in image")
	}

	// Sharpness issues
	if metrics.ImageSharpness < fqa.sharpnessThresh {
		issues = append(issues, "Image appears blurry or out of focus")
	}

	// Lighting issues
	if metrics.LightingQuality < 0.5 {
		issues = append(issues, "Poor lighting conditions")
	}

	// Feature strength issues
	if metrics.FeatureStrength < 0.4 {
		issues = append(issues, "Weak facial features or low contrast")
	}

	// Overall quality issues
	if metrics.OverallQuality < 0.3 {
		issues = append(issues, "Overall image quality is poor")
	} else if metrics.OverallQuality < 0.6 {
		issues = append(issues, "Image quality could be improved")
	}

	return issues
}

// SetThresholds allows customization of quality assessment thresholds
func (fqa *FeatureQualityAssessor) SetThresholds(minFaceSize, maxFaceSize, sharpnessThresh, lightingThresh, featureThresh float64) {
	if minFaceSize > 0 && minFaceSize < maxFaceSize {
		fqa.minFaceSize = minFaceSize
	}
	if maxFaceSize > minFaceSize && maxFaceSize <= 100 {
		fqa.maxFaceSize = maxFaceSize
	}
	if sharpnessThresh > 0 {
		fqa.sharpnessThresh = sharpnessThresh
	}
	if lightingThresh > 0 && lightingThresh <= 255 {
		fqa.lightingThresh = lightingThresh
	}
	if featureThresh > 0 && featureThresh <= 1.0 {
		fqa.featureThresh = featureThresh
	}
}

// GetThresholds returns the current quality assessment thresholds
func (fqa *FeatureQualityAssessor) GetThresholds() (float64, float64, float64, float64, float64) {
	return fqa.minFaceSize, fqa.maxFaceSize, fqa.sharpnessThresh, fqa.lightingThresh, fqa.featureThresh
}

// IsGoodQuality determines if the overall quality meets minimum standards
func (fqa *FeatureQualityAssessor) IsGoodQuality(metrics FeatureQualityMetrics) bool {
	return metrics.OverallQuality >= 0.6 && len(metrics.QualityIssues) <= 2
}

// GetQualityLevel returns a descriptive quality level based on the overall score
func (fqa *FeatureQualityAssessor) GetQualityLevel(overallScore float64) string {
	if overallScore >= 0.8 {
		return "excellent"
	} else if overallScore >= 0.6 {
		return "good"
	} else if overallScore >= 0.4 {
		return "fair"
	} else if overallScore >= 0.2 {
		return "poor"
	} else {
		return "very poor"
	}
}
