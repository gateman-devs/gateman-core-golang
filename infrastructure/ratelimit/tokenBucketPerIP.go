package ratelimit

import (
	"encoding/json"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/didip/tollbooth_gin"
	"github.com/gin-gonic/gin"
)

func TokenBucketPerIP() gin.HandlerFunc {
	message := map[string]any{
		"message": "You are going too fast! You have been ratelimited.",
	}
	jsonMessage, _ := json.Marshal(message)

	tlbthLimiter := tollbooth.NewLimiter(25, &limiter.ExpirableOptions{
		DefaultExpirationTTL: time.Minute * 1,
	})
	tlbthLimiter.SetMessageContentType("application/json")
	tlbthLimiter.SetMessage(string(jsonMessage))

	return tollbooth_gin.LimitHandler(tlbthLimiter)
}
