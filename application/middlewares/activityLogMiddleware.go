package middlewares

import (
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"time"

	"gateman.io/application/repository"
	"gateman.io/entities"
	"gateman.io/infrastructure/logger"
	"github.com/gin-gonic/gin"
)

// responseWriter wraps gin.ResponseWriter to capture response body
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// ActivityLogMiddleware logs request and response data to MongoDB
func ActivityLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Capture request body
		var requestBody string
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				requestBody = string(bodyBytes)
				// Restore the body for the next handler
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// Wrap response writer to capture response body
		responseBodyBuffer := &bytes.Buffer{}
		wrappedWriter := &responseWriter{
			ResponseWriter: c.Writer,
			body:           responseBodyBuffer,
		}
		c.Writer = wrappedWriter

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime).Milliseconds()

		// Extract IP address
		ipAddress := getClientIP(c)

		// Extract App ID from header
		var appID *string
		if appIDHeader := c.GetHeader("X-App-Id"); appIDHeader != "" {
			appID = &appIDHeader
		}

		// Extract User Agent
		var userAgent *string
		if ua := c.GetHeader("User-Agent"); ua != "" {
			userAgent = &ua
		}

		// Get query parameters
		var queryParams *string
		if c.Request.URL.RawQuery != "" {
			queryParams = &c.Request.URL.RawQuery
		}

		// Get response body
		responseBody := responseBodyBuffer.String()
		var responseBodyPtr *string
		if responseBody != "" {
			responseBodyPtr = &responseBody
		}

		// Get request body pointer
		var requestBodyPtr *string
		if requestBody != "" {
			requestBodyPtr = &requestBody
		}

		// Skip logging if appID is not present
		if appID == nil {
			return
		}

		// Create activity log entry
		activityLog := entities.RequestActivityLog{
			AppID:        *appID,
			IPAddress:    ipAddress,
			Method:       c.Request.Method,
			URL:          c.Request.URL.Path,
			QueryParams:  queryParams,
			RequestBody:  requestBodyPtr,
			ResponseBody: responseBodyPtr,
			StatusCode:   c.Writer.Status(),
			UserAgent:    userAgent,
			Timestamp:    startTime,
			Duration:     duration,
		}

		// Save to MongoDB asynchronously to avoid blocking the response
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			repo := repository.RequestActivityLogRepo()
			_, err := repo.CreateOne(ctx, activityLog)
			if err != nil {
				logger.Error("failed to save activity log", logger.LoggerOptions{
					Key:  "error",
					Data: err,
				}, logger.LoggerOptions{
					Key:  "activityLog",
					Data: activityLog,
				})
			}
		}()
	}
}

// getClientIP extracts the real client IP address from various headers
func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header (most common)
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" && ip != "unknown" {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" && xri != "unknown" {
		return xri
	}

	// Check X-Client-IP header
	if xci := c.GetHeader("X-Client-IP"); xci != "" && xci != "unknown" {
		return xci
	}

	// Check CF-Connecting-IP header (Cloudflare)
	if cfip := c.GetHeader("CF-Connecting-IP"); cfip != "" && cfip != "unknown" {
		return cfip
	}

	// Fallback to RemoteAddr
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}
