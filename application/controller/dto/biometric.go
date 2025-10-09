package dto

import (
	"fmt"
	"strings"
	"time"
)

// LivenessDetectionDTO represents the request for liveness detection
type LivenessDetectionDTO struct {
	Image     string  `json:"image" validate:"required"`    // Base64 encoded image or URL
	Threshold float64 `json:"threshold,omitempty"`         // Liveness threshold (0.0-1.0, default: 0.6)
	Verbose   bool    `json:"verbose,omitempty"`           // Enable verbose analysis reporting
}

// FaceComparisonDTO represents the request for face comparison
type FaceComparisonDTO struct {
	ReferenceImage string  `json:"reference_image" validate:"required"` // Base64 encoded image or URL
	TestImage      string  `json:"test_image" validate:"required"`      // Base64 encoded image or URL
	Threshold      float64 `json:"threshold,omitempty"`                 // Custom similarity threshold (0.0-1.0)
	RequestID      string  `json:"request_id,omitempty"`                // Optional request ID for tracking
}

// ImageQualityDTO represents the request for image quality verification
type ImageQualityDTO struct {
	Image     string `json:"image" validate:"required"` // Base64 encoded image or URL
	RequestID string `json:"request_id,omitempty"`      // Optional request ID for tracking
}

// LivenessDetectionResponse represents the response for liveness detection
type LivenessDetectionResponse struct {
	IsLive            bool                  `json:"is_live"`
	LivenessScore     float64               `json:"liveness_score"`
	ThresholdUsed     float64               `json:"threshold_used"`
	SpoofScore        float64               `json:"spoof_score"`
	Confidence        float64               `json:"confidence"`
	ProcessingTime    int64                 `json:"processing_time_ms"`
	AnalysisBreakdown *AnalysisBreakdownDTO `json:"analysis_breakdown,omitempty"`
	QualityMetrics    *QualityMetricsDTO    `json:"quality_metrics,omitempty"`
	SpoofReasons      []string              `json:"spoof_reasons,omitempty"`
	Recommendations   []string              `json:"recommendations,omitempty"`
	RequestID         string                `json:"request_id"`
	Timestamp         time.Time             `json:"timestamp"`
	Error             string                `json:"error,omitempty"`
}

// FaceComparisonResponse represents the response for face comparison
type FaceComparisonResponse struct {
	IsMatch          bool               `json:"is_match"`
	Similarity       float64            `json:"similarity"`
	Confidence       float64            `json:"confidence"`
	ProcessingTime   int64              `json:"processing_time_ms"`
	ReferenceQuality *QualityMetricsDTO `json:"reference_quality,omitempty"`
	TestQuality      *QualityMetricsDTO `json:"test_quality,omitempty"`
	MatchMetadata    *MatchMetadataDTO  `json:"match_metadata,omitempty"`
	RequestID        string             `json:"request_id"`
	Timestamp        time.Time          `json:"timestamp"`
	Error            string             `json:"error,omitempty"`
}

// ImageQualityResponse represents the response for image quality verification
type ImageQualityResponse struct {
	IsGoodQuality   bool      `json:"is_good_quality"`
	HasFace         bool      `json:"has_face"`
	FaceCount       int       `json:"face_count"`
	FaceSize        float64   `json:"face_size_percent"`
	ImageResolution string    `json:"image_resolution"`
	QualityScore    float64   `json:"quality_score"`
	Issues          []string  `json:"issues,omitempty"`
	Recommendations []string  `json:"recommendations,omitempty"`
	RequestID       string    `json:"request_id"`
	Timestamp       time.Time `json:"timestamp"`
	Error           string    `json:"error,omitempty"`
}

// AnalysisBreakdownDTO represents detailed analysis breakdown
type AnalysisBreakdownDTO struct {
	LBPScore              float64                `json:"lbp_score"`
	LPQScore              float64                `json:"lpq_score"`
	ReflectionConsistency float64                `json:"reflection_consistency"`
	ColorSpaceAnalysis    *ColorSpaceScoresDTO   `json:"color_space_analysis,omitempty"`
	EdgeAnalysis          *EdgeAnalysisScoresDTO `json:"edge_analysis,omitempty"`
	FrequencyAnalysis     *FrequencyScoresDTO    `json:"frequency_analysis,omitempty"`
	TextureAnalysis       *TextureScoresDTO      `json:"texture_analysis,omitempty"`
}

// QualityMetricsDTO represents image quality metrics
type QualityMetricsDTO struct {
	Resolution       string   `json:"resolution"`
	Sharpness        float64  `json:"sharpness"`
	Brightness       float64  `json:"brightness"`
	Contrast         float64  `json:"contrast"`
	FaceSize         float64  `json:"face_size_percent"`
	FacePosition     Point2D  `json:"face_position"`
	CompressionLevel float64  `json:"compression_level"`
	QualityScore     float64  `json:"quality_score"`
	Issues           []string `json:"issues,omitempty"`
	Recommendations  []string `json:"recommendations,omitempty"`
}

// MatchMetadataDTO represents face comparison metadata
type MatchMetadataDTO struct {
	FeatureVector1Length int     `json:"feature_vector1_length"`
	FeatureVector2Length int     `json:"feature_vector2_length"`
	SimilarityMethod     string  `json:"similarity_method"`
	ThresholdUsed        float64 `json:"threshold_used"`
	ConfidenceLevel      string  `json:"confidence_level"`
}

// ColorSpaceScoresDTO represents color space analysis scores
type ColorSpaceScoresDTO struct {
	RGBVariance float64 `json:"rgb_variance"`
	HSVVariance float64 `json:"hsv_variance"`
	LABVariance float64 `json:"lab_variance"`
}

// EdgeAnalysisScoresDTO represents edge analysis scores
type EdgeAnalysisScoresDTO struct {
	EdgeDensity     float64 `json:"edge_density"`
	EdgeSharpness   float64 `json:"edge_sharpness"`
	EdgeConsistency float64 `json:"edge_consistency"`
}

// FrequencyScoresDTO represents frequency analysis scores
type FrequencyScoresDTO struct {
	HighFrequency        float64 `json:"high_frequency"`
	MidFrequency         float64 `json:"mid_frequency"`
	LowFrequency         float64 `json:"low_frequency"`
	CompressionArtifacts float64 `json:"compression_artifacts"`
}

// TextureScoresDTO represents texture analysis scores
type TextureScoresDTO struct {
	TextureVariance   float64 `json:"texture_variance"`
	TextureUniformity float64 `json:"texture_uniformity"`
	TextureEntropy    float64 `json:"texture_entropy"`
}

// Point2D represents a 2D point
type Point2D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// SystemHealthResponse represents system health status
type SystemHealthResponse struct {
	Status                string          `json:"status"`
	ModelsLoaded          bool            `json:"models_loaded"`
	SystemUptime          time.Duration   `json:"system_uptime"`
	ProcessedRequests     int64           `json:"processed_requests"`
	AverageProcessingTime float64         `json:"average_processing_time_ms"`
	ErrorRate             float64         `json:"error_rate_percent"`
	MemoryUsage           *MemoryUsageDTO `json:"memory_usage,omitempty"`
	ModelInfo             []ModelInfoDTO  `json:"model_info,omitempty"`
	Timestamp             time.Time       `json:"timestamp"`
}

// MemoryUsageDTO represents memory usage information
type MemoryUsageDTO struct {
	AllocatedMB float64 `json:"allocated_mb"`
	SystemMB    float64 `json:"system_mb"`
	GCCycles    uint32  `json:"gc_cycles"`
}

// ModelInfoDTO represents model information
type ModelInfoDTO struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Loaded   bool      `json:"loaded"`
	LoadTime time.Time `json:"load_time,omitempty"`
	Version  string    `json:"version,omitempty"`
}

// EnhancedFaceComparisonRequest represents the enhanced request for face comparison with liveness detection
type EnhancedFaceComparisonRequest struct {
	Image1    string  `json:"image1" validate:"required"`   // Base64 encoded image or URL
	Image2    string  `json:"image2" validate:"required"`   // Base64 encoded image or URL
	Threshold float64 `json:"threshold,omitempty"`         // Custom similarity threshold (0.0-1.0)
	Verbose   bool    `json:"verbose,omitempty"`           // Enable verbose analysis reporting
	RequestID string  `json:"request_id,omitempty"`        // Optional request ID for tracking
}

// EnhancedFaceComparisonResponse represents the enhanced response for face comparison with liveness results
type EnhancedFaceComparisonResponse struct {
	IsMatch        bool    `json:"is_match"`
	Similarity     float64 `json:"similarity"`
	Confidence     float64 `json:"confidence"`
	ProcessingTime int64   `json:"processing_time_ms"`

	// Liveness Detection Results
	ReferenceLiveness   *LivenessResultDTO `json:"reference_liveness"`
	TestLiveness        *LivenessResultDTO `json:"test_liveness"`
	LivenessProcessTime int64              `json:"liveness_process_time_ms"`

	// Enhanced Comparison Metadata
	FeatureQuality     *FeatureQualityMetricsDTO      `json:"feature_quality,omitempty"`
	ComparisonMetadata *EnhancedComparisonMetadataDTO `json:"comparison_metadata,omitempty"`

	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
}

// LivenessResultDTO represents liveness detection result for a single image
type LivenessResultDTO struct {
	IsLive       bool     `json:"is_live"`
	SpoofScore   float64  `json:"spoof_score"`
	Confidence   float64  `json:"confidence"`
	SpoofReasons []string `json:"spoof_reasons,omitempty"`
}

// FeatureQualityMetricsDTO represents enhanced feature quality assessment
type FeatureQualityMetricsDTO struct {
	FaceSize        float64 `json:"face_size_percent"`
	FacePosition    string  `json:"face_position"`
	ImageSharpness  float64 `json:"image_sharpness"`
	LightingQuality float64 `json:"lighting_quality"`
	FeatureStrength float64 `json:"feature_strength"`
}

// EnhancedComparisonMetadataDTO represents enhanced comparison processing metadata
type EnhancedComparisonMetadataDTO struct {
	SimilarityMethod  string              `json:"similarity_method"`
	ThresholdUsed     float64             `json:"threshold_used"`
	QualityAdjustment float64             `json:"quality_adjustment"`
	ConfidenceLevel   string              `json:"confidence_level"`
	FeatureStrength   float64             `json:"feature_strength"`
	ProcessingSteps   []ProcessingStepDTO `json:"processing_steps"`
}

// ProcessingStepDTO represents a single processing step with timing
type ProcessingStepDTO struct {
	Step     string `json:"step"`
	Duration int64  `json:"duration_ms"`
	Success  bool   `json:"success"`
	Details  string `json:"details,omitempty"`
}

// ValidateEnhancedFaceComparisonRequest validates the enhanced face comparison request
func ValidateEnhancedFaceComparisonRequest(req *EnhancedFaceComparisonRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	// Validate reference image
	if err := validateImageInput(req.Image1, "image1"); err != nil {
		return err
	}

	// Validate test image
	if err := validateImageInput(req.Image2, "image2"); err != nil {
		return err
	}

	// Validate threshold if provided
	if req.Threshold != 0 && (req.Threshold < 0.0 || req.Threshold > 1.0) {
		return fmt.Errorf("threshold must be between 0.0 and 1.0, got: %f", req.Threshold)
	}

	return nil
}

// validateImageInput validates image input with production-grade checks
func validateImageInput(image, fieldName string) error {
	if image == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}

	// Check if it's a URL
	if strings.HasPrefix(image, "http://") || strings.HasPrefix(image, "https://") {
		// Basic URL validation
		if len(image) > 2048 {
			return fmt.Errorf("%s URL too long (max 2048 characters)", fieldName)
		}

		// Additional URL security checks
		if strings.Contains(strings.ToLower(image), "localhost") ||
			strings.Contains(strings.ToLower(image), "127.0.0.1") ||
			strings.Contains(strings.ToLower(image), "0.0.0.0") {
			return fmt.Errorf("%s localhost URLs not allowed for security reasons", fieldName)
		}

		return nil
	}

	// Check if it's base64
	if strings.Contains(image, ",") {
		// Data URL format
		parts := strings.Split(image, ",")
		if len(parts) != 2 {
			return fmt.Errorf("%s invalid data URL format", fieldName)
		}
		image = parts[1]
	}

	// Validate base64 length (approximate size check)
	if len(image) > 67108864 { // ~50MB in base64
		return fmt.Errorf("%s too large (max ~50MB)", fieldName)
	}

	if len(image) < 100 {
		return fmt.Errorf("%s too small (minimum 100 characters)", fieldName)
	}

	// Basic base64 validation
	if len(image)%4 != 0 {
		return fmt.Errorf("%s invalid base64 encoding", fieldName)
	}

	return nil
}

// NewEnhancedFaceComparisonResponse creates a new enhanced face comparison response
func NewEnhancedFaceComparisonResponse(requestID string) *EnhancedFaceComparisonResponse {
	return &EnhancedFaceComparisonResponse{
		RequestID: requestID,
		Timestamp: time.Now(),
	}
}

// SetError sets an error on the response
func (r *EnhancedFaceComparisonResponse) SetError(err error) {
	if err != nil {
		r.Error = err.Error()
	}
}

// SetComparisonResult sets the main comparison results
func (r *EnhancedFaceComparisonResponse) SetComparisonResult(isMatch bool, similarity, confidence float64, processingTime int64) {
	r.IsMatch = isMatch
	r.Similarity = similarity
	r.Confidence = confidence
	r.ProcessingTime = processingTime
}

// SetLivenessResults sets the liveness detection results for both images
func (r *EnhancedFaceComparisonResponse) SetLivenessResults(refLiveness, testLiveness *LivenessResultDTO, livenessProcessTime int64) {
	r.ReferenceLiveness = refLiveness
	r.TestLiveness = testLiveness
	r.LivenessProcessTime = livenessProcessTime
}

// SetFeatureQuality sets the feature quality metrics
func (r *EnhancedFaceComparisonResponse) SetFeatureQuality(quality *FeatureQualityMetricsDTO) {
	r.FeatureQuality = quality
}

// SetComparisonMetadata sets the comparison metadata
func (r *EnhancedFaceComparisonResponse) SetComparisonMetadata(metadata *EnhancedComparisonMetadataDTO) {
	r.ComparisonMetadata = metadata
}

// AddProcessingStep adds a processing step to the comparison metadata
func (r *EnhancedFaceComparisonResponse) AddProcessingStep(step string, duration int64, success bool, details string) {
	if r.ComparisonMetadata == nil {
		r.ComparisonMetadata = &EnhancedComparisonMetadataDTO{
			ProcessingSteps: make([]ProcessingStepDTO, 0),
		}
	}

	r.ComparisonMetadata.ProcessingSteps = append(r.ComparisonMetadata.ProcessingSteps, ProcessingStepDTO{
		Step:     step,
		Duration: duration,
		Success:  success,
		Details:  details,
	})
}

// IsSuccessful returns true if the response represents a successful operation (no error)
func (r *EnhancedFaceComparisonResponse) IsSuccessful() bool {
	return r.Error == ""
}

// GetTotalProcessingTime returns the total processing time including liveness detection
func (r *EnhancedFaceComparisonResponse) GetTotalProcessingTime() int64 {
	return r.ProcessingTime + r.LivenessProcessTime
}

// NewLivenessResultDTO creates a new liveness result DTO
func NewLivenessResultDTO(isLive bool, spoofScore, confidence float64, spoofReasons []string) *LivenessResultDTO {
	return &LivenessResultDTO{
		IsLive:       isLive,
		SpoofScore:   spoofScore,
		Confidence:   confidence,
		SpoofReasons: spoofReasons,
	}
}

// NewFeatureQualityMetricsDTO creates a new feature quality metrics DTO
func NewFeatureQualityMetricsDTO(faceSize float64, facePosition string, imageSharpness, lightingQuality, featureStrength float64) *FeatureQualityMetricsDTO {
	return &FeatureQualityMetricsDTO{
		FaceSize:        faceSize,
		FacePosition:    facePosition,
		ImageSharpness:  imageSharpness,
		LightingQuality: lightingQuality,
		FeatureStrength: featureStrength,
	}
}

// NewEnhancedComparisonMetadataDTO creates a new enhanced comparison metadata DTO
func NewEnhancedComparisonMetadataDTO(similarityMethod string, thresholdUsed, qualityAdjustment, featureStrength float64, confidenceLevel string) *EnhancedComparisonMetadataDTO {
	return &EnhancedComparisonMetadataDTO{
		SimilarityMethod:  similarityMethod,
		ThresholdUsed:     thresholdUsed,
		QualityAdjustment: qualityAdjustment,
		ConfidenceLevel:   confidenceLevel,
		FeatureStrength:   featureStrength,
		ProcessingSteps:   make([]ProcessingStepDTO, 0),
	}
}

// New API DTOs for face comparison and liveness detection

// FaceComparisonRequest represents the request for face comparison
type FaceComparisonRequest struct {
	Image1 string `json:"image1" validate:"required"` // Base64 encoded image or URL
	Image2 string `json:"image2" validate:"required"` // Base64 encoded image or URL
}

// LivenessCheckRequest represents the request for image liveness check
type LivenessCheckRequest struct {
	Image         string `json:"image" validate:"required"` // Base64 encoded image or URL
	LenientBlurry bool   `json:"lenient_blurry,omitempty"`  // Enable lenient mode for slightly blurry images
}

// VideoLivenessVerificationRequest represents the request for video liveness verification
type VideoLivenessVerificationRequest struct {
	ChallengeID string `json:"challenge_id" validate:"required"` // Challenge ID from generate-challenge
}
