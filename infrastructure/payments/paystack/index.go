package paystack_local_payment_processor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"gateman.io/infrastructure/logger"
	"gateman.io/infrastructure/network"
	payment_types "gateman.io/infrastructure/payments/types"
)

var LocalPaymentProcessor *PaystackPaymentProcessor

type PaystackPaymentProcessor struct {
	Network   *network.NetworkController
	AuthToken string
}

func (paystack *PaystackPaymentProcessor) InitialisePaymentProcessor() {
	LocalPaymentProcessor = &PaystackPaymentProcessor{
		Network: &network.NetworkController{
			BaseUrl: os.Getenv("PAYSTACK_BASE_URL"),
		},
		AuthToken: os.Getenv("PAYSTACK_ACCESS_TOKEN"),
	}
}

func (paystack *PaystackPaymentProcessor) GeneratePaymentLink(email string, metadata map[string]any, amount uint32, channels []payment_types.PaymentChannel) (*payment_types.GeneratePaymentLinkResponse, error) {
	response, statusCode, err := paystack.Network.Post("/transaction/initialize", &map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", paystack.AuthToken),
	}, map[string]any{
		"currency":  "NGN",
		"amount":    amount,
		"email":     email,
		"channels":  channels,
		"metadata": metadata,
	}, nil, false, nil)
	if err != nil {
		logger.Error("an error occured while trying to call GeneratePaymentLink", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, errors.New("failed to generate payment link")
	}
	var paystackResponse PaystackGenertePaymentLinkResponse
	err = json.Unmarshal(*response, &paystackResponse)
	if err != nil {
		logger.Error("an error occured while trying to unmarshal GeneratePaymentLink response", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, errors.New("failed to generate payment link")
	}
	if *statusCode != 200 || !paystackResponse.Status {
		err = errors.New("failed to generate payment link")
		logger.Error("an error occured while trying to run GeneratePaymentLink", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "body",
			Data: paystackResponse,
		})
		return nil, nil
	}
	return &payment_types.GeneratePaymentLinkResponse{
		Link: *paystackResponse.Data.AuthURL,
	}, nil
}

func (paystack *PaystackPaymentProcessor) VerifyTransaction(id string) (any, error) {
	response, statusCode, err := paystack.Network.Get(fmt.Sprintf("/transaction/verify/%s", id), &map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", paystack.AuthToken),
		"Content-Type":  "application/json",
	}, nil)
	if err != nil {
		logger.Error("an error occured while trying to call VerifyTransaction", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, errors.New("failed to verify transaction on Paystack")
	}
	var paystackResponse PaystackTransactionVerificationResponse
	err = json.Unmarshal(*response, &paystackResponse)
	if err != nil {
		logger.Error("an error occured while trying to unmarshal VerifyTransaction response", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, errors.New("failed to verify transaction")
	}
	if *statusCode != 200 || !paystackResponse.Status {
		err = errors.New("failed to verify transaction")
		logger.Error("an error occured while trying to verify transaction", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "body",
			Data: paystackResponse,
		})
		return nil, nil
	}
	return paystackResponse.Data, nil
}

func (paystack *PaystackPaymentProcessor) ReverseTransaction(id string, reason string) (interface{}, error) {
	response, statusCode, err := paystack.Network.Post("/refund", &map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", paystack.AuthToken),
		"Content-Type":  "application/json",
	}, map[string]any{
		"transaction":   id,
		"merchant_note": reason,
	}, nil, false, nil)
	if err != nil {
		logger.Error("an error occured while trying to call ReverseTransaction", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, errors.New("failed to reverse transaction on Paystack")
	}
	var paystackResponse PaystackTransactionVerificationResponse
	err = json.Unmarshal(*response, &paystackResponse)
	if err != nil {
		logger.Error("an error occured while trying to unmarshal ReverseTransaction response", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, errors.New("failed to reverse transaction")
	}
	if *statusCode != 200 || !paystackResponse.Status {
		err = errors.New("failed to reverse transaction")
		logger.Error("an error occured while trying to reverse transaction", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "body",
			Data: paystackResponse,
		})
		return nil, nil
	}
	return paystackResponse.Data, nil
}

func (paystack *PaystackPaymentProcessor) ChargeCard(authorization_code string, email string, amount uint32, metadata map[string]any) (interface{}, error) {
	response, statusCode, err := paystack.Network.Post("/transaction/charge_authorization", &map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", paystack.AuthToken),
		"Content-Type":  "application/json",
	}, map[string]any{
		"authorization_code": authorization_code,
		"email":              email,
		"amount":             amount,
		"metadata":           metadata,
	}, nil, false, nil)
	if err != nil {
		logger.Error("an error occured while trying to call ChargeCard", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, errors.New("failed to charge card")
	}
	var paystackResponse PaystackTransactionVerificationResponse
	err = json.Unmarshal(*response, &paystackResponse)
	if err != nil {
		logger.Error("an error occured while trying to unmarshal ChargeCard response", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, errors.New("failed to charge card")
	}
	if *statusCode != 200 || !paystackResponse.Status {
		err = errors.New("failed to charge card")
		logger.Error("an error occured while trying to charge card", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "body",
			Data: paystackResponse,
		})
		return nil, nil
	}
	return paystackResponse.Data, nil
}
