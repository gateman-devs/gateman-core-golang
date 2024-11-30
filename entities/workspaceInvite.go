package entities

import (
	"time"

	"authone.usepolymer.co/application/utils"
)

type WorkspaceInvite struct {
	WorkspaceID   string              `bson:"workspaceID" json:"workspaceID"`
	WorkspaceName string              `bson:"workspaceName" json:"workspaceName"`
	Email         string              `bson:"email" json:"email"`
	InvitedByID   string              `bson:"invitedByID" json:"invitedByID"`
	Accepted      *bool               `bson:"accepted" json:"accepted"`
	ResentAt      time.Time           `bson:"resentAt" json:"resentAt"`
	Permissions   []MemberPermissions `bson:"permissions" json:"permissions"`

	ID        string    `bson:"_id" json:"id"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

func (model WorkspaceInvite) ParseModel() any {
	now := time.Now()
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
		model.ID = utils.GenerateUULDString()
	}
	model.UpdatedAt = now
	return &model
}
