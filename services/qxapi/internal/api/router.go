package api

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
)

func NewRouter(db *gorm.DB, authSvc *auth.Service, corsOrigin string) *gin.Engine {
	r := gin.New()
	r.Use(RecoveryLogger(), RequestLogger(), CORSMiddleware(corsOrigin))

	authH := &AuthHandler{Service: authSvc}
	usersH := &UsersHandler{Service: authSvc}
	health := &HealthHandler{DB: db}

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", health.Liveness)
		v1.GET("/health/ready", health.Readiness)

		v1.POST("/auth/register", authH.Register)
		v1.POST("/auth/login", authH.Login)
		v1.POST("/auth/refresh", authH.Refresh)
		v1.POST("/auth/guest", authH.Guest)
		v1.POST("/auth/logout", AuthMiddleware(authSvc.Tokens()), authH.Logout)

		authed := v1.Group("")
		authed.Use(AuthMiddleware(authSvc.Tokens()))
		{
			authed.GET("/users/me", usersH.Me)
			authed.PATCH("/users/me/password", usersH.ChangePassword)
			authed.PATCH("/users/me/email", usersH.ChangeEmail)
		}
	}

	return r
}
