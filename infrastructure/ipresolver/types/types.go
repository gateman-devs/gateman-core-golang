package types

type IPResolver interface {
	LookUp(ipAddress string) (*IPResult, error)
}

type IPResult struct {
	AcuracyRadius int     `bson:"acuracyRadius" json:"acuracyRadius"`
	IPAddress     string  `bson:"ipAddress" json:"ipAddress"`
	Longitude     float64 `bson:"longitude" json:"longitude"`
	Latitude      float64 `bson:"latitude" json:"latitude"`
	City          string  `bson:"city" json:"city"`
	Anonymous     bool    `bson:"anonymous" json:"anonymous"`
	CountryCode   string  `bson:"countryCode" json:"countryCode"`
}
