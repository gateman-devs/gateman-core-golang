package controller

import (
	"encoding/hex"
	"net/http"

	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	auth_usecases "authone.usepolymer.co/application/usecases/auth"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
)

func KeyExchange(ctx *interfaces.ApplicationContext[dto.KeyExchangeDTO]) {
	serverPublicKey := auth_usecases.InitiateKeyExchange(ctx.Ctx, ctx.DeviceID, ctx.Body.ClientPublicKey, ctx.DeviceID, ctx.Nonce)
	if serverPublicKey == nil {
		return
	}
	server_response.Responder.UnEncryptedRespond(ctx.Ctx, http.StatusCreated, "key exchanged", hex.EncodeToString(serverPublicKey), nil, nil)
}
