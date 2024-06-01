package cryptography

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
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

func DecryptData(stringToDecrypt string, keyString *string, ivText *string) ([]byte, error) {
	if keyString == nil {
		keyString = utils.GetStringPointer(os.Getenv("ENC_KEY"))
	}
	key, _ := hex.DecodeString(*keyString)
	ciphertext, _ := base64.URLEncoding.DecodeString(stringToDecrypt)

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}
	var iv []byte
	if ivText == nil {
		iv = ciphertext[:aes.BlockSize]
	} else {
		iv, _ = base64.URLEncoding.DecodeString(*ivText)
	}

	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext, nil
}

func EncryptData(payload []byte, keyString *string) (encryptedString *string, err error) {
	// convert key to bytes
	if keyString == nil {
		keyString = utils.GetStringPointer(os.Getenv("ENC_KEY"))
	}
	key, _ := hex.DecodeString(*keyString)

	//Create a new Cipher Block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		logger.Error(err.Error())
		panic(err.Error())
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(payload))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], payload)

	// convert to base64
	return utils.GetStringPointer(base64.URLEncoding.EncodeToString(ciphertext)), nil
}
