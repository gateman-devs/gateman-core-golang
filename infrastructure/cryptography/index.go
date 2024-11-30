package cryptography

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/infrastructure/database/repository/cache"
	"authone.usepolymer.co/infrastructure/logger"
)

var CryptoHahser Hasher = argonHasher{}

func GeneratePublicKey(sessionID string, clientPubKey *ecdh.PublicKey) *ecdh.PublicKey {
	serverCurve := ecdh.P256()
	serverPrivKey, err := serverCurve.GenerateKey(rand.Reader)
	if err != nil {
		logger.Error("error generating public key for key exchange", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil
	}
	serverPubKey := serverPrivKey.PublicKey()
	serverSecret, err := serverPrivKey.ECDH(clientPubKey)
	if err != nil {
		logger.Error("error generating server secret for key exchange", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil
	}
	cache.Cache.CreateEntry(sessionID, string(serverSecret), time.Minute*20)
	return serverPubKey
}

func DecryptData(stringToDecrypt string, keyString *string) ([]byte, error) {
	if keyString == nil {
		keyString = utils.GetStringPointer(os.Getenv("ENC_KEY"))
	}

	key, err := hex.DecodeString(*keyString)
	if err != nil {
		return nil, fmt.Errorf("invalid key format: %w", err)
	}

	ciphertext, err := base64.URLEncoding.DecodeString(stringToDecrypt)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 input: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short: must be at least %d bytes", aes.BlockSize)
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}

func EncryptData(payload []byte, keyString *string) (*string, error) {
	if keyString == nil {
		keyString = utils.GetStringPointer(os.Getenv("ENC_KEY"))
	}

	key, err := hex.DecodeString(*keyString)
	if err != nil {
		return nil, fmt.Errorf("invalid key format: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	ciphertext := make([]byte, aes.BlockSize+len(payload))
	iv := ciphertext[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], payload)

	encoded := base64.URLEncoding.EncodeToString(ciphertext)
	return utils.GetStringPointer(encoded), nil
}
