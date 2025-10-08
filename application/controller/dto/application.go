package dto

import "gateman.io/entities"

type ApplicationDTO struct {
	Name              string                        `json:"name" validate:"required,max=100,min=2"`
	Description       string                        `json:"description" validate:"required,max=200,min=10"`
	Verifications     *[]entities.VerificationType  `json:"verifications" validate:"omitempty,dive"`
	LocaleRestriction *[]entities.LocaleRestriction `json:"localeRestriction" validate:"omitempty,dive"`
	RequestedFields   []entities.RequestedField     `json:"requestedFields" validate:"required,dive"`
	CustomFormFields  *[]entities.CustomFormField   `json:"customFormFields" validate:"omitempty,dive"`
}

type UpdateApplications struct {
	Name                *string                           `json:"name" validate:"omitempty,max=100,min=2"`
	Description         *string                           `json:"description" validate:"omitempty,max=200,min=10"`
	PaymentCard         *string                           `json:"paymentCard" validate:"omitempty,ulid"`
	SubscriptionID      *string                           `json:"subscriptionID" validate:"omitempty,ulid"`
	Interval            *entities.SubscriptionFrequency   `json:"interval" validate:"omitempty,oneof=monthly annually"`
	Verifications       *[]entities.VerificationType      `json:"verifications" validate:"omitempty,dive"`
	LocaleRestriction   *[]entities.LocaleRestriction     `json:"localeRestriction" validate:"omitempty,dive"`
	RequestedFields     []entities.RequestedField         `json:"requestedFields" validate:"omitempty,dive"`
	CustomFormFields    *[]entities.CustomFormField       `json:"customFormFields" validate:"omitempty,dive"`
}

type ApplicationSignUpDTO struct {
	AppID string  `json:"appID" validate:"required,ulid"`
	Pin   *string `json:"pin"`
}

type SubmitCustomAppFormDTO struct {
	AppID string         `json:"appID" validate:"required,ulid"`
	Page  uint8          `json:"page" validate:"required,min=1,max=10"`
	Data  map[string]any `json:"data" validate:"required"`
}

type FetchAppUsersDTO struct {
	AppID    string  `json:"appID" validate:"required,ulid"`
	PageSize int64   `json:"pageSize" validate:"required"`
	LastID   *string `json:"lastID" validate:"omitempty,ulid"`
	Blocked  *bool   `json:"blocked"`
	Deleted  *bool   `json:"deleted"`
	Sort     int8    `json:"sort"`
}

type BlockAccountsDTO struct {
	IDs []string `json:"ids" validate:"dive,ulid"`
	Reason string `json:"reason" validate:"omitempty,max=200"`
}

type FetchAppMetrics struct {
	ID string `json:"id" validate:"ulid"`
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

type TogglePinProtectionSettingDTO struct {
	Activated bool `json:"activated"`
}

type ToggleMFAProtectionSettingDTO struct {
	Activated bool   `json:"activated"`
	ID        string `json:"id" validate:"ulid"`
}
