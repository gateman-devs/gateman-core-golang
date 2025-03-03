package totp

type TOTPGeneratorType interface {
	ValidateTOTP(token string, secret string) bool
	GenerateTOTPCode(secret string) (*string, error)
	GenerateSecret(userID string) (secretKey *string, url *string, err error)
}
