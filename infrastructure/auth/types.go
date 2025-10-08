package auth

type TokenType string

var AccessToken TokenType = "access_token"
var RefreshToken TokenType = "refresh_token"

type ClaimsData struct {
	Issuer          string
	UserID          string
	FirstName       string
	LastName        string
	VerifiedAccount bool
	Email           *string
	PhoneNum        *string
	ExpiresAt       int64
	IssuedAt        int64
	UserAgent       string
	DeviceID        string
	Intent          string
	WorkspaceID     *string
	TokenType       TokenType
	Payload         map[string]any
}

type InterserviceClaimsData struct {
	Issuer      string
	Origination string
	ExpiresAt   int64
	IssuedAt    int64
}
