package apperrors

import (
	"fmt"
	"net/http"

	"gateman.io/infrastructure/logger"
	server_response "gateman.io/infrastructure/serverResponse"
)

func NotFoundError(ctx interface{}, message string, deviceID *string) {
	server_response.Responder.Respond(ctx, http.StatusNotFound, message, nil, nil, nil, deviceID)
}

func ValidationFailedError(ctx interface{}, errMessages *[]error, deviceID string) {
	server_response.Responder.Respond(ctx, http.StatusUnprocessableEntity, "Payload validation failed 🙄", nil, *errMessages, nil, &deviceID)
}

func EntityAlreadyExistsError(ctx interface{}, message string, deviceID string) {
	server_response.Responder.Respond(ctx, http.StatusConflict, message, nil, nil, nil, &deviceID)
}

func AuthenticationError(ctx interface{}, message string, deviceID string) {
	server_response.Responder.Respond(ctx, http.StatusUnauthorized, message, nil, nil, nil, &deviceID)
}

func ExternalDependencyError(ctx interface{}, serviceName string, statusCode string, err error, deviceID string) {
	logger.Error(err.Error(), logger.LoggerOptions{
		Key: fmt.Sprintf("error with %s. status code %s", serviceName, statusCode),
	})
	// logger.MetricMonitor.ReportError(fmt.Errorf(fmt.Sprintf("error with %s", serviceName)), []logger.LoggerOptions{
	// 	{
	// 		Key: "statusCode",
	// 		Data: statusCode,
	// 	},
	// })
	// logger.MetricMonitor.ReportError(err, nil, nil, nil)
	server_response.Responder.Respond(ctx, http.StatusServiceUnavailable,
		"Omo! Our service is temporarily down 😢. Our team is working to fix it. Please check back later.", nil, nil, nil, &deviceID)
}

func ErrorProcessingPayload(ctx interface{}, deviceID *string) {
	server_response.Responder.Respond(ctx, http.StatusBadRequest, "Abnormal payload passed 🤨", nil, nil, nil, deviceID)
}

func FatalServerError(ctx interface{}, err error, deviceID string) {
	// logger.MetricMonitor.ReportError(err, nil, nil, nil)
	server_response.Responder.Respond(ctx, http.StatusInternalServerError,
		"Omo! Our service is temporarily down 😢. Our team is working to fix it. Please check back later.", nil, nil, nil, &deviceID)
}

func UnknownError(ctx interface{}, err error, responseCode *uint, deviceID string) {
	// logger.MetricMonitor.ReportError(err, nil, nil, nil)
	server_response.Responder.Respond(ctx, http.StatusBadRequest,
		"Omo! Something went wrong somewhere 😭. Please check back later.", nil, nil, responseCode, &deviceID)
}

func CustomError(ctx interface{}, msg string, responseCode *uint, deviceID string) {
	server_response.Responder.Respond(ctx, http.StatusBadRequest, msg, nil, nil, responseCode, &deviceID)
}

func UnsupportedAppVersion(ctx interface{}, deviceID string) {
	server_response.Responder.Respond(ctx, http.StatusBadRequest,
		"Uh oh! Seems you're using an old version of the app. 🤦🏻‍♂️\n Upgrade to the latest version to continue enjoying our blazing fast services! 🚀", nil, nil, nil, &deviceID)
}

func UnsupportedUserAgent(ctx interface{}, deviceID string) {
	// logger.MetricMonitor.ReportError(errors.New("unspported user agent"), []logger.LoggerOptions{
	// 	{Key: "ctx",
	// 	Data: ctx,},
	// })
	server_response.Responder.Respond(ctx, http.StatusBadRequest,
		"unsupported user agent 👮🏻‍♂️", nil, nil, nil, &deviceID)
}

func MalformedHeader(ctx interface{}, deviceID *string) {
	// logger.MetricMonitor.ReportError(errors.New("unspported user agent"), []logger.LoggerOptions{
	// 	{Key: "ctx",
	// 	Data: ctx,},
	// })
	server_response.Responder.Respond(ctx, http.StatusBadRequest,
		"malformed header information 👮🏻‍♂️", nil, nil, nil, deviceID)
}

func ClientError(ctx interface{}, msg string, errs []error, responseCode *uint, deviceID string) {
	server_response.Responder.Respond(ctx, http.StatusBadRequest, msg, nil, errs, responseCode, &deviceID)
}
