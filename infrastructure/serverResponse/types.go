package server_response

type serverResponder interface {
	// Used to send a JSON response to the client.
	Respond(ctx interface{}, code int, message string, payload any, errs []error, responseCode *uint, deviceID *string)
	UnEncryptedRespond(ctx interface{}, code int, message string, payload any, errs []error, responseCode *uint)
}
