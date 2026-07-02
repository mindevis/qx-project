package api

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/launcher"
)

func (h *InstancesHandler) ListFiles(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	path := c.Query("path")
	entries, err := h.Service.ListInstanceFiles(c.Request.Context(), owner, c.Param("id"), path)
	if err != nil {
		mapInstanceBridgeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": entries})
}

func (h *InstancesHandler) ReadFile(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	path := strings.TrimSpace(c.Query("path"))
	if path == "" {
		JSONValidation(c, "path query parameter required")
		return
	}
	result, err := h.Service.ReadInstanceFile(c.Request.Context(), owner, c.Param("id"), path)
	if err != nil {
		mapInstanceBridgeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

type writeInstanceFileBody struct {
	Content string `json:"content"`
}

func (h *InstancesHandler) WriteFile(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	path := strings.TrimSpace(c.Query("path"))
	if path == "" {
		JSONValidation(c, "path query parameter required")
		return
	}
	var body writeInstanceFileBody
	if err := c.ShouldBindJSON(&body); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	if err := h.Service.WriteInstanceFile(c.Request.Context(), owner, c.Param("id"), path, body.Content); err != nil {
		mapInstanceBridgeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *InstancesHandler) UploadResource(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		JSONValidation(c, "missing file")
		return
	}
	if file.Size > launcher.MaxResourceUploadBytes {
		JSONValidation(c, "file too large")
		return
	}
	f, err := file.Open()
	if err != nil {
		JSONInternal(c)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, launcher.MaxResourceUploadBytes+1))
	if err != nil {
		JSONInternal(c)
		return
	}
	if int64(len(data)) > launcher.MaxResourceUploadBytes {
		JSONValidation(c, "file too large")
		return
	}
	resourceType := c.PostForm("resource_type")
	view, err := h.Service.CreateInstanceResourceUpload(
		c.Request.Context(),
		owner,
		c.Param("id"),
		file.Filename,
		resourceType,
		data,
	)
	if err != nil {
		mapInstanceBridgeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, view)
}

func mapInstanceBridgeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, launcher.ErrNotFound):
		JSONError(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
	case errors.Is(err, launcher.ErrValidation):
		JSONValidation(c, "invalid request")
	case errors.Is(err, launcher.ErrDeviceNotLinked):
		JSONError(c, http.StatusForbidden, "FORBIDDEN", "device not linked to account")
	case errors.Is(err, launcher.ErrBridgeTimeout):
		JSONError(c, http.StatusGatewayTimeout, "LAUNCHER_TIMEOUT", "launcher did not respond in time")
	default:
		if err != nil && strings.TrimSpace(err.Error()) != "" && !errors.Is(err, launcher.ErrBridgeTimeout) {
			JSONError(c, http.StatusBadGateway, "LAUNCHER_ERROR", err.Error())
			return
		}
		JSONInternal(c)
	}
}
