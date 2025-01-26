package infrastructure

import (
	"crypto/ecdh"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/subscription"
	"authone.usepolymer.co/infrastructure/logger"
	middlewares "authone.usepolymer.co/infrastructure/middleware"
	ratelimit "authone.usepolymer.co/infrastructure/ratelimit"
	publicRouter "authone.usepolymer.co/infrastructure/routes/ginRouter/web/publicAPI/v1"
	webRoutev1 "authone.usepolymer.co/infrastructure/routes/ginRouter/web/v1"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
	startup "authone.usepolymer.co/infrastructure/startUp"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type ginServer struct{}

func (s *ginServer) Start() {
	err := godotenv.Load()
	startup.StartServices()

	if err != nil {
		logger.Info("error loading env variables")
	}

	defer startup.CleanUpServices()

	server := gin.Default()
	origins := strings.Split(os.Getenv("ORIGINS"), ",")
	corsConfig := cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "x-device-id", "User-Agent", "x-workspace-id"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	server.Use(cors.New(corsConfig))
	server.Use(ratelimit.TokenBucketPerIP())
	server.MaxMultipartMemory = 0 // 8 MiB
	subscription.SeedSubscriptionData()

	// server.Use(logger.MetricMonitor.MetricMiddleware().(gin.HandlerFunc))
	server.Use(logger.RequestMetricMonitor.RequestMetricMiddleware().(func(*gin.Context)))

	api := server.Group("/api")
	api.Use(middlewares.UserAgentMiddleware())

	// initiate key exchange for encryption
	api.POST("/v1/auth/handshake", func(ctx *gin.Context) {
		clientPubKeyBytes, _ := ctx.GetRawData()
		clientPubKey, _ := ecdh.P256().NewPublicKey(clientPubKeyBytes)
		appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
		controller.KeyExchange(&interfaces.ApplicationContext[dto.KeyExchangeDTO]{
			Ctx: ctx,
			Body: &dto.KeyExchangeDTO{
				ClientPublicKey: clientPubKey,
			},
			DeviceID: appContext.DeviceID,
		})
	})

	routerV1 := api.Group("/v1")
	// routerV1.Use(middlewares.DecryptPayloadMiddleware())
	{
		webRoutev1.AuthRouter(routerV1)
		webRoutev1.AppRouter(routerV1)
		webRoutev1.UserRouter(routerV1)
		webRoutev1.OrgRouter(routerV1)
	}

	publicAPI := api.Group("/public")
	publicV1 := publicAPI.Group("/v1")
	{
		publicRouter.AppRouter(publicV1)
	}

	server.GET("/ping", func(ctx *gin.Context) {
		server_response.Responder.Respond(ctx, http.StatusOK, "pong!", nil, nil, nil, nil, nil)
	})

	server.NoRoute(func(ctx *gin.Context) {
		apperrors.NotFoundError(ctx, fmt.Sprintf("%s %s does not exist", ctx.Request.Method, ctx.Request.URL))
	})

	gin_mode := os.Getenv("GIN_MODE")
	port := os.Getenv("PORT")
	if gin_mode == "debug" || gin_mode == "release" {
		logger.Info(fmt.Sprintf("Server starting on PORT %s", port))
		server.Run(fmt.Sprintf(":%s", port))
	} else {
		panic(fmt.Sprintf("invalid gin mode used - %s", gin_mode))
	}
}
