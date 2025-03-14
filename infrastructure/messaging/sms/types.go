package sms

type TermiiOTPResponse struct {
	PinID     *string `json:"pinId"`
	Recipient *string `json:"to"`
	SmsStatus *string `json:"smsStatus"`
	Msg       *string `json:"message"`
	Code      *string `json:"code"`
}

type TermiiOTPVerifiedResponse struct {
	PinID    string `json:"pinId"`
	Verified bool   `json:"verified"`
	Msg      string `json:"message"`
}

type SMSServiceType interface {
	SendOTP(phone string, whatsapp bool, otp *string) *string
	VerifyOTP(otpID string, otp string) bool
}
