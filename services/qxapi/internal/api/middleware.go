package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
)

const UserIDKey = "user_id"

func AuthMiddleware(tokens *auth.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			JSONUnauthorized(c)
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := tokens.Parse(token)
		if err != nil || claims.Kind != auth.TokenAccess {
			JSONUnauthorized(c)
			return
		}
		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}

func CORSMiddleware(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Device-Token")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
