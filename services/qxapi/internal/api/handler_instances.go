package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/launcher"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

type InstancesHandler struct {
	Service *launcher.Service
}

type instanceResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	MCVersion     string   `json:"mc_version"`
	Loader        string   `json:"loader"`
	LoaderVersion *string  `json:"loader_version,omitempty"`
	MaxMemoryMB   *int     `json:"max_memory_mb,omitempty"`
	MinMemoryMB   *int     `json:"min_memory_mb,omitempty"`
	ExtraJVMArgs  []string `json:"extra_jvm_args,omitempty"`
	WindowWidth   *int     `json:"window_width,omitempty"`
	WindowHeight  *int     `json:"window_height,omitempty"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

func instanceFromModel(inst models.LauncherInstance) instanceResponse {
	return instanceResponse{
		ID:            inst.ID,
		Name:          inst.Name,
		MCVersion:     inst.MCVersion,
		Loader:        inst.Loader,
		LoaderVersion: inst.LoaderVersion,
		MaxMemoryMB:   inst.MaxMemoryMB,
		MinMemoryMB:   inst.MinMemoryMB,
		ExtraJVMArgs:  []string(inst.ExtraJVMArgs),
		WindowWidth:   inst.WindowWidth,
		WindowHeight:  inst.WindowHeight,
		CreatedAt:     inst.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     inst.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type createInstanceRequest struct {
	Name          string `json:"name" binding:"required"`
	MCVersion     string `json:"mc_version" binding:"required"`
	Loader        string `json:"loader"`
	LoaderVersion string `json:"loader_version"`
}

type updateInstanceRequest struct {
	MaxMemoryMB  *int      `json:"max_memory_mb"`
	MinMemoryMB  *int      `json:"min_memory_mb"`
	ExtraJVMArgs *[]string `json:"extra_jvm_args"`
	WindowWidth  *int      `json:"window_width"`
	WindowHeight *int      `json:"window_height"`
}

func hasInstanceUpdateFields(req updateInstanceRequest) bool {
	return req.MaxMemoryMB != nil || req.MinMemoryMB != nil || req.ExtraJVMArgs != nil ||
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
	inst, err := h.Service.CreateInstance(c.Request.Context(), owner, launcher.CreateInstanceInput{
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
	c.JSON(http.StatusCreated, instanceFromModel(*inst))
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
