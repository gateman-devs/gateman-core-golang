package dto

import (
	"gateman.io/entities"
	"gateman.io/infrastructure/file_upload/types"
)

type GeneratedSignedURLDTO struct {
	Permission   types.SignedURLPermission `json:"permission"`
	AccountImage bool                      `json:"accountImage"`
	FilePath     string                    `json:"filePath"  validate:"max=100,min=26"`
}

type GeneratePaymentLinkDTO struct {
	PlanID    string                         `json:"planID"  validate:"required,ulid"`
	AppID     string                         `json:"appID"  validate:"required,ulid"`
	AutoRenew bool                           `json:"autoRenew"`
	Frequency entities.SubscriptionFrequency `json:"frequency"  validate:"required,oneof=monthly annually"`
}

type GenerateAddCardLinkDTO struct {
	AppID *string `json:"appID"`
}
