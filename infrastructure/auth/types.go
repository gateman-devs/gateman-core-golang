package auth

type ClaimsData struct {
	Issuer      string
	UserID      string
	FirstName   string
	LastName    string
	Email       *string
	PhoneNum    *string
	WorkspaceID *string
	ExpiresAt   int64
	IssuedAt    int64
	UserAgent   string
	DeviceID    string
	Intent      string
}

type InterserviceClaimsData struct {
	Issuer      string
	Origination string
	ExpiresAt   int64
	IssuedAt    int64
}
