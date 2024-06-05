package infrastructure

import (
	"crypto/ecdh"
	"fmt"
	"net/http"
	"os"
	"time"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/infrastructure/logger"
	middlewares "authone.usepolymer.co/infrastructure/middleware"
	ratelimit "authone.usepolymer.co/infrastructure/ratelimit"
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

	if err != nil {
		logger.Info("error loading env variables")
	}

	startup.StartServices()
	defer startup.CleanUpServices()

	server := gin.Default()
	origins := []string{}
	if os.Getenv("GIN_MODE") == "debug" {
		origins = append(origins, "http://localhost:5174")
	} else if os.Getenv("GIN_MODE") == "release" {
		origins = append(origins, "https://authone.usepolymer.co", "https://www.authone.usepolymer.co", "https://www.authone.usepolymer.co/")
	}
	corsConfig := cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "web-api-key", "ctx.DeviceID", "User-Agent"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	server.Use(cors.New(corsConfig))
	server.Use(ratelimit.TokenBucketPerIP())
	server.MaxMultipartMemory = 15 << 20 // 8 MiB

	// server.Use(logger.MetricMonitor.MetricMiddleware().(gin.HandlerFunc))
	server.Use(logger.RequestMetricMonitor.RequestMetricMiddleware().(func(*gin.Context)))

	v1 := server.Group("/api")
	v1.Use(middlewares.UserAgentMiddleware())

	// initiate key exchange for encryption
	v1.POST("/v1/auth/handshake", func(ctx *gin.Context) {
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

	routerV1 := v1.Group("/v1")
	// routerV1.Use(middlewares.DecryptPayloadMiddleware())
	{
		webRoutev1.AuthRouter(routerV1)
		webRoutev1.AppRouter(routerV1)
		webRoutev1.UserRouter(routerV1)
	}

	server.GET("/ping", func(ctx *gin.Context) {
		server_response.Responder.UnEncryptedRespond(ctx, http.StatusOK, "pong!", nil, nil, nil)
	})

	server.NoRoute(func(ctx *gin.Context) {
		apperrors.NotFoundError(ctx, fmt.Sprintf("%s %s does not exist", ctx.Request.Method, ctx.Request.URL), nil)
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
