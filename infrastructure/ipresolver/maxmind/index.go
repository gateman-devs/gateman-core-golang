package maxmind

import (
	"context"
	"encoding/json"
	"os"

	"gateman.io/infrastructure/ipresolver/types"
	"github.com/savaki/geoip2"
)

type MaxMindIPResolver struct{}

func (mmResolver *MaxMindIPResolver) LookUp(ipAddress string) (*types.IPResult, error) {
	if os.Getenv("APP_ENV") != "production" {
		return &types.IPResult{
			Longitude:     6.789,
			Latitude:      6.543,
			City:          "lagos",
			CountryCode:   "ng",
			AcuracyRadius: 0,
			IPAddress:     ipAddress,
			Anonymous:     false,
		}, nil
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
