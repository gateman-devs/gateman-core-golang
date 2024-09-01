package auth_usecases

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/hex"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/infrastructure/cryptography"
)

func InitiateKeyExchange(ctx any, clientPublicKey string, deviceID *string) ([]byte, *string) {
	serverPrivateKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		apperrors.FatalServerError(ctx, err, deviceID)
		return nil, nil
	}
	clientPublicKeyByteArr, _ := hex.DecodeString(clientPublicKey)
	parsedClientPublicKey, _ := ecdh.P256().NewPublicKey(clientPublicKeyByteArr)
	sharedSecret, err := serverPrivateKey.ECDH(parsedClientPublicKey)
	if err != nil {
		apperrors.FatalServerError(ctx, err, deviceID)
		return nil, nil
	}

	serverPublicKey := serverPrivateKey.PublicKey()
	encryptedSecret, err := cryptography.EncryptData([]byte((sharedSecret)), nil)
	if err != nil {
		apperrors.FatalServerError(ctx, err, deviceID)
		return nil, nil
	}
	return serverPublicKey.Bytes(), encryptedSecret
}
