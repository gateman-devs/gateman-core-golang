package org_usecases

import (
	"context"
	"fmt"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/constants"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/repository"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/logger"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateOrgUseCase(ctx any, payload *dto.CreateOrgDTO, deviceID string, userAgent string, userID string, email string) error {
	workspaceMemberRepo := repository.WorkspaceMemberRepo()
	workspaceRepo := repository.WorkspaceRepository()
	createdWorkspaces, err := workspaceRepo.CountDocs(map[string]interface{}{
		"createdBy": userID,
	})
	if err != nil {
		apperrors.UnknownError(ctx, err)
		return err
	}
	if createdWorkspaces >= constants.MAX_ORGANISATIONS_CREATED {
		err = fmt.Errorf("you have exceeded the number of organisations a user can create on Gateman. if you think this is a mistake reach out to our support team at %s", constants.SUPPORT_EMAIL)
		apperrors.ClientError(ctx, err.Error(), nil, nil)
		return err
	}
	err = workspaceRepo.StartTransaction(func(sc mongo.Session, c context.Context) error {
		workspaceID := utils.GenerateUULDString()
		orgMember := entities.WorkspaceMember{
			Permissions:   []entities.MemberPermissions{entities.SUPER_ACCESS},
			ID:            utils.GenerateUULDString(),
			WorkspaceID:   workspaceID,
			WorkspaceName: payload.WorkspaceName,
			UserID:        userID,
		}
		orgData := entities.Workspace{
			Name:        payload.WorkspaceName,
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

		_, trxErr = workspaceMemberRepo.CreateOne(context.TODO(), orgMember)
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
		apperrors.UnknownError(ctx, err)
		return err
	}
	return nil
}
