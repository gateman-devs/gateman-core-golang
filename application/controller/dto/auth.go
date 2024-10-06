package dto

import (
	"crypto/ecdh"

	"authone.usepolymer.co/entities"
)

type KeyExchangeDTO struct {
	ClientPublicKey *ecdh.PublicKey `json:"clientPubKey"`
}

type VerifyOTPDTO struct {
	OTP   string  `json:"otp"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}

type CreateUserDTO struct {
	Email           *string               `json:"email,omitempty" validate:"omitempty,email,max=100"`
	Password        *string               `json:"password" validate:"omitempty,min=8"`
	Phone           *entities.PhoneNumber `json:"phone,omitempty"`
	DeviceID        string                `json:"deviceID" validate:"required,max=50"`
	DeviceName      string                `json:"deviceName" validate:"required,max=30"`
	UserAgent       string                `json:"userAgent" validate:"required,max=1000"`
	ClientPublicKey string                `json:"clientPublicKey" validate:"required"`
}

type LoginDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ResendOTPDTO struct {
	Email       *string `json:"email"`
	Phone       *string `json:"phone"`
	PhonePrefix *string `json:"phonePrefix"`
}

type VerifyDevice struct {
	ImgURL   string `json:"imgURL"`
	DeviceID string `json:"deviceID"`
}
