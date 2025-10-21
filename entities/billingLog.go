package entities

import (
	"time"

	"gateman.io/application/utils"
)

type BillingLog struct {
	ID              string    `bson:"_id" json:"id"`
	WorkspaceID     string    `bson:"workspaceID" json:"workspaceID"`
	RequestID       string    `bson:"requestID" json:"requestID"`
	ActivityLogID   string    `bson:"activityLogID" json:"activityLogID"`
	Operation       string    `bson:"operation" json:"operation"` // "liveness" or "comparison"
	Amount          int64     `bson:"amount" json:"amount"`       // Amount deducted in kobo
	Status          string    `bson:"status" json:"status"`       // "success", "failed", "reverted"
	TransactionID   string    `bson:"transactionID" json:"transactionID"`
	ErrorMessage    *string   `bson:"errorMessage" json:"errorMessage"`
	CreatedAt       time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time `bson:"updatedAt" json:"updatedAt"`
}

func (model BillingLog) ParseModel() any {
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
