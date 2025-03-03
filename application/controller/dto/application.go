package dto

import "gateman.io/entities"

type ApplicationDTO struct {
	Name                  string                        `json:"name" validate:"required,max=100,min=2"`
	Description           string                        `json:"description" validate:"required,max=200,min=10"`
	RequiredVerifications *[]entities.VerificationType  `json:"requiredVerifications" validate:"dive"`
	LocaleRestriction     *[]entities.LocaleRestriction `json:"localeRestriction" validate:"dive"`
	RequestedFields       []entities.RequestedField     `json:"requestedFields" validate:"required,dive"`
	CustomFormFields      *[]entities.CustomFormField   `json:"customFormFields" validate:"dive"`
}

type UpdateApplications struct {
	Name                  *string                       `json:"name" validate:"max=100,min=2"`
	Description           *string                       `json:"description" validate:"max=200,min=10"`
	PaymentCard           *string                       `json:"paymentCard" validate:"ulid"`
	RequiredVerifications *[]string                     `json:"requiredVerifications" validate:"dive,min=2,max=50"`
	LocaleRestriction     *[]entities.LocaleRestriction `json:"localeRestriction"`
	RequestedFields       []entities.RequestedField     `json:"requestedFields"`
	CustomFormFields      *[]entities.CustomFormField   `json:"customFormFields" validate:"dive"`
}

type ApplicationSignUpDTO struct {
	AppID string `json:"appID" validate:"required,max=100"`
}

type SubmitCustomAppFormDTO struct {
	AppID string         `json:"appID" validate:"required,max=100"`
	Page  uint8          `json:"page" validate:"required,min=1,max=10"`
	Data  map[string]any `json:"data" validate:"required"`
}

type FetchAppUsersDTO struct {
	AppID    string  `json:"appID" validate:"required"`
	PageSize int64   `json:"pageSize" validate:"required"`
	LastID   *string `json:"lastID" validate:"ulid"`
	Blocked  *bool   `json:"blocked"`
	Deleted  *bool   `json:"deleted"`
	Sort     int8    `json:"sort"`
}

type BlockAccountsDTO struct {
	IDs []string `json:"ids" validate:"dive,ulid"`
}

type FetchAppMetrics struct {
	ID string `json:"id" validate:"max=26"`
}

type UpdateAccessRefreshTokenTTL struct {
	RefreshTokenTTL        *uint32 `json:"refreshTokenTTL" validate:"min=60"`
	AccessTokenTTL         *uint16 `json:"accessTokenTTL" validate:"min=60"`
	SandboxRefreshTokenTTL *uint32 `json:"sandboxRefreshTokenTTL" validate:"min=60"`
	SandboxAccessTokenTTL  *uint16 `json:"sandboxAccessTokenTTL" validate:"min=60"`
}

type UpdateWhitelistIPDTO struct {
	IPs []string `json:"ips" validate:"required,dive,ip"`
}
