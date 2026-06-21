package api

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/agenthub"
	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/crypto"
	"github.com/qxproject/qx/services/qxapi/internal/deploy"
	"github.com/qxproject/qx/services/qxapi/internal/launcher"
	"github.com/qxproject/qx/services/qxapi/internal/servers"
)

type DeploySettings struct {
	PublicAPIURL    string
	AgentBinaryPath string
	DeployExecutor  servers.DeployExecutor // optional; tests inject a capturer
}

func NewRouter(db *gorm.DB, authSvc *auth.Service, corsOrigin, sshMasterKey string, deployCfg DeploySettings) *gin.Engine {
	r := gin.New()
	r.Use(RecoveryLogger(), RequestLogger(), CORSMiddleware(corsOrigin))

	tokens := authSvc.Tokens()
	launcherSvc := launcher.NewService(db, tokens, corsOrigin)

	enc, err := crypto.NewEncryptor(sshMasterKey)
	if err != nil {
		slog.Error("ssh encryptor init failed", "error", err)
		enc, _ = crypto.NewEncryptor(devSSHMasterKey())
	}
	hub := agenthub.New(nil)
	deployer := deployCfg.DeployExecutor
	if deployer == nil {
		deployer = deploy.NewSSH(deploy.SSHConfig{
			Encryptor:  enc,
			APIBaseURL: deployCfg.PublicAPIURL,
			BinaryPath: deployCfg.AgentBinaryPath,
		})
	}
	serversSvc := servers.NewService(db, tokens, enc, hub, deployer)
	hub.SetOnEvent(serversSvc.OnAgentEvent)

	authH := &AuthHandler{Service: authSvc}
	usersH := &UsersHandler{Service: authSvc}
	devicesH := &DevicesHandler{Service: launcherSvc}
	instancesH := &InstancesHandler{Service: launcherSvc}
	profilesH := &ProfilesHandler{Service: launcherSvc}
	launchH := &LaunchRequestsHandler{Service: launcherSvc, Tokens: tokens}
	serversH := &ServersHandler{Service: serversSvc}
	consoleH := &ServerConsoleHandler{Servers: serversSvc, Tokens: tokens}
	agentWS := &AgentWSHandler{Hub: hub, Tokens: tokens, Servers: serversSvc}
	health := &HealthHandler{DB: db}

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", health.Liveness)
		v1.GET("/health/ready", health.Readiness)

		v1.POST("/auth/register", authH.Register)
		v1.POST("/auth/login", authH.Login)
		v1.POST("/auth/refresh", authH.Refresh)
		v1.POST("/auth/guest", authH.Guest)
		v1.POST("/auth/logout", AuthMiddleware(tokens), authH.Logout)

		// WebSocket clients pass access_token as a query param (no Authorization header).
		v1.GET("/servers/:id/console", consoleH.Connect)

		v1.POST("/launcher/devices/register", devicesH.Register)
		v1.GET("/launcher/devices/:id/status", devicesH.Status)

		link := v1.Group("")
		link.Use(OptionalAuthMiddleware(tokens))
		{
			link.POST("/launcher/devices/link", devicesH.Link)
		}

		authed := v1.Group("")
		authed.Use(AuthMiddleware(tokens))
		{
			authed.GET("/users/me", usersH.Me)
			authed.GET("/users/me/launcher-device", devicesH.UserLinkedDevice)
			authed.PATCH("/users/me/password", usersH.ChangePassword)
			authed.PATCH("/users/me/email", usersH.ChangeEmail)

			authed.GET("/servers", serversH.List)
			authed.POST("/servers", serversH.Create)
			authed.GET("/servers/:id", serversH.Get)
			authed.DELETE("/servers/:id", serversH.Delete)
			authed.POST("/servers/:id/deploy", serversH.Deploy)
			authed.POST("/servers/:id/start", serversH.Start)
			authed.POST("/servers/:id/stop", serversH.Stop)
			authed.POST("/servers/:id/restart", serversH.Restart)
		}

		launcherOwner := v1.Group("")
		launcherOwner.Use(LauncherOwnerMiddleware(tokens))
		{
			launcherOwner.GET("/instances", instancesH.List)
			launcherOwner.POST("/instances", instancesH.Create)
			launcherOwner.GET("/instances/:id", instancesH.Get)
			launcherOwner.GET("/instances/:id/manifest", instancesH.Manifest)
			launcherOwner.DELETE("/instances/:id", instancesH.Delete)

			launcherOwner.GET("/launcher/profiles", profilesH.List)
			launcherOwner.POST("/launcher/profiles", profilesH.Create)
			launcherOwner.DELETE("/launcher/profiles/:id", profilesH.Delete)

			launcherOwner.POST("/launcher/launch-requests", launchH.Create)
			launcherOwner.GET("/launcher/launch-requests/:id", launchH.Get)
		}

		deviceLauncher := v1.Group("")
		deviceLauncher.Use(DeviceOrLauncherOwnerMiddleware(tokens))
		{
			deviceLauncher.POST("/launcher/devices/unlink", devicesH.Unlink)
		}

		deviceAuth := v1.Group("")
		deviceAuth.Use(DeviceAuthMiddleware(tokens))
		{
			deviceAuth.GET("/launcher/devices/me", devicesH.Me)
			deviceAuth.GET("/launcher/devices/me/instances", devicesH.MeInstances)
			deviceAuth.GET("/launcher/launch-requests/pending", launchH.Pending)
			deviceAuth.PATCH("/launcher/launch-requests/:id", launchH.Update)
		}
	}

	r.GET("/agent/v1/connect", agentWS.Connect)

	mountSwagger(r)

	return r
}

func devSSHMasterKey() string {
	return "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
}
