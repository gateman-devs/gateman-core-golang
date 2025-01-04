package middlewares

import (
	"errors"
	"fmt"
	"strings"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/infrastructure/useragent"
)

func UserAgentMiddleware(ctx *interfaces.ApplicationContext[any], minAppVersion string, clientIP string) (*interfaces.ApplicationContext[any], bool) {
	agent := ctx.GetHeader("User-Agent")
	if agent == nil {
		apperrors.ClientError(ctx.Ctx, "why your user-agent header no dey? You be criminal?🤨", []error{errors.New("user agent header missing")}, nil)
		return nil, false
	}
	agentDetails := useragent.ParseUserAgent(*agent)
	if agentDetails.Bot {
		apperrors.UnsupportedUserAgent(ctx.Ctx)
		return nil, false
	}

	reqSemVers := strings.Split(agentDetails.OSVersion, ".")
	if len(reqSemVers) < 3 {
		apperrors.UnsupportedUserAgent(ctx.Ctx)
		return nil, false
	}
	minAppVersionSemVers := strings.Split(minAppVersion, ".")
	if len(minAppVersionSemVers) < 3 {
		apperrors.UnsupportedUserAgent(ctx.Ctx)
		return nil, false
	}

	if minAppVersionSemVers[0] > reqSemVers[0] {
		apperrors.UnsupportedUserAgent(ctx.Ctx)
		return nil, false
	}
	if minAppVersionSemVers[1] > reqSemVers[1] {
		apperrors.UnsupportedUserAgent(ctx.Ctx)
		return nil, false
	}
	if minAppVersionSemVers[2] > reqSemVers[2] {
		apperrors.UnsupportedUserAgent(ctx.Ctx)
		return nil, false
	}
	ctx.UserAgent = *agent
	ctx.DeviceName = fmt.Sprintf("%s/%s", agentDetails.Device, agentDetails.Name)
	deviceID := ctx.GetHeader("X-Device-Id")
	if deviceID == nil || *deviceID == "" {
		apperrors.MalformedHeader(ctx.Ctx)
		return nil, false
	}
	ctx.DeviceID = *deviceID
	return ctx, true
}
