package entities

import (
	"time"

	"authone.usepolymer.co/application/utils"
)

type MemberPermissions string

const (
	// org users
	USER_RESTRICT MemberPermissions = "user_restrict"
	USER_BLOCK    MemberPermissions = "user_block"
	USER_DELETE   MemberPermissions = "user_delete"

	// org members
	MEMBER_INVITE      MemberPermissions = "member_invite"
	MEMBER_BLOCK       MemberPermissions = "member_block"
	MEMBER_REMOVE      MemberPermissions = "member_remove"
	MEMBER_EDIT_ACCESS MemberPermissions = "member_edit_access"

	// organisation
	ORG_EDIT_INFO             MemberPermissions = "org_edit_info"
	ORG_EDIT_DEFAULT_TEMPLATE MemberPermissions = "org_edit_default_template"
	ORG_EDIT_TEMPLATE_INFO    MemberPermissions = "org_edit_template_info"
	ORG_BILLING               MemberPermissions = "org_billing"
	ORG_CREATE_APPLICATIONS   MemberPermissions = "create_apps"

	// all
	SUPER_ACCESS MemberPermissions = "*"
)

type OrgMember struct {
	Email       string              `bson:"email" json:"email"`
	UserAgent   string              `bson:"userAgent" json:"userAgent"`
	UserID      string              `bson:"userID" json:"userID"`
	DeviceID    string              `bson:"deviceID" json:"deviceID"`
	OrgID       string              `bson:"orgID" json:"orgID"`
	Deactivated bool                `bson:"deactivated" json:"deactivated"`
	Permissions []MemberPermissions `bson:"permissions" json:"permissions"`

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
