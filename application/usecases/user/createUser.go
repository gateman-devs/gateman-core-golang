package user_usecases

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/repository"
	polymercore "authone.usepolymer.co/application/services/polymer-core"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/database/repository/cache"
	"authone.usepolymer.co/infrastructure/logger"
)

func CreateUserUseCase(ctx any, payload *dto.CreateUserDTO, deviceID *string, userAgent *string) error {
	payload.Email = strings.ToLower(payload.Email)
	userRepo := repository.UserRepo()
	exists, err := userRepo.CountDocs(map[string]any{
		"email": payload.Email,
	})
	if err != nil {
		apperrors.UnknownError(ctx, err, deviceID)
		return err
	}
	if exists != 0 {
		apperrors.ClientError(ctx, "User with email already exists", nil, nil, deviceID)
		return errors.New("")
	}

	if os.Getenv("ENV") == "prod" {
		found := cache.Cache.FindOne(fmt.Sprintf("%s-email-blacklist", payload.Email))
		if found != nil {
			apperrors.ClientError(ctx, fmt.Sprintf(`email address "%s" has been flagged as unacceptable on our system`, payload.Email), nil, nil, deviceID)
			return errors.New("")
		}
		result, err := polymercore.PolymerService.EmailStatus(payload.Email)
		if err != nil {
			apperrors.ExternalDependencyError(ctx, "polymer-core", "500", err, deviceID)
			return err
		}
		if !result {
			apperrors.ClientError(ctx, fmt.Sprintf(`email address "%s" has been flagged as unacceptable on our system`, payload.Email), nil, nil, deviceID)
			cache.Cache.CreateEntry(fmt.Sprintf("%s-email-blacklist", payload.Email), payload.Email, time.Minute*0)
			return errors.New("")
		}
	}

	hashedPassword, err := cryptography.CryptoHahser.HashString(payload.Password)
	if err != nil {
		logger.Error("an error occured while hashing user password", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return err
	}
	id, _ := polymercore.PolymerService.GetPolymerID(payload.Email)
	if id == nil {
		polymercore.PolymerService.CreateAccount(payload.Email, payload.Password)
		id, _ = polymercore.PolymerService.GetPolymerID(payload.Email)
	}
	if id == nil {
		logger.Error("could not create Polymer details for AuthOne user", logger.LoggerOptions{
			Key:  "email",
			Data: payload.Email,
		})
		id = utils.GetStringPointer("")
	}
	_, err = userRepo.CreateOne(context.TODO(), entities.User{
		Email:     payload.Email,
		Password:  string(hashedPassword),
		DeviceID:  *deviceID,
		UserAgent: *userAgent,
		PolymerID: *id,
	})
	if err != nil {
		logger.Error("could not create user after", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
	}
	return nil
}
