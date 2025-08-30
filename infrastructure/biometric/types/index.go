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
}

// FaceAnalysisResponse represents the complete response from face analysis
type BiometricLivenessResponse struct {
	AnalysisDetails  AnalysisDetails `json:"analysis_details"`
	Confidence       float64         `json:"confidence"`
	Error            *string         `json:"error"`
	FailureReason    string          `json:"failure_reason"`
	IsLive           bool            `json:"is_live"`
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
