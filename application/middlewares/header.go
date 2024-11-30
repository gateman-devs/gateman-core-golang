package middlewares

import (
	"errors"
	"fmt"
	"strings"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/infrastructure/ipresolver"
	"authone.usepolymer.co/infrastructure/logger"
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

	ipLookupRes, err := ipresolver.IPResolverInstance.LookUp(clientIP)
	if err != nil {
		logger.Error("error looking up ip", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		},
			logger.LoggerOptions{
				Key:  "ip",
				Data: clientIP,
			}, logger.LoggerOptions{
				Key:  "user agent",
				Data: agent,
			})
		return nil, false
	}
	logger.Info("request-ip-lookup", logger.LoggerOptions{
		Key:  "ip-data",
		Data: ipLookupRes,
	}, logger.LoggerOptions{
		Key:  "user-agent",
		Data: *agent,
	})

	ctx.SetContextData("Latitude", ipLookupRes.Latitude)
	ctx.SetContextData("Longitude", ipLookupRes.Longitude)

	ctx.UserAgent = agent
	ctx.DeviceName = utils.GetStringPointer(fmt.Sprintf("%s/%s", agentDetails.Device, agentDetails.Name))
	ctx.DeviceID = ctx.GetHeader("X-Device-Id")
	if ctx.DeviceID == nil || *ctx.DeviceID == "" {
		apperrors.MalformedHeader(ctx.Ctx)
		return nil, false
	}
	return ctx, true
}
