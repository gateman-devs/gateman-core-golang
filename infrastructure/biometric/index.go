package biometric

import (
	"os"

	"gateman.io/infrastructure/biometric/types"
	"gateman.io/infrastructure/database/repository/cache"
	"gateman.io/infrastructure/network"
)

func init() {
	BiometricService = &GatemanFace{
		Network: &network.NetworkController{
			BaseUrl: os.Getenv("GATEMAN_FACE_BASE_URL"),
		},
		Cache: &cache.Cache,
	}
}

var BiometricService types.BiometricServiceType
