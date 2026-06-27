package api

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
)

const DeviceIDKey = "device_id"

func DeviceAuthMiddleware(tokens *auth.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := deviceTokenFromRequest(c)
		if token == "" {
			JSONUnauthorized(c)
			return
		}
		claims, err := tokens.Parse(token)
		if err != nil || claims.Kind != auth.TokenDevice || claims.DeviceID == "" {
			JSONUnauthorized(c)
			return
		}
		c.Set(DeviceIDKey, claims.DeviceID)
		c.Next()
	}
}

func deviceTokenFromRequest(c *gin.Context) string {
	if header := strings.TrimSpace(c.GetHeader("X-Device-Token")); header != "" {
		return strings.TrimPrefix(header, "Bearer ")
	}
	header := c.GetHeader("Authorization")
	if header == "" || !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(header, "Bearer ")
}

func deviceIDFromContext(c *gin.Context) (string, bool) {
	v, ok := c.Get(DeviceIDKey)
	if !ok {
		return "", false
	}
	id, ok := v.(string)
	return id, ok && id != ""
}

func DeviceOrLauncherOwnerMiddleware(tokens *auth.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := deviceTokenFromRequest(c)
		if token == "" {
			LauncherOwnerMiddleware(tokens)(c)
			return
		}
		claims, err := tokens.Parse(token)
		if err != nil {
			JSONUnauthorized(c)
			return
		}
		switch claims.Kind {
		case auth.TokenDevice:
			if claims.DeviceID == "" {
				JSONUnauthorized(c)
				return
			}
			c.Set(DeviceIDKey, claims.DeviceID)
			c.Next()
		case auth.TokenAccess:
			c.Set(UserIDKey, claims.UserID)
			c.Next()
		default:
			JSONUnauthorized(c)
		}
	}
}
