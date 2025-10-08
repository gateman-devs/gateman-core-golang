package public_controller

import (
	"net/http"

	"gateman.io/application/interfaces"
	identityverification "gateman.io/infrastructure/identity_verification"
	server_response "gateman.io/infrastructure/serverResponse"
)

func APIFetchNINDetails(ctx *interfaces.ApplicationContext[any]) {
	if ctx.Param["nin"] == nil {
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "id is required", nil, nil, nil, &ctx.DeviceID)
		return
	}
	fetchedNIN, _ := identityverification.IdentityVerifier.FetchNINDetails(ctx.Param["nin"].(string))
	if fetchedNIN == nil {
		server_response.Responder.Respond(ctx.Ctx, http.StatusNotFound, "NIN details not found", nil, nil, nil, &ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "NIN details fetched successfully", fetchedNIN, nil, nil, &ctx.DeviceID)
}

func APIFetchBVNDetails(ctx *interfaces.ApplicationContext[any]) {
	if ctx.Param["bvn"] == nil {
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "bvn is required", nil, nil, nil, &ctx.DeviceID)
		return
	}
	fetchedBVN, _ := identityverification.IdentityVerifier.FetchBVNDetails(ctx.Param["bvn"].(string))
	if fetchedBVN == nil {
		server_response.Responder.Respond(ctx.Ctx, http.StatusNotFound, "BVN details not found", nil, nil, nil, &ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "BVN details fetched successfully", fetchedBVN, nil, nil, &ctx.DeviceID)
}

func APIFetchVotersCardDetails(ctx *interfaces.ApplicationContext[any]) {
	if ctx.Param["votersID"] == nil {
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "voters id is required", nil, nil, nil, &ctx.DeviceID)
		return
	}
	fetchedVoterID, _ := identityverification.IdentityVerifier.FetchVoterIDDetails(ctx.Param["votersID"].(string))
	if fetchedVoterID == nil {
		server_response.Responder.Respond(ctx.Ctx, http.StatusNotFound, "Voter ID details not found", nil, nil, nil, &ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Voter ID details fetched successfully", fetchedVoterID, nil, nil, &ctx.DeviceID)
}

func APIFetchDriversLicenseDetails(ctx *interfaces.ApplicationContext[any]) {
	if ctx.Param["driversLicense"] == nil {
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "drivers license is required", nil, nil, nil, &ctx.DeviceID)
		return
	}
	fetchedDriversID, _ := identityverification.IdentityVerifier.FetchDriverIDDetails(ctx.Param["driversLicense"].(string))
	if fetchedDriversID == nil {
		server_response.Responder.Respond(ctx.Ctx, http.StatusNotFound, "Driver's License details not found", nil, nil, nil, &ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Driver's License details fetched successfully", fetchedDriversID, nil, nil, &ctx.DeviceID)
}
