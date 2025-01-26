package dto

import "authone.usepolymer.co/entities"

type ApplicationDTO struct {
	Name                  string                        `json:"name" validate:"required"`
	Description           string                        `json:"description" validate:"required"`
	RequiredVerifications *[]string                     `json:"requiredVerifications"`
	LocaleRestriction     *[]entities.LocaleRestriction `json:"localeRestriction"`
	RequestedFields       []entities.RequestedField     `json:"requestedFields" validate:"required"`
}

type UpdateApplications struct {
	Name                  *string                       `json:"name" validate:"required"`
	Description           *string                       `json:"description" validate:"required"`
	RequiredVerifications *[]string                     `json:"requiredVerifications"`
	LocaleRestriction     *[]entities.LocaleRestriction `json:"localeRestriction"`
	RequestedFields       []entities.RequestedField     `json:"requestedFields"`
}

type ApplicationSignUpDTO struct {
	AppID string `json:"appID" validate:"required"`
}

type FetchAppUsersDTO struct {
	AppID    string  `json:"appID" validate:"required"`
	PageSize int64   `json:"pageSize" validate:"required"`
	LastID   *string `json:"lastID"`
	Blocked  *bool   `json:"blocked"`
	Deleted  *bool   `json:"deleted"`
	Sort     int8    `json:"sort"`
}

type BlockAccountsDTO struct {
	IDs []string `json:"ids"`
}

type FetchAppMetrics struct {
	ID string `json:"id"`
}

type UpdateAccessRefreshTokenTTL struct {
	RefreshTokenTTL        *uint32 `json:"refreshTokenTTL"`
	AccessTokenTTL         *uint16 `json:"accessTokenTTL"`
	SandboxRefreshTokenTTL *uint32 `json:"sandboxRefreshTokenTTL"`
	SandboxAccessTokenTTL  *uint16 `json:"sandboxAccessTokenTTL"`
}
