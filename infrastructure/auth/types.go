package auth

type ClaimsData struct {
	Issuer    string
	UserID    string
	FirstName string
	LastName  string
	Email     *string
	ExpiresAt int64
	IssuedAt  int64
	UserAgent string
	DeviceID  string
	OTPIntent string
}
