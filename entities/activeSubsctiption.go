package entities

import (
	"time"

	"gateman.io/application/utils"
)

type ActiveSubscription struct {
	SubscriptionID string                `bson:"subscriptionID" json:"subscriptionID"`
	Active         bool                  `bson:"active" json:"active"`
	ActiveSubID    string                `bson:"activeSubID" json:"activeSubID"`
	ActiveSubName  SubscriptionPlanName  `bson:"activeSubName" json:"activeSubName"`
	AutoRenew      bool                  `bson:"autoRenew" json:"autoRenew"`
	AppID          string                `bson:"appID" json:"appID"`
	WorkspaceID    string                `bson:"workspaceID" json:"workspaceID"`
	Name           SubscriptionPlanName  `bson:"name" json:"name"`
	Interval       SubscriptionFrequency `bson:"interval" json:"interval"`
	ExpiresOn      *time.Time            `bson:"expiresOn" json:"expiresOn"`
	RenewedOn      *time.Time            `bson:"renewedOn" json:"renewedOn"`
	CancelledOn    *time.Time            `bson:"cancelledOn" json:"cancelledOn"`

	ID            string     `bson:"_id" json:"id"`
	CreatedAt     time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt     *time.Time `bson:"deletedAt" json:"deletedAt"`
	DeletedReason *string    `bson:"deletedReason" json:"deletedReason"`
}

func (model ActiveSubscription) ParseModel() any {
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
