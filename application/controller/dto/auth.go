package dto

import (
	"crypto/ecdh"

	"gateman.io/entities"
)

type KeyExchangeDTO struct {
	ClientPublicKey *ecdh.PublicKey `json:"clientPubKey"`
}

type VerifyOTPDTO struct {
	OTP   string  `json:"otp" validate:"required,max=5,min=5"`
	Email *string `json:"email" validate:"omitempty,email,max=100,min=6"`
	Phone *string `json:"phone" validate:"omitempty,max=11,min=11"`
}

type CreateUserDTO struct {
	Email      *string               `json:"email,omitempty" validate:"omitempty,email,max=100"`
	Phone      *entities.PhoneNumber `json:"phone,omitempty"`
	DeviceID   string                `json:"deviceID" validate:"required,max=50"`
	DeviceName string                `json:"deviceName" validate:"required,max=30"`
	UserAgent  string                `json:"userAgent" validate:"required,max=1000"`
}

type ResendOTPDTO struct {
	Email       *string `json:"email" validate:"omitempty,email,max=100,min=6"`
	Phone       *string `json:"phone" validate:"omitempty,eq=11"`
	PhonePrefix *string `json:"phonePrefix" validate:"omitempty,max=3,min=1"`
}

type VerifyDeviceDTO struct {
	Email *string `json:"email" validate:"omitempty,email,max=100,min=6"`
	Phone *string `json:"phone" validate:"omitempty,eq=11"`
}
