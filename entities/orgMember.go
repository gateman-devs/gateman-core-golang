package entities

import (
	"time"

	"authone.usepolymer.co/application/utils"
)

type MemberAccess string

const (
	// org users
	USER_RESTRICT MemberAccess = "user_restrict"
	USER_BLOCK    MemberAccess = "user_block"
	USER_DELETE   MemberAccess = "user_delete"

	// org members
	MEMBER_INVITE      MemberAccess = "member_invite"
	MEMBER_BLOCK       MemberAccess = "member_block"
	MEMBER_REMOVE      MemberAccess = "member_remove"
	MEMBER_EDIT_ACCESS MemberAccess = "member_edit_access"

	// organisation
	ORG_EDIT_INFO             MemberAccess = "org_edit_info"
	ORG_EDIT_DEFAULT_TEMPLATE MemberAccess = "org_edit_default_template"
	ORG_EDIT_TEMPLATE_INFO    MemberAccess = "org_edit_template_info"
	ORG_BILLING               MemberAccess = "org_billing"

	// all
	SUPER_ACCESS MemberAccess = "*"
)

type OrgMember struct {
	FirstName     string         `bson:"firstName" json:"firstName"`
	LastName      string         `bson:"lastName" json:"lastName"`
	Email         string         `bson:"email" json:"email"`
	Password      string         `bson:"password" json:"password"`
	AppVersion    string         `bson:"appVersion" json:"appVersion"`
	DeviceID      string         `bson:"deviceID" json:"deviceID"`
	Deactivated   bool           `bson:"deactivated" json:"deactivated"`
	VerifiedEmail bool           `bson:"verifiedEmail" json:"verifiedEmail"`
	Access        []MemberAccess `bson:"access" json:"access"`

	ID        string    `bson:"_id" json:"id"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

func (model OrgMember) ParseModel() any {
	now := time.Now()
	if model.ID == "" {
		model.CreatedAt = now
		model.ID = utils.GenerateUULDString()
	}
	model.UpdatedAt = now
	return &model
}
