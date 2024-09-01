package polymercore

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	"authone.usepolymer.co/application/utils"
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

func (pc *PolymerCore) generateAuthToken() (*string, error) {
	now := time.Now()
	token, err := auth.GenerateInterserviceAuthToken(auth.InterserviceClaimsData{
		Origination: os.Getenv("SERVICE_NAME"),
		IssuedAt:    now.Unix(),
		ExpiresAt:   now.Local().Add(time.Second * 30).Unix(), //lasts for 30 sec
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
	token, err := pc.generateAuthToken()
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
	token, err := pc.generateAuthToken()
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

func (pc *PolymerCore) GetPolymerID(email string) (*string, error) {
	token, err := pc.generateAuthToken()
	if err != nil {
		return nil, err
	}
	response, statusCode, err := pc.Network.Get("/api/v1/web/authone/user/"+email, &map[string]string{
		"x-is-token": *token,
	}, nil)
	if err != nil {
		logger.Error("could not complete request to polymer main to fetch users polymer id", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "email",
			Data: email,
		})
		return nil, err
	}
	if *statusCode != 200 && *statusCode != 404 {
		err = errors.New("could not complete request to polymer main to fetch polymer id")
		logger.Error(err.Error(), logger.LoggerOptions{
			Key:  "status code",
			Data: *statusCode,
		}, logger.LoggerOptions{
			Key:  "email",
			Data: email,
		})
		return nil, err
	}
	if *statusCode == 404 {
		logger.Info("user not found on Polymer main db")
		return nil, errors.New("")
	}
	var exists PolymerAccountExists
	err = json.Unmarshal(*response, &exists)
	if err != nil {
		err = errors.New("could not unmarshal response from polymer main to fetch polymer id")
		logger.Error(err.Error(), logger.LoggerOptions{
			Key:  "status code",
			Data: *statusCode,
		}, logger.LoggerOptions{
			Key:  "email",
			Data: email,
		})
		return nil, err
	}

	return utils.GetStringPointer(exists.Body["_id"]), nil
}

func (pc *PolymerCore) CreateAccount(email string, password string) bool {
	token, err := pc.generateAuthToken()
	if err != nil {
		return false
	}
	response, statusCode, err := pc.Network.Post("/api/v1/web/authone/account/create", &map[string]string{
		"x-is-token": *token,
	}, map[string]any{
		"email":    email,
		"password": password,
	}, nil, false, nil)
	if err != nil {
		logger.Error("could not complete request to polymer main to create user polymer account", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "email",
			Data: email,
		})
		return false
	}
	var res any
	err = json.Unmarshal(*response, &res)
	if err != nil {
		logger.Error(err.Error(), logger.LoggerOptions{
			Key:  "status code",
			Data: *statusCode,
		}, logger.LoggerOptions{
			Key:  "body",
			Data: res,
		}, logger.LoggerOptions{
			Key:  "email",
			Data: email,
		})
		return false
	}
	if *statusCode != 201 {
		err = errors.New("could not complete request to polymer main to  create user polymer account")
		logger.Error(err.Error(), logger.LoggerOptions{
			Key:  "status code",
			Data: *statusCode,
		}, logger.LoggerOptions{
			Key:  "body",
			Data: res,
		}, logger.LoggerOptions{
			Key:  "email",
			Data: email,
		})
		return false
	}

	return true
}

func (pc *PolymerCore) VerifyAccount(email string) bool {
	token, err := pc.generateAuthToken()
	if err != nil {
		return false
	}
	response, statusCode, err := pc.Network.Post("/api/v1/web/authone/account/verify", &map[string]string{
		"x-is-token": *token,
	}, map[string]any{
		"email":    email,
	}, nil, false, nil)
	if err != nil {
		logger.Error("could not complete request to polymer main to verify user polymer account", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "email",
			Data: email,
		})
		return false
	}
	var res any
	err = json.Unmarshal(*response, &res)
	if err != nil {
		logger.Error(err.Error(), logger.LoggerOptions{
			Key:  "status code",
			Data: *statusCode,
		}, logger.LoggerOptions{
			Key:  "body",
			Data: res,
		}, logger.LoggerOptions{
			Key:  "email",
			Data: email,
		})
		return false
	}
	if *statusCode != 201 {
		err = errors.New("could not complete request to polymer main to verify user polymer account")
		logger.Error(err.Error(), logger.LoggerOptions{
			Key:  "status code",
			Data: *statusCode,
		}, logger.LoggerOptions{
			Key:  "body",
			Data: res,
		}, logger.LoggerOptions{
			Key:  "email",
			Data: email,
		})
		return false
	}

	return true
}
