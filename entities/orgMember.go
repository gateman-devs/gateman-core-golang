package entities

import (
	"time"

	"gateman.io/application/utils"
)

type MemberPermissions string

const (
	// org users
	USER_RESTRICT MemberPermissions = "user_restrict"
	USER_VIEW     MemberPermissions = "user_view"
	USER_BLOCK    MemberPermissions = "user_block"
	USER_DELETE   MemberPermissions = "user_delete"

	// org members
	MEMBER_INVITE      MemberPermissions = "member_invite"
	MEMBER_BLOCK       MemberPermissions = "member_block"
	MEMBER_REMOVE      MemberPermissions = "member_remove"
	MEMBER_EDIT_ACCESS MemberPermissions = "member_edit_access"

	// workspace
	WORKSPACE_EDIT_INFO             MemberPermissions = "workspace_edit_info"
	WORKSPACE_EDIT_DEFAULT_TEMPLATE MemberPermissions = "workspace_edit_default_template"
	WORKSPACE_EDIT_TEMPLATE_INFO    MemberPermissions = "workspace_edit_template_info"
	WORKSPACE_BILLING               MemberPermissions = "workspace_billing"
	WORKSPACE_CREATE_APPLICATIONS   MemberPermissions = "workspace_create_apps"
	WORKSPACE_VIEW_APPLICATIONS     MemberPermissions = "workspace_view_apps"
	WORKSPACE_DELETE_APPLICATIONS   MemberPermissions = "workspace_delete_apps"
	WORKSPACE_EDIT_APPLICATIONS     MemberPermissions = "workspace_edit_apps"

	// all
	SUPER_ACCESS MemberPermissions = "*"
)

type WorkspaceMember struct {
	FirstName     string              `bson:"firstName" json:"firstName"`
	LastName      string              `bson:"lastName" json:"lastName"`
	Email         string              `bson:"email" json:"email"`
	Username      string              `bson:"username" json:"username"`
	Password      string              `bson:"password" json:"-"`
	WorkspaceID   string              `bson:"workspaceID" json:"workspaceID"`
	WorkspaceName string              `bson:"workspaceName" json:"workspaceName"`
	Deactivated   bool                `bson:"deactivated" json:"deactivated"`
	VerifiedEmail bool                `bson:"verifiedEmail" json:"verifiedEmail"`
	Permissions   []MemberPermissions `bson:"permissions" json:"permissions"`
	UserAgent     string              `bson:"userAgent" json:"userAgent"`
	Devices       []Device            `bson:"devices" json:"devices"`

	ID            string     `bson:"_id" json:"id"`
	CreatedAt     time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt     *time.Time `bson:"deletedAt" json:"deletedAt"`
	DeletedReason *string    `bson:"deletedReason" json:"deletedReason"`
}

func (model WorkspaceMember) ParseModel() any {
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
