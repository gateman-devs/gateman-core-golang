package entities

import (
	"fmt"
	"time"

	"authone.usepolymer.co/application/utils"
)

type Device struct {
	Name     string `bson:"name" json:"name"`
	ID       string `bson:"id" json:"id"`
	Secret   string `bson:"secret" json:"-"`
	Verified bool   `bson:"verified" json:"-"`
}

type PhoneNumber struct {
	ISOCode     string `bson:"isoCode" json:"isoCode" validate:"iso3166_1_alpha2"` // Two-letter country code (ISO 3166-1 alpha-2)
	LocalNumber string `bson:"localNumber" json:"localNumber"`
	Prefix      string `bson:"prefix" json:"prefix"`
}

func (pn *PhoneNumber) ParsePhoneNumber() string {
	return fmt.Sprintf("+%s%s", pn.Prefix, pn.LocalNumber)
}

// This represents a user signed up to authone
type User struct {
	PolymerID       string       `bson:"polymerID" json:"polymerID"`
	AllowedOrgs     []string     `bson:"allowedOrgs" json:"allowedOrgs"`
	Email           *string      `bson:"email" json:"email,omitempty"`
	Phone           *PhoneNumber `bson:"phone" json:"phone,omitempty"`
	Image           string       `bson:"image" json:"image"`
	Password        string       `bson:"password" json:"-"`
	UserAgent       string       `bson:"userAgent" json:"userAgent"`
	Deactivated     bool         `bson:"deactivated" json:"deactivated"`
	Blocked         bool         `bson:"blocked" json:"-"`
	BlockedReason   *string      `bson:"blockedReason" json:"blockedReason"`
	VerifiedAccount bool         `bson:"verifiedAccount" json:"verifiedAccount"`
	Devices         []Device     `bson:"devices" json:"devices"`

	ID            string     `bson:"_id" json:"id"`
	CreatedAt     time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt     *time.Time `bson:"deletedAt" json:"deletedAt"`
	DeletedReason *string    `bson:"deletedReason" json:"deletedReason"`
}

func (model User) ParseModel() any {
	now := time.Now()
	if model.ID == "" {
		model.CreatedAt = now
		model.ID = utils.GenerateUULDString()
	}
	model.UpdatedAt = now
	return &model
}
