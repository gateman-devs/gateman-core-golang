package controller

import (
	"math"
	"net/http"
	"net/url"
	"strings"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/application/repository"
	"gateman.io/application/usecases/wallet"
	"gateman.io/application/utils"
	"gateman.io/infrastructure/biometric"
	server_response "gateman.io/infrastructure/serverResponse"
	"gateman.io/infrastructure/validator"
	"github.com/gin-gonic/gin"
)

// EnhancedFaceComparison performs enhanced face comparison with detailed analysis
func EnhancedFaceComparison(ctx *interfaces.ApplicationContext[dto.EnhancedFaceComparisonRequest]) {
	validationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if validationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, validationErr, ctx.DeviceID)
		return
	}

	// Get workspace ID
	workspaceID := ctx.GetStringContextData("WorkspaceID")
	if workspaceID == "" {
		apperrors.ClientError(ctx.Ctx, "workspace ID not found", nil, nil, ctx.DeviceID)
		return
	}

	// Check and deduct balance
	walletService := wallet.NewWalletService()
	billingLog, err := walletService.CheckAndDeductBalance(workspaceID, wallet.COMPARISON_PRICE, "comparison", ctx.Body.RequestID)
	if err != nil {
		apperrors.ClientError(ctx.Ctx, "insufficient balance", nil, nil, ctx.DeviceID)
		return
	}

	// Store billing log ID in context for potential reversion and activity log
	ctx.SetContextData("billingLogID", billingLog.ID)
	if ginCtx, ok := ctx.Ctx.(*gin.Context); ok {
		ginCtx.Set("transactionID", billingLog.ID)
	}

	// Validate threshold
	threshold := ctx.Body.Threshold
	if threshold == 0 {
		threshold = 0.5 // Default threshold for comparison
	}
	if threshold < 0.0 || threshold > 1.0 {
		apperrors.ClientError(ctx.Ctx, "threshold must be between 0.0 and 1.0", nil, nil, ctx.DeviceID)
		return
	}

	// Check if sandbox environment - return mock response
	sandboxEnv, exists := ctx.GetContextData("SandboxEnv")
	if exists && sandboxEnv.(bool) {
		// Check query param for fail response
		failParam, exists := ctx.Query["fail"]
		if exists && failParam == "true" {
			server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Enhanced face comparison failed (sandbox)", map[string]interface{}{
				"is_match":        false,
				"similarity":      0.2,
				"confidence":      0.1,
				"processing_time": 150,
				"error":           "Mock failure response",
			}, nil, nil, nil)
			return
		}

		// Return success mock response
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Enhanced face comparison completed (sandbox)", map[string]interface{}{
			"is_match":        true,
			"similarity":      0.95,
			"confidence":      0.92,
			"processing_time": 150,
		}, nil, nil, nil)
		return
	}

	// Validate image inputs - check if they're URLs or base64
	if strings.HasPrefix(ctx.Body.Image1, "http://") || strings.HasPrefix(ctx.Body.Image1, "https://") {
		// Validate URL format
		_, err := url.ParseRequestURI(ctx.Body.Image1)
		if err != nil {
			apperrors.ClientError(ctx.Ctx, "invalid image1 URL format", nil, nil, ctx.DeviceID)
			return
		}
	} else {
		// Validate base64 format
		_, err := utils.DecodeBase64Image(ctx.Body.Image1)
		if err != nil {
			apperrors.ClientError(ctx.Ctx, "invalid image1 format - must be valid URL or base64", nil, nil, ctx.DeviceID)
			return
		}
	}

	if strings.HasPrefix(ctx.Body.Image2, "http://") || strings.HasPrefix(ctx.Body.Image2, "https://") {
		// Validate URL format
		_, err := url.ParseRequestURI(ctx.Body.Image2)
		if err != nil {
			apperrors.ClientError(ctx.Ctx, "invalid image2 URL format", nil, nil, ctx.DeviceID)
			return
		}
	} else {
		// Validate base64 format
		_, err := utils.DecodeBase64Image(ctx.Body.Image2)
		if err != nil {
			apperrors.ClientError(ctx.Ctx, "invalid image2 format - must be valid URL or base64", nil, nil, ctx.DeviceID)
			return
		}
	}

	// Use local face service for enhanced comparison
	localService := biometric.NewLocalFaceService()
	defer localService.Close()

	// Perform face comparison using service's default thresholds first, then apply custom threshold if needed
	result, err := localService.CompareFaces(&ctx.Body.Image1, &ctx.Body.Image2)
	if err != nil {
		// Revert balance deduction on failure
		billingLogID := ctx.GetStringContextData("billingLogID")
		if billingLogID != "" {
			billingRepo := repository.BillingLogRepository()
			updateFilter := map[string]interface{}{"_id": billingLogID}
			updateData := map[string]interface{}{
				"status":       "failed",
				"errorMessage": err.Error(),
			}
			billingRepo.UpdateOrCreateByField(updateFilter, updateData)

			// Revert balance
			revertFilter := map[string]interface{}{"_id": workspaceID}
			revertUpdate := map[string]interface{}{
				"$inc": map[string]interface{}{
					"balance": wallet.COMPARISON_PRICE,
				},
			}
			repository.WorkspaceRepository().UpdateWithOperator(revertFilter, revertUpdate)
		}
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}

	// Apply custom threshold to override match decision if necessary
	customMatch := result.Match
	if result.Confidence >= threshold {
		customMatch = true
	} else if result.Confidence < threshold {
		customMatch = false
	}

	// Use custom match decision if it differs from service decision
	finalMatch := customMatch

	// Liveness detection removed from enhanced compare - only basic comparison now

	// Create enhanced response
	response := dto.NewEnhancedFaceComparisonResponse("")
	response.SetComparisonResult(
		finalMatch,
		result.Confidence,
		result.Confidence,
		int64(result.ProcessingTimeMs),
	)

	// Liveness results removed from payload as requested

	// Add detailed analysis if verbose mode is enabled
	if ctx.Body.Verbose {
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
		threshold = ctx.Body.Threshold
		if threshold == 0 {
			threshold = 0.6 // Default threshold
		}

		metadata := dto.NewEnhancedComparisonMetadataDTO(
			"yunet_facenet", // Updated to reflect YuNet + FaceNet
			threshold,
			result.Confidence,
			"high", // Confidence level
		)
		response.SetComparisonMetadata(metadata)
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

	// Get workspace ID
	workspaceID := ctx.GetStringContextData("WorkspaceID")
	if workspaceID == "" {
		apperrors.ClientError(ctx.Ctx, "workspace ID not found", nil, nil, ctx.DeviceID)
		return
	}

	// Check and deduct balance
	walletService := wallet.NewWalletService()
	billingLog, err := walletService.CheckAndDeductBalance(workspaceID, wallet.LIVENESS_PRICE, "liveness", ctx.Body.RequestID)
	if err != nil {
		apperrors.ClientError(ctx.Ctx, "insufficient balance", nil, nil, ctx.DeviceID)
		return
	}

	// Store billing log ID in context for potential reversion and activity log
	ctx.SetContextData("billingLogID", billingLog.ID)
	if ginCtx, ok := ctx.Ctx.(*gin.Context); ok {
		ginCtx.Set("transactionID", billingLog.ID)
	}

	// Validate threshold
	threshold := ctx.Body.Threshold
	if threshold == 0 {
		threshold = 0.6 // Default threshold
	}
	if threshold < 0.0 || threshold > 1.0 {
		apperrors.ClientError(ctx.Ctx, "threshold must be between 0.0 and 1.0", nil, nil, ctx.DeviceID)
		return
	}

	// Check if sandbox environment - return mock response
	sandboxEnv, exists := ctx.GetContextData("SandboxEnv")
	if exists && sandboxEnv.(bool) {
		// Check query param for fail response
		failParam, failExists := ctx.Query["fail"]
		if failExists && failParam == "true" {
			server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Enhanced liveness check failed (sandbox)", map[string]interface{}{
				"is_live":         false,
				"liveness_score":  0.1,
				"threshold_used":  threshold,
				"spoof_score":     0.9,
				"confidence":      0.1,
				"processing_time": 100,
				"error":           "Mock failure response",
			}, nil, nil, nil)
			return
		}

		// Return success mock response
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Enhanced liveness check completed (sandbox)", map[string]interface{}{
			"is_live":         true,
			"liveness_score":  0.95,
			"threshold_used":  threshold,
			"spoof_score":     0.05,
			"confidence":      0.95,
			"processing_time": 100,
		}, nil, nil, nil)
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
	result, err := localService.ImageLivenessCheck(&ctx.Body.Image, false)
	if err != nil {
		// Revert balance deduction on failure
		billingLogID := ctx.GetStringContextData("billingLogID")
		if billingLogID != "" {
			billingRepo := repository.BillingLogRepository()
			updateFilter := map[string]interface{}{"_id": billingLogID}
			updateData := map[string]interface{}{
				"status":       "failed",
				"errorMessage": err.Error(),
			}
			billingRepo.UpdateOrCreateByField(updateFilter, updateData)

			// Revert balance
			revertFilter := map[string]interface{}{"_id": workspaceID}
			revertUpdate := map[string]interface{}{
				"$inc": map[string]interface{}{
					"balance": wallet.LIVENESS_PRICE,
				},
			}
			repository.WorkspaceRepository().UpdateWithOperator(revertFilter, revertUpdate)
		}
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}

	// ============================================================================
	// PRODUCTION-GRADE MULTI-LAYER VALIDATION SYSTEM
	// ============================================================================
	// Apply comprehensive validation with multiple security layers
	// This ensures paintings, printed photos, and sophisticated spoofs are rejected

	// Layer 1: Base threshold check
	customIsLive := result.IsLive
	if result.LivenessScore >= threshold {
		customIsLive = true
	} else if result.LivenessScore < threshold {
		customIsLive = false
	}

	// Layer 2: Production minimum threshold (security baseline)
	// Regardless of user threshold, enforce a production minimum for safety
	productionMinThreshold := 0.55 // Production safety baseline
	if result.LivenessScore < productionMinThreshold {
		customIsLive = false
	}

	// Layer 3: Confidence validation
	// Low confidence indicates uncertain results - reject for safety
	if result.Confidence < 0.4 {
		customIsLive = false
	}

	// Layer 4: Quality check
	// Very poor quality images are more likely to be spoofs
	if result.QualityScore < 0.3 {
		customIsLive = false
	}

	// Layer 5: Spoof score check
	// High spoof detection score indicates likely attack
	spoofScore := result.AnalysisDetails.SpoofDetectionScore
	if !math.IsNaN(spoofScore) && !math.IsInf(spoofScore, 0) && spoofScore > 0.6 {
		customIsLive = false
	}

	// Layer 6: Critical metrics validation
	// Check for painting/spoof indicators in detailed analysis
	textureScore := result.AnalysisDetails.TextureScore
	edgeSharpness := result.AnalysisDetails.EdgeSharpness

	// Extremely low texture + low edges = likely painting/print
	if textureScore < 0.15 && edgeSharpness < 0.15 {
		customIsLive = false
	}

	// Layer 7: Combined risk assessment
	// Multiple weak indicators together = reject
	riskFactors := 0
	if result.LivenessScore < 0.65 {
		riskFactors++
	}
	if result.Confidence < 0.6 {
		riskFactors++
	}
	if result.QualityScore < 0.5 {
		riskFactors++
	}
	if textureScore < 0.3 {
		riskFactors++
	}
	if edgeSharpness < 0.3 {
		riskFactors++
	}

	// If 3+ risk factors present, reject
	if riskFactors >= 3 {
		customIsLive = false
	}

	// Final decision
	finalIsLive := customIsLive

	// Create enhanced response with NaN checks
	// Re-assign spoofScore for response (already declared above)
	spoofScore = result.AnalysisDetails.SpoofDetectionScore
	confidence := result.Confidence

	// Fix NaN values to prevent JSON serialization errors
	if math.IsNaN(spoofScore) || math.IsInf(spoofScore, 0) {
		spoofScore = 0.0
	}
	if math.IsNaN(confidence) || math.IsInf(confidence, 0) {
		confidence = 0.5
	}

	response := &dto.LivenessDetectionResponse{
		IsLive:         finalIsLive,
		LivenessScore:  result.LivenessScore,
		ThresholdUsed:  threshold, // Use custom threshold
		SpoofScore:     spoofScore,
		Confidence:     confidence,
		ProcessingTime: int64(result.ProcessingTimeMs),
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
			Sharpness:    safeValue(result.AnalysisDetails.SharpnessScore),
			Brightness:   safeValue(result.AnalysisDetails.LightingScore),
			Contrast:     safeValue(result.AnalysisDetails.LightingScore),
			FaceSize:     qualityScore * 100,
			QualityScore: qualityScore,
		}

		// Add specific spoof reasons if detected (not generic)
		if !finalIsLive {
			// Only add spoof reasons if we have specific indicators
			if result.AnalysisDetails.TextureScore < 0.2 {
				response.SpoofReasons = append(response.SpoofReasons, "Low natural skin texture detected")
			}
			if result.AnalysisDetails.EdgeSharpness < 0.2 {
				response.SpoofReasons = append(response.SpoofReasons, "Unnatural edge patterns detected")
			}
			if result.AnalysisDetails.SpoofDetectionScore > 0.7 {
				response.SpoofReasons = append(response.SpoofReasons, "High spoof probability indicators")
			}
		}
	}

	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Enhanced liveness check completed", response, nil, nil, nil)
}
