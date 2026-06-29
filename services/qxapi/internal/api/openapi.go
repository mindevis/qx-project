// Package api OpenAPI metadata and route docs for swag.
//
// Regenerate: make swagger (from repo root).
package api

// @title           QX API
// @version         1.0
// @description     QXSystem REST API. Base path `/api/v1`. Agent WSS: `GET /agent/v1/connect`.
// @description     Interactive docs: `GET /swagger/index.html`

// @contact.name   QXSystem
// @contact.url    https://github.com/mindevis/qx-project

// @license.name   MIT

// @host      localhost:3000
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT access token. Format: `Bearer {access_token}`

// @securityDefinitions.apikey DeviceAuth
// @in header
// @name Authorization
// @description Device JWT from POST /launcher/devices/link. Format: `Bearer {device_token}`

// --- Health ---

// swaggerHealthLiveness documents GET /health.
//
// @Summary      Liveness probe
// @Description  Returns ok if the process is running.
// @Tags         health
// @Produce      json
// @Success      200  {object}  swaggerHealthStatus
// @Router       /health [get]
func swaggerHealthLiveness() {}

// swaggerHealthReadiness documents GET /health/ready.
//
// @Summary      Readiness probe
// @Description  Checks database connectivity.
// @Tags         health
// @Produce      json
// @Success      200  {object}  swaggerHealthStatus
// @Failure      503  {object}  swaggerHealthStatus
// @Router       /health/ready [get]
func swaggerHealthReadiness() {}

// --- Auth ---

// swaggerAuthRegister documents POST /auth/register.
//
// @Summary  Register user
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body  body      swaggerRegisterRequest  true  "Registration"
// @Success  201   {object}  swaggerTokenResponse
// @Failure  400   {object}  swaggerErrorResponse
// @Failure  409   {object}  swaggerErrorResponse
// @Router   /auth/register [post]
func swaggerAuthRegister() {}

// swaggerAuthLogin documents POST /auth/login.
//
// @Summary  Login
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body  body      swaggerLoginRequest  true  "Credentials"
// @Success  200   {object}  swaggerTokenResponse
// @Failure  401   {object}  swaggerErrorResponse
// @Router   /auth/login [post]
func swaggerAuthLogin() {}

// swaggerAuthRefresh documents POST /auth/refresh.
//
// @Summary  Refresh tokens
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body  body      swaggerRefreshRequest  true  "Refresh token"
// @Success  200   {object}  swaggerTokenResponse
// @Failure  401   {object}  swaggerErrorResponse
// @Router   /auth/refresh [post]
func swaggerAuthRefresh() {}

// swaggerAuthGuest documents POST /auth/guest.
//
// @Summary  Issue guest token
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body  body      swaggerGuestRequest  true  "Device id"
// @Success  200   {object}  swaggerGuestTokenResponse
// @Router   /auth/guest [post]
func swaggerAuthGuest() {}

// swaggerAuthLogout documents POST /auth/logout.
//
// @Summary  Logout
// @Tags     auth
// @Security BearerAuth
// @Success  204
// @Failure  401  {object}  swaggerErrorResponse
// @Router   /auth/logout [post]
func swaggerAuthLogout() {}

// --- Users ---

// swaggerUsersMe documents GET /users/me.
//
// @Summary  Current user profile
// @Tags     users
// @Security BearerAuth
// @Produce  json
// @Success  200  {object}  swaggerUserProfile
// @Failure  401  {object}  swaggerErrorResponse
// @Router   /users/me [get]
func swaggerUsersMe() {}

// swaggerUsersChangePassword documents PATCH /users/me/password.
//
// @Summary  Change password
// @Tags     users
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Success  204
// @Failure  401  {object}  swaggerErrorResponse
// @Router   /users/me/password [patch]
func swaggerUsersChangePassword() {}

// swaggerUsersChangeEmail documents PATCH /users/me/email.
//
// @Summary  Change email
// @Tags     users
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Success  200  {object}  swaggerUserProfile
// @Failure  401  {object}  swaggerErrorResponse
// @Router   /users/me/email [patch]
func swaggerUsersChangeEmail() {}

// swaggerUsersLauncherDevice documents GET /users/me/launcher-device.
//
// @Summary  Linked launcher device status
// @Tags     users
// @Security BearerAuth
// @Produce  json
// @Success  200  {object}  map[string]interface{}
// @Failure  401  {object}  swaggerErrorResponse
// @Router   /users/me/launcher-device [get]
func swaggerUsersLauncherDevice() {}

// --- Launcher devices ---

// swaggerDevicesRegister documents POST /launcher/devices/register.
//
// @Summary  Register launcher device (pending link)
// @Tags     launcher-devices
// @Accept   json
// @Produce  json
// @Success  201  {object}  map[string]interface{}
// @Router   /launcher/devices/register [post]
func swaggerDevicesRegister() {}

// swaggerDevicesLink documents POST /launcher/devices/link.
//
// @Summary  Confirm device link (guest or user)
// @Tags     launcher-devices
// @Accept   json
// @Produce  json
// @Success  200  {object}  map[string]interface{}
// @Router   /launcher/devices/link [post]
func swaggerDevicesLink() {}

// swaggerDevicesStatus documents GET /launcher/devices/{id}/status.
//
// @Summary  Poll device link status
// @Tags     launcher-devices
// @Produce  json
// @Param    id   path      string  true  "Device ID"
// @Success  200  {object}  map[string]interface{}
// @Router   /launcher/devices/{id}/status [get]
func swaggerDevicesStatus() {}

// swaggerDevicesUnlink documents POST /launcher/devices/unlink.
//
// @Summary  Unlink device
// @Tags     launcher-devices
// @Security DeviceAuth
// @Success  204
// @Failure  401  {object}  swaggerErrorResponse
// @Router   /launcher/devices/unlink [post]
func swaggerDevicesUnlink() {}

// swaggerDevicesMe documents GET /launcher/devices/me.
//
// @Summary  Device profile
// @Tags     launcher-devices
// @Security DeviceAuth
// @Produce  json
// @Success  200  {object}  map[string]interface{}
// @Router   /launcher/devices/me [get]
func swaggerDevicesMe() {}

// swaggerDevicesMeInstances documents GET /launcher/devices/me/instances.
//
// @Summary  Instances for linked device
// @Tags     launcher-devices
// @Security DeviceAuth
// @Produce  json
// @Success  200  {object}  map[string]interface{}
// @Router   /launcher/devices/me/instances [get]
func swaggerDevicesMeInstances() {}

// --- Instances ---

// swaggerInstancesList documents GET /instances.
//
// @Summary  List instances
// @Tags     instances
// @Security BearerAuth
// @Produce  json
// @Success  200  {object}  map[string]interface{}
// @Router   /instances [get]
func swaggerInstancesList() {}

// swaggerInstancesCreate documents POST /instances.
//
// @Summary  Create instance
// @Tags     instances
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Success  201  {object}  map[string]interface{}
// @Router   /instances [post]
func swaggerInstancesCreate() {}

// swaggerInstancesGet documents GET /instances/{id}.
//
// @Summary  Get instance
// @Tags     instances
// @Security BearerAuth
// @Produce  json
// @Param    id   path  string  true  "Instance ID"
// @Success  200  {object}  map[string]interface{}
// @Router   /instances/{id} [get]
func swaggerInstancesGet() {}

// swaggerInstancesManifest documents GET /instances/{id}/manifest.
//
// @Summary  Launch manifest
// @Tags     instances
// @Security BearerAuth
// @Produce  json
// @Param    id   path  string  true  "Instance ID"
// @Success  200  {object}  map[string]interface{}
// @Router   /instances/{id}/manifest [get]
func swaggerInstancesManifest() {}

// swaggerInstancesDelete documents DELETE /instances/{id}.
//
// @Summary  Delete instance
// @Tags     instances
// @Security BearerAuth
// @Param    id   path  string  true  "Instance ID"
// @Success  204
// @Router   /instances/{id} [delete]
func swaggerInstancesDelete() {}

// --- Profiles ---

// swaggerProfilesList documents GET /launcher/profiles.
//
// @Summary  List offline profiles
// @Tags     profiles
// @Security BearerAuth
// @Produce  json
// @Success  200  {object}  map[string]interface{}
// @Router   /launcher/profiles [get]
func swaggerProfilesList() {}

// swaggerProfilesCreate documents POST /launcher/profiles.
//
// @Summary  Create offline profile
// @Tags     profiles
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Success  201  {object}  map[string]interface{}
// @Router   /launcher/profiles [post]
func swaggerProfilesCreate() {}

// swaggerProfilesDelete documents DELETE /launcher/profiles/{id}.
//
// @Summary  Delete profile
// @Tags     profiles
// @Security BearerAuth
// @Param    id   path  string  true  "Profile ID"
// @Success  204
// @Router   /launcher/profiles/{id} [delete]
func swaggerProfilesDelete() {}

// --- Launch requests ---

// swaggerLaunchCreate documents POST /launcher/launch-requests.
//
// @Summary  Queue launch (web → QXLauncher)
// @Tags     launch
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Success  201  {object}  map[string]interface{}
// @Router   /launcher/launch-requests [post]
func swaggerLaunchCreate() {}

// swaggerLaunchGet documents GET /launcher/launch-requests/{id}.
//
// @Summary  Launch request status
// @Tags     launch
// @Security BearerAuth
// @Param    id   path  string  true  "Request ID"
// @Success  200  {object}  map[string]interface{}
// @Router   /launcher/launch-requests/{id} [get]
func swaggerLaunchGet() {}

// swaggerLaunchPending documents GET /launcher/launch-requests/pending.
//
// @Summary  Pending launch for device (QXLauncher poll)
// @Tags     launch
// @Security DeviceAuth
// @Produce  json
// @Success  200  {object}  map[string]interface{}
// @Router   /launcher/launch-requests/pending [get]
func swaggerLaunchPending() {}

// swaggerLaunchUpdate documents PATCH /launcher/launch-requests/{id}.
//
// @Summary  Update launch status (QXLauncher)
// @Tags     launch
// @Security DeviceAuth
// @Param    id   path  string  true  "Request ID"
// @Success  200  {object}  map[string]interface{}
// @Router   /launcher/launch-requests/{id} [patch]
func swaggerLaunchUpdate() {}

// --- Servers ---

// swaggerServersList documents GET /servers.
//
// @Summary  List servers
// @Tags     servers
// @Security BearerAuth
// @Produce  json
// @Success  200  {object}  map[string]interface{}
// @Router   /servers [get]
func swaggerServersList() {}

// swaggerServersCreate documents POST /servers.
//
// @Summary  Create Dedicated server
// @Tags     servers
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Success  201  {object}  map[string]interface{}
// @Router   /servers [post]
func swaggerServersCreate() {}

// swaggerServersGet documents GET /servers/{id}.
//
// @Summary  Server detail (agent_online, minecraft_running)
// @Tags     servers
// @Security BearerAuth
// @Param    id   path  string  true  "Server ID"
// @Success  200  {object}  map[string]interface{}
// @Router   /servers/{id} [get]
func swaggerServersGet() {}

// swaggerServersDelete documents DELETE /servers/{id}.
//
// @Summary  Delete server
// @Tags     servers
// @Security BearerAuth
// @Param    id   path  string  true  "Server ID"
// @Success  204
// @Router   /servers/{id} [delete]
func swaggerServersDelete() {}

// swaggerServersDeploy documents POST /servers/{id}/deploy.
//
// @Summary  SSH deploy QXAgent
// @Tags     servers
// @Security BearerAuth
// @Param    id   path  string  true  "Server ID"
// @Success  200  {object}  map[string]interface{}
// @Router   /servers/{id}/deploy [post]
func swaggerServersDeploy() {}

// swaggerServersStart documents POST /servers/{id}/start.
//
// @Summary  Start Minecraft JAR via agent
// @Tags     servers
// @Security BearerAuth
// @Param    id   path  string  true  "Server ID"
// @Success  200  {object}  map[string]interface{}
// @Router   /servers/{id}/start [post]
func swaggerServersStart() {}

// swaggerServersStop documents POST /servers/{id}/stop.
//
// @Summary  Stop Minecraft JAR
// @Tags     servers
// @Security BearerAuth
// @Param    id   path  string  true  "Server ID"
// @Success  200  {object}  map[string]interface{}
// @Router   /servers/{id}/stop [post]
func swaggerServersStop() {}

// swaggerServersRestart documents POST /servers/{id}/restart.
//
// @Summary  Restart Minecraft JAR
// @Tags     servers
// @Security BearerAuth
// @Param    id   path  string  true  "Server ID"
// @Success  200  {object}  map[string]interface{}
// @Router   /servers/{id}/restart [post]
func swaggerServersRestart() {}

// swaggerServerConsole documents GET /servers/{id}/console.
//
// @Summary  Live console WebSocket
// @Description  Upgrade to WebSocket. Pass `access_token` query param or Authorization header.
// @Tags     servers
// @Security BearerAuth
// @Param    id            path   string  true   "Server ID"
// @Param    access_token  query  string  false  "User JWT (alternative to Authorization header)"
// @Success  101
// @Router   /servers/{id}/console [get]
func swaggerServerConsole() {}
