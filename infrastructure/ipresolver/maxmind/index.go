package maxmind

import (
	"context"
	"encoding/json"
	"os"

	"authone.usepolymer.co/infrastructure/ipresolver/types"
	"github.com/savaki/geoip2"
)

type MaxMindIPResolver struct{}

func (mmResolver *MaxMindIPResolver) LookUp(ipAddress string) (*types.IPResult, error) {
	if os.Getenv("ENV") == "development" {
		ipAddress = "102.88.111.43"
	}

	api := geoip2.New(os.Getenv("MAXMIND_USER_ID"), os.Getenv("MAXMIND_LICENSE_KEY"))
	result, _ := api.Insights(context.TODO(), ipAddress)
	json.NewEncoder(os.Stdout).Encode(result)
	return &types.IPResult{
		Longitude:     result.Location.Longitude,
		Latitude:      result.Location.Latitude,
		City:          result.City.Names["en"],
		CountryCode:   result.Country.IsoCode,
		AcuracyRadius: result.Location.AccuracyRadius,
		IPAddress:     ipAddress,
		Anonymous:     result.Traits.IsAnonymousProxy,
	}, nil
}
