package dojah_identity_verification

import (
	"encoding/json"
	"errors"
	"fmt"

	identity_verification_types "gateman.io/infrastructure/identity_verification/types"
	"gateman.io/infrastructure/logger"
	"gateman.io/infrastructure/network"
)

type DojahIdentityVerification struct {
	Network *network.NetworkController
	API_KEY string
	APP_ID  string
}

func (div *DojahIdentityVerification) ImgLivenessCheck(img string) (bool, error) {
	response, statusCode, err := div.Network.Post("/ml/liveness/", &map[string]string{
		"Authorization": div.API_KEY,
		"AppId":         div.APP_ID,
	}, map[string]any{
		"image": img,
	}, nil, false, nil)
	if err != nil {
		logger.Error("error liveness check result from dojah", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return false, errors.New("something went wrong while retireving liveness check result from dojah")
	}
	var dojahResponse identity_verification_types.LivenessCheckResult
	err = json.Unmarshal(*response, &dojahResponse)
	if err != nil {
		logger.Error("error parsing liveness check result from dojah", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return false, errors.New("something went wrong while parsing liveness check result from dojah")
	}
	if *statusCode != 200 {
		logger.Error("request to Dojah for liveness check was unsuccessful", logger.LoggerOptions{
			Key:  "statusCode",
			Data: fmt.Sprintf("%d", statusCode),
		}, logger.LoggerOptions{
			Key:  "data",
			Data: dojahResponse,
		})
		return false, errors.New("error retireving liveness check result ")
	}
	logger.Info("liveness check completed by Dojah")
	if !dojahResponse.Entity.Liveness.LivenessCheck {
		return false, errors.New("Face verification failed. Please ensure you are in a well lit environment and have no coverings on your face.")
	}
	if dojahResponse.Entity.Liveness.LivenessProbability < 50.0 {
		return false, errors.New("Face verification failed. Please ensure you are in a well lit environment and have no coverings on your face.")
	}
	return true, nil
}

func (div *DojahIdentityVerification) FetchBVNDetails(bvn string) (*identity_verification_types.BVNData, error) {
	response, statusCode, err := div.Network.Get(fmt.Sprintf("/kyc/bvn/advance?bvn=%s", bvn), &map[string]string{
		"Authorization": div.API_KEY,
		"AppId":         div.APP_ID,
	}, nil)
	var dojahResponse DojahBVNResponse
	json.Unmarshal(*response, &dojahResponse)
	if err != nil {
		logger.Error("error retireving bvn data from dojah", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, errors.New("something went wrong while retireving bvn data from dojah")
	}
	if *statusCode != 200 {
		logger.Error("request to Dojah for BVN fetch was unsuccessful", logger.LoggerOptions{
			Key:  "statusCode",
			Data: fmt.Sprintf("%d", statusCode),
		}, logger.LoggerOptions{
			Key:  "data",
			Data: dojahResponse,
		})
		return nil, errors.New("error retireving bvn")
	}
	logger.Info("BVN information retireved by Dojah")
	return &dojahResponse.Data, nil
}

func (div *DojahIdentityVerification) FetchNINDetails(nin string) (*identity_verification_types.NINData, error) {
	response, statusCode, err := div.Network.Get(fmt.Sprintf("/kyc/nin/advance?nin=%s", nin), &map[string]string{
		"Authorization": div.API_KEY,
		"AppId":         div.APP_ID,
	}, nil)
	var dojahResponse DojahNINResponse
	json.Unmarshal(*response, &dojahResponse)
	if err != nil {
		logger.Error("error retireving nin data from dojah", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, errors.New("something went wrong while retireving nin data from dojah")
	}
	if *statusCode != 200 {
		logger.Error("request to Dojah for nin fetch was unsuccessful", logger.LoggerOptions{
			Key:  "statusCode",
			Data: fmt.Sprintf("%d", statusCode),
		}, logger.LoggerOptions{
			Key:  "data",
			Data: dojahResponse,
		})
		if dojahResponse.Error == "Wrong NIN Inputted" {
			return nil, errors.New("NIN not found. Crosscheck the number inputed")
		}
		return nil, errors.New("error retireving nin")
	}
	logger.Info("NIN information retireved by Dojah")
	return &dojahResponse.Data, nil
}

func (div *DojahIdentityVerification) EmailVerification(email string) (bool, error) {
	response, statusCode, err := div.Network.Get(fmt.Sprintf("/fraud/email?email_address=%s", email), &map[string]string{
		"Authorization": div.API_KEY,
		"AppId":         div.APP_ID,
	}, nil)
	var dojahResponse DojahEmailVerification
	json.Unmarshal(*response, &dojahResponse)
	if err != nil {
		logger.Error("error verifying email from dojah", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return false, errors.New("something went wrong while verifying email from dojah")
	}
	if *statusCode != 200 {
		logger.Error("request to Dojah email verification was unsuccessful", logger.LoggerOptions{
			Key:  "statusCode",
			Data: fmt.Sprintf("%d", statusCode),
		}, logger.LoggerOptions{
			Key:  "data",
			Data: dojahResponse,
		})
		return false, errors.New("error verifying email")
	}
	logger.Info("Email verification successful", logger.LoggerOptions{
		Key:  "email",
		Data: email,
	}, logger.LoggerOptions{
		Key:  "result",
		Data: dojahResponse,
	})
	return dojahResponse.Entity.Deliverable && !dojahResponse.Entity.DomainDetails.SusTLD && dojahResponse.Entity.DomainDetails.Registered && (dojahResponse.Entity.Score == 1), nil
}

func (div *DojahIdentityVerification) FetchDriverIDDetails(n string) (*identity_verification_types.DriversID, error) {
	response, statusCode, err := div.Network.Get(fmt.Sprintf("/kyc/dl?dl=%s", n), &map[string]string{
		"Authorization": div.API_KEY,
		"AppId":         div.APP_ID,
	}, nil)
	var dojahResponse DojahDriversLicenseResponse
	json.Unmarshal(*response, &dojahResponse)
	if err != nil {
		logger.Error("error retireving driver id data from dojah", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, errors.New("something went wrong while retireving driver id data from dojah")
	}
	if *statusCode != 200 {
		logger.Error("request to Dojah for driver id fetch was unsuccessful", logger.LoggerOptions{
			Key:  "statusCode",
			Data: fmt.Sprintf("%d", statusCode),
		}, logger.LoggerOptions{
			Key:  "data",
			Data: dojahResponse,
		})
		if dojahResponse.Error == "Wrong driver id Inputted" {
			return nil, errors.New("Drives License not found. Crosscheck the number inputed")
		}
		return nil, errors.New("error retireving driver id")
	}
	logger.Info("driver id information retireved by Dojah")
	return &dojahResponse.Data, nil
}

func (div *DojahIdentityVerification) FetchVoterIDDetails(vin string) (*identity_verification_types.VoterID, error) {
	response, statusCode, err := div.Network.Get(fmt.Sprintf("/kyc/vin?vin=%s", vin), &map[string]string{
		"Authorization": div.API_KEY,
		"AppId":         div.APP_ID,
	}, nil)
	var dojahResponse DojahVoterIDResponse
	json.Unmarshal(*response, &dojahResponse)
	if err != nil {
		logger.Error("error retireving voter id data from dojah", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, errors.New("something went wrong while retireving voter id data from dojah")
	}
	if *statusCode != 200 {
		logger.Error("request to Dojah for voter id fetch was unsuccessful", logger.LoggerOptions{
			Key:  "statusCode",
			Data: fmt.Sprintf("%d", statusCode),
		}, logger.LoggerOptions{
			Key:  "data",
			Data: dojahResponse,
		})
		if dojahResponse.Error == "Wrong voter id Inputted" {
			return nil, errors.New("Voter License not found. Crosscheck the number inputed")
		}
		return nil, errors.New("error retireving voter id")
	}
	logger.Info("voter id information retireved by Dojah")
	return &dojahResponse.Data, nil
}
