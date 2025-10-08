package env

import (
	"gateman.io/infrastructure/logger"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		logger.Info("error loading env variables")
	}
}

func LoadEnv() {
}
