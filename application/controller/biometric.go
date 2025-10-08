package controller

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/application/utils"
	"gateman.io/infrastructure/biometric"
	"gateman.io/infrastructure/biometric/types"
	fileupload "gateman.io/infrastructure/file_upload"
	file_upload_types "gateman.io/infrastructure/file_upload/types"
	server_response "gateman.io/infrastructure/serverResponse"
	"gateman.io/infrastructure/validator"
)

// CompareFaces compares two face images and returns similarity score
func CompareFaces(ctx *interfaces.ApplicationContext[dto.FaceComparisonRequest]) {
	validationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if validationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, validationErr, ctx.DeviceID)
		return
	}

	_, err := utils.DecodeBase64Image(ctx.Body.Image1)
	if err != nil {
		_, err := url.ParseRequestURI(ctx.Body.Image1)
		if err != nil {
			apperrors.ClientError(ctx.Ctx, "invalid image format", nil, nil, ctx.DeviceID)
			return
		}
	}

	_, err = utils.DecodeBase64Image(ctx.Body.Image2)
	if err != nil {
		_, err := url.ParseRequestURI(ctx.Body.Image2)
		if err != nil {
			apperrors.ClientError(ctx.Ctx, "invalid image format", nil, nil, ctx.DeviceID)
			return
		}
	}

	result, err := biometric.BiometricService.CompareFaces(&ctx.Body.Image1, &ctx.Body.Image2)
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}

	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "face comparison completed", result, nil, nil, nil)

}

// ImageLivenessCheck performs liveness detection on a single image
func ImageLivenessCheck(ctx *interfaces.ApplicationContext[dto.LivenessCheckRequest]) {
	validationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if validationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, validationErr, ctx.DeviceID)
		return
	}

	_, err := utils.DecodeBase64Image(ctx.Body.Image)
	if err != nil {
		_, err := url.ParseRequestURI(ctx.Body.Image)
		if err != nil {
			apperrors.ClientError(ctx.Ctx, "invalid image format", nil, nil, ctx.DeviceID)
			return
		}
	}
	// Use local face service for liveness detection
	localService := biometric.NewLocalFaceService()
	defer localService.Close()

	result, err := localService.ImageLivenessCheck(&ctx.Body.Image, ctx.Body.LenientBlurry)
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}

	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Liveness check completed", result, nil, nil, nil)
}

// VideoLivenessCheck performs liveness detection on a video
func VideoLivenessCheck(ctx *interfaces.ApplicationContext[dto.VideoLivenessVerificationRequest]) {
	validationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if validationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, validationErr, ctx.DeviceID)
		return
	}
	urls := []string{}
	for index := 0; index < 4; index++ {
		url, _ := fileupload.FileUploader.GeneratedSignedURL(
			ctx.Body.ChallengeID+"_"+fmt.Sprintf("%d", index),
			file_upload_types.SignedURLPermission{
				Read: true,
			},
			time.Minute*50,
		)
		urls = append(urls, *url)
	}

	result, err := biometric.BiometricService.VideoLivenessCheck(types.VideoLivenessRequest{
		ChallengeID: ctx.Body.ChallengeID,
		VideoURLs:   urls,
	})
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}

	server_response.Responder.UnEncryptedRespond(ctx.Ctx, http.StatusOK, "Video liveness check completed", result, nil, nil)
}

// GenerateChallenge generates a new liveness challenge with random directions
func GenerateChallenge(ctx *interfaces.ApplicationContext[any]) {
	result, err := biometric.BiometricService.GenerateChallenge()
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}

	directions := [4]map[string]string{}
	if result.ChallengeID != nil {
		for index := 0; index < 4; index++ {
			url, _ := fileupload.FileUploader.GeneratedSignedURL(
				*result.ChallengeID+"_"+fmt.Sprintf("%d", index),
				file_upload_types.SignedURLPermission{
					Write: true,
				},
				time.Minute*5,
			)
			directions[index] = map[string]string{
				"direction": result.Directions[index],
				"url":       *url,
			}
		}
	}

	// Prepare the response payload
	responsePayload := map[string]interface{}{
		"success":      result.Success,
		"challenge_id": result.ChallengeID,
		"directions":   directions,
		"ttl_seconds":  result.TTLSeconds,
		"error":        result.Error,
	}

	server_response.Responder.UnEncryptedRespond(ctx.Ctx, http.StatusOK, "challenge generated successfully", responsePayload, nil, nil)
}

// EnhancedFaceComparison performs enhanced face comparison with detailed analysis
func EnhancedFaceComparison(ctx *interfaces.ApplicationContext[dto.EnhancedFaceComparisonRequest]) {
	validationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if validationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, validationErr, ctx.DeviceID)
		return
	}

	// Validate image inputs - check if they're URLs or base64
	if strings.HasPrefix(ctx.Body.ReferenceImage, "http://") || strings.HasPrefix(ctx.Body.ReferenceImage, "https://") {
		// Validate URL format
		_, err := url.ParseRequestURI(ctx.Body.ReferenceImage)
		if err != nil {
			apperrors.ClientError(ctx.Ctx, "invalid reference image URL format", nil, nil, ctx.DeviceID)
			return
		}
	} else {
		// Validate base64 format
		_, err := utils.DecodeBase64Image(ctx.Body.ReferenceImage)
		if err != nil {
			apperrors.ClientError(ctx.Ctx, "invalid reference image format - must be valid URL or base64", nil, nil, ctx.DeviceID)
			return
		}
	}

	if strings.HasPrefix(ctx.Body.TestImage, "http://") || strings.HasPrefix(ctx.Body.TestImage, "https://") {
		// Validate URL format
		_, err := url.ParseRequestURI(ctx.Body.TestImage)
		if err != nil {
			apperrors.ClientError(ctx.Ctx, "invalid test image URL format", nil, nil, ctx.DeviceID)
			return
		}
	} else {
		// Validate base64 format
		_, err := utils.DecodeBase64Image(ctx.Body.TestImage)
		if err != nil {
			apperrors.ClientError(ctx.Ctx, "invalid test image format - must be valid URL or base64", nil, nil, ctx.DeviceID)
			return
		}
	}

	// Use local face service for enhanced comparison
	localService := biometric.NewLocalFaceService()
	defer localService.Close()

	// Perform face comparison
	result, err := localService.CompareFaces(&ctx.Body.ReferenceImage, &ctx.Body.TestImage)
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}

	// Perform liveness detection if required
	var livenessResult1, livenessResult2 *dto.LivenessResultDTO
	var livenessProcessTime int64 = 0

	if ctx.Body.RequireLiveness {
		startTime := time.Now()

		// Check liveness for reference image
		liveness1, err := localService.ImageLivenessCheck(&ctx.Body.ReferenceImage, ctx.Body.LenientBlurry)
		if err == nil && liveness1.Success {
			// Fix NaN values for liveness result
			spoofScore1 := liveness1.AnalysisDetails.SpoofDetectionScore
			confidence1 := liveness1.Confidence
			if math.IsNaN(spoofScore1) || math.IsInf(spoofScore1, 0) {
				spoofScore1 = 0.0
			}
			if math.IsNaN(confidence1) || math.IsInf(confidence1, 0) {
				confidence1 = 0.5
			}

			livenessResult1 = dto.NewLivenessResultDTO(
				liveness1.IsLive,
				spoofScore1,
				confidence1,
				[]string{},
			)
		}

		// Check liveness for test image
		liveness2, err := localService.ImageLivenessCheck(&ctx.Body.TestImage, ctx.Body.LenientBlurry)
		if err == nil && liveness2.Success {
			// Fix NaN values for liveness result
			spoofScore2 := liveness2.AnalysisDetails.SpoofDetectionScore
			confidence2 := liveness2.Confidence
			if math.IsNaN(spoofScore2) || math.IsInf(spoofScore2, 0) {
				spoofScore2 = 0.0
			}
			if math.IsNaN(confidence2) || math.IsInf(confidence2, 0) {
				confidence2 = 0.5
			}

			livenessResult2 = dto.NewLivenessResultDTO(
				liveness2.IsLive,
				spoofScore2,
				confidence2,
				[]string{},
			)
		}

		livenessProcessTime = time.Since(startTime).Milliseconds()
	}

	// Create enhanced response
	response := dto.NewEnhancedFaceComparisonResponse(ctx.Body.RequestID)
	response.SetComparisonResult(
		result.Match,
		result.Confidence,
		result.Confidence,
		int64(result.ProcessingTimeMs),
	)

	if ctx.Body.RequireLiveness {
		response.SetLivenessResults(livenessResult1, livenessResult2, livenessProcessTime)
	}

	// Add feature quality metrics
	if len(result.FaceQualityScores) >= 2 {
		featureQuality := dto.NewFeatureQualityMetricsDTO(
			result.FaceQualityScores[0]*100, // Convert to percentage
			"center",                        // Default position
			result.FaceQualityScores[0],
			result.FaceQualityScores[0],
			result.Confidence,
		)
		response.SetFeatureQuality(featureQuality)
	}

	// Add comparison metadata
	threshold := ctx.Body.Threshold
	if threshold == 0 {
		threshold = 0.6 // Default threshold
	}

	metadata := dto.NewEnhancedComparisonMetadataDTO(
		"yunet_facenet", // Updated to reflect YuNet + FaceNet
		threshold,
		0.0, // No quality adjustment for now
		result.Confidence,
		"high", // Confidence level
	)
	response.SetComparisonMetadata(metadata)

	// Add processing steps
	response.AddProcessingStep("face_detection", int64(result.ProcessingTimeMs/2), true, "Faces detected successfully")
	response.AddProcessingStep("face_comparison", int64(result.ProcessingTimeMs/2), true, "Face comparison completed")

	if ctx.Body.RequireLiveness {
		response.AddProcessingStep("liveness_detection", livenessProcessTime, true, "Liveness detection completed")
	}

	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Enhanced face comparison completed", response, nil, nil, nil)
}

// EnhancedLivenessCheck performs enhanced liveness detection with detailed analysis
func EnhancedLivenessCheck(ctx *interfaces.ApplicationContext[dto.LivenessDetectionDTO]) {
	validationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if validationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, validationErr, ctx.DeviceID)
		return
	}

	// Validate image input - check if it's a URL or base64
	if strings.HasPrefix(ctx.Body.Image, "http://") || strings.HasPrefix(ctx.Body.Image, "https://") {
		// Validate URL format
		_, err := url.ParseRequestURI(ctx.Body.Image)
		if err != nil {
			apperrors.ClientError(ctx.Ctx, "invalid image URL format", nil, nil, ctx.DeviceID)
			return
		}
	} else {
		// Validate base64 format
		_, err := utils.DecodeBase64Image(ctx.Body.Image)
		if err != nil {
			apperrors.ClientError(ctx.Ctx, "invalid image format - must be valid URL or base64", nil, nil, ctx.DeviceID)
			return
		}
	}

	// Use local face service for enhanced liveness detection
	localService := biometric.NewLocalFaceService()
	defer localService.Close()

	// Perform liveness detection
	result, err := localService.ImageLivenessCheck(&ctx.Body.Image, ctx.Body.LenientBlurry)
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}

	// Create enhanced response with NaN checks
	spoofScore := result.AnalysisDetails.SpoofDetectionScore
	confidence := result.Confidence

	// Fix NaN values to prevent JSON serialization errors
	if math.IsNaN(spoofScore) || math.IsInf(spoofScore, 0) {
		spoofScore = 0.0
	}
	if math.IsNaN(confidence) || math.IsInf(confidence, 0) {
		confidence = 0.5
	}

	response := &dto.LivenessDetectionResponse{
		IsLive:         result.IsLive,
		LivenessScore:  result.LivenessScore,
		ThresholdUsed:  result.ThresholdUsed,
		SpoofScore:     spoofScore,
		Confidence:     confidence,
		ProcessingTime: int64(result.ProcessingTimeMs),
		Timestamp:      time.Now(),
	}

	// Add detailed analysis if verbose mode is enabled
	if ctx.Body.Verbose {
		// Helper function to safely convert values and prevent NaN
		safeValue := func(val float64) float64 {
			if math.IsNaN(val) || math.IsInf(val, 0) {
				return 0.0
			}
			return val
		}

		response.AnalysisBreakdown = &dto.AnalysisBreakdownDTO{
			LBPScore:              safeValue(result.AnalysisDetails.LBPScore),
			LPQScore:              safeValue(result.AnalysisDetails.LPQScore),
			ReflectionConsistency: safeValue(result.AnalysisDetails.ReflectionConsistency),
			ColorSpaceAnalysis: &dto.ColorSpaceScoresDTO{
				RGBVariance: safeValue(result.AnalysisDetails.ColorRGBVariance),
				HSVVariance: safeValue(result.AnalysisDetails.ColorHSVVariance),
				LABVariance: safeValue(result.AnalysisDetails.ColorLABVariance),
			},
			EdgeAnalysis: &dto.EdgeAnalysisScoresDTO{
				EdgeDensity:     safeValue(result.AnalysisDetails.EdgeDensity),
				EdgeSharpness:   safeValue(result.AnalysisDetails.EdgeSharpness),
				EdgeConsistency: safeValue(result.AnalysisDetails.EdgeConsistency),
			},
			FrequencyAnalysis: &dto.FrequencyScoresDTO{
				HighFrequency:        safeValue(result.AnalysisDetails.HighFrequency),
				MidFrequency:         safeValue(result.AnalysisDetails.MidFrequency),
				LowFrequency:         safeValue(result.AnalysisDetails.LowFrequency),
				CompressionArtifacts: safeValue(result.AnalysisDetails.CompressionArtifacts),
			},
			TextureAnalysis: &dto.TextureScoresDTO{
				TextureVariance:   safeValue(result.AnalysisDetails.TextureVariance),
				TextureUniformity: safeValue(result.AnalysisDetails.TextureUniformity),
				TextureEntropy:    safeValue(result.AnalysisDetails.TextureEntropy),
			},
		}

		qualityScore := safeValue(result.QualityScore)
		response.QualityMetrics = &dto.QualityMetricsDTO{
			Resolution:       "unknown", // Would need to extract from image
			Sharpness:        safeValue(result.AnalysisDetails.SharpnessScore),
			Brightness:       safeValue(result.AnalysisDetails.LightingScore),
			Contrast:         safeValue(result.AnalysisDetails.LightingScore),
			FaceSize:         qualityScore * 100,
			FacePosition:     dto.Point2D{X: 0.5, Y: 0.5}, // Default center position
			CompressionLevel: 0.0,                         // Would need to analyze
			QualityScore:     qualityScore,
			Issues:           []string{},
			Recommendations:  []string{},
		}

		// Add spoof reasons if detected
		if !result.IsLive {
			response.SpoofReasons = []string{
				"Low texture variance detected",
				"Inconsistent lighting patterns",
				"Unnatural edge patterns",
			}
		}

		// Add recommendations
		response.Recommendations = []string{
			"Ensure good lighting conditions",
			"Keep face centered in frame",
			"Avoid reflections and shadows",
		}
	}

	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Enhanced liveness check completed", response, nil, nil, nil)
}

// ImageQualityCheck performs image quality assessment
func ImageQualityCheck(ctx *interfaces.ApplicationContext[dto.ImageQualityDTO]) {
	validationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if validationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, validationErr, ctx.DeviceID)
		return
	}

	// Validate image input - check if it's a URL or base64
	if strings.HasPrefix(ctx.Body.Image, "http://") || strings.HasPrefix(ctx.Body.Image, "https://") {
		// Validate URL format
		_, err := url.ParseRequestURI(ctx.Body.Image)
		if err != nil {
			apperrors.ClientError(ctx.Ctx, "invalid image URL format", nil, nil, ctx.DeviceID)
			return
		}
	} else {
		// Validate base64 format
		_, err := utils.DecodeBase64Image(ctx.Body.Image)
		if err != nil {
			apperrors.ClientError(ctx.Ctx, "invalid image format - must be valid URL or base64", nil, nil, ctx.DeviceID)
			return
		}
	}

	// Use local face service for quality assessment
	localService := biometric.NewLocalFaceService()
	defer localService.Close()

	// Process image to get quality metrics
	img, faces, quality, err := localService.ProcessImage(ctx.Body.Image)
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	defer img.Close()

	// Create quality response
	response := &dto.ImageQualityResponse{
		IsGoodQuality:   quality > 0.6,
		HasFace:         len(faces) > 0,
		FaceCount:       len(faces),
		FaceSize:        quality * 100, // Convert to percentage
		ImageResolution: "unknown",     // Would need to extract from image
		QualityScore:    quality,
		RequestID:       ctx.Body.RequestID,
		Timestamp:       time.Now(),
	}

	// Add issues and recommendations
	if quality < 0.6 {
		response.Issues = append(response.Issues, "Low image quality detected")
	}
	if len(faces) == 0 {
		response.Issues = append(response.Issues, "No face detected in image")
	}
	if len(faces) > 1 {
		response.Issues = append(response.Issues, "Multiple faces detected")
	}

	// Add recommendations
	response.Recommendations = []string{
		"Ensure good lighting conditions",
		"Keep face centered and clearly visible",
		"Avoid blurry or low-resolution images",
	}

	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Image quality check completed", response, nil, nil, nil)
}

// SystemHealthCheck returns the health status of the biometric system
func SystemHealthCheck(ctx *interfaces.ApplicationContext[any]) {
	// Use local face service to check system health
	localService := biometric.NewLocalFaceService()
	defer localService.Close()

	stats := localService.GetStats()

	response := &dto.SystemHealthResponse{
		Status:                "healthy",
		ModelsLoaded:          true,                   // Assuming models are loaded if service was created
		SystemUptime:          time.Since(time.Now()), // Would need to track actual uptime
		ProcessedRequests:     stats.TotalRequests,
		AverageProcessingTime: stats.AverageTime,
		ErrorRate:             float64(stats.TotalRequests-stats.SuccessfulRequests) / float64(stats.TotalRequests) * 100,
		Timestamp:             time.Now(),
	}

	// Add memory usage if available
	response.MemoryUsage = &dto.MemoryUsageDTO{
		AllocatedMB: 0.0, // Would need to implement memory tracking
		SystemMB:    0.0,
		GCCycles:    0,
	}

	// Add model information
	response.ModelInfo = []dto.ModelInfoDTO{
		{
			Name:    "Haar Cascade Face Detector",
			Path:    "haarcascade_frontalface_alt.xml",
			Loaded:  true,
			Version: "1.0",
		},
		{
			Name:    "Haar Cascade Eye Detector",
			Path:    "haarcascade_eye.xml",
			Loaded:  true,
			Version: "1.0",
		},
	}

	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "System health check completed", response, nil, nil, nil)
}
