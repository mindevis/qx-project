package api

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/agenthub"
	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/blob"
	"github.com/qxproject/qx/services/qxapi/internal/cosmetics"
	"github.com/qxproject/qx/services/qxapi/internal/crypto"
	"github.com/qxproject/qx/services/qxapi/internal/deploy"
	"github.com/qxproject/qx/services/qxapi/internal/launcher"
	"github.com/qxproject/qx/services/qxapi/internal/mods"
	"github.com/qxproject/qx/services/qxapi/internal/mojang"
	"github.com/qxproject/qx/services/qxapi/internal/servers"
)

type DeploySettings struct {
	PublicAPIURL    string
	AgentBinaryPath string
	DeployExecutor  servers.DeployExecutor // optional; tests inject a capturer
}

type MojangSettings struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	WebBaseURL   string
	JWTSecret    string
}

type ModsSettings struct {
	CurseForgeAPIKey  string
	ModrinthUserAgent string
}

type CosmeticsSettings struct {
	DataDir             string
	PublicAPIURL        string
	SkinServerPublicURL string
}

type LauncherSettings struct {
	Version     string
	DownloadURL string
	Blobs       blob.Store
}

func NewRouter(db *gorm.DB, authSvc *auth.Service, corsOrigin, sshMasterKey string, deployCfg DeploySettings, mojangCfg MojangSettings, modsCfg ModsSettings, cosmeticsCfg CosmeticsSettings, launcherCfg LauncherSettings) *gin.Engine {
	r := gin.New()
	r.Use(RecoveryLogger(), RequestLogger(), CORSMiddleware(corsOrigin))

	tokens := authSvc.Tokens()
	enc, err := crypto.NewEncryptor(sshMasterKey)
	if err != nil {
		slog.Error("ssh encryptor init failed", "error", err)
		enc, _ = crypto.NewEncryptor(devSSHMasterKey())
	}
	launcherSvc := launcher.NewService(db, tokens, corsOrigin)
	launcherSvc.SetRelease(launcherCfg.Version, launcherCfg.DownloadURL)
	launcherSvc.SetBlobs(launcherCfg.Blobs)
	mojangSvc := mojang.NewService(db, enc, mojang.Config{
		ClientID:     mojangCfg.ClientID,
		ClientSecret: mojangCfg.ClientSecret,
		RedirectURI:  mojangCfg.RedirectURI,
		WebBaseURL:   mojangCfg.WebBaseURL,
	}, mojangCfg.JWTSecret)
	launcherSvc.SetMojang(mojangSvc)
	cosmeticsSvc := cosmetics.NewService(db, cosmetics.Config{
		DataDir:             cosmeticsCfg.DataDir,
		PublicAPIURL:        cosmeticsCfg.PublicAPIURL,
		SkinServerPublicURL: cosmeticsCfg.SkinServerPublicURL,
	})
	launcherSvc.SetCosmetics(cosmeticsSvc)

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
	instancesH := &InstancesHandler{Service: launcherSvc, ServerService: serversSvc}
	profilesH := &ProfilesHandler{Service: launcherSvc}
	launchH := &LaunchRequestsHandler{Service: launcherSvc, Tokens: tokens}
	modInstallH := &ModInstallRequestsHandler{Service: launcherSvc, Tokens: tokens}
	prepareH := &PrepareRequestsHandler{Service: launcherSvc}
	instanceFileH := &InstanceFileRequestsHandler{Service: launcherSvc}
	modUninstallH := &ModUninstallRequestsHandler{Service: launcherSvc}
	resourceUploadH := &ResourceUploadRequestsHandler{Service: launcherSvc}
	resourceExportH := &ResourceExportRequestsHandler{Service: launcherSvc}
	updateH := &UpdateRequestsHandler{Service: launcherSvc}
	releaseH := &ReleaseHandler{Service: launcherSvc}
	mcVersionsH := &McVersionsHandler{}
	serversH := &ServersHandler{Service: serversSvc}
	gameServersH := &GameServersHandler{Service: serversSvc}
	gameServerNetworksH := &GameServerNetworksHandler{Service: serversSvc}
	monitoringH := &MonitoringHandler{Service: serversSvc, LauncherService: launcherSvc}
	consoleH := &ServerConsoleHandler{Servers: serversSvc, Tokens: tokens}
	mojangH := &MojangHandler{Service: mojangSvc}
	cosmeticsH := &CosmeticsHandler{Service: cosmeticsSvc}
	sessionH := &SessionHandler{Service: cosmeticsSvc}

	// QX Skin Server — Yggdrasil-compatible session/profile (Ely.by-style, no client mods).
	r.GET("/", sessionH.Meta)
	r.GET("/sessionserver/session/minecraft/profile/:uuid", sessionH.Profile)

	modsSvc := mods.NewService(mods.Config{
		CurseForgeAPIKey:  modsCfg.CurseForgeAPIKey,
		ModrinthUserAgent: modsCfg.ModrinthUserAgent,
	})
	modsH := &ModsHandler{Service: modsSvc}
	agentWS := &AgentWSHandler{Hub: hub, Tokens: tokens, Servers: serversSvc}
	health := &HealthHandler{DB: db}

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", health.Liveness)
		v1.GET("/health/ready", health.Readiness)
		v1.GET("/cosmetics/skins/:userId", cosmeticsH.ServeSkin)
		v1.GET("/cosmetics/capes/:userId", cosmeticsH.ServeCape)
		v1.GET("/cosmetics/skin-catalog", cosmeticsH.ListSkinCatalog)

		v1.POST("/auth/register", authH.Register)
		v1.POST("/auth/login", authH.Login)
		v1.POST("/auth/refresh", authH.Refresh)
		v1.GET("/mojang/oauth/callback", mojangH.OAuthCallback)
		v1.POST("/auth/logout", AuthMiddleware(tokens), authH.Logout)

		// WebSocket clients pass access_token as a query param (no Authorization header).
		v1.GET("/servers/:id/console", consoleH.Connect)

		v1.POST("/launcher/devices/register", devicesH.Register)
		v1.GET("/launcher/devices/:id/status", devicesH.Status)
		v1.GET("/launcher/mc-versions", mcVersionsH.List)
		v1.GET("/launcher/release", releaseH.Get)
		v1.GET("/monitoring/servers", monitoringH.List)

		link := v1.Group("")
		link.Use(AuthMiddleware(tokens))
		{
			link.POST("/launcher/devices/link", devicesH.Link)
		}

		authed := v1.Group("")
		authed.Use(AuthMiddleware(tokens))
		{
			authed.GET("/users/me", usersH.Me)
			authed.GET("/users/me/launcher-device", devicesH.UserLinkedDevice)
			authed.POST("/launcher/update-requests", updateH.Create)
			authed.PATCH("/users/me/password", usersH.ChangePassword)
			authed.PATCH("/users/me/email", usersH.ChangeEmail)
			authed.GET("/users/me/mojang", mojangH.Status)
			authed.POST("/users/me/mojang/oauth/start", mojangH.StartOAuth)
			authed.DELETE("/users/me/mojang", mojangH.Unlink)

			authed.GET("/users/me/cosmetics", cosmeticsH.GetMine)
			authed.PUT("/users/me/cosmetics", cosmeticsH.Equip)
			authed.POST("/users/me/cosmetics/skin", cosmeticsH.UploadSkin)
			authed.POST("/users/me/cosmetics/skin/apply", cosmeticsH.ApplySkin)
			authed.DELETE("/users/me/cosmetics/skin", cosmeticsH.DeleteSkin)
			authed.POST("/users/me/cosmetics/cape", cosmeticsH.UploadCape)
			authed.DELETE("/users/me/cosmetics/cape", cosmeticsH.DeleteCape)

			authed.GET("/servers", serversH.List)
			authed.POST("/servers", serversH.Create)
			authed.GET("/servers/:id", serversH.Get)
			authed.DELETE("/servers/:id", serversH.Delete)
			authed.POST("/servers/:id/deploy", serversH.Deploy)
			authed.POST("/servers/:id/start", serversH.Start)
			authed.POST("/servers/:id/stop", serversH.Stop)
			authed.POST("/servers/:id/restart", serversH.Restart)

			authed.GET("/servers/:id/game-servers", gameServersH.List)
			authed.POST("/servers/:id/game-servers", gameServersH.Create)
			authed.POST("/servers/:id/game-servers/:gameServerId/clone", gameServersH.Clone)
			authed.GET("/servers/:id/game-servers/:gameServerId", gameServersH.Get)
			authed.PATCH("/servers/:id/game-servers/:gameServerId", gameServersH.Update)
			authed.GET("/servers/:id/game-servers/:gameServerId/properties", gameServersH.GetProperties)
			authed.PATCH("/servers/:id/game-servers/:gameServerId/properties", gameServersH.PatchProperties)
			authed.GET("/servers/:id/game-servers/:gameServerId/resources", gameServersH.ListContentResources)
			authed.PATCH("/servers/:id/game-servers/:gameServerId/resources", gameServersH.PatchContentResource)
			authed.GET("/servers/:id/game-servers/:gameServerId/mods", gameServersH.ListMods)
			authed.GET("/servers/:id/game-servers/:gameServerId/client-mods", gameServersH.ListClientMods)
			authed.POST("/servers/:id/game-servers/:gameServerId/mods/sync", gameServersH.SyncMod)
			authed.DELETE("/servers/:id/game-servers/:gameServerId/mods", gameServersH.DeleteMod)
			authed.POST("/servers/:id/game-servers/:gameServerId/mods/upload", gameServersH.UploadMod)
			authed.GET("/servers/:id/game-servers/:gameServerId/resourcepacks", gameServersH.ListResourcepacks)
			authed.GET("/servers/:id/game-servers/:gameServerId/client-resourcepacks", gameServersH.ListClientResourcepacks)
			authed.POST("/servers/:id/game-servers/:gameServerId/resourcepacks/sync", gameServersH.SyncResourcepack)
			authed.DELETE("/servers/:id/game-servers/:gameServerId/resourcepacks", gameServersH.DeleteResourcepack)
			authed.GET("/servers/:id/game-servers/:gameServerId/shaders", gameServersH.ListShaders)
			authed.GET("/servers/:id/game-servers/:gameServerId/client-shaders", gameServersH.ListClientShaders)
			authed.POST("/servers/:id/game-servers/:gameServerId/shaders/sync", gameServersH.SyncShader)
			authed.DELETE("/servers/:id/game-servers/:gameServerId/shaders", gameServersH.DeleteShader)
			authed.GET("/servers/:id/game-servers/:gameServerId/plugins", gameServersH.ListPlugins)
			authed.POST("/servers/:id/game-servers/:gameServerId/plugins/sync", gameServersH.SyncPlugin)
			authed.DELETE("/servers/:id/game-servers/:gameServerId/plugins", gameServersH.DeletePlugin)
			authed.GET("/servers/:id/game-servers/:gameServerId/datapacks", gameServersH.ListDatapacks)
			authed.POST("/servers/:id/game-servers/:gameServerId/datapacks/sync", gameServersH.SyncDatapack)
			authed.DELETE("/servers/:id/game-servers/:gameServerId/datapacks", gameServersH.DeleteDatapack)
			authed.GET("/servers/:id/game-servers/:gameServerId/files", gameServersH.ListFiles)
			authed.DELETE("/servers/:id/game-servers/:gameServerId/files", gameServersH.DeleteFile)
			authed.GET("/servers/:id/game-servers/:gameServerId/files/content", gameServersH.ReadFile)
			authed.PUT("/servers/:id/game-servers/:gameServerId/files/content", gameServersH.WriteFile)
			authed.POST("/servers/:id/game-servers/:gameServerId/version", gameServersH.ChangeVersion)
			authed.POST("/servers/:id/game-servers/:gameServerId/reinstall", gameServersH.Reinstall)
			authed.POST("/servers/:id/game-servers/:gameServerId/start", gameServersH.Start)
			authed.POST("/servers/:id/game-servers/:gameServerId/stop", gameServersH.Stop)
			authed.POST("/servers/:id/game-servers/:gameServerId/restart", gameServersH.Restart)
			authed.DELETE("/servers/:id/game-servers/:gameServerId", gameServersH.Delete)

			authed.GET("/servers/:id/networks", gameServerNetworksH.List)
			authed.POST("/servers/:id/networks", gameServerNetworksH.Create)
			authed.PATCH("/servers/:id/networks/:networkId", gameServerNetworksH.Update)
			authed.POST("/servers/:id/networks/:networkId/apply", gameServerNetworksH.Apply)
			authed.DELETE("/servers/:id/networks/:networkId", gameServerNetworksH.Delete)

			authed.POST("/monitoring/servers/:id/like", monitoringH.Like)
			authed.POST("/monitoring/servers/:id/rate", monitoringH.Rate)
			authed.GET("/monitoring/bindings", monitoringH.ListBindings)
			authed.GET("/monitoring/bindable-servers", monitoringH.ListBindable)
			authed.POST("/monitoring/servers/:id/connect-instance", monitoringH.EnsureConnectInstance)
			authed.PUT("/monitoring/servers/:id/binding", monitoringH.SetBinding)
			authed.DELETE("/monitoring/servers/:id/binding", monitoringH.ClearBinding)
			authed.GET("/monitoring/servers/:id/connect-mod-status", monitoringH.GetConnectModStatus)
			authed.PUT("/monitoring/servers/:id/client-mod-prefs", monitoringH.SetClientModPrefs)
			authed.POST("/monitoring/servers/:id/prepare-connect-mods", monitoringH.PrepareConnectMods)

			authed.GET("/mods/search", modsH.Search)
			authed.GET("/mods/browse", modsH.Browse)
			authed.GET("/mods/:source/:projectId", modsH.GetProject)
			authed.GET("/mods/:source/:projectId/versions", modsH.ListVersions)
			authed.GET("/mods/:source/:projectId/versions/:versionId", modsH.GetVersion)
		}

		launcherOwner := v1.Group("")
		launcherOwner.Use(LauncherOwnerMiddleware(tokens))
		{
			launcherOwner.GET("/instances", instancesH.List)
			launcherOwner.POST("/instances", instancesH.Create)
			launcherOwner.POST("/instances/:id/clone", instancesH.Clone)
			launcherOwner.GET("/instances/:id", instancesH.Get)
			launcherOwner.PATCH("/instances/:id", instancesH.Update)
			launcherOwner.GET("/instances/:id/resources", instancesH.ListResources)
			launcherOwner.PATCH("/instances/:id/resources", instancesH.PatchResource)
			launcherOwner.DELETE("/instances/:id/resources", instancesH.DeleteResource)
			launcherOwner.POST("/instances/:id/resources/upload", instancesH.UploadResource)
			launcherOwner.POST("/instances/:id/resources/sync-to-game-server", instancesH.SyncUploadedResource)
			launcherOwner.GET("/instances/:id/files", instancesH.ListFiles)
			launcherOwner.GET("/instances/:id/files/content", instancesH.ReadFile)
			launcherOwner.PUT("/instances/:id/files/content", instancesH.WriteFile)
			launcherOwner.GET("/instances/:id/manifest", instancesH.Manifest)
			launcherOwner.DELETE("/instances/:id", instancesH.Delete)

			launcherOwner.GET("/launcher/profiles", profilesH.List)
			launcherOwner.POST("/launcher/profiles", profilesH.Create)
			launcherOwner.DELETE("/launcher/profiles/:id", profilesH.Delete)

			launcherOwner.POST("/launcher/launch-requests", launchH.Create)
			launcherOwner.GET("/launcher/launch-requests/:id", launchH.Get)
			launcherOwner.POST("/launcher/mod-install-requests", modInstallH.Create)
			launcherOwner.GET("/launcher/mod-install-requests/:id", modInstallH.Get)
			launcherOwner.GET("/launcher/prepare-requests/:id", prepareH.Get)
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
			deviceAuth.GET("/launcher/mod-install-requests/pending", modInstallH.Pending)
			deviceAuth.PATCH("/launcher/mod-install-requests/:id", modInstallH.Update)
			deviceAuth.GET("/launcher/prepare-requests/pending", prepareH.Pending)
			deviceAuth.PATCH("/launcher/prepare-requests/:id", prepareH.Update)
			deviceAuth.GET("/launcher/instance-file-requests/pending", instanceFileH.Pending)
			deviceAuth.PATCH("/launcher/instance-file-requests/:id", instanceFileH.Update)
			deviceAuth.GET("/launcher/mod-uninstall-requests/pending", modUninstallH.Pending)
			deviceAuth.PATCH("/launcher/mod-uninstall-requests/:id", modUninstallH.Update)
			deviceAuth.GET("/launcher/resource-upload-requests/pending", resourceUploadH.Pending)
			deviceAuth.GET("/launcher/resource-upload-requests/:id/content", resourceUploadH.Content)
			deviceAuth.PATCH("/launcher/resource-upload-requests/:id", resourceUploadH.Update)
			deviceAuth.GET("/launcher/resource-export-requests/pending", resourceExportH.Pending)
			deviceAuth.PATCH("/launcher/resource-export-requests/:id", resourceExportH.Update)
			deviceAuth.GET("/launcher/update-requests/pending", updateH.Pending)
			deviceAuth.PATCH("/launcher/update-requests/:id", updateH.Complete)
		}
	}

	r.GET("/agent/v1/connect", agentWS.Connect)

	mountSwagger(r)

	return r
}

func devSSHMasterKey() string {
	return "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
}
