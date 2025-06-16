package entities

import (
	"fmt"
	"time"

	"gateman.io/application/utils"
)

type Device struct {
	LastLogin         time.Time `bson:"lastLogin" json:"lastLogin"`
	LastLoginLocation string    `bson:"lastLoginLocation" json:"lastLoginLocation"`
	Name              string    `bson:"name" json:"name"`
	ID                string    `bson:"id" json:"id"`
	Verified          bool      `bson:"verified" json:"-"`
}

type PhoneNumber struct {
	ISOCode     string `bson:"isoCode" json:"isoCode" validate:"iso3166_1_alpha2"` // Two-letter country code (ISO 3166-1 alpha-2)
	LocalNumber string `bson:"localNumber" json:"localNumber" validate:"len=10"`
	Prefix      string `bson:"prefix" json:"prefix" validate:"max=3,min=1"`
}

func (pn *PhoneNumber) ParsePhoneNumber() string {
	return fmt.Sprintf("+%s%s", pn.Prefix, pn.LocalNumber)
}

type KYCData[T any] struct {
	Value    *T   `bson:"value" json:"value"`
	Verified bool `bson:"verified" json:"verified"`
}

type Address struct {
	Value    *string `bson:"value" json:"value"`
	Country  *string `bson:"country" json:"country"`
	State    *string `bson:"state" json:"state"`
	LGA      *string `bson:"lga" json:"lga"`
	City     *string `bson:"city" json:"city"`
	Landmark *string `bson:"landmark" json:"landmark"`
	Verified bool    `bson:"verified" json:"verified"`
}

// This represents a user signed up to authone
type User struct {
	FirstName       *KYCData[string]    `bson:"firstName" json:"firstName"`
	LastName        *KYCData[string]    `bson:"lastName" json:"lastName"`
	MiddleName      *KYCData[string]    `bson:"middleName" json:"middleName"`
	DOB             *KYCData[time.Time] `bson:"dob" json:"dob"`
	Gender          *KYCData[string]    `bson:"gender" json:"gender"`
	Address         *Address            `bson:"address" json:"address"`
	NIN             *string             `bson:"nin" json:"nin"`
	BVN             *string             `bson:"bvn" json:"bvn"`
	VoterID         *string             `bson:"voterID" json:"voterID"`
	DriverID        *string             `bson:"driverID" json:"driverID"`
	AllowedOrgs     []string            `bson:"allowedOrgs" json:"allowedOrgs"`
	Email           *string             `bson:"email" json:"email,omitempty"`
	Phone           *PhoneNumber        `bson:"phone" json:"phone,omitempty"`
	Image           string              `bson:"image" json:"image"`
	UserAgent       string              `bson:"userAgent" json:"userAgent"`
	Deactivated     bool                `bson:"deactivated" json:"deactivated"`
	Blocked         bool                `bson:"blocked" json:"-"`
	BlockedReason   *string             `bson:"blockedReason" json:"blockedReason"`
	VerifiedAccount bool                `bson:"verifiedAccount" json:"verifiedAccount"`
	Devices         []Device            `bson:"devices" json:"devices"`

	ID            string     `bson:"_id" json:"id"`
	CreatedAt     time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt     *time.Time `bson:"deletedAt" json:"deletedAt"`
	DeletedReason *string    `bson:"deletedReason" json:"deletedReason"`
}

func (model User) ParseModel() any {
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
