package polymercore

import (
	"errors"
	"os"
	"time"

	"authone.usepolymer.co/infrastructure/auth"
	"authone.usepolymer.co/infrastructure/logger"
	"authone.usepolymer.co/infrastructure/network"
)

var PolymerService PolymerCore

type PolymerCore struct {
	Network *network.NetworkController
}

func (pc *PolymerCore) Initialise() {
	PolymerService = PolymerCore{
		Network: &network.NetworkController{
			BaseUrl: os.Getenv("POLYMER_CORE_URL"),
		},
	}
}

func (pc *PolymerCore) GenerateAuthToken() (*string, error) {
	now := time.Now()
	token, err := auth.GenerateInterserviceAuthToken(auth.InterserviceClaimsData{
		Origination: os.Getenv("SERVICE_NAME"),
		IssuedAt:    now.Unix(),
		ExpiresAt:   now.Local().Add(time.Second * time.Duration(30)).Unix(), //lasts for 30 sec
	})
	if err != nil {
		logger.Error("an error occured while generating interservice auth token", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, err
	}
	return token, nil
}

func (pc *PolymerCore) SendEmail(template string, email string, subject string, opts map[string]any) error {
	token, err := pc.GenerateAuthToken()
	if err != nil {
		return err
	}
	_, statusCode, err := pc.Network.Post("/api/v1/web/authone/email/send", &map[string]string{
		"x-is-token": *token,
	}, map[string]any{
		"email":    email,
		"template": template,
		"subject":  subject,
		"opts":     opts,
	}, nil, false, nil)
	if err != nil {
		logger.Error("could not complete request to polymer main to send email", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return err
	}
	if *statusCode != 200 {
		err = errors.New("could not successfully send email")
		logger.Error(err.Error(), logger.LoggerOptions{
			Key:  "status code",
			Data: *statusCode,
		})
		return err
	}
	return nil
}

func (pc *PolymerCore) EmailStatus(email string) (bool, error) {
	token, err := pc.GenerateAuthToken()
	if err != nil {
		return false, err
	}
	response, statusCode, err := pc.Network.Post("/api/v1/web/authone/email/status", &map[string]string{
		"x-is-token": *token,
	}, email, nil, false, nil)
	if err != nil {
		logger.Error("could not complete request to polymer main to fetch email status", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return false, err
	}
	if *statusCode != 200 {
		err = errors.New("could not complete request to polymer main to fetch email status")
		logger.Error(err.Error(), logger.LoggerOptions{
			Key:  "status code",
			Data: *statusCode,
		})
		return false, err
	}
	if len(*response) == 0 {
		return false, nil
	}
	return (*response)[0] != 0, nil
}
