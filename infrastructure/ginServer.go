package infrastructure

import (
	"crypto/ecdh"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/application/subscription"
	"gateman.io/infrastructure/facematch"
	"gateman.io/infrastructure/logger"
	middlewares "gateman.io/infrastructure/middleware"
	publicRouter "gateman.io/infrastructure/routes/ginRouter/web/publicAPI/v1"
	webRoutev1 "gateman.io/infrastructure/routes/ginRouter/web/v1"
	server_response "gateman.io/infrastructure/serverResponse"
	startup "gateman.io/infrastructure/startUp"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ginServer struct{}

func (s *ginServer) Start() {
	err := godotenv.Load()
	startup.StartServices()

	// Perform liveness check immediately after starting services

	err = facematch.InitializeFaceMatcherService()
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
		return
	}
	logger.Info("Performing face matcher liveness check...")

	result := facematch.GlobalFaceMatcher.DetectAntiSpoof("https://t4.ftcdn.net/jpg/01/24/84/83/360_F_124848388_cjx0VIF3BdR6yveB7ZguDSlEpM91tbrM.jpg")
	fmt.Println("result")
	fmt.Println("result")
	fmt.Println("result")
	fmt.Println("result")
	fmt.Println("result")
	fmt.Println("result")
	fmt.Println("result")
	fmt.Println("result")

	fmt.Println(result)

	if err != nil {
		logger.Info("error loading env variables")
	}

	defer startup.CleanUpServices()

	subscription.SeedSubscriptionData()

	server := gin.Default()

	server.GET("/metrics", gin.WrapH(promhttp.Handler()))

	origins := strings.Split(os.Getenv("ORIGINS"), ",")
	corsConfig := cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "x-device-id", "User-Agent", "x-workspace-id", "x-api-key", "x-app-id"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	server.Use(cors.New(corsConfig))
	server.MaxMultipartMemory = 0 // 8 MiB

	// server.Use(logger.MetricMonitor.MetricMiddleware().(gin.HandlerFunc))
	server.Use(logger.RequestMetricMonitor.RequestMetricMiddleware().(func(*gin.Context)))

	api := server.Group("/api")

	routerV1 := api.Group("/v1")
	routerV1.Use(middlewares.UserAgentMiddleware())

	routerV1.POST("/key-exchange", func(ctx *gin.Context) {
		// clientPubKeyBytes, _ := ctx.GetRawData()
		var body map[string]any
		if err := ctx.ShouldBindJSON(&body); err != nil {
			apperrors.ErrorProcessingPayload(ctx, nil)
			return
		}
		it, _ := hex.DecodeString(body["clientPubKey"].(string))
		clientPubKey, _ := ecdh.P256().NewPublicKey([]byte(it))
		deviceID := ctx.GetHeader("X-Device-Id")
		controller.KeyExchange(&interfaces.ApplicationContext[dto.KeyExchangeDTO]{
			Ctx: ctx,
			Body: &dto.KeyExchangeDTO{
				ClientPublicKey: clientPubKey,
			},
			DeviceID: deviceID,
		})
	})

	routerV1.Use(middlewares.DecryptPayloadMiddleware())
	{
		webRoutev1.AuthRouter(routerV1)
		webRoutev1.AppRouter(routerV1)
		webRoutev1.UserRouter(routerV1)
		webRoutev1.WorkspaceRouter(routerV1)
		webRoutev1.MiscRouter(routerV1)
	}

	publicAPI := api.Group("/public")
	publicV1 := publicAPI.Group("/v1")
	{
		publicRouter.AppRouter(publicV1)
		publicRouter.WebhookRouter(publicV1)
	}

	server.GET("/ping", func(ctx *gin.Context) {
		server_response.Responder.Respond(ctx, http.StatusOK, "pong!", nil, nil, nil, nil)
	})

	server.NoRoute(func(ctx *gin.Context) {
		deviceID := ctx.GetHeader("X-Device-Id")
		apperrors.NotFoundError(ctx, fmt.Sprintf("%s %s does not exist", ctx.Request.Method, ctx.Request.URL), &deviceID)
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
