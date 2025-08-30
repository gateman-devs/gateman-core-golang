package controller

import (
	"fmt"
	"net/http"
	"net/url"
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
	result, err := biometric.BiometricService.ImageLivenessCheck(&ctx.Body.Image)
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
