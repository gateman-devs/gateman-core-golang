package auth_usecases

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/repository/cache"
)

func InitiateKeyExchange(ctx any, deviceID string, clientPublicKey *ecdh.PublicKey) ([]byte, *string, error) {
	serverPrivateKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		// apperrors.FatalServerError(ctx, err, deviceID)
		return nil, nil, err
	}
	sharedSecret, err := serverPrivateKey.ECDH(clientPublicKey)
	if err != nil {
		// apperrors.FatalServerError(ctx, err, deviceID)
		return nil, nil, err
	}

	serverPublicKey := serverPrivateKey.PublicKey()

	parsedSharedSecret := hex.EncodeToString(sharedSecret)

	encryptedSecret, err := cryptography.EncryptData([]byte(parsedSharedSecret), nil)
	if err != nil {
		// apperrors.FatalServerError(ctx, err, deviceID)
		return nil, nil, err
	}
	success := cache.Cache.CreateEntry(fmt.Sprintf("%s-key", deviceID), *encryptedSecret, time.Minute*15)
	if !success {
		// apperrors.FatalServerError(ctx, nil, deviceID)
		return nil, nil, err
	}
	return serverPublicKey.Bytes(), &parsedSharedSecret, nil
}
