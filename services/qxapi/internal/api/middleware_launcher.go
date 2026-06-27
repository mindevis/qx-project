package api

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/launcher"
)

func LauncherOwnerMiddleware(tokens *auth.TokenService) gin.HandlerFunc {
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

func ownerFromContext(c *gin.Context) (launcher.Owner, bool) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		return launcher.Owner{}, false
	}
	return launcher.Owner{UserID: userID.(string)}, true
}
