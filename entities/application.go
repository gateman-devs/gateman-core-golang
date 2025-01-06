package entities

import (
	"time"

	"authone.usepolymer.co/application/utils"
)

type RestrictionType string

var (
	Restrict RestrictionType = "restrict"
	Allow    RestrictionType = "allow"
)

type LocaleRestriction struct {
	States          *[]string       `bson:"states" json:"states"`
	Country         string          `bson:"country" json:"country" validate:"required,iso3166_1_alpha2"`
	RestrictionType RestrictionType `bson:"restrictionType" json:"restrictionType"`
}

type RequestedField struct {
	Name     string `bson:"name" json:"name"`
	Verified bool   `bson:"verified" json:"verified"`
}

type Application struct {
	Name                  string               `bson:"name" json:"name"`
	Description           string               `bson:"description" json:"description"`
	WorkspaceID           string               `bson:"workspaceID" json:"-"`
	AppImg                string               `bson:"appImg" json:"appImg"`
	CreatorID             string               `bson:"creatorID" json:"-"`
	AppSigningKey         string               `bson:"appSigningKey" json:"-"`
	SandboxAppSigningKey  string               `bson:"sandBoxAppSigningKey" json:"-"`
	SandboxAPIKey         string               `bson:"sandBoxAPIKey" json:"-"`
	APIKey                string               `bson:"apiKey" json:"-"`
	VPN                   bool                 `bson:"vpn" json:"vpn"`
	RefreshTokenTTL       uint32               `bson:"refreshTokenTTL" json:"refreshTokenTTL"`
	AccessTokenTTL        uint16               `bson:"accessTokenTTL" json:"accessTokenTTL"`
	RequiredVerifications *[]string            `bson:"requiredVerifications" json:"requiredVerifications"` // the verifications that must be completed before signup is approved
	RequestedFields       []RequestedField     `bson:"requestedFields" json:"requestedFields"`             // the fields the application are interested in recieving. MUST NOT BE EMPTY
	LocaleRestriction     *[]LocaleRestriction `bson:"localeRestriction" json:"localeRestriction"`
	PaymentCard           *string              `bson:"paymentCard" json:"-"`

	ID            string     `bson:"_id" json:"id"`
	CreatedAt     time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt     *time.Time `bson:"deletedAt" json:"deletedAt"`
	DeletedReason *string    `bson:"deletedReason" json:"deletedReason"`
}

func (model Application) ParseModel() any {
	now := time.Now()
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
		model.ID = utils.GenerateUULDString()
	}
	model.UpdatedAt = now
	return &model
}
