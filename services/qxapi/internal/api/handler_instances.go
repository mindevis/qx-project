package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/launcher"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/servers"
)

type InstancesHandler struct {
	Service       *launcher.Service
	ServerService *servers.Service
}

type instanceResponse struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	MCVersion             string   `json:"mc_version"`
	Loader                string   `json:"loader"`
	LoaderVersion         *string  `json:"loader_version,omitempty"`
	MaxMemoryMB           *int     `json:"max_memory_mb,omitempty"`
	MinMemoryMB           *int     `json:"min_memory_mb,omitempty"`
	ExtraJVMArgs          []string `json:"extra_jvm_args,omitempty"`
	WindowWidth           *int     `json:"window_width,omitempty"`
	WindowHeight          *int     `json:"window_height,omitempty"`
	PrepareRequestID      *string  `json:"prepare_request_id,omitempty"`
	ManagedByGameServerID *string  `json:"managed_by_game_server_id,omitempty"`
	ContentLocked         bool     `json:"content_locked"`
	CreatedAt             string   `json:"created_at"`
	UpdatedAt             string   `json:"updated_at"`
}

func instanceFromModel(inst models.LauncherInstance) instanceResponse {
	return instanceResponse{
		ID:                    inst.ID,
		Name:                  inst.Name,
		MCVersion:             inst.MCVersion,
		Loader:                inst.Loader,
		LoaderVersion:         inst.LoaderVersion,
		MaxMemoryMB:           inst.MaxMemoryMB,
		MinMemoryMB:           inst.MinMemoryMB,
		ExtraJVMArgs:          []string(inst.ExtraJVMArgs),
		WindowWidth:           inst.WindowWidth,
		WindowHeight:          inst.WindowHeight,
		ManagedByGameServerID: inst.ManagedByGameServerID,
		ContentLocked:         inst.ManagedByGameServerID != nil && strings.TrimSpace(*inst.ManagedByGameServerID) != "",
		CreatedAt:             inst.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:             inst.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func instanceCreateResponse(inst models.LauncherInstance, prepareRequestID *string) instanceResponse {
	resp := instanceFromModel(inst)
	resp.PrepareRequestID = prepareRequestID
	return resp
}

type createInstanceRequest struct {
	Name          string `json:"name" binding:"required"`
	MCVersion     string `json:"mc_version" binding:"required"`
	Loader        string `json:"loader"`
	LoaderVersion string `json:"loader_version"`
}

type updateInstanceRequest struct {
	Name         *string   `json:"name"`
	MaxMemoryMB  *int      `json:"max_memory_mb"`
	MinMemoryMB  *int      `json:"min_memory_mb"`
	ExtraJVMArgs *[]string `json:"extra_jvm_args"`
	WindowWidth  *int      `json:"window_width"`
	WindowHeight *int      `json:"window_height"`
}

func hasInstanceUpdateFields(req updateInstanceRequest) bool {
	return req.Name != nil || req.MaxMemoryMB != nil || req.MinMemoryMB != nil || req.ExtraJVMArgs != nil ||
		req.WindowWidth != nil || req.WindowHeight != nil
}

func (h *InstancesHandler) List(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	items, err := h.Service.ListInstances(c.Request.Context(), owner)
	if err != nil {
		JSONInternal(c)
		return
	}
	out := make([]instanceResponse, 0, len(items))
	for _, item := range items {
		out = append(out, instanceFromModel(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *InstancesHandler) Create(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req createInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	result, err := h.Service.CreateInstance(c.Request.Context(), owner, launcher.CreateInstanceInput{
		Name:          req.Name,
		MCVersion:     req.MCVersion,
		Loader:        req.Loader,
		LoaderVersion: req.LoaderVersion,
	})
	if err != nil {
		if errors.Is(err, launcher.ErrValidation) {
			JSONValidation(c, "invalid instance data")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusCreated, instanceCreateResponse(*result.Instance, result.PrepareRequestID))
}

func (h *InstancesHandler) Clone(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	result, err := h.Service.CloneInstance(c.Request.Context(), owner, c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, launcher.ErrNotFound):
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "instance not found")
		case errors.Is(err, launcher.ErrValidation):
			JSONValidation(c, "invalid instance data")
		case errors.Is(err, launcher.ErrDeviceNotLinked):
			JSONError(c, http.StatusForbidden, "FORBIDDEN", "device not linked to account")
		case errors.Is(err, launcher.ErrBridgeTimeout):
			JSONError(c, http.StatusGatewayTimeout, "LAUNCHER_TIMEOUT", "launcher did not respond in time")
		default:
			JSONInternal(c)
		}
		return
	}
	c.JSON(http.StatusCreated, instanceCreateResponse(*result.Instance, result.PrepareRequestID))
}

func (h *InstancesHandler) Get(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	inst, err := h.Service.GetInstance(c.Request.Context(), owner, c.Param("id"))
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "instance not found")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, instanceFromModel(*inst))
}

func (h *InstancesHandler) Update(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req updateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	if !hasInstanceUpdateFields(req) {
		JSONValidation(c, "at least one instance setting required")
		return
	}
	inst, err := h.Service.UpdateInstance(c.Request.Context(), owner, c.Param("id"), launcher.UpdateInstanceInput{
		Name:         req.Name,
		MaxMemoryMB:  req.MaxMemoryMB,
		MinMemoryMB:  req.MinMemoryMB,
		ExtraJVMArgs: req.ExtraJVMArgs,
		WindowWidth:  req.WindowWidth,
		WindowHeight: req.WindowHeight,
	})
	if err != nil {
		if errors.Is(err, launcher.ErrValidation) {
			JSONValidation(c, "invalid instance settings")
			return
		}
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "instance not found")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, instanceFromModel(*inst))
}

func (h *InstancesHandler) ListResources(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	items, err := h.Service.ListInstanceResources(c.Request.Context(), owner, c.Param("id"))
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "instance not found")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

type deleteInstanceResourceRequest struct {
	Source       string `json:"source" binding:"required"`
	ProjectID    string `json:"project_id"`
	Filename     string `json:"filename"`
	ResourceType string `json:"resource_type"`
}

func (h *InstancesHandler) DeleteResource(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req deleteInstanceResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	if req.ProjectID == "" && req.Filename == "" {
		JSONValidation(c, "project_id or filename required")
		return
	}
	err := h.Service.DeleteInstanceResourceWithBridge(c.Request.Context(), owner, c.Param("id"), launcher.DeleteInstanceResourceInput{
		Source:       req.Source,
		ProjectID:    req.ProjectID,
		Filename:     req.Filename,
		ResourceType: req.ResourceType,
	})
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
			return
		}
		if errors.Is(err, launcher.ErrValidation) {
			JSONValidation(c, "invalid resource data")
			return
		}
		if errors.Is(err, launcher.ErrInstanceManaged) {
			JSONError(c, http.StatusForbidden, "INSTANCE_MANAGED", "this instance is managed by a game server")
			return
		}
		if errors.Is(err, launcher.ErrDeviceNotLinked) {
			JSONError(c, http.StatusForbidden, "FORBIDDEN", "device not linked to account")
			return
		}
		if errors.Is(err, launcher.ErrBridgeTimeout) {
			JSONError(c, http.StatusGatewayTimeout, "LAUNCHER_TIMEOUT", "launcher did not respond in time")
			return
		}
		JSONInternal(c)
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

type syncUploadedResourceBody struct {
	VpsID        string `json:"vps_id" binding:"required"`
	GameServerID string `json:"game_server_id" binding:"required"`
	Filename     string `json:"filename" binding:"required"`
	ResourceType string `json:"resource_type"`
	ModTarget    string `json:"mod_target"`
}

type patchInstanceResourceRequest struct {
	Source       string `json:"source" binding:"required"`
	ProjectID    string `json:"project_id"`
	Filename     string `json:"filename"`
	ResourceType string `json:"resource_type"`
	SideOverride string `json:"side_override"`
}

func (h *InstancesHandler) PatchResource(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req patchInstanceResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	if req.ProjectID == "" && req.Filename == "" {
		JSONValidation(c, "project_id or filename required")
		return
	}
	err := h.Service.UpdateInstanceResourceSide(c.Request.Context(), owner, c.Param("id"), launcher.UpdateInstanceResourceSideInput{
		Source:       req.Source,
		ProjectID:    req.ProjectID,
		Filename:     req.Filename,
		ResourceType: req.ResourceType,
		SideOverride: req.SideOverride,
	})
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
			return
		}
		if errors.Is(err, launcher.ErrValidation) {
			JSONValidation(c, "invalid resource data")
			return
		}
		if errors.Is(err, launcher.ErrInstanceManaged) {
			JSONError(c, http.StatusForbidden, "INSTANCE_MANAGED", "this instance is managed by a game server")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *InstancesHandler) SyncUploadedResource(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	if h.ServerService == nil {
		JSONInternal(c)
		return
	}
	var body syncUploadedResourceBody
	if err := c.ShouldBindJSON(&body); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	instanceID := c.Param("id")
	resourceType := body.ResourceType
	if resourceType == "" {
		resourceType = "mod"
	}
	resourceType = strings.ToLower(strings.TrimSpace(resourceType))
	switch resourceType {
	case "mod", "resourcepack", "shader":
	default:
		JSONValidation(c, "unsupported resource type for server sync")
		return
	}

	inst, err := h.Service.GetInstance(c.Request.Context(), owner, instanceID)
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "instance not found")
			return
		}
		JSONInternal(c)
		return
	}
	if err := h.Service.AssertInstanceContentMutable(c.Request.Context(), instanceID); err != nil {
		if errors.Is(err, launcher.ErrInstanceManaged) {
			JSONError(c, http.StatusForbidden, "INSTANCE_MANAGED", "this instance is managed by a game server")
			return
		}
		JSONInternal(c)
		return
	}
	if launcher.FindUploadedInstanceResource(inst, body.Filename, resourceType) == nil {
		JSONError(c, http.StatusNotFound, "NOT_FOUND", "uploaded resource not found on instance")
		return
	}

	modTarget := strings.TrimSpace(body.ModTarget)
	entries, listErr := listUploadedSyncEntries(
		c.Request.Context(),
		h.ServerService,
		owner.UserID,
		body.VpsID,
		body.GameServerID,
		resourceType,
		modTarget,
	)
	if listErr != nil {
		gameServerError(c, listErr)
		return
	}
	for _, entry := range entries {
		if entry.Dir {
			continue
		}
		if strings.EqualFold(entry.Name, body.Filename) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "already_installed",
				"message": resourceType + " file already exists on server",
			})
			return
		}
	}

	data, err := h.Service.ExportInstanceResource(
		c.Request.Context(),
		owner,
		instanceID,
		body.Filename,
		resourceType,
	)
	if err != nil {
		mapInstanceBridgeError(c, err)
		return
	}

	result, err := h.ServerService.UploadGameServerContent(
		c.Request.Context(),
		owner.UserID,
		body.VpsID,
		body.GameServerID,
		resourceType,
		modTarget,
		body.Filename,
		data,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":   "installed",
		"message":  resourceType + " installed on server",
		"filename": result.Filename,
		"path":     result.RelPath,
	})
}

func listUploadedSyncEntries(
	ctx context.Context,
	service *servers.Service,
	userID, vpsID, gameServerID, resourceType, modTarget string,
) ([]protocol.FileEntry, error) {
	switch resourceType {
	case "mod":
		if strings.EqualFold(modTarget, "client-mods") {
			return service.ListGameServerClientMods(ctx, userID, vpsID, gameServerID)
		}
		return service.ListGameServerMods(ctx, userID, vpsID, gameServerID)
	case "resourcepack":
		if strings.EqualFold(modTarget, "client-resourcepacks") {
			return service.ListGameServerClientResourcepacks(ctx, userID, vpsID, gameServerID)
		}
		return service.ListGameServerResourcepacks(ctx, userID, vpsID, gameServerID)
	case "shader":
		if strings.EqualFold(modTarget, "client-shaders") {
			return service.ListGameServerClientShaders(ctx, userID, vpsID, gameServerID)
		}
		return service.ListGameServerShaders(ctx, userID, vpsID, gameServerID)
	default:
		return nil, servers.ErrValidation
	}
}

func (h *InstancesHandler) Delete(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	if err := h.Service.DeleteInstance(c.Request.Context(), owner, c.Param("id")); err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "instance not found")
			return
		}
		JSONInternal(c)
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

func (h *InstancesHandler) Manifest(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	manifest, err := h.Service.InstanceManifest(c.Request.Context(), owner, c.Param("id"))
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "instance not found")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, manifest)
}
