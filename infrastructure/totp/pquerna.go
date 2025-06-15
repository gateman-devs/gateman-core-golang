package totp

import (
	"time"

	"gateman.io/application/utils"
	"gateman.io/infrastructure/logger"
	"github.com/pquerna/otp/totp"
)

type PquernaTOTPService struct {
}

func (pq *PquernaTOTPService) GenerateSecret(userID string) (secretKey *string, url *string, err error) {
	secret, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Gateman Service",
		AccountName: userID,
	})
	if err != nil {
		logger.Error("an error occured while generating TOTP secret and QR code", logger.LoggerOptions{
			Key:  "err",
			Data: err,
		})
		return nil, nil, err
	}
	return utils.GetStringPointer(secret.Secret()), utils.GetStringPointer(secret.URL()), nil
}

func (pq *PquernaTOTPService) GenerateTOTPCode(secret string) (*string, error) {
	token, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		logger.Error("an error occured while generating TOTP token", logger.LoggerOptions{
			Key:  "err",
			Data: err,
		})
		return nil, err
	}
	return utils.GetStringPointer(token), err
}

func (pq *PquernaTOTPService) ValidateTOTP(token string, secret string) bool {
	success := totp.Validate(token, secret)
	return success
}
