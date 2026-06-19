package api

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/launcher"
)

const (
	GuestSessionIDKey = "guest_session_id"
	IsGuestKey        = "is_guest"
)

func OptionalAuthMiddleware(tokens *auth.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.Next()
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := tokens.Parse(token)
		if err != nil {
			JSONUnauthorized(c)
			return
		}
		switch claims.Kind {
		case auth.TokenAccess:
			c.Set(UserIDKey, claims.UserID)
		case auth.TokenGuest:
			c.Set(GuestSessionIDKey, claims.UserID)
			c.Set(IsGuestKey, true)
		default:
			JSONUnauthorized(c)
			return
		}
		c.Next()
	}
}

func LauncherOwnerMiddleware(tokens *auth.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			JSONUnauthorized(c)
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := tokens.Parse(token)
		if err != nil {
			JSONUnauthorized(c)
			return
		}
		switch claims.Kind {
		case auth.TokenAccess:
			c.Set(UserIDKey, claims.UserID)
		case auth.TokenGuest:
			c.Set(GuestSessionIDKey, claims.UserID)
			c.Set(IsGuestKey, true)
		default:
			JSONUnauthorized(c)
			return
		}
		c.Next()
	}
}

func ownerFromContext(c *gin.Context) (launcher.Owner, bool) {
	if guestID, ok := c.Get(GuestSessionIDKey); ok {
		return launcher.Owner{GuestSessionID: guestID.(string), IsGuest: true}, true
	}
	if userID, ok := c.Get(UserIDKey); ok {
		return launcher.Owner{UserID: userID.(string), IsGuest: false}, true
	}
	return launcher.Owner{}, false
}
