package types

import "time"

type BiometricServiceType interface {
	CompareFaces(image1 *string, image2 *string) (*BiometricFaceMatchResponse, error)
	ImageLivenessCheck(image *string) (*BiometricLivenessResponse, error)
	VideoLivenessCheck(payload VideoLivenessRequest) (*VideoLivenessResponse, error)
	GenerateChallenge() (*ChallengeResponse, error)
}

type AnalysisDetails struct {
	ImageQuality        float64 `json:"image_quality"`
	LandmarkConsistency float64 `json:"landmark_consistency"`
	LightingScore       float64 `json:"lighting_score"`
	SharpnessScore      float64 `json:"sharpness_score"`
	SpoofDetectionScore float64 `json:"spoof_detection_score"`
	TextureScore        float64 `json:"texture_score"`

	// Detailed breakdown scores
	LBPScore              float64 `json:"lbp_score"`
	LPQScore              float64 `json:"lpq_score"`
	ReflectionConsistency float64 `json:"reflection_consistency"`
	ColorRGBVariance      float64 `json:"color_rgb_variance"`
	ColorHSVVariance      float64 `json:"color_hsv_variance"`
	ColorLABVariance      float64 `json:"color_lab_variance"`
	EdgeDensity           float64 `json:"edge_density"`
	EdgeSharpness         float64 `json:"edge_sharpness"`
	EdgeConsistency       float64 `json:"edge_consistency"`
	HighFrequency         float64 `json:"high_frequency"`
	MidFrequency          float64 `json:"mid_frequency"`
	LowFrequency          float64 `json:"low_frequency"`
	CompressionArtifacts  float64 `json:"compression_artifacts"`
	TextureVariance       float64 `json:"texture_variance"`
	TextureUniformity     float64 `json:"texture_uniformity"`
	TextureEntropy        float64 `json:"texture_entropy"`
}

// DetailedAnalysisResult contains detailed breakdown of liveness analysis
type DetailedAnalysisResult struct {
	LBPScore              float64
	LPQScore              float64
	ReflectionConsistency float64
	ColorRGBVariance      float64
	ColorHSVVariance      float64
	ColorLABVariance      float64
	EdgeDensity           float64
	EdgeSharpness         float64
	EdgeConsistency       float64
	HighFrequency         float64
	MidFrequency          float64
	LowFrequency          float64
	CompressionArtifacts  float64
	TextureVariance       float64
	TextureUniformity     float64
	TextureEntropy        float64
}

// FaceAnalysisResponse represents the complete response from face analysis
type BiometricLivenessResponse struct {
	AnalysisDetails  AnalysisDetails `json:"analysis_details"`
	Confidence       float64         `json:"confidence"`
	Error            *string         `json:"error"`
	FailureReason    *string         `json:"failure_reason"`
	IsLive           bool            `json:"is_live"`
	LivenessScore    float64         `json:"liveness_score"` // The final score compared to threshold
	ThresholdUsed    float64         `json:"threshold_used"` // The threshold that was applied
	ProcessingTimeMs int             `json:"processing_time_ms"`
	QualityScore     float64         `json:"quality_score"`
	Success          bool            `json:"success"`
}

type Liveness struct {
	Image1 bool `json:"image1"`
	Image2 bool `json:"image2"`
}

type BiometricFaceMatchResponse struct {
	Confidence        float64   `json:"confidence"`
	Error             *string   `json:"error"`
	FaceQualityScores []float64 `json:"face_quality_scores"`
	Liveness          Liveness  `json:"liveness"`
	Match             bool      `json:"match"`
	ProcessingTimeMs  int       `json:"processing_time_ms"`
	Success           bool      `json:"success"`
}

// Challenge-related types
type ChallengeRequest struct {
	TTLSeconds int `json:"ttl_seconds,omitempty"`
}

type ChallengeResponse struct {
	Success     bool     `json:"success"`
	ChallengeID *string  `json:"challenge_id"`
	Directions  []string `json:"directions"`
	TTLSeconds  int      `json:"ttl_seconds"`
	Error       error    `json:"error"`
}

type VideoLivenessRequest struct {
	ChallengeID string   `json:"challenge_id"`
	VideoURLs   []string `json:"video_urls"`
}

type VideoLivenessResponse struct {
	Success            bool     `json:"success"`
	Result             bool     `json:"result"`
	ExpectedDirections []string `json:"expected_directions"`
	DetectedDirections []string `json:"detected_directions"`
	Confidence         float32  `json:"confidence"`
	Message            string   `json:"message"`
	Error              *string  `json:"error"`
}

// Internal challenge storage
type Challenge struct {
	ID         string    `json:"id"`
	Directions []string  `json:"directions"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Used       bool      `json:"used"`
}

// Face comparison request
type FaceComparisonRequest struct {
	Image1 *string `json:"image1"`
	Image2 *string `json:"image2"`
}

// Liveness check request
type LivenessCheckRequest struct {
	Image *string `json:"image"`
}
