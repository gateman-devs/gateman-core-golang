package biometric

import (
	"encoding/json"

	"gateman.io/application/utils"
	"gateman.io/infrastructure/biometric/types"
	"gateman.io/infrastructure/database/repository/cache"
	"gateman.io/infrastructure/logger"
	"gateman.io/infrastructure/network"
)

type GatemanFace struct {
	Network *network.NetworkController
	Cache   *cache.RedisRepository
}

func (g *GatemanFace) CompareFaces(image1 *string, image2 *string) (*types.BiometricFaceMatchResponse, error) {
	requestBody := types.FaceComparisonRequest{
		Image1: image1,
		Image2: image2,
	}

	response, statusCode, err := g.Network.Post("/compare-faces", &map[string]string{}, requestBody, nil, false, nil)
	if err != nil {
		logger.Error("error performing face comparison", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, err
	}

	if statusCode == nil || *statusCode != 200 {
		logger.Error("face comparison failed with status code", logger.LoggerOptions{
			Key:  "status_code",
			Data: statusCode,
		})
		return &types.BiometricFaceMatchResponse{
			Success: false,
			Error:   utils.GetStringPointer("Face comparison failed"),
		}, nil
	}

	var result types.BiometricFaceMatchResponse
	if err := json.Unmarshal(*response, &result); err != nil {
		logger.Error("error unmarshaling face comparison response", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, err
	}

	return &result, nil
}

func (g *GatemanFace) ImageLivenessCheck(image *string) (*types.BiometricLivenessResponse, error) {
	requestBody := types.LivenessCheckRequest{
		Image: image,
	}

	response, statusCode, err := g.Network.Post("/liveness-check", &map[string]string{}, requestBody, nil, false, nil)
	if err != nil {
		logger.Error("error performing image liveness check", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, err
	}

	if statusCode == nil || *statusCode != 200 {
		logger.Error("image liveness check failed with status code", logger.LoggerOptions{
			Key:  "status_code",
			Data: statusCode,
		})
		return &types.BiometricLivenessResponse{
			Success:       false,
			FailureReason: utils.GetStringPointer("service_error"),
			Error:         utils.GetStringPointer("Image liveness check failed"),
		}, nil
	}

	var result types.BiometricLivenessResponse
	if err := json.Unmarshal(*response, &result); err != nil {
		logger.Error("error unmarshaling image liveness response", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, err
	}

	return &result, nil
}

func (g *GatemanFace) VideoLivenessCheck(payload types.VideoLivenessRequest) (*types.VideoLivenessResponse, error) {
	response, statusCode, err := g.Network.Post("/verify-video-liveness", &map[string]string{}, payload, nil, false, nil)
	if err != nil {
		logger.Error("error performing video liveness check", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, err
	}

	if statusCode == nil || *statusCode != 200 {
		logger.Error("video liveness check failed with status code", logger.LoggerOptions{
			Key:  "status_code",
			Data: statusCode,
		})
		return &types.VideoLivenessResponse{
			Error: utils.GetStringPointer("Video liveness check failed"),
		}, nil
	}

	var result types.VideoLivenessResponse
	if err := json.Unmarshal(*response, &result); err != nil {
		logger.Error("error unmarshaling video liveness response", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, err
	}

	return &result, nil
}

func (g *GatemanFace) GenerateChallenge() (*types.ChallengeResponse, error) {
	response, statusCode, err := g.Network.Post("/generate-challenge", &map[string]string{}, nil, nil, false, nil)
	if err != nil {
		logger.Error("error generating challenge", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, err
	}

	if statusCode == nil || *statusCode != 200 {
		logger.Error("challenge generation failed with status code", logger.LoggerOptions{
			Key:  "status_code",
			Data: statusCode,
		})
		return &types.ChallengeResponse{
			Success:     false,
			ChallengeID: nil,
			Directions:  []string{},
			TTLSeconds:  0,
		}, nil
	}

	var result types.ChallengeResponse
	if err := json.Unmarshal(*response, &result); err != nil {
		logger.Error("error unmarshaling challenge response", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, err
	}

	return &result, nil
}
