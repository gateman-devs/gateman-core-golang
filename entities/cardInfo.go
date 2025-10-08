package entities

type CardInfo struct {
	ID                string  `json:"id" bson:"id"`
	AuthorizationCode string  `json:"authorization_code" bson:"authorizationCode"`
	Bin               string  `json:"bin" bson:"bin"`
	Last4             string  `json:"last4" bson:"last4"`
	ExpMonth          string  `json:"exp_month" bson:"expMonth"`
	ExpYear           string  `json:"exp_year" bson:"expYear"`
	Channel           string  `json:"channel" bson:"channel"`
	CardType          string  `json:"card_type" bson:"cardType"`
	Bank              string  `json:"bank" bson:"bank"`
	CountryCode       string  `json:"country_code" bson:"countryCode"`
	Brand             string  `json:"brand" bson:"brand"`
	Reusable          bool    `json:"reusable" bson:"reusable"`
	Signature         string  `json:"signature" bson:"signature"`
	AccountName       *string `json:"account_name" bson:"accountName"`
}
