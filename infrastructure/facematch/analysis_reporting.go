package facematch

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// AnalysisReport represents a comprehensive report of the anti-spoofing and face comparison analysis
type AnalysisReport struct {
	// Basic information
	RequestID        string    `json:"request_id"`
	Timestamp        time.Time `json:"timestamp"`
	ProcessingTimeMs int64     `json:"processing_time_ms"`

	// Analysis results
	IsLive       bool     `json:"is_live"`
	SpoofScore   float64  `json:"spoof_score"`
	Confidence   float64  `json:"confidence"`
	SpoofReasons []string `json:"spoof_reasons,omitempty"`

	// Detailed analysis breakdown
	AnalysisBreakdown AnalysisBreakdown `json:"analysis_breakdown"`

	// Quality metrics
	QualityMetrics QualityMetrics `json:"quality_metrics"`

	// Recommendations
	Recommendations []string `json:"recommendations,omitempty"`

	// Debug information (only included in verbose mode)
	DebugInfo *DebugInfo `json:"debug_info,omitempty"`

	// Metadata for additional context
	Metadata ReportMetadata `json:"metadata,omitempty"`
}

// QualityMetrics provides detailed information about image quality
type QualityMetrics struct {
	Resolution       string   `json:"resolution"`
	Sharpness        float64  `json:"sharpness"`
	Brightness       float64  `json:"brightness"`
	Contrast         float64  `json:"contrast"`
	FaceSize         float64  `json:"face_size_percent"`
	FacePosition     Point2D  `json:"face_position"`
	CompressionLevel float64  `json:"compression_level"`
	QualityScore     float64  `json:"quality_score"`
	Issues           []string `json:"issues"`
	Recommendations  []string `json:"recommendations"`
}

// Point2D represents a 2D point for face position
type Point2D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// DebugInfo contains additional debug information for verbose mode
type DebugInfo struct {
	ConfigSettings      map[string]interface{} `json:"config_settings"`
	IntermediateResults map[string]interface{} `json:"intermediate_results"`
	ModelVersions       map[string]string      `json:"model_versions"`
	PerformanceMetrics  PerformanceMetrics     `json:"performance_metrics"`
}

// PerformanceMetrics contains detailed timing information for performance analysis
type PerformanceMetrics struct {
	ImageLoadTimeMs         int64   `json:"image_load_time_ms"`
	FaceDetectionTimeMs     int64   `json:"face_detection_time_ms"`
	AnalysisTimeMs          int64   `json:"analysis_time_ms"`
	TextureAnalysisTimeMs   int64   `json:"texture_analysis_time_ms"`
	ColorAnalysisTimeMs     int64   `json:"color_analysis_time_ms"`
	FrequencyAnalysisTimeMs int64   `json:"frequency_analysis_time_ms"`
	EdgeAnalysisTimeMs      int64   `json:"edge_analysis_time_ms"`
	DecisionTimeMs          int64   `json:"decision_time_ms"`
	TotalProcessingTimeMs   int64   `json:"total_processing_time_ms"`
	MemoryUsageKB           int64   `json:"memory_usage_kb,omitempty"`
	CPUUsagePercent         float64 `json:"cpu_usage_percent,omitempty"`
}

// ReportMetadata contains additional contextual information about the analysis
type ReportMetadata struct {
	// System information
	SystemInfo SystemInfo `json:"system_info,omitempty"`

	// Input information
	InputInfo InputInfo `json:"input_info,omitempty"`

	// Analysis context
	AnalysisContext AnalysisContext `json:"analysis_context,omitempty"`

	// Custom metadata fields
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

// SystemInfo contains information about the system performing the analysis
type SystemInfo struct {
	HostID            string `json:"host_id,omitempty"`
	HostName          string `json:"host_name,omitempty"`
	ServiceVersion    string `json:"service_version,omitempty"`
	OperatingSystem   string `json:"operating_system,omitempty"`
	ProcessorInfo     string `json:"processor_info,omitempty"`
	AvailableMemoryMB int64  `json:"available_memory_mb,omitempty"`
}

// InputInfo contains information about the input being analyzed
type InputInfo struct {
	ImageSource      string `json:"image_source,omitempty"`
	ImageFormat      string `json:"image_format,omitempty"`
	ImageSizeBytes   int64  `json:"image_size_bytes,omitempty"`
	ImageDimensions  string `json:"image_dimensions,omitempty"`
	ImageOrientation string `json:"image_orientation,omitempty"`
	DeviceInfo       string `json:"device_info,omitempty"`
}

// AnalysisContext contains information about the context of the analysis
type AnalysisContext struct {
	SecurityLevel      string    `json:"security_level,omitempty"`
	AnalysisMode       string    `json:"analysis_mode,omitempty"`
	ThresholdProfile   string    `json:"threshold_profile,omitempty"`
	BatchID            string    `json:"batch_id,omitempty"`
	SessionID          string    `json:"session_id,omitempty"`
	PreviousAnalysisID string    `json:"previous_analysis_id,omitempty"`
	ClientIP           string    `json:"client_ip,omitempty"`
	UserAgent          string    `json:"user_agent,omitempty"`
	AnalysisTimestamp  time.Time `json:"analysis_timestamp,omitempty"`
}

// NewAnalysisReport creates a new analysis report with basic information
func NewAnalysisReport(requestID string) *AnalysisReport {
	return &AnalysisReport{
		RequestID: requestID,
		Timestamp: time.Now(),
		QualityMetrics: QualityMetrics{
			Issues:          []string{},
			Recommendations: []string{},
		},
		Recommendations: []string{},
		SpoofReasons:    []string{},
		Metadata: ReportMetadata{
			AnalysisContext: AnalysisContext{
				AnalysisTimestamp: time.Now(),
			},
			CustomFields: make(map[string]interface{}),
		},
	}
}

// GenerateQualityRecommendations analyzes quality metrics and generates recommendations
func (ar *AnalysisReport) GenerateQualityRecommendations() {
	qm := &ar.QualityMetrics

	// Clear existing recommendations
	qm.Recommendations = []string{}

	// Check sharpness
	if qm.Sharpness < VERY_BLURRY_SHARPNESS {
		qm.Issues = append(qm.Issues, "Image is very blurry")
		qm.Recommendations = append(qm.Recommendations, "Use a clearer, sharper image with good focus")
	} else if qm.Sharpness < SOMEWHAT_BLURRY_SHARPNESS {
		qm.Issues = append(qm.Issues, "Image is somewhat blurry")
		qm.Recommendations = append(qm.Recommendations, "Improve image focus for better results")
	}

	// Check brightness
	if qm.Brightness < MIN_BRIGHTNESS {
		qm.Issues = append(qm.Issues, "Image is too dark")
		qm.Recommendations = append(qm.Recommendations, "Use better lighting conditions")
	} else if qm.Brightness > MAX_BRIGHTNESS {
		qm.Issues = append(qm.Issues, "Image is too bright/overexposed")
		qm.Recommendations = append(qm.Recommendations, "Reduce lighting or camera exposure")
	}

	// Check face size
	if qm.FaceSize < MIN_FACE_SIZE_PERCENT {
		qm.Issues = append(qm.Issues, fmt.Sprintf("Face too small (%.1f%% of image)", qm.FaceSize))
		qm.Recommendations = append(qm.Recommendations, "Move closer to camera or crop image to focus on face")
	} else if qm.FaceSize > MAX_FACE_SIZE_PERCENT {
		qm.Issues = append(qm.Issues, fmt.Sprintf("Face too large (%.1f%% of image)", qm.FaceSize))
		qm.Recommendations = append(qm.Recommendations, "Move back from camera or include more background")
	}

	// Check compression level
	if qm.CompressionLevel > HEAVY_COMPRESSION_LEVEL {
		qm.Issues = append(qm.Issues, "Image is heavily compressed")
		qm.Recommendations = append(qm.Recommendations, "Use higher quality image with less compression")
	} else if qm.CompressionLevel > MODERATE_COMPRESSION_LEVEL {
		qm.Issues = append(qm.Issues, "Image has noticeable compression artifacts")
		qm.Recommendations = append(qm.Recommendations, "Use less compressed image format or higher quality settings")
	}

	// Calculate overall quality score
	qm.QualityScore = calculateQualityScore(qm)

	// Add overall recommendation based on quality score
	if qm.QualityScore < 0.5 {
		ar.Recommendations = append(ar.Recommendations, "Image quality is poor. Consider using a higher quality image for better results.")
	} else if qm.QualityScore < 0.7 {
		ar.Recommendations = append(ar.Recommendations, "Image quality is acceptable but could be improved for better results.")
	}
}

// calculateQualityScore calculates an overall quality score based on metrics
func calculateQualityScore(qm *QualityMetrics) float64 {
	score := 1.0

	// Deduct for issues
	if qm.Sharpness < VERY_BLURRY_SHARPNESS {
		score -= 0.3
	} else if qm.Sharpness < SOMEWHAT_BLURRY_SHARPNESS {
		score -= 0.15
	}

	if qm.Brightness < MIN_BRIGHTNESS || qm.Brightness > MAX_BRIGHTNESS {
		score -= 0.2
	}

	if qm.FaceSize < MIN_FACE_SIZE_PERCENT || qm.FaceSize > MAX_FACE_SIZE_PERCENT {
		score -= 0.2
	}

	if qm.CompressionLevel > HEAVY_COMPRESSION_LEVEL {
		score -= 0.25
	} else if qm.CompressionLevel > MODERATE_COMPRESSION_LEVEL {
		score -= 0.15
	} else if qm.CompressionLevel > LIGHT_COMPRESSION_LEVEL {
		score -= 0.05
	}

	// Ensure score is between 0 and 1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// EnableVerboseMode adds debug information to the report
func (ar *AnalysisReport) EnableVerboseMode(configSettings map[string]interface{}, modelVersions map[string]string) {
	ar.DebugInfo = &DebugInfo{
		ConfigSettings:      configSettings,
		ModelVersions:       modelVersions,
		IntermediateResults: make(map[string]interface{}),
		PerformanceMetrics:  PerformanceMetrics{},
	}
}

// AddIntermediateResult adds an intermediate result to the debug info
func (ar *AnalysisReport) AddIntermediateResult(name string, value interface{}) {
	if ar.DebugInfo != nil {
		ar.DebugInfo.IntermediateResults[name] = value
	}
}

// RecordPerformanceMetric records a performance metric
func (ar *AnalysisReport) RecordPerformanceMetric(metricName string, durationMs int64) {
	if ar.DebugInfo == nil {
		return
	}

	switch metricName {
	case "image_load":
		ar.DebugInfo.PerformanceMetrics.ImageLoadTimeMs = durationMs
	case "face_detection":
		ar.DebugInfo.PerformanceMetrics.FaceDetectionTimeMs = durationMs
	case "analysis":
		ar.DebugInfo.PerformanceMetrics.AnalysisTimeMs = durationMs
	case "texture_analysis":
		ar.DebugInfo.PerformanceMetrics.TextureAnalysisTimeMs = durationMs
	case "color_analysis":
		ar.DebugInfo.PerformanceMetrics.ColorAnalysisTimeMs = durationMs
	case "frequency_analysis":
		ar.DebugInfo.PerformanceMetrics.FrequencyAnalysisTimeMs = durationMs
	case "edge_analysis":
		ar.DebugInfo.PerformanceMetrics.EdgeAnalysisTimeMs = durationMs
	case "decision":
		ar.DebugInfo.PerformanceMetrics.DecisionTimeMs = durationMs
	case "total":
		ar.DebugInfo.PerformanceMetrics.TotalProcessingTimeMs = durationMs
		return // Skip automatic calculation when explicitly set
	}

	// Always calculate total processing time
	ar.DebugInfo.PerformanceMetrics.TotalProcessingTimeMs =
		ar.DebugInfo.PerformanceMetrics.ImageLoadTimeMs +
			ar.DebugInfo.PerformanceMetrics.FaceDetectionTimeMs +
			ar.DebugInfo.PerformanceMetrics.AnalysisTimeMs +
			ar.DebugInfo.PerformanceMetrics.DecisionTimeMs
}

// LogAnalysisReport logs the analysis report with appropriate detail level
func LogAnalysisReport(report *AnalysisReport, verboseMode bool) {
	// Basic logging for all reports
	log.Printf("[Analysis Report] RequestID: %s, IsLive: %v, SpoofScore: %.3f, Confidence: %.3f, ProcessingTime: %dms",
		report.RequestID, report.IsLive, report.SpoofScore, report.Confidence, report.ProcessingTimeMs)

	// Log spoof reasons if any
	if len(report.SpoofReasons) > 0 {
		log.Printf("[Analysis Report] SpoofReasons: %v", report.SpoofReasons)
	}

	// Log quality metrics summary
	log.Printf("[Analysis Report] Quality: Score=%.2f, Resolution=%s, Sharpness=%.2f, Compression=%.2f",
		report.QualityMetrics.QualityScore, report.QualityMetrics.Resolution,
		report.QualityMetrics.Sharpness, report.QualityMetrics.CompressionLevel)

	// Log recommendations if any
	if len(report.Recommendations) > 0 {
		log.Printf("[Analysis Report] Recommendations: %v", report.Recommendations)
	}

	// Detailed logging in verbose mode
	if verboseMode && report.DebugInfo != nil {
		// Log performance metrics
		pm := report.DebugInfo.PerformanceMetrics
		log.Printf("[Analysis Report] Performance: Total=%dms, ImageLoad=%dms, FaceDetection=%dms, Analysis=%dms, Decision=%dms",
			pm.TotalProcessingTimeMs, pm.ImageLoadTimeMs, pm.FaceDetectionTimeMs, pm.AnalysisTimeMs, pm.DecisionTimeMs)

		// Log detailed analysis breakdown
		ab := report.AnalysisBreakdown
		log.Printf("[Analysis Report] LBP=%.3f, LPQ=%.3f, Reflection=%.3f, TextureScore=%.3f",
			ab.LBPScore, ab.LPQScore, ab.ReflectionConsistency, report.QualityMetrics.Sharpness)

		// Log color space analysis
		log.Printf("[Analysis Report] ColorAnalysis: YCrCb=%.3f, HSV=%.3f, LAB=%.3f",
			ab.ColorSpaceAnalysis.YCrCbConsistency, ab.ColorSpaceAnalysis.HSVConsistency, ab.ColorSpaceAnalysis.LABConsistency)

		// Log edge analysis
		log.Printf("[Analysis Report] EdgeAnalysis: Density=%.3f, Orientation=%.3f, Sharpness=%.3f",
			ab.EdgeAnalysis.EdgeDensity, ab.EdgeAnalysis.EdgeOrientation, ab.EdgeAnalysis.EdgeSharpness)

		// Log frequency analysis
		log.Printf("[Analysis Report] FrequencyAnalysis: HighFreq=%.3f, Distribution=%.3f, Noise=%.3f",
			ab.FrequencyAnalysis.HighFrequencyContent, ab.FrequencyAnalysis.FrequencyDistribution, ab.FrequencyAnalysis.NoiseLevel)

		// Log metadata if available
		if report.Metadata.InputInfo.ImageFormat != "" {
			log.Printf("[Analysis Report] InputInfo: Format=%s, Dimensions=%s, Size=%d bytes",
				report.Metadata.InputInfo.ImageFormat, report.Metadata.InputInfo.ImageDimensions, report.Metadata.InputInfo.ImageSizeBytes)
		}

		if report.Metadata.AnalysisContext.SecurityLevel != "" {
			log.Printf("[Analysis Report] Context: SecurityLevel=%s, Mode=%s, Profile=%s",
				report.Metadata.AnalysisContext.SecurityLevel, report.Metadata.AnalysisContext.AnalysisMode, report.Metadata.AnalysisContext.ThresholdProfile)
		}

		// Log any intermediate results
		if len(report.DebugInfo.IntermediateResults) > 0 {
			log.Printf("[Analysis Report] IntermediateResults: %d values stored", len(report.DebugInfo.IntermediateResults))
		}
	}
}

// ToJSON converts the analysis report to JSON
func (ar *AnalysisReport) ToJSON(prettyPrint bool) (string, error) {
	var bytes []byte
	var err error

	if prettyPrint {
		bytes, err = json.MarshalIndent(ar, "", "  ")
	} else {
		bytes, err = json.Marshal(ar)
	}

	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// SetInputInfo sets the input information in the metadata
func (ar *AnalysisReport) SetInputInfo(format string, dimensions string, sizeBytes int64, source string) {
	ar.Metadata.InputInfo.ImageFormat = format
	ar.Metadata.InputInfo.ImageDimensions = dimensions
	ar.Metadata.InputInfo.ImageSizeBytes = sizeBytes
	ar.Metadata.InputInfo.ImageSource = source
}

// SetSystemInfo sets the system information in the metadata
func (ar *AnalysisReport) SetSystemInfo(hostID string, hostName string, serviceVersion string, os string) {
	ar.Metadata.SystemInfo.HostID = hostID
	ar.Metadata.SystemInfo.HostName = hostName
	ar.Metadata.SystemInfo.ServiceVersion = serviceVersion
	ar.Metadata.SystemInfo.OperatingSystem = os
}

// SetAnalysisContext sets the analysis context in the metadata
func (ar *AnalysisReport) SetAnalysisContext(securityLevel string, analysisMode string, thresholdProfile string) {
	ar.Metadata.AnalysisContext.SecurityLevel = securityLevel
	ar.Metadata.AnalysisContext.AnalysisMode = analysisMode
	ar.Metadata.AnalysisContext.ThresholdProfile = thresholdProfile
}

// SetSessionInfo sets the session information in the metadata
func (ar *AnalysisReport) SetSessionInfo(sessionID string, batchID string, clientIP string, userAgent string) {
	ar.Metadata.AnalysisContext.SessionID = sessionID
	ar.Metadata.AnalysisContext.BatchID = batchID
	ar.Metadata.AnalysisContext.ClientIP = clientIP
	ar.Metadata.AnalysisContext.UserAgent = userAgent
}

// AddCustomMetadata adds a custom metadata field
func (ar *AnalysisReport) AddCustomMetadata(key string, value interface{}) {
	ar.Metadata.CustomFields[key] = value
}

// GenerateDetailedAnalysisBreakdown creates a comprehensive analysis breakdown with explanations
func (ar *AnalysisReport) GenerateDetailedAnalysisBreakdown() map[string]interface{} {
	// Create a detailed breakdown with explanations
	breakdown := make(map[string]interface{})

	// Add texture analysis details
	textureAnalysis := make(map[string]interface{})
	textureAnalysis["lbp_score"] = ar.AnalysisBreakdown.LBPScore
	textureAnalysis["lpq_score"] = ar.AnalysisBreakdown.LPQScore
	textureAnalysis["explanation"] = getTextureAnalysisExplanation(ar.AnalysisBreakdown.LBPScore, ar.AnalysisBreakdown.LPQScore)
	breakdown["texture_analysis"] = textureAnalysis

	// Add reflection analysis details
	reflectionAnalysis := make(map[string]interface{})
	reflectionAnalysis["reflection_consistency"] = ar.AnalysisBreakdown.ReflectionConsistency
	reflectionAnalysis["explanation"] = getReflectionAnalysisExplanation(ar.AnalysisBreakdown.ReflectionConsistency)
	breakdown["reflection_analysis"] = reflectionAnalysis

	// Add color space analysis details
	colorAnalysis := make(map[string]interface{})
	colorAnalysis["ycrcb_consistency"] = ar.AnalysisBreakdown.ColorSpaceAnalysis.YCrCbConsistency
	colorAnalysis["hsv_consistency"] = ar.AnalysisBreakdown.ColorSpaceAnalysis.HSVConsistency
	colorAnalysis["lab_consistency"] = ar.AnalysisBreakdown.ColorSpaceAnalysis.LABConsistency
	colorAnalysis["explanation"] = getColorAnalysisExplanation(ar.AnalysisBreakdown.ColorSpaceAnalysis)
	breakdown["color_analysis"] = colorAnalysis

	// Add edge analysis details
	edgeAnalysis := make(map[string]interface{})
	edgeAnalysis["edge_density"] = ar.AnalysisBreakdown.EdgeAnalysis.EdgeDensity
	edgeAnalysis["edge_orientation"] = ar.AnalysisBreakdown.EdgeAnalysis.EdgeOrientation
	edgeAnalysis["edge_sharpness"] = ar.AnalysisBreakdown.EdgeAnalysis.EdgeSharpness
	edgeAnalysis["explanation"] = getEdgeAnalysisExplanation(ar.AnalysisBreakdown.EdgeAnalysis)
	breakdown["edge_analysis"] = edgeAnalysis

	// Add frequency analysis details
	freqAnalysis := make(map[string]interface{})
	freqAnalysis["high_frequency_content"] = ar.AnalysisBreakdown.FrequencyAnalysis.HighFrequencyContent
	freqAnalysis["frequency_distribution"] = ar.AnalysisBreakdown.FrequencyAnalysis.FrequencyDistribution
	freqAnalysis["noise_level"] = ar.AnalysisBreakdown.FrequencyAnalysis.NoiseLevel
	freqAnalysis["explanation"] = getFrequencyAnalysisExplanation(ar.AnalysisBreakdown.FrequencyAnalysis)
	breakdown["frequency_analysis"] = freqAnalysis

	// Add quality metrics details
	qualityAnalysis := make(map[string]interface{})
	qualityAnalysis["quality_score"] = ar.QualityMetrics.QualityScore
	qualityAnalysis["sharpness"] = ar.QualityMetrics.Sharpness
	qualityAnalysis["brightness"] = ar.QualityMetrics.Brightness
	qualityAnalysis["contrast"] = ar.QualityMetrics.Contrast
	qualityAnalysis["compression_level"] = ar.QualityMetrics.CompressionLevel
	qualityAnalysis["face_size"] = ar.QualityMetrics.FaceSize
	qualityAnalysis["issues"] = ar.QualityMetrics.Issues
	qualityAnalysis["recommendations"] = ar.QualityMetrics.Recommendations
	breakdown["quality_analysis"] = qualityAnalysis

	// Add overall analysis summary
	summary := make(map[string]interface{})
	summary["is_live"] = ar.IsLive
	summary["spoof_score"] = ar.SpoofScore
	summary["confidence"] = ar.Confidence
	summary["spoof_reasons"] = ar.SpoofReasons
	summary["processing_time_ms"] = ar.ProcessingTimeMs
	breakdown["summary"] = summary

	return breakdown
}

// Helper functions for generating explanations

func getTextureAnalysisExplanation(lbpScore, lpqScore float64) string {
	if lbpScore > LBP_THRESHOLD && lpqScore > LPQ_THRESHOLD {
		return "Both LBP and LPQ analysis indicate highly uniform texture patterns, which is characteristic of printed or displayed images."
	} else if lbpScore > LBP_THRESHOLD {
		return "LBP analysis indicates uniform texture patterns, which may suggest a printed or displayed image."
	} else if lpqScore > LPQ_THRESHOLD {
		return "LPQ analysis indicates unusual phase patterns, which may suggest a printed or displayed image."
	}
	return "Texture analysis indicates natural skin texture variations consistent with a live face."
}

func getReflectionAnalysisExplanation(reflectionScore float64) string {
	if reflectionScore > REFLECTION_THRESHOLD {
		return "Reflection analysis detected unnatural light reflection patterns, which may indicate a flat surface like a screen or printed photo."
	}
	return "Reflection analysis indicates natural light interaction consistent with three-dimensional facial features."
}

func getColorAnalysisExplanation(colorScores ColorSpaceScores) string {
	avgConsistency := (colorScores.YCrCbConsistency + colorScores.HSVConsistency + colorScores.LABConsistency) / 3

	if avgConsistency > COLOR_THRESHOLD {
		return "Color analysis detected unusual color distribution across multiple color spaces, which may indicate artificial reproduction."
	}
	return "Color analysis indicates natural color distribution consistent with human skin tones."
}

func getEdgeAnalysisExplanation(edgeScores EdgeAnalysisScores) string {
	if edgeScores.EdgeDensity < MIN_EDGE_DENSITY {
		return "Edge analysis detected unusually low edge density, which may indicate a smooth artificial surface."
	} else if edgeScores.EdgeDensity > MAX_EDGE_DENSITY {
		return "Edge analysis detected unusually high edge density, which may indicate digital artifacts or noise."
	}
	return "Edge analysis indicates natural edge patterns consistent with facial features."
}

func getFrequencyAnalysisExplanation(freqScores FrequencyScores) string {
	if freqScores.HighFrequencyContent < FREQUENCY_THRESHOLD {
		return "Frequency analysis detected low high-frequency content, which may indicate a smoothed or filtered image."
	} else if freqScores.NoiseLevel > MAX_NOISE_LEVEL {
		return "Frequency analysis detected unusual noise patterns, which may indicate digital compression artifacts."
	}
	return "Frequency analysis indicates natural frequency distribution consistent with a live face."
}
