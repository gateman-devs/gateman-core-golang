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
	"gateman.io/infrastructure/logger"
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

	// ========================================================================
	// LIVENESS PRE-CHECK - Fail fast if either image is not live
	// ========================================================================
	logger.Info("🔍 Performing liveness pre-checks before face comparison", logger.LoggerOptions{
		Key: "liveness_precheck_start",
	})

	// Check liveness of image1
	liveness1Result, err := localService.ImageLivenessCheck(&ctx.Body.Image1, false)
	if err != nil {
		// Revert balance
		billingLogID := ctx.GetStringContextData("billingLogID")
		if billingLogID != "" {
			billingRepo := repository.BillingLogRepository()
			updateFilter := map[string]interface{}{"_id": billingLogID}
			updateData := map[string]interface{}{
				"status":       "failed",
				"errorMessage": "Liveness check failed for image1: " + err.Error(),
			}
			billingRepo.UpdateOrCreateByField(updateFilter, updateData)

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

	// Apply strict liveness validation for image1
	if !liveness1Result.IsLive || liveness1Result.LivenessScore < 0.55 {
		// Revert balance
		billingLogID := ctx.GetStringContextData("billingLogID")
		if billingLogID != "" {
			billingRepo := repository.BillingLogRepository()
			updateFilter := map[string]interface{}{"_id": billingLogID}
			updateData := map[string]interface{}{
				"status":       "failed",
				"errorMessage": "Image1 failed liveness check",
			}
			billingRepo.UpdateOrCreateByField(updateFilter, updateData)

			revertFilter := map[string]interface{}{"_id": workspaceID}
			revertUpdate := map[string]interface{}{
				"$inc": map[string]interface{}{
					"balance": wallet.COMPARISON_PRICE,
				},
			}
			repository.WorkspaceRepository().UpdateWithOperator(revertFilter, revertUpdate)
		}

		logger.Error("❌ Image1 failed liveness check - aborting comparison", logger.LoggerOptions{
			Key: "liveness1_failed",
			Data: map[string]interface{}{
				"is_live":        liveness1Result.IsLive,
				"liveness_score": liveness1Result.LivenessScore,
				"spoof_score":    liveness1Result.AnalysisDetails.SpoofDetectionScore,
				"painting_prob":  liveness1Result.AnalysisDetails.PaintingProbability,
				"screen_prob":    liveness1Result.AnalysisDetails.ScreenProbability,
				"print_prob":     liveness1Result.AnalysisDetails.PrintProbability,
				"mask_prob":      liveness1Result.AnalysisDetails.MaskProbability,
				"skin_tone":      liveness1Result.AnalysisDetails.SkinToneRealism,
			},
		})

		apperrors.ClientError(ctx.Ctx, "Image1 failed liveness verification - possible spoof detected", nil, nil, ctx.DeviceID)
		return
	}

	// Check liveness of image2
	liveness2Result, err := localService.ImageLivenessCheck(&ctx.Body.Image2, false)
	if err != nil {
		// Revert balance
		billingLogID := ctx.GetStringContextData("billingLogID")
		if billingLogID != "" {
			billingRepo := repository.BillingLogRepository()
			updateFilter := map[string]interface{}{"_id": billingLogID}
			updateData := map[string]interface{}{
				"status":       "failed",
				"errorMessage": "Liveness check failed for image2: " + err.Error(),
			}
			billingRepo.UpdateOrCreateByField(updateFilter, updateData)

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

	// Apply strict liveness validation for image2
	if !liveness2Result.IsLive || liveness2Result.LivenessScore < 0.55 {
		// Revert balance
		billingLogID := ctx.GetStringContextData("billingLogID")
		if billingLogID != "" {
			billingRepo := repository.BillingLogRepository()
			updateFilter := map[string]interface{}{"_id": billingLogID}
			updateData := map[string]interface{}{
				"status":       "failed",
				"errorMessage": "Image2 failed liveness check",
			}
			billingRepo.UpdateOrCreateByField(updateFilter, updateData)

			revertFilter := map[string]interface{}{"_id": workspaceID}
			revertUpdate := map[string]interface{}{
				"$inc": map[string]interface{}{
					"balance": wallet.COMPARISON_PRICE,
				},
			}
			repository.WorkspaceRepository().UpdateWithOperator(revertFilter, revertUpdate)
		}

		logger.Error("❌ Image2 failed liveness check - aborting comparison", logger.LoggerOptions{
			Key: "liveness2_failed",
			Data: map[string]interface{}{
				"is_live":        liveness2Result.IsLive,
				"liveness_score": liveness2Result.LivenessScore,
				"spoof_score":    liveness2Result.AnalysisDetails.SpoofDetectionScore,
				"painting_prob":  liveness2Result.AnalysisDetails.PaintingProbability,
				"screen_prob":    liveness2Result.AnalysisDetails.ScreenProbability,
				"print_prob":     liveness2Result.AnalysisDetails.PrintProbability,
				"mask_prob":      liveness2Result.AnalysisDetails.MaskProbability,
				"skin_tone":      liveness2Result.AnalysisDetails.SkinToneRealism,
			},
		})

		apperrors.ClientError(ctx.Ctx, "Image2 failed liveness verification - possible spoof detected", nil, nil, ctx.DeviceID)
		return
	}

	logger.Info("✅ Both images passed liveness pre-check - proceeding with comparison", logger.LoggerOptions{
		Key: "liveness_precheck_passed",
		Data: map[string]interface{}{
			"image1_liveness_score": liveness1Result.LivenessScore,
			"image2_liveness_score": liveness2Result.LivenessScore,
		},
	})

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

	// Layer 6: Comprehensive spoof type checks (CRITICAL)
	// Direct checks for all spoof types - hard fail if detected
	paintingProbability := result.AnalysisDetails.PaintingProbability
	screenProbability := result.AnalysisDetails.ScreenProbability
	printProbability := result.AnalysisDetails.PrintProbability
	maskProbability := result.AnalysisDetails.MaskProbability
	skinToneRealism := result.AnalysisDetails.SkinToneRealism

	// Painting detection - BALANCED THRESHOLD (0.45)
	// Calibrated to catch paintings while allowing legitimate images
	if !math.IsNaN(paintingProbability) && !math.IsInf(paintingProbability, 0) && paintingProbability >= 0.45 {
		customIsLive = false
		logger.Error("🎨 PAINTING DETECTED - Hard fail", logger.LoggerOptions{
			Key: "painting_hard_fail",
			Data: map[string]interface{}{
				"painting_probability": paintingProbability,
				"threshold":            0.45,
			},
		})
	}

	// Combined painting indicators check - more conservative thresholds
	// This catches paintings with moderate probability + clear painting characteristics
	if !math.IsNaN(paintingProbability) && paintingProbability >= 0.35 {
		textureScore := result.AnalysisDetails.TextureScore
		edgeSharpness := result.AnalysisDetails.EdgeSharpness

		// High painting probability + very low texture + very low edges = definitely painting
		if textureScore < 0.25 && edgeSharpness < 0.25 {
			customIsLive = false
			logger.Error("🎨 PAINTING DETECTED via combined indicators - Hard fail", logger.LoggerOptions{
				Key: "painting_combined_fail",
				Data: map[string]interface{}{
					"painting_probability": paintingProbability,
					"texture_score":        textureScore,
					"edge_sharpness":       edgeSharpness,
				},
			})
		}

		// High liveness score but very suspicious painting characteristics
		// More conservative - only trigger on very low texture
		if result.LivenessScore > 0.6 && paintingProbability >= 0.4 && textureScore < 0.20 {
			customIsLive = false
			logger.Error("🎨 HIGH-SCORING PAINTING DETECTED - Hard fail", logger.LoggerOptions{
				Key: "high_score_painting_fail",
				Data: map[string]interface{}{
					"liveness_score":       result.LivenessScore,
					"painting_probability": paintingProbability,
					"texture_score":        textureScore,
				},
			})
		}
	}

	// Screen detection
	if !math.IsNaN(screenProbability) && !math.IsInf(screenProbability, 0) && screenProbability >= 0.5 {
		customIsLive = false
		logger.Error("🖥️ SCREEN DETECTED - Hard fail", logger.LoggerOptions{
			Key: "screen_hard_fail",
			Data: map[string]interface{}{
				"screen_probability": screenProbability,
				"threshold":          0.5,
			},
		})
	}

	// Print detection
	if !math.IsNaN(printProbability) && !math.IsInf(printProbability, 0) && printProbability >= 0.5 {
		customIsLive = false
		logger.Error("🖨️ PRINT DETECTED - Hard fail", logger.LoggerOptions{
			Key: "print_hard_fail",
			Data: map[string]interface{}{
				"print_probability": printProbability,
				"threshold":         0.5,
			},
		})
	}

	// Mask detection
	if !math.IsNaN(maskProbability) && !math.IsInf(maskProbability, 0) && maskProbability >= 0.5 {
		customIsLive = false
		logger.Error("🎭 MASK DETECTED - Hard fail", logger.LoggerOptions{
			Key: "mask_hard_fail",
			Data: map[string]interface{}{
				"mask_probability": maskProbability,
				"threshold":        0.5,
			},
		})
	}

	// Skin tone realism check
	if !math.IsNaN(skinToneRealism) && !math.IsInf(skinToneRealism, 0) && skinToneRealism < 0.4 {
		customIsLive = false
		logger.Error("🎨 UNREALISTIC SKIN TONE - Hard fail", logger.LoggerOptions{
			Key: "skin_tone_hard_fail",
			Data: map[string]interface{}{
				"skin_tone_realism": skinToneRealism,
				"threshold":         0.4,
			},
		})
	}

	// Layer 7: Critical metrics validation
	// Check for painting/spoof indicators in detailed analysis
	textureScore := result.AnalysisDetails.TextureScore
	edgeSharpness := result.AnalysisDetails.EdgeSharpness

	// Extremely low texture + low edges = likely painting/print
	// BALANCED threshold - only fail on very clear spoofs
	if textureScore < 0.15 && edgeSharpness < 0.15 {
		customIsLive = false
		logger.Error("📊 CRITICAL TEXTURE/EDGE METRICS - Hard fail", logger.LoggerOptions{
			Key: "critical_metrics_fail",
			Data: map[string]interface{}{
				"texture_score":  textureScore,
				"edge_sharpness": edgeSharpness,
			},
		})
	}

	// Layer 8: Combined risk assessment
	// Multiple weak indicators together = reject
	riskFactors := 0
	if result.LivenessScore < 0.60 { // More lenient
		riskFactors++
	}
	if result.Confidence < 0.55 { // More lenient
		riskFactors++
	}
	if result.QualityScore < 0.45 { // More lenient
		riskFactors++
	}
	if textureScore < 0.25 { // More lenient
		riskFactors++
	}
	if edgeSharpness < 0.25 { // More lenient
		riskFactors++
	}
	// Add all spoof type probabilities to risk factors
	// Painting gets 1.5x weight (reduced from 2x)
	if paintingProbability > 0.4 {
		riskFactors += 2 // Still high weight for clear paintings
	} else if paintingProbability > 0.3 {
		riskFactors++ // Single weight for moderate suspicion
	}
	if screenProbability > 0.4 { // Slightly higher threshold
		riskFactors++
	}
	if printProbability > 0.4 { // Slightly higher threshold
		riskFactors++
	}
	if maskProbability > 0.4 { // Slightly higher threshold
		riskFactors++
	}
	if skinToneRealism < 0.45 { // Slightly lower threshold
		riskFactors++
	}

	// If 3+ risk factors present, reject
	// Back to 3 for balanced approach
	if riskFactors >= 3 {
		customIsLive = false
		logger.Error("⚠️ MULTIPLE RISK FACTORS DETECTED - Hard fail", logger.LoggerOptions{
			Key: "risk_factors_fail",
			Data: map[string]interface{}{
				"risk_factors": riskFactors,
				"threshold":    3,
			},
		})
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

		// Add specific spoof reasons if detected (priority order: most specific first)
		if !finalIsLive {
			// Spoof type detections (high confidence)
			if paintingProbability >= 0.5 {
				response.SpoofReasons = append(response.SpoofReasons, "Painting or artwork detected")
			} else if paintingProbability >= 0.35 {
				response.SpoofReasons = append(response.SpoofReasons, "Painting-like characteristics detected")
			}

			if screenProbability >= 0.5 {
				response.SpoofReasons = append(response.SpoofReasons, "Screen or display detected")
			} else if screenProbability >= 0.35 {
				response.SpoofReasons = append(response.SpoofReasons, "Screen-like characteristics detected")
			}

			if printProbability >= 0.5 {
				response.SpoofReasons = append(response.SpoofReasons, "Printed photograph detected")
			} else if printProbability >= 0.35 {
				response.SpoofReasons = append(response.SpoofReasons, "Print-like characteristics detected")
			}

			if maskProbability >= 0.5 {
				response.SpoofReasons = append(response.SpoofReasons, "Facial mask detected")
			} else if maskProbability >= 0.35 {
				response.SpoofReasons = append(response.SpoofReasons, "Mask-like characteristics detected")
			}

			if skinToneRealism < 0.4 {
				response.SpoofReasons = append(response.SpoofReasons, "Unrealistic skin tone detected")
			} else if skinToneRealism < 0.5 {
				response.SpoofReasons = append(response.SpoofReasons, "Low skin tone realism")
			}

			// Texture/edge indicators
			if result.AnalysisDetails.TextureScore < 0.2 {
				response.SpoofReasons = append(response.SpoofReasons, "Low natural skin texture")
			}
			if result.AnalysisDetails.EdgeSharpness < 0.2 {
				response.SpoofReasons = append(response.SpoofReasons, "Unnatural edge patterns")
			}
			if result.AnalysisDetails.SpoofDetectionScore > 0.7 {
				response.SpoofReasons = append(response.SpoofReasons, "High spoof probability indicators")
			}
		}
	}

	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Enhanced liveness check completed", response, nil, nil, nil)
}
