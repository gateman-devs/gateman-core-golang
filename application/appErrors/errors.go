package apperrors

import (
	"fmt"
	"net/http"

	"gateman.io/infrastructure/logger"
	server_response "gateman.io/infrastructure/serverResponse"
)

func NotFoundError(ctx interface{}, message string) {
	server_response.Responder.Respond(ctx, http.StatusNotFound, message, nil, nil, nil, nil, nil)
}

func ValidationFailedError(ctx interface{}, errMessages *[]error) {
	server_response.Responder.Respond(ctx, http.StatusUnprocessableEntity, "Payload validation failed 🙄", nil, *errMessages, nil, nil, nil)
}

func EntityAlreadyExistsError(ctx interface{}, message string) {
	server_response.Responder.Respond(ctx, http.StatusConflict, message, nil, nil, nil, nil, nil)
}

func AuthenticationError(ctx interface{}, message string) {
	server_response.Responder.Respond(ctx, http.StatusUnauthorized, message, nil, nil, nil, nil, nil)
}

func ExternalDependencyError(ctx interface{}, serviceName string, statusCode string, err error) {
	logger.Error(err.Error(), logger.LoggerOptions{
		Key: fmt.Sprintf("error with %s. status code %s", serviceName, statusCode),
	})
	// logger.MetricMonitor.ReportError(fmt.Errorf(fmt.Sprintf("error with %s", serviceName)), []logger.LoggerOptions{
	// 	{
	// 		Key: "statusCode",
	// 		Data: statusCode,
	// 	},
	// })
	// logger.MetricMonitor.ReportError(err, nil, nil)
	server_response.Responder.Respond(ctx, http.StatusServiceUnavailable,
		"Omo! Our service is temporarily down 😢. Our team is working to fix it. Please check back later.", nil, nil, nil, nil, nil)
}

func ErrorProcessingPayload(ctx interface{}) {
	server_response.Responder.Respond(ctx, http.StatusBadRequest, "Abnormal payload passed 🤨", nil, nil, nil, nil, nil)
}

func FatalServerError(ctx interface{}, err error) {
	// logger.MetricMonitor.ReportError(err, nil, nil)
	server_response.Responder.Respond(ctx, http.StatusInternalServerError,
		"Omo! Our service is temporarily down 😢. Our team is working to fix it. Please check back later.", nil, nil, nil, nil, nil)
}

func UnknownError(ctx interface{}, err error, responseCode *uint) {
	// logger.MetricMonitor.ReportError(err, nil, nil)
	server_response.Responder.Respond(ctx, http.StatusBadRequest,
		"Omo! Something went wrong somewhere 😭. Please check back later.", nil, nil, responseCode, nil, nil)
}

func CustomError(ctx interface{}, msg string, responseCode *uint) {
	server_response.Responder.Respond(ctx, http.StatusBadRequest, msg, nil, nil, responseCode, nil, nil)
}

func UnsupportedAppVersion(ctx interface{}) {
	server_response.Responder.Respond(ctx, http.StatusBadRequest,
		"Uh oh! Seems you're using an old version of the app. 🤦🏻‍♂️\n Upgrade to the latest version to continue enjoying our blazing fast services! 🚀", nil, nil, nil, nil, nil)
}

func UnsupportedUserAgent(ctx interface{}) {
	// logger.MetricMonitor.ReportError(errors.New("unspported user agent"), []logger.LoggerOptions{
	// 	{Key: "ctx",
	// 	Data: ctx,},
	// })
	server_response.Responder.Respond(ctx, http.StatusBadRequest,
		"unsupported user agent 👮🏻‍♂️", nil, nil, nil, nil, nil)
}

func MalformedHeader(ctx interface{}) {
	// logger.MetricMonitor.ReportError(errors.New("unspported user agent"), []logger.LoggerOptions{
	// 	{Key: "ctx",
	// 	Data: ctx,},
	// })
	server_response.Responder.Respond(ctx, http.StatusBadRequest,
		"malformed header information 👮🏻‍♂️", nil, nil, nil, nil, nil)
}

func ClientError(ctx interface{}, msg string, errs []error, responseCode *uint) {
	server_response.Responder.Respond(ctx, http.StatusBadRequest, msg, nil, errs, responseCode, nil, nil)
}
