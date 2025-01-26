package services

import (
	"encoding/json"
	"reflect"

	"authone.usepolymer.co/entities"
)

func ProcessUserSignUp(app *entities.Application, user *entities.User) (bool, string, map[string]any, map[string]any) {
	var eligible = true
	outstandingIDs := []string{}
	for _, id := range *app.RequiredVerifications {
		if id == "nin" {
			if user.NIN == nil {
				outstandingIDs = append(outstandingIDs, "nin")
				eligible = false
			}
		} else {
			if user.BVN == nil {
				outstandingIDs = append(outstandingIDs, "bvn")
				eligible = false
			}
		}
	}
	var results []string
	requestedFields := map[string]any{}
	userValue := reflect.ValueOf(*user)

	for _, field := range app.RequestedFields {
		userField := userValue.FieldByName(field.Name)
		if !userField.IsValid() {
			results = append(results, field.Name)
			eligible = false
			continue
		}
		var userFieldData entities.KYCData[any]
		actualValue := userField.Interface()
		jsonBytes, _ := json.Marshal(actualValue)
		json.Unmarshal(jsonBytes, &userFieldData)

		// If Verified field doesn't exist or is not true, add to results
		if userFieldData.Value == nil || !userFieldData.Verified {
			results = append(results, field.Name)
			eligible = false
		}
		requestedFields[field.Name] = userFieldData.Value
	}

	payload := map[string]any{}
	var msg string
	if eligible {
		msg = "Sign up successful"
	} else {
		msg = "Additional info is required to sign up to this app"
		payload["missingIDs"] = outstandingIDs
		payload["unverifiedFields"] = results
	}
	return eligible, msg, payload, requestedFields
}
