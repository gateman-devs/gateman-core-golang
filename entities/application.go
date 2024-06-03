package entities

import (
	"time"

	"authone.usepolymer.co/application/utils"
)

type LocaleRestriction struct {
	States  *[]string `bson:"states" json:"states"`
	Country string    `bson:"country" json:"country" validate:"required,iso3166_1_alpha2"`
}

type Application struct {
	Name                  string               `bson:"name" json:"name"`
	OrgID                 string               `bson:"orgID" json:"-"`
	CreatorID             string               `bson:"creatorID" json:"-"`
	AppID                 string               `bson:"appID" json:"appID"`
	APIKey                string               `bson:"apiKey" json:"-"`
	RequiredVerifications *[]string            `bson:"requiredVerifications" json:"requiredVerifications"` // the verifications that must be completed before signup is approved
	RequestedFields       []string             `bson:"requestedFields" json:"requestedFields"`             // the fields the application are interested in recieving. MUST NOT BE EMPTY
	LocaleRestriction     *[]LocaleRestriction `bson:"localeRestriction" json:"localeRestriction"`

	ID        string    `bson:"_id" json:"id"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

func (model Application) ParseModel() any {
	now := time.Now()
	if model.ID == "" {
		model.CreatedAt = now
		model.ID = utils.GenerateUULDString()
	}
	model.UpdatedAt = now
	return &model
}
