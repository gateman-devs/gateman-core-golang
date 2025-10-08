package middlewares

import (
	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/interfaces"
	"gateman.io/infrastructure/ipresolver"
	"gateman.io/infrastructure/logger"
)

func IPAddressMiddleware(ctx *interfaces.ApplicationContext[any], clientIP string) (*interfaces.ApplicationContext[any], bool) {
	return ctx, true
	ipLookupRes, err := ipresolver.IPResolverInstance.LookUp(clientIP)
	if err != nil {
		logger.Error("error looking up ip", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "ip",
			Data: clientIP,
		})
		apperrors.FatalServerError(ctx.Ctx, err, ctx.DeviceID)
		return nil, false
	}
	logger.Info("request-ip-lookup", logger.LoggerOptions{
		Key:  "ip-data",
		Data: ipLookupRes,
	})

	ctx.SetContextData("Latitude", ipLookupRes.Latitude)
	ctx.SetContextData("Longitude", ipLookupRes.Longitude)
	ctx.SetContextData("VPN", ipLookupRes.Anonymous)
	return ctx, true
}
