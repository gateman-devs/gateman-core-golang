package routev1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"gateman.io/application/interfaces"
	"gateman.io/infrastructure/facematch"
	serverResponse "gateman.io/infrastructure/serverResponse"
	"github.com/gin-gonic/gin"
)

// ProductionLivenessRouter creates production-ready API endpoints for liveness detection and face comparison
func ProductionLivenessRouter(router *gin.RouterGroup) {
	livenessRouter := router.Group("/production/liveness")

	livenessRouter.Use(productionRequestValidationMiddleware())
	livenessRouter.Use(productionAuditLoggingMiddleware())

	{
		// Production Liveness Detection Endpoint
		livenessRouter.POST("/detect", func(ctx *gin.Context) {
			// Get application context
			// appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			fmt.Println("omo")
			var body ProductionLivenessRequest
			if err := bindAndValidateProductionRequest(ctx, &body); err != nil {
				handleValidationError(ctx, "Request binding failed", err)
				return
			}

			fmt.Println("before")
			// Additional validation for liveness detection
			if err := validateProductionImageInput(body.Image); err != nil {
				handleValidationError(ctx, "Image validation failed", err)
				return
			}

			// Process liveness detection
			processProductionLivenessDetection(ctx, body)
		})

		// Production Liveness Detection with Verbose Analysis
		livenessRouter.POST("/detect/verbose", func(ctx *gin.Context) {
			// Get application context

			var body ProductionLivenessRequest
			if err := bindAndValidateProductionRequest(ctx, &body); err != nil {
				handleValidationError(ctx, "Request binding failed", err)
				return
			}

			// Force verbose mode
			body.Verbose = true

			// Additional validation for verbose analysis
			if err := validateProductionImageInput(body.Image); err != nil {
				handleValidationError(ctx, "Image validation failed", err)
				return
			}

			// Process verbose liveness detection
			processProductionLivenessDetection(ctx, body)
		})

		// Production Face Comparison Endpoint
		livenessRouter.POST("/compare", func(ctx *gin.Context) {
			// Get application context

			var body ProductionFaceComparisonRequest
			if err := bindAndValidateProductionRequest(ctx, &body); err != nil {
				handleValidationError(ctx, "Request binding failed", err)
				return
			}

			// Validate both images
			if err := validateProductionImageInput(body.ReferenceImage); err != nil {
				handleValidationError(ctx, "Reference image validation failed", err)
				return
			}

			if err := validateProductionImageInput(body.TestImage); err != nil {
				handleValidationError(ctx, "Test image validation failed", err)
				return
			}

			// Validate threshold if provided
			if body.Threshold != 0 && (body.Threshold < 0.0 || body.Threshold > 1.0) {
				handleValidationError(ctx, "Threshold validation failed", fmt.Errorf("threshold must be between 0.0 and 1.0, got: %f", body.Threshold))
				return
			}

			// Process face comparison
			processProductionFaceComparison(ctx, body)
		})

		// Production Image Quality Assessment Endpoint
		livenessRouter.POST("/quality", func(ctx *gin.Context) {
			// Get application context
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])

			var body ProductionImageQualityRequest
			if err := bindAndValidateProductionRequest(ctx, &body); err != nil {
				handleValidationError(ctx, "Request binding failed", err)
				return
			}

			// Validate image
			if err := validateProductionImageInput(body.Image); err != nil {
				handleValidationError(ctx, "Image validation failed", err)
				return
			}

			// Process image quality assessment
			processProductionImageQuality(ctx, appContext, body)
		})

		// Production Batch Liveness Detection (with heavy rate limiting)
		livenessRouter.POST("/batch/detect", func(ctx *gin.Context) {
			// Get application context
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])

			var body ProductionBatchLivenessRequest
			if err := bindAndValidateProductionRequest(ctx, &body); err != nil {
				handleValidationError(ctx, "Request binding failed", err)
				return
			}

			// Validate batch size
			if len(body.Images) == 0 {
				handleValidationError(ctx, "Batch validation failed", fmt.Errorf("images array cannot be empty"))
				return
			}

			if len(body.Images) > 10 {
				handleValidationError(ctx, "Batch validation failed", fmt.Errorf("batch size cannot exceed 10 images, got: %d", len(body.Images)))
				return
			}

			// Validate each image
			for i, image := range body.Images {
				if err := validateProductionImageInput(image); err != nil {
					handleValidationError(ctx, "Image validation failed", fmt.Errorf("image %d validation failed: %w", i+1, err))
					return
				}
			}

			// Process batch liveness detection
			processProductionBatchLiveness(ctx, appContext, body)
		})

		// Production System Health Endpoint (no rate limiting for monitoring)
		livenessRouter.GET("/health", func(ctx *gin.Context) {
			processProductionSystemHealth(ctx)
		})

		// Production System Metrics Endpoint
		livenessRouter.GET("/metrics", func(ctx *gin.Context) {
			processProductionSystemMetrics(ctx)
		})

		// Production Configuration Endpoint (development only)
		if os.Getenv("ENV") == "dev" {
			livenessRouter.GET("/config", func(ctx *gin.Context) {
				processProductionSystemConfig(ctx)
			})
		}
	}
}

// Production request structures
type ProductionLivenessRequest struct {
	Image     string `json:"image" validate:"required"`
	RequestID string `json:"request_id,omitempty"`
	Verbose   bool   `json:"verbose,omitempty"`
}

type ProductionFaceComparisonRequest struct {
	ReferenceImage string  `json:"reference_image" validate:"required"`
	TestImage      string  `json:"test_image" validate:"required"`
	Threshold      float64 `json:"threshold,omitempty"`
	RequestID      string  `json:"request_id,omitempty"`
}

type ProductionImageQualityRequest struct {
	Image     string `json:"image" validate:"required"`
	RequestID string `json:"request_id,omitempty"`
}

type ProductionBatchLivenessRequest struct {
	Images    []string `json:"images" validate:"required,min=1,max=10"`
	RequestID string   `json:"request_id,omitempty"`
	Verbose   bool     `json:"verbose,omitempty"`
}

// Production response structures
type ProductionLivenessResponse struct {
	IsLive            bool                         `json:"is_live"`
	SpoofScore        float64                      `json:"spoof_score"`
	Confidence        float64                      `json:"confidence"`
	ProcessingTime    int64                        `json:"processing_time_ms"`
	AnalysisBreakdown *ProductionAnalysisBreakdown `json:"analysis_breakdown,omitempty"`
	QualityMetrics    *ProductionQualityMetrics    `json:"quality_metrics,omitempty"`
	SpoofReasons      []string                     `json:"spoof_reasons,omitempty"`
	Recommendations   []string                     `json:"recommendations,omitempty"`
	RequestID         string                       `json:"request_id"`
	Timestamp         time.Time                    `json:"timestamp"`
	Error             string                       `json:"error,omitempty"`
}

type ProductionFaceComparisonResponse struct {
	IsMatch          bool                      `json:"is_match"`
	Similarity       float64                   `json:"similarity"`
	Confidence       float64                   `json:"confidence"`
	ProcessingTime   int64                     `json:"processing_time_ms"`
	ReferenceQuality *ProductionQualityMetrics `json:"reference_quality,omitempty"`
	TestQuality      *ProductionQualityMetrics `json:"test_quality,omitempty"`
	MatchMetadata    *ProductionMatchMetadata  `json:"match_metadata,omitempty"`
	RequestID        string                    `json:"request_id"`
	Timestamp        time.Time                 `json:"timestamp"`
	Error            string                    `json:"error,omitempty"`
}

type ProductionImageQualityResponse struct {
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

type ProductionBatchLivenessResponse struct {
	Results        []ProductionLivenessResponse `json:"results"`
	TotalImages    int                          `json:"total_images"`
	SuccessCount   int                          `json:"success_count"`
	ErrorCount     int                          `json:"error_count"`
	ProcessingTime int64                        `json:"total_processing_time_ms"`
	RequestID      string                       `json:"request_id"`
	Timestamp      time.Time                    `json:"timestamp"`
}

type ProductionAnalysisBreakdown struct {
	LBPScore              float64                       `json:"lbp_score"`
	LPQScore              float64                       `json:"lpq_score"`
	ReflectionConsistency float64                       `json:"reflection_consistency"`
	ColorSpaceAnalysis    *ProductionColorSpaceScores   `json:"color_space_analysis,omitempty"`
	EdgeAnalysis          *ProductionEdgeAnalysisScores `json:"edge_analysis,omitempty"`
	FrequencyAnalysis     *ProductionFrequencyScores    `json:"frequency_analysis,omitempty"`
	TextureAnalysis       *ProductionTextureScores      `json:"texture_analysis,omitempty"`
}

type ProductionQualityMetrics struct {
	Resolution       string   `json:"resolution"`
	Sharpness        float64  `json:"sharpness"`
	Brightness       float64  `json:"brightness"`
	Contrast         float64  `json:"contrast"`
	FaceSize         float64  `json:"face_size_percent"`
	CompressionLevel float64  `json:"compression_level"`
	QualityScore     float64  `json:"quality_score"`
	Issues           []string `json:"issues,omitempty"`
	Recommendations  []string `json:"recommendations,omitempty"`
}

type ProductionMatchMetadata struct {
	SimilarityMethod string  `json:"similarity_method"`
	ThresholdUsed    float64 `json:"threshold_used"`
	ConfidenceLevel  string  `json:"confidence_level"`
}

type ProductionColorSpaceScores struct {
	RGBVariance float64 `json:"rgb_variance"`
	HSVVariance float64 `json:"hsv_variance"`
	LABVariance float64 `json:"lab_variance"`
}

type ProductionEdgeAnalysisScores struct {
	EdgeDensity     float64 `json:"edge_density"`
	EdgeSharpness   float64 `json:"edge_sharpness"`
	EdgeConsistency float64 `json:"edge_consistency"`
}

type ProductionFrequencyScores struct {
	HighFrequency        float64 `json:"high_frequency"`
	MidFrequency         float64 `json:"mid_frequency"`
	LowFrequency         float64 `json:"low_frequency"`
	CompressionArtifacts float64 `json:"compression_artifacts"`
}

type ProductionTextureScores struct {
	TextureVariance   float64 `json:"texture_variance"`
	TextureUniformity float64 `json:"texture_uniformity"`
	TextureEntropy    float64 `json:"texture_entropy"`
}

// bindAndValidateProductionRequest binds and validates production requests
func bindAndValidateProductionRequest(ctx *gin.Context, body interface{}) error {
	if os.Getenv("ENV") != "dev" {
		decryptedPayload, exists := ctx.Get("DecryptedBody")
		if !exists {
			return fmt.Errorf("encrypted payload required in production")
		}
		return json.Unmarshal([]byte(decryptedPayload.(string)), body)
	} else {
		return ctx.ShouldBindJSON(body)
	}
}

// validateProductionImageInput validates image input with production-grade checks
func validateProductionImageInput(image string) error {
	if image == "" {
		return fmt.Errorf("image cannot be empty")
	}

	// Check if it's a URL
	if strings.HasPrefix(image, "http://") || strings.HasPrefix(image, "https://") {
		// Basic URL validation
		if len(image) > 2048 {
			return fmt.Errorf("image URL too long (max 2048 characters)")
		}

		// Additional URL security checks
		if strings.Contains(strings.ToLower(image), "localhost") ||
			strings.Contains(strings.ToLower(image), "127.0.0.1") ||
			strings.Contains(strings.ToLower(image), "0.0.0.0") {
			return fmt.Errorf("localhost URLs not allowed for security reasons")
		}

		return nil
	}

	// Check if it's base64
	if strings.Contains(image, ",") {
		// Data URL format
		parts := strings.Split(image, ",")
		if len(parts) != 2 {
			return fmt.Errorf("invalid data URL format")
		}
		image = parts[1]
	}

	// Validate base64 length (approximate size check)
	if len(image) > 67108864 { // ~50MB in base64
		return fmt.Errorf("image too large (max ~50MB)")
	}

	if len(image) < 100 {
		return fmt.Errorf("image too small (minimum 100 characters)")
	}

	// Basic base64 validation
	if len(image)%4 != 0 {
		return fmt.Errorf("invalid base64 encoding")
	}

	return nil
}

// generateRequestID generates a unique request ID
func generateRequestID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// handleValidationError handles validation errors consistently
func handleValidationError(ctx *gin.Context, message string, err error) {
	requestID := generateRequestID("error")

	serverResponse.Responder.Respond(
		ctx,
		http.StatusBadRequest,
		message,
		map[string]interface{}{
			"error":      "VALIDATION_ERROR",
			"message":    err.Error(),
			"details":    message,
			"request_id": requestID,
			"timestamp":  time.Now(),
		},
		nil,
		nil,
		nil,
	)
}

// productionRequestValidationMiddleware validates production requests
func productionRequestValidationMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Check content type for POST requests
		if ctx.Request.Method == "POST" {
			contentType := ctx.GetHeader("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				serverResponse.Responder.Respond(
					ctx,
					http.StatusBadRequest,
					"Invalid content type",
					map[string]interface{}{
						"error":    "INVALID_CONTENT_TYPE",
						"message":  "Content-Type must be application/json",
						"received": contentType,
					},
					nil,
					nil,
					nil,
				)
				ctx.Abort()
				return
			}
		}

		// Check request size (50MB limit for production)
		if ctx.Request.ContentLength > 50*1024*1024 {
			serverResponse.Responder.Respond(
				ctx,
				http.StatusRequestEntityTooLarge,
				"Request too large",
				map[string]interface{}{
					"error":   "REQUEST_TOO_LARGE",
					"message": "Request size exceeds 50MB limit",
					"size":    ctx.Request.ContentLength,
				},
				nil,
				nil,
				nil,
			)
			ctx.Abort()
			return
		}

		// Validate required headers
		requiredHeaders := []string{"X-App-Id"}
		for _, header := range requiredHeaders {
			if ctx.GetHeader(header) == "" {
				serverResponse.Responder.Respond(
					ctx,
					http.StatusBadRequest,
					"Missing required header",
					map[string]interface{}{
						"error":   "MISSING_HEADER",
						"message": fmt.Sprintf("Header %s is required", header),
						"header":  header,
					},
					nil,
					nil,
					nil,
				)
				ctx.Abort()
				return
			}
		}

		ctx.Next()
	}
}

// productionAuditLoggingMiddleware logs audit information for production requests
func productionAuditLoggingMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startTime := time.Now()
		ctx.Set("StartTime", startTime)

		// Generate request ID if not provided
		requestID := ctx.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("prod_%d", time.Now().UnixNano())
			ctx.Header("X-Request-ID", requestID)
		}
		ctx.Set("RequestID", requestID)

		// Process request
		ctx.Next()

		// Log audit information
		processingTime := time.Since(startTime)

		auditLog := map[string]interface{}{
			"timestamp":       startTime,
			"request_id":      requestID,
			"method":          ctx.Request.Method,
			"path":            ctx.Request.URL.Path,
			"client_ip":       ctx.ClientIP(),
			"user_agent":      ctx.GetHeader("User-Agent"),
			"app_id":          ctx.GetHeader("X-App-Id"),
			"processing_time": processingTime.Milliseconds(),
			"status_code":     ctx.Writer.Status(),
			"response_size":   ctx.Writer.Size(),
		}

		// In production, this would go to a proper audit system
		fmt.Printf("PRODUCTION_AUDIT: %+v\n", auditLog)
	}
}

// getConfidenceLevel determines confidence level from similarity score
func getConfidenceLevel(similarity float64) string {
	if similarity >= 0.9 {
		return "high"
	} else if similarity >= 0.7 {
		return "medium"
	} else if similarity >= 0.5 {
		return "low"
	}
	return "very_low"
}

// processProductionLivenessDetection processes liveness detection requests
func processProductionLivenessDetection(ctx *gin.Context, request ProductionLivenessRequest) {
	startTime := time.Now()
	fmt.Println("start time")
	fmt.Println("start time")
	fmt.Println("start time")
	fmt.Println("start time")
	fmt.Println("start time")
	fmt.Println("start time")
	// Generate request ID if not provided
	requestID := request.RequestID
	if requestID == "" {
		requestID = generateRequestID("liveness")
	}

	// Validate that face matcher is initialized
	if facematch.GlobalFaceMatcher == nil {
		serverResponse.Responder.Respond(
			ctx,
			http.StatusInternalServerError,
			"Biometric service not initialized",
			ProductionLivenessResponse{
				RequestID: requestID,
				Timestamp: time.Now(),
				Error:     "Biometric service not initialized",
			},
			nil,
			nil,
			nil,
		)
		return
	}

	// Use goroutines for concurrent processing when verbose mode is enabled
	if request.Verbose {
		// Concurrent processing for verbose mode
		type verboseResult struct {
			livenessResult facematch.AdvancedAntiSpoofResult
			qualityResult  facematch.ImageQualityResult
		}

		resultChan := make(chan verboseResult, 1)

		go func() {
			var vResult verboseResult

			// Create channels for concurrent operations
			livenessChan := make(chan facematch.AdvancedAntiSpoofResult, 1)
			qualityChan := make(chan facematch.ImageQualityResult, 1)

			// Start liveness detection
			go func() {
				livenessChan <- facematch.GlobalFaceMatcher.DetectAdvancedAntiSpoof(
					request.Image,
					requestID,
					true,
				)
			}()

			// Start quality assessment
			go func() {
				qualityChan <- facematch.GlobalFaceMatcher.VerifyImageQuality(request.Image)
			}()

			// Collect results
			vResult.livenessResult = <-livenessChan
			vResult.qualityResult = <-qualityChan

			resultChan <- vResult
		}()

		// Get concurrent results
		concurrentResult := <-resultChan
		result := concurrentResult.livenessResult
		qualityResult := concurrentResult.qualityResult

		processingTime := time.Since(startTime).Milliseconds()

		// Handle errors
		if result.Error != "" {
			serverResponse.Responder.Respond(
				ctx,
				http.StatusBadRequest,
				"Liveness detection failed",
				ProductionLivenessResponse{
					RequestID:      requestID,
					Timestamp:      time.Now(),
					ProcessingTime: processingTime,
					Error:          result.Error,
				},
				nil,
				nil,
				nil,
			)
			return
		}

		// Build response with quality metrics
		response := ProductionLivenessResponse{
			IsLive:         result.IsReal,
			SpoofScore:     result.SpoofScore,
			Confidence:     result.Confidence,
			ProcessingTime: processingTime,
			SpoofReasons:   result.SpoofReasons,
			RequestID:      requestID,
			Timestamp:      time.Now(),
		}

		// Add detailed analysis
		response.AnalysisBreakdown = convertToProductionAnalysisBreakdown(result.AnalysisBreakdown)

		// Add quality metrics if available
		if qualityResult.Error == "" {
			response.QualityMetrics = &ProductionQualityMetrics{
				Resolution:   qualityResult.ImageResolution,
				QualityScore: qualityResult.QualityScore,
				Issues:       qualityResult.Issues,
			}
		}

		serverResponse.Responder.Respond(
			ctx,
			http.StatusOK,
			"Liveness detection completed successfully",
			response,
			nil,
			nil,
			nil,
		)
		return
	}

	// Standard processing for non-verbose mode
	basicResult := facematch.GlobalFaceMatcher.DetectAntiSpoof(request.Image)
	// Convert to AdvancedAntiSpoofResult for consistency
	result := facematch.AdvancedAntiSpoofResult{
		IsReal:       basicResult.IsReal,
		SpoofScore:   basicResult.SpoofScore,
		Confidence:   basicResult.Confidence,
		HasFace:      basicResult.HasFace,
		ProcessTime:  basicResult.ProcessTime,
		SpoofReasons: basicResult.SpoofReasons,
		Error:        basicResult.Error,
	}

	processingTime := time.Since(startTime).Milliseconds()

	// Handle errors
	if result.Error != "" {
		serverResponse.Responder.Respond(
			ctx,
			http.StatusBadRequest,
			"Liveness detection failed",
			ProductionLivenessResponse{
				RequestID:      requestID,
				Timestamp:      time.Now(),
				ProcessingTime: processingTime,
				Error:          result.Error,
			},
			nil,
			nil,
			nil,
		)
		return
	}

	// Build response
	response := ProductionLivenessResponse{
		IsLive:         result.IsReal,
		SpoofScore:     result.SpoofScore,
		Confidence:     result.Confidence,
		ProcessingTime: processingTime,
		SpoofReasons:   result.SpoofReasons,
		RequestID:      requestID,
		Timestamp:      time.Now(),
	}

	serverResponse.Responder.Respond(
		ctx,
		http.StatusOK,
		"Liveness detection completed successfully",
		response,
		nil,
		nil,
		nil,
	)
}

// processProductionFaceComparison processes face comparison requests
func processProductionFaceComparison(ctx *gin.Context, request ProductionFaceComparisonRequest) {
	startTime := time.Now()

	// Generate request ID if not provided
	requestID := request.RequestID
	if requestID == "" {
		requestID = generateRequestID("compare")
	}

	// Validate that face matcher is initialized
	if facematch.GlobalFaceMatcher == nil {
		serverResponse.Responder.Respond(
			ctx,
			http.StatusInternalServerError,
			"Biometric service not initialized",
			ProductionFaceComparisonResponse{
				RequestID: requestID,
				Timestamp: time.Now(),
				Error:     "Biometric service not initialized",
			},
			nil,
			nil,
			nil,
		)
		return
	}

	// Set default threshold if not provided
	threshold := request.Threshold
	if threshold == 0 {
		threshold = 0.7 // Default threshold from design
	}

	// Use goroutines to perform face comparison and quality assessments concurrently
	type comparisonResult struct {
		compareResult facematch.CompareResult
		refQuality    facematch.ImageQualityResult
		testQuality   facematch.ImageQualityResult
		err           error
	}

	resultChan := make(chan comparisonResult, 1)

	go func() {
		var result comparisonResult

		// Create channels for concurrent operations
		compareChan := make(chan facematch.CompareResult, 1)
		refQualityChan := make(chan facematch.ImageQualityResult, 1)
		testQualityChan := make(chan facematch.ImageQualityResult, 1)

		// Start face comparison
		go func() {
			compareChan <- facematch.GlobalFaceMatcher.Compare(
				request.ReferenceImage,
				request.TestImage,
				threshold,
			)
		}()

		// Start reference image quality assessment
		go func() {
			refQualityChan <- facematch.GlobalFaceMatcher.VerifyImageQuality(request.ReferenceImage)
		}()

		// Start test image quality assessment
		go func() {
			testQualityChan <- facematch.GlobalFaceMatcher.VerifyImageQuality(request.TestImage)
		}()

		// Collect results
		result.compareResult = <-compareChan
		result.refQuality = <-refQualityChan
		result.testQuality = <-testQualityChan

		resultChan <- result
	}()

	// Get the concurrent results
	concurrentResult := <-resultChan
	processingTime := time.Since(startTime).Milliseconds()

	// Handle errors
	if concurrentResult.compareResult.Error != "" {
		serverResponse.Responder.Respond(
			ctx,
			http.StatusBadRequest,
			"Face comparison failed",
			ProductionFaceComparisonResponse{
				RequestID:      requestID,
				Timestamp:      time.Now(),
				ProcessingTime: processingTime,
				Error:          concurrentResult.compareResult.Error,
			},
			nil,
			nil,
			nil,
		)
		return
	}

	// Build quality metrics from concurrent assessments
	var refQuality, testQuality *ProductionQualityMetrics
	if concurrentResult.refQuality.Error == "" {
		refQuality = &ProductionQualityMetrics{
			Resolution:   concurrentResult.refQuality.ImageResolution,
			QualityScore: concurrentResult.refQuality.QualityScore,
			Issues:       concurrentResult.refQuality.Issues,
		}
	}
	if concurrentResult.testQuality.Error == "" {
		testQuality = &ProductionQualityMetrics{
			Resolution:   concurrentResult.testQuality.ImageResolution,
			QualityScore: concurrentResult.testQuality.QualityScore,
			Issues:       concurrentResult.testQuality.Issues,
		}
	}

	// Build response
	response := ProductionFaceComparisonResponse{
		IsMatch:          concurrentResult.compareResult.Match,
		Similarity:       concurrentResult.compareResult.Similarity,
		Confidence:       concurrentResult.compareResult.Similarity, // Using similarity as confidence for now
		ProcessingTime:   processingTime,
		ReferenceQuality: refQuality,
		TestQuality:      testQuality,
		RequestID:        requestID,
		Timestamp:        time.Now(),
		MatchMetadata: &ProductionMatchMetadata{
			SimilarityMethod: "cosine",
			ThresholdUsed:    threshold,
			ConfidenceLevel:  getConfidenceLevel(concurrentResult.compareResult.Similarity),
		},
	}

	serverResponse.Responder.Respond(
		ctx,
		http.StatusOK,
		"Face comparison completed successfully",
		response,
		nil,
		nil,
		nil,
	)
}

// processProductionImageQuality processes image quality assessment requests
func processProductionImageQuality(ctx *gin.Context, appContext *interfaces.ApplicationContext[any], request ProductionImageQualityRequest) {
	startTime := time.Now()

	// Generate request ID if not provided
	requestID := request.RequestID
	if requestID == "" {
		requestID = generateRequestID("quality")
	}

	// Validate that face matcher is initialized
	if facematch.GlobalFaceMatcher == nil {
		serverResponse.Responder.Respond(
			ctx,
			http.StatusInternalServerError,
			"Biometric service not initialized",
			ProductionImageQualityResponse{
				RequestID: requestID,
				Timestamp: time.Now(),
				Error:     "Biometric service not initialized",
			},
			nil,
			nil,
			nil,
		)
		return
	}

	// Perform image quality verification
	result := facematch.GlobalFaceMatcher.VerifyImageQuality(request.Image)

	processingTime := time.Since(startTime).Milliseconds()

	// Handle errors
	if result.Error != "" {
		serverResponse.Responder.Respond(
			ctx,
			http.StatusBadRequest,
			"Image quality verification failed",
			ProductionImageQualityResponse{
				RequestID: requestID,
				Timestamp: time.Now(),
				Error:     result.Error,
			},
			nil,
			nil,
			nil,
		)
		return
	}

	// Build response
	response := ProductionImageQualityResponse{
		IsGoodQuality:   result.IsGoodQuality,
		HasFace:         result.HasFace,
		FaceCount:       result.FaceCount,
		FaceSize:        result.FaceSize,
		ImageResolution: result.ImageResolution,
		QualityScore:    result.QualityScore,
		Issues:          result.Issues,
		Recommendations: result.Recommendations,
		RequestID:       requestID,
		Timestamp:       time.Now(),
	}

	// Note: processingTime is calculated but not used in this response structure
	// This is intentional as the ImageQualityResponse doesn't include processing time
	_ = processingTime

	serverResponse.Responder.Respond(
		ctx,
		http.StatusOK,
		"Image quality verification completed successfully",
		response,
		nil,
		nil,
		nil,
	)
}

// processProductionBatchLiveness processes batch liveness detection requests
func processProductionBatchLiveness(ctx *gin.Context, appContext *interfaces.ApplicationContext[any], request ProductionBatchLivenessRequest) {
	startTime := time.Now()

	// Generate request ID if not provided
	requestID := request.RequestID
	if requestID == "" {
		requestID = generateRequestID("batch")
	}

	// Validate that face matcher is initialized
	if facematch.GlobalFaceMatcher == nil {
		serverResponse.Responder.Respond(
			ctx,
			http.StatusInternalServerError,
			"Biometric service not initialized",
			ProductionBatchLivenessResponse{
				RequestID: requestID,
				Timestamp: time.Now(),
			},
			nil,
			nil,
			nil,
		)
		return
	}

	results := make([]ProductionLivenessResponse, len(request.Images))
	successCount := 0
	errorCount := 0

	// Use goroutines to process images concurrently for better performance
	type imageResult struct {
		index  int
		result ProductionLivenessResponse
	}

	resultChan := make(chan imageResult, len(request.Images))

	// Process each image concurrently
	for i, image := range request.Images {
		go func(index int, img string) {
			imageRequestID := fmt.Sprintf("%s_img_%d", requestID, index+1)

			var result facematch.AdvancedAntiSpoofResult

			if request.Verbose {
				result = facematch.GlobalFaceMatcher.DetectAdvancedAntiSpoof(
					img,
					imageRequestID,
					true,
				)
			} else {
				basicResult := facematch.GlobalFaceMatcher.DetectAntiSpoof(img)
				result = facematch.AdvancedAntiSpoofResult{
					IsReal:       basicResult.IsReal,
					SpoofScore:   basicResult.SpoofScore,
					Confidence:   basicResult.Confidence,
					HasFace:      basicResult.HasFace,
					ProcessTime:  basicResult.ProcessTime,
					SpoofReasons: basicResult.SpoofReasons,
					Error:        basicResult.Error,
				}
			}

			// Build individual result
			individualResult := ProductionLivenessResponse{
				IsLive:         result.IsReal,
				SpoofScore:     result.SpoofScore,
				Confidence:     result.Confidence,
				ProcessingTime: result.ProcessTime,
				SpoofReasons:   result.SpoofReasons,
				RequestID:      imageRequestID,
				Timestamp:      time.Now(),
			}

			if result.Error != "" {
				individualResult.Error = result.Error
			}

			// Add detailed analysis if verbose mode
			if request.Verbose {
				individualResult.AnalysisBreakdown = convertToProductionAnalysisBreakdown(result.AnalysisBreakdown)
				// Note: Recommendations are included in the analysis breakdown
			}

			resultChan <- imageResult{index: index, result: individualResult}
		}(i, image)
	}

	// Collect results from goroutines
	for i := 0; i < len(request.Images); i++ {
		imgResult := <-resultChan
		results[imgResult.index] = imgResult.result

		if imgResult.result.Error != "" {
			errorCount++
		} else {
			successCount++
		}
	}

	totalProcessingTime := time.Since(startTime).Milliseconds()

	// Build batch response
	response := ProductionBatchLivenessResponse{
		Results:        results,
		TotalImages:    len(request.Images),
		SuccessCount:   successCount,
		ErrorCount:     errorCount,
		ProcessingTime: totalProcessingTime,
		RequestID:      requestID,
		Timestamp:      time.Now(),
	}

	serverResponse.Responder.Respond(
		ctx,
		http.StatusOK,
		"Batch liveness detection completed",
		response,
		nil,
		nil,
		nil,
	)
}

// processProductionSystemHealth processes system health requests
func processProductionSystemHealth(ctx *gin.Context) {
	// Check if models are loaded
	modelsLoaded := facematch.GlobalFaceMatcher != nil

	// Determine overall status
	status := "healthy"
	if !modelsLoaded {
		status = "unhealthy"
	}

	// Build model info
	var modelInfo []map[string]interface{}
	if modelsLoaded {
		modelInfo = []map[string]interface{}{
			{
				"name":    "YuNet Face Detector",
				"path":    "./models/yunet.onnx",
				"loaded":  true,
				"version": "1.0",
			},
			{
				"name":    "ArcFace Feature Extractor",
				"path":    "./models/arcface.onnx",
				"loaded":  true,
				"version": "1.0",
			},
		}
	}

	response := map[string]interface{}{
		"status":        status,
		"models_loaded": modelsLoaded,
		"model_info":    modelInfo,
		"timestamp":     time.Now(),
		"version":       "1.0.0",
	}

	serverResponse.Responder.Respond(
		ctx,
		http.StatusOK,
		"System health retrieved successfully",
		response,
		nil,
		nil,
		nil,
	)
}

// processProductionSystemMetrics processes system metrics requests
func processProductionSystemMetrics(ctx *gin.Context) {
	response := map[string]interface{}{
		"timestamp":                time.Now(),
		"uptime_seconds":           time.Since(time.Now().Add(-time.Hour)).Seconds(), // Placeholder
		"requests_per_minute":      0,                                                // Would be calculated from actual metrics
		"average_response_time_ms": 0,                                                // Would be calculated from actual metrics
		"error_rate_percent":       0,                                                // Would be calculated from actual metrics
		"active_connections":       0,                                                // Would be calculated from actual metrics
		"memory_usage_mb":          0,                                                // Would be calculated from actual metrics
	}

	serverResponse.Responder.Respond(
		ctx,
		http.StatusOK,
		"System metrics retrieved successfully",
		response,
		nil,
		nil,
		nil,
	)
}

// processProductionSystemConfig processes system configuration requests (dev only)
func processProductionSystemConfig(ctx *gin.Context) {
	config := map[string]interface{}{
		"environment":   os.Getenv("ENV"),
		"models_loaded": facematch.GlobalFaceMatcher != nil,
		"rate_limits": map[string]interface{}{
			"standard_requests_per_minute": 10,
			"heavy_requests_per_2_minutes": 5,
		},
		"image_limits": map[string]interface{}{
			"max_size_mb":       50,
			"supported_formats": []string{"JPEG", "PNG", "WebP"},
			"max_dimension":     4000,
			"min_dimension":     50,
		},
		"thresholds": map[string]interface{}{
			"face_comparison_threshold": 0.7,
			"quality_score_threshold":   0.7,
		},
	}

	serverResponse.Responder.Respond(
		ctx,
		http.StatusOK,
		"System configuration retrieved successfully",
		config,
		nil,
		nil,
		nil,
	)
}

// convertToProductionAnalysisBreakdown converts facematch analysis breakdown to production format
func convertToProductionAnalysisBreakdown(breakdown facematch.AnalysisBreakdown) *ProductionAnalysisBreakdown {
	if breakdown == (facematch.AnalysisBreakdown{}) {
		return nil
	}

	return &ProductionAnalysisBreakdown{
		LBPScore:              breakdown.LBPScore,
		LPQScore:              breakdown.LPQScore,
		ReflectionConsistency: breakdown.ReflectionConsistency,
		ColorSpaceAnalysis: &ProductionColorSpaceScores{
			RGBVariance: breakdown.ColorSpaceAnalysis.YCrCbConsistency, // Map YCrCb to RGB for API consistency
			HSVVariance: breakdown.ColorSpaceAnalysis.HSVConsistency,
			LABVariance: breakdown.ColorSpaceAnalysis.LABConsistency,
		},
		EdgeAnalysis: &ProductionEdgeAnalysisScores{
			EdgeDensity:     breakdown.EdgeAnalysis.EdgeDensity,
			EdgeSharpness:   breakdown.EdgeAnalysis.EdgeSharpness,
			EdgeConsistency: breakdown.EdgeAnalysis.EdgeOrientation, // Map EdgeOrientation to EdgeConsistency
		},
		FrequencyAnalysis: &ProductionFrequencyScores{
			HighFrequency:        breakdown.FrequencyAnalysis.HighFrequencyContent,
			MidFrequency:         breakdown.FrequencyAnalysis.FrequencyDistribution,
			LowFrequency:         breakdown.FrequencyAnalysis.NoiseLevel,
			CompressionArtifacts: breakdown.FrequencyAnalysis.NoiseLevel, // Use NoiseLevel as compression artifacts indicator
		},
		// TextureAnalysis is not available in the current AnalysisBreakdown, so we'll omit it
		TextureAnalysis: nil,
	}
}
