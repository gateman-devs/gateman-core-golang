package dto

type FetchAppDTO struct {
	GenerateImgURL bool `json:"generateImgURL"`
}

type FetchAppUsersDTO struct {
	AppID    string  `json:"appID" validate:"required"`
	PageSize int64   `json:"pageSize" validate:"required"`
	LastID   *string `json:"lastID" validate:"ulid"`
	Blocked  *bool   `json:"blocked"`
	Deleted  *bool   `json:"deleted"`
	Sort     int8    `json:"sort"`
}

type BlockAccountsDTO struct {
	IDs []string `json:"ids" validate:"dive,ulid"`
}
