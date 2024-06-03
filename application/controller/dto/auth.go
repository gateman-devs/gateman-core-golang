package dto

import "crypto/ecdh"

type KeyExchangeDTO struct {
	ClientPublicKey *ecdh.PublicKey `json:"clientPubKey"`
}

type VerifyOTPDTO struct {
	OTP   string  `json:"otp"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}

type LoginDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
