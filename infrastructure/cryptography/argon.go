package cryptography

import (
	"authone.usepolymer.co/infrastructure/logger"
	"github.com/matthewhartstonge/argon2"
)

type argonHasher struct{}

func (ah argonHasher) HashString(data string, salt []byte) ([]byte, error) {
	config := argon2.DefaultConfig()
	raw, err := config.Hash([]byte(data), salt)
	if err != nil {
		logger.Error("argon - error while hashing data", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, err
	}

	return raw.Encode(), nil
}

func (ah argonHasher) VerifyHashData(hash string, data string) bool {
	raw, err := argon2.Decode([]byte(hash))
	if err != nil {
		logger.Error("argon - could not decode data", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "data",
			Data: hash,
		})
		return false
	}
	ok, err := raw.Verify([]byte(data))
	if err != nil {
		logger.Error("argon - error while verifying data", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "data",
			Data: data,
		}, logger.LoggerOptions{
			Key:  "hash",
			Data: hash,
		})
		ok = false
	}

	return ok
}
