package org_usecases

import (
	"context"
	"fmt"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/repository"
	"authone.usepolymer.co/entities"
)

func CreateOrgUseCase(ctx any, payload *dto.CreateOrgDTO, deviceID *string, nonce *string) error {
	orgModel := repository.OrgRepository
	fmt.Println(orgModel)
	fmt.Println(orgModel)
	fmt.Println(orgModel)
	fmt.Println(orgModel)
	fmt.Println(orgModel)
	fmt.Println(orgModel)
	org, err := orgModel.CreateOne(context.TODO(), entities.Organisation{
		Name:    payload.OrgName,
		Email:   payload.Email,
		Sector:  payload.Sector,
		Country: payload.Country,
	})
	if err != nil {
		apperrors.FatalServerError(ctx, err, deviceID, nonce)
		return err
	}
	fmt.Println(org)
	return nil
}
