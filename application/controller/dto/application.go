package dto

import "authone.usepolymer.co/entities"

type ApplicationDTO struct {
	Name                  string                        `json:"name" validate:"required"`
	Description           string                        `json:"description" validate:"required"`
	RequiredVerifications *[]string                     `json:"requiredVerifications"`
	LocaleRestriction     *[]entities.LocaleRestriction `json:"localeRestriction"`
	RequestedFields       []string                      `json:"requestedFields" validate:"required"`
}

type ApplicationSignUpDTO struct {
	AppID string `json:"appID" validate:"required"`
}
