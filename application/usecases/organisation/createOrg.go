package org_usecases

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
	"authone.usepolymer.co/infrastructure/auth"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/database/repository/cache"
	"authone.usepolymer.co/infrastructure/logger"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateOrgUseCase(ctx any, payload *dto.CreateOrgDTO, deviceID *string, userAgent *string) error {
	payload.Email = strings.ToLower(payload.Email)
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
	orgMemberRepo := repository.OrgMemberRepo()
	exists, err := orgMemberRepo.CountDocs(map[string]interface{}{
		"email": payload.Email,
	})
	if err != nil {
		apperrors.FatalServerError(ctx, err, deviceID)
		return err
	}
	if exists != 0 {
		apperrors.EntityAlreadyExistsError(ctx, "organisation with email already exists", deviceID)
		return errors.New("")
	}
	orgRepo := repository.OrgRepo()
	err = orgRepo.StartTransaction(func(sc mongo.Session, c context.Context) error {
		hashedPassword, err := cryptography.CryptoHahser.HashString(payload.Password)
		if err != nil {
			logger.Error("an error occured while hashing org member password", logger.LoggerOptions{
				Key:  "error",
				Data: err,
			})
			sc.AbortTransaction(c)
			return err
		}
		orgID := utils.GenerateUULDString()
		orgMember := entities.OrgMember{
			FirstName:   payload.FirstName,
			LastName:    payload.LastName,
			UserAgent:   *userAgent,
			Password:    string(hashedPassword),
			Email:       payload.Email,
			DeviceID:    *deviceID,
			Permissions: []entities.MemberPermissions{entities.SUPER_ACCESS},
			ID:          utils.GenerateUULDString(),
			OrgID:       orgID,
		}
		orgData := entities.Organisation{
			Name:        payload.OrgName,
			Email:       payload.Email,
			Sector:      payload.Sector,
			Country:     payload.Country,
			SuperMember: orgMember.ID,
			ID:          orgID,
		}
		_, trxErr := orgRepo.CreateOne(context.TODO(), orgData)
		if trxErr != nil {
			logger.Error("an error occured while creating an org", logger.LoggerOptions{
				Key:  "error",
				Data: trxErr,
			}, logger.LoggerOptions{
				Key:  "payload",
				Data: orgData,
			})
			sc.AbortTransaction(c)
			return trxErr
		}

		_, trxErr = orgMemberRepo.CreateOne(context.TODO(), orgMember)
		if trxErr != nil {
			orgMember.Password = ""
			logger.Error("an error occured while creating an org member", logger.LoggerOptions{
				Key:  "error",
				Data: trxErr,
			}, logger.LoggerOptions{
				Key:  "payload",
				Data: orgMember,
			})
			sc.AbortTransaction(c)
			return trxErr
		}

		(sc).CommitTransaction(c)
		return nil
	})

	if err != nil {
		apperrors.UnknownError(ctx, err, deviceID)
		return err
	}
	otp, err := auth.GenerateOTP(6, payload.Email)
	if err != nil {
		logger.Error("an error occured while generating an otp", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return err
	}
	err = polymercore.PolymerService.SendEmail("authone_org_welcome", payload.Email, "Welcome to AuthOne!", map[string]any{
		"FIRSTNAME": payload.FirstName,
		"ORGNAME":   payload.OrgName,
		"OTP":       otp,
	})
	if err != nil {
		apperrors.UnknownError(ctx, err, deviceID)
		return err
	}
	success := cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", payload.Email), "org_verification", time.Minute*10)
	if !success {
		apperrors.UnknownError(ctx, err, deviceID)
		return err
	}
	return nil
}
