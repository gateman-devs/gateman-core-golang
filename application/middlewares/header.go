package middlewares

import (
	"errors"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/interfaces"
	"gateman.io/infrastructure/useragent"
)

func UserAgentMiddleware(ctx *interfaces.ApplicationContext[any], minAppVersion string, clientIP string) (*interfaces.ApplicationContext[any], bool) {
	agent := ctx.GetHeader("User-Agent")
	if agent == nil {
		apperrors.ClientError(ctx.Ctx, "why your user-agent header no dey? You be criminal?🤨", []error{errors.New("user agent header missing")}, nil, *ctx.GetHeader("X-Device-Id"))
		return nil, false
	}
	agentDetails := useragent.ParseUserAgent(*agent)
	ctx.UserAgent = *agent
	ctx.DeviceName = agentDetails.Name
	deviceID := ctx.GetHeader("X-Device-Id")
	if deviceID == nil || *deviceID == "" {
		apperrors.MalformedHeader(ctx.Ctx, nil)
		return nil, false
	}
	ctx.DeviceID = *deviceID
	return ctx, true
}
