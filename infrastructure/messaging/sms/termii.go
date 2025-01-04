package sms

import (
	"encoding/json"
	"fmt"
	"os"

	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/infrastructure/logger"
	"authone.usepolymer.co/infrastructure/network"
)

type TermiiService struct {
	Network *network.NetworkController
	API_KEY string
}

func (ts *TermiiService) SendOTP(phone string, whatsapp bool, otp *string) *string {
	if os.Getenv("ENV") != "production" {
		return utils.GetStringPointer(utils.GenerateUULDString())
	}
	var response *[]byte
	var statusCode *int
	var err error
	if whatsapp {
		response, statusCode, err = ts.Network.Post("/sms/send", nil, map[string]any{
			"api_key":         ts.API_KEY,
			"from":            "Polymer",
			"to":              phone,
			"sms":             *otp,
			"type":            "plain",
			"channel":         "whatsapp_otp",
			"time_in_minutes": "10 minutes",
		}, nil, false, nil)
	} else {
		response, statusCode, err = ts.Network.Post("/sms/otp/send", nil, map[string]any{
			"api_key":          ts.API_KEY,
			"message_type":     "NUMERIC",
			"from":             "N-Alert",
			"to":               phone,
			"channel":          "dnd",
			"pin_attempts":     4,
			"pin_time_to_live": 10,
			"pin_length":       6,
			"pin_placeholder":  "< 123456 >",
			"message_text":     "Your Polymer confirmation code is < 123456 >. Valid for 10 minutes, one-time use only.",
			"pin_type":         "NUMERIC",
		}, nil, false, nil)
	}
	var termiiResponse TermiiOTPResponse
	json.Unmarshal(*response, &termiiResponse)
	if err != nil {
		logger.Error("error sending sms", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil
	}
	if *statusCode != 200 {
		logger.Error("request to termii for sms delivery was unsuccessful", logger.LoggerOptions{
			Key:  "statusCode",
			Data: fmt.Sprintf("%d", statusCode),
		}, logger.LoggerOptions{
			Key:  "data",
			Data: termiiResponse,
		})
		return nil
	}
	logger.Info(fmt.Sprintf("SMS OTP sent to %s", phone), logger.LoggerOptions{
		Key:  "res",
		Data: termiiResponse,
	})
	if whatsapp {
		return termiiResponse.Code
	}
	return termiiResponse.PinID
}

func (ts *TermiiService) VerifyOTP(otpID string, otp string) bool {
	if os.Getenv("ENV") != "production" {
		return otp == "000000"
	}
	response, statusCode, err := ts.Network.Post("/sms/otp/verify", nil, map[string]any{
		"api_key": ts.API_KEY,
		"pin":     otp,
		"pin_id":  otpID,
	}, nil, false, nil)
	var termiiResponse TermiiOTPVerifiedResponse
	json.Unmarshal(*response, &termiiResponse)
	if err != nil {
		logger.Error("error verifying termii otp", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return false
	}
	if *statusCode != 200 {
		logger.Error("error verifying termii otp", logger.LoggerOptions{
			Key:  "statusCode",
			Data: fmt.Sprintf("%d", statusCode),
		}, logger.LoggerOptions{
			Key:  "data",
			Data: termiiResponse,
		})
		return false
	}
	return termiiResponse.Verified
}
