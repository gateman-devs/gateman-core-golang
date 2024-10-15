package org_usecases

import (
	"context"
	"errors"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/repository"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/logger"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateOrgUseCase(ctx any, payload *dto.CreateOrgDTO, deviceID *string, userAgent *string, userID string, email string) error {
	WorkspaceMemberRepo := repository.WorkspaceMemberRepo()
	exists, err := WorkspaceMemberRepo.CountDocs(map[string]interface{}{
		"email": email,
	})
	if err != nil {
		apperrors.FatalServerError(ctx, err, deviceID)
		return err
	}
	if exists != 0 {
		apperrors.EntityAlreadyExistsError(ctx, "organisation with email already exists", deviceID)
		return errors.New("")
	}
	workspaceRepo := repository.WorkspaceRepository()
	err = workspaceRepo.StartTransaction(func(sc mongo.Session, c context.Context) error {
		if err != nil {
			logger.Error("an error occured while hashing org member password", logger.LoggerOptions{
				Key:  "error",
				Data: err,
			})
			sc.AbortTransaction(c)
			return err
		}
		workspaceID := utils.GenerateUULDString()
		orgMember := entities.WorkspaceMember{
			Permissions:   []entities.MemberPermissions{entities.SUPER_ACCESS},
			ID:            utils.GenerateUULDString(),
			WorkspaceID:   workspaceID,
			WorkspaceName: payload.WorkspaceName,
		}
		orgData := entities.Workspace{
			Name:        payload.WorkspaceName,
			Email:       email,
			Sector:      payload.Sector,
			Country:     payload.Country,
			SuperMember: orgMember.ID,
			CreatedBy:   userID,
			ID:          workspaceID,
		}
		_, trxErr := workspaceRepo.CreateOne(context.TODO(), orgData)
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

		_, trxErr = WorkspaceMemberRepo.CreateOne(context.TODO(), orgMember)
		if trxErr != nil {
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
	return nil
}
