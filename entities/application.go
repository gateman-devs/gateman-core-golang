package entities

import (
	"time"

	"gateman.io/application/utils"
)

type RestrictionType string

var (
	Restrict RestrictionType = "restrict"
	Allow    RestrictionType = "allow"
)

type LocaleRestriction struct {
	States          *[]string       `bson:"states" json:"states" validate:"dive,min=2,max=15"`
	Country         string          `bson:"country" json:"country" validate:"required,iso3166_1_alpha2"`
	RestrictionType RestrictionType `bson:"restrictionType" json:"restrictionType" validate:"oneof=allow restrict"`
}

type RequestedField struct {
	Name     string `bson:"name" json:"name" validate:"required,oneof=BVN NIN FirstName LastName Gender MiddleName DOB Image Email Phone LoginLocale"`
	Verified bool   `bson:"verified" json:"verified"`
}

type VerificationType struct {
	Name     string `bson:"name" json:"name" validate:"required,min=2,max=50"`
	Required bool   `bson:"required" json:"required"`
}

type CustomFormField struct {
	Name      string                 `json:"name" bson:"name" validate:"required"`
	DBKey     string                 `json:"dbKey" bson:"dbKey" validate:"required"`
	FieldType string                 `json:"fieldType" bson:"fieldType" validate:"oneof=long_text short_text switch dropdown number secret pin"`
	Rules     []CustomValidationRule `json:"rules" bson:"rules"`
	Page      uint8                  `json:"page" bson:"page" validate:"required,min=1,max=10"`
}

type CustomValidationRule struct {
	Name  string  `json:"name" bson:"name" validate:"required"`
	Value *string `json:"value" bson:"value"`
}

type ValidationRule struct {
	AppliesTo *[]string `json:"appliesTo" bson:"-"`
	Tag       string    `json:"-" bson:"-"`
}

var ValidationRules map[string]ValidationRule = map[string]ValidationRule{
	"Required": {
		AppliesTo: &[]string{"long_text", "short_text", "switch", "dropdown", "number", "secret", "pin", "date"},
		Tag:       "required",
	},
	"Email": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "email",
	},
	"Minimum Length": {
		AppliesTo: &[]string{"long_text", "short_text", "secret", "pin"},
		Tag:       "min",
	},
	"Maximum Length": {
		AppliesTo: &[]string{"long_text", "short_text", "secret", "pin"},
		Tag:       "max",
	},
	"Exact Length": {
		AppliesTo: &[]string{"long_text", "short_text", "secret", "pin"},
		Tag:       "len",
	},
	"Numeric": {
		AppliesTo: &[]string{"long_text", "short_text", "number"},
		Tag:       "numeric",
	},
	"Alpha": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "alpha",
	},
	"Alpha Numeric": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "alphanum",
	},
	"Boolean": {
		AppliesTo: &[]string{"switch"},
		Tag:       "boolean",
	},
	"URL": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "url",
	},
	"UUID": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "uuid",
	},
	"IP Address": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "ip",
	},
	"IPv4 Address": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "ipv4",
	},
	"IPv6 Address": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "ipv6",
	},
	"Lowercase": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "lowercase",
	},
	"Uppercase": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "uppercase",
	},
	"Starts With": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "startswith",
	},
	"Ends With": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "endswith",
	},
	"Contains": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "contains",
	},
	"Excludes": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "excludes",
	},
	"Date": {
		AppliesTo: &[]string{"long_text", "short_text", "date"},
		Tag:       "datetime=2006-01-02",
	},
	"Date Time": {
		AppliesTo: &[]string{"long_text", "short_text", "date"},
		Tag:       "datetime=2006-01-02 15:04:05",
	},
	"Time": {
		AppliesTo: &[]string{"long_text", "short_text", "date"},
		Tag:       "datetime=15:04:05",
	},
	"Before Date": {
		AppliesTo: &[]string{"long_text", "short_text", "date"},
		Tag:       "ltfield",
	},
	"After Date": {
		AppliesTo: &[]string{"long_text", "short_text", "date"},
		Tag:       "gtfield",
	},
	"Minimum Tag": {
		AppliesTo: &[]string{"number"},
		Tag:       "gte",
	},
	"Maximum Tag": {
		AppliesTo: &[]string{"number"},
		Tag:       "lte",
	},
	"Equal To": {
		AppliesTo: &[]string{"number"},
		Tag:       "eq",
	},
	"Not Equal To": {
		AppliesTo: &[]string{"number"},
		Tag:       "ne",
	},
	"One Of": {
		AppliesTo: &[]string{"dropdown"},
		Tag:       "oneof",
	},
	"Not One Of": {
		AppliesTo: &[]string{"dropdown"},
		Tag:       "notoneof",
	},
	"Credit Card": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "credit_card",
	},
	"ISBN": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "isbn",
	},
	"JSON": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "json",
	},
	"Latitude": {
		AppliesTo: &[]string{"long_text", "short_text", "number"},
		Tag:       "latitude",
	},
	"Longitude": {
		AppliesTo: &[]string{"long_text", "short_text", "number"},
		Tag:       "longitude",
	},
	"Base64": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "base64",
	},
	"File Path": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "filepath",
	},
	"Hexadecimal": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "hexadecimal",
	},
	"ASCII": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "ascii",
	},
	"Printable ASCII": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "printascii",
	},
	"Multi Byte": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "multibyte",
	},
	"Data URI": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "datauri",
	},
	"MongoDB ObjectID": {
		AppliesTo: &[]string{"long_text", "short_text"},
		Tag:       "mongodb",
	},
	"Country Code (2 Letters)": {
		AppliesTo: &[]string{"long_text", "short_text", "dropdown"},
		Tag:       "iso3166_1_alpha2",
	},
	"Country Code (3 Letters)": {
		AppliesTo: &[]string{"long_text", "short_text", "dropdown"},
		Tag:       "iso3166_1_alpha3",
	},
	"Currency Code": {
		AppliesTo: &[]string{"long_text", "short_text", "dropdown"},
		Tag:       "iso4217",
	},
	"Language Code": {
		AppliesTo: &[]string{"long_text", "short_text", "dropdown"},
		Tag:       "bcp47_language_tag",
	},
}

type Application struct {
	Name                   string               `bson:"name" json:"name"`
	Disabled               bool                 `bson:"disabled" json:"disabled"`
	Description            string               `bson:"description" json:"description"`
	WorkspaceID            string               `bson:"workspaceID" json:"-"`
	AppImg                 string               `bson:"appImg" json:"appImg"`
	Email                  string               `bson:"email" json:"email"`
	AppID                  string               `bson:"appID" json:"appID"`
	PinProtected           bool                 `bson:"pinProtected" json:"pinProtected"`
	RequireAppMFA          bool                 `bson:"requireAppMFA" json:"requireAppMFA"`
	CreatorID              string               `bson:"creatorID" json:"-"`
	AppSigningKey          string               `bson:"appSigningKey" json:"-"`
	SandboxAppSigningKey   string               `bson:"sandBoxAppSigningKey" json:"-"`
	SandboxAPIKey          string               `bson:"sandBoxAPIKey" json:"-"`
	APIKey                 string               `bson:"apiKey" json:"-"`
	VPN                    bool                 `bson:"vpn" json:"vpn"`
	RefreshTokenTTL        uint32               `bson:"refreshTokenTTL" json:"refreshTokenTTL"`
	AccessTokenTTL         uint16               `bson:"accessTokenTTL" json:"accessTokenTTL"`
	SandboxRefreshTokenTTL uint32               `bson:"sandboxRefreshTokenTTL" json:"sandboxRefreshTokenTTL"`
	SandboxAccessTokenTTL  uint16               `bson:"sandboxAccessTokenTTL" json:"sandboxAccessTokenTTL"`
	Verifications          *[]VerificationType  `bson:"verifications" json:"verifications"`     // the verifications that must be completed before signup is approved
	RequestedFields        []RequestedField     `bson:"requestedFields" json:"requestedFields"` // the fields the application are interested in recieving. MUST NOT BE EMPTY
	LocaleRestriction      *[]LocaleRestriction `bson:"localeRestriction" json:"localeRestriction"`
	CustomFields           *[]CustomFormField   `bson:"customFields" json:"customFields"`
	PaymentCard            *string              `bson:"paymentCard" json:"paymentCard"`
	WhiteListedIPs         *[]string            `bson:"whiteListedIPs" json:"whiteListedIPs"`

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
		if model.ID == "" {
			model.ID = utils.GenerateUULDString()
		}
	}
	model.UpdatedAt = now
	return &model
}
