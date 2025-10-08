package entities

import (
	"time"

	"gateman.io/application/utils"
)

type RequestActivityLog struct {
	ID           string    `bson:"_id" json:"id"`
	AppID        string    `bson:"appID" json:"appID"`
	IPAddress    string    `bson:"ipAddress" json:"ipAddress"`
	Method       string    `bson:"method" json:"method"`
	URL          string    `bson:"url" json:"url"`
	QueryParams  *string   `bson:"queryParams" json:"queryParams"`
	RequestBody  *string   `bson:"requestBody" json:"requestBody"`
	ResponseBody *string   `bson:"responseBody" json:"responseBody"`
	StatusCode   int       `bson:"statusCode" json:"statusCode"`
	UserAgent    *string   `bson:"userAgent" json:"userAgent"`
	Timestamp    time.Time `bson:"timestamp" json:"timestamp"`
	Duration     int64     `bson:"duration" json:"duration"` // Duration in milliseconds
	CreatedAt    time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time `bson:"updatedAt" json:"updatedAt"`
}

func (r RequestActivityLog) ParseModel() interface{} {
	if r.ID == "" {
		r.ID = utils.GenerateUULDString()
	}
	now := time.Now()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	r.UpdatedAt = now
	if r.Timestamp.IsZero() {
		r.Timestamp = now
	}
	return &r
}
