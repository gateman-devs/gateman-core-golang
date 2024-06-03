package maxmind

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"authone.usepolymer.co/infrastructure/ipresolver/types"
	"authone.usepolymer.co/infrastructure/logger"
	"github.com/oschwald/maxminddb-golang"
)

var db *maxminddb.Reader

type MaxMindIPResolver struct{}

func (mmResolver *MaxMindIPResolver) ConnectToDB() {
	var err error
	dbPath, _ := filepath.Abs("/infrastructure/ipresolver/maxmind/GeoLite2-City.mmdb")
	basePath, _ := filepath.Abs("")
	db, err = maxminddb.Open(fmt.Sprintf("%s%s", basePath, dbPath))
	if err != nil {
		logger.Error("could not connect to mmdb", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		panic(err)
	}
	logger.Info("connected to maxmind db successfully")
}

type maxmindLookupResult struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Location struct {
		Longitude      float64 `maxminddb:"longitude"`
		Latitude       float64 `maxminddb:"latitude"`
		AccuracyRadius int     `maxminddb:"accuracy_radius"`
	} `maxminddb:"location"`
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

func (mmResolver *MaxMindIPResolver) LookUp(ipAddress string) (*types.IPResult, error) {
	if os.Getenv("ENV") == "development" {
		ipAddress = "102.89.23.187"
	}
	ip := net.ParseIP(ipAddress)
	var result maxmindLookupResult
	err := db.Lookup(ip, &result)
	if err != nil {
		return nil, err
	}
	logger.Info("ip data fetched successfully")
	return &types.IPResult{
		Longitude:     result.Location.Longitude,
		Latitude:      result.Location.Latitude,
		City:          result.City.Names["en"],
		CountryCode:   result.Country.ISOCode,
		AcuracyRadius: result.Location.AccuracyRadius,
		IPAddress:     ipAddress,
	}, nil
}
