package auth_usecases

import (
	"crypto/ecdh"
	"crypto/rand"
	"fmt"
	"time"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/database/repository/cache"
)

func InitiateKeyExchange(ctx any, deviceID *string, clientPublicKey *ecdh.PublicKey, device_id *string) []byte {
	serverPrivateKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		apperrors.FatalServerError(ctx, err, device_id)
		return nil
	}
	sharedSecret, err := serverPrivateKey.ECDH(clientPublicKey)
	if err != nil {
		apperrors.FatalServerError(ctx, err, device_id)
		return nil
	}

	serverPublicKey := serverPrivateKey.PublicKey()
	encryptedSecret, err := cryptography.EncryptData([]byte((sharedSecret)), nil)
	if err != nil {
		apperrors.FatalServerError(ctx, err, device_id)
		return nil
	}
	success := cache.Cache.CreateEntry(fmt.Sprintf("%s-key", *deviceID), *encryptedSecret, time.Minute*15)
	if !success {
		apperrors.FatalServerError(ctx, nil, device_id)
		return nil
	}
	return serverPublicKey.Bytes()
}
