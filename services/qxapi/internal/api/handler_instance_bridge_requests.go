package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/launcher"
)

type InstanceFileRequestsHandler struct {
	Service *launcher.Service
}

func (h *InstanceFileRequestsHandler) Pending(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.FetchPendingInstanceFile(c.Request.Context(), deviceID)
	if err != nil {
		mapInstanceBridgeError(c, err)
		return
	}
	if view == nil {
		c.JSON(http.StatusOK, gin.H{"item": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": view})
}

type patchInstanceFileRequestBody struct {
	Status     string  `json:"status" binding:"required"`
	ResultJSON string  `json:"result_json"`
	ErrorCode  *string `json:"error_code"`
}

func (h *InstanceFileRequestsHandler) Update(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req patchInstanceFileRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.UpdateInstanceFileRequest(c.Request.Context(), deviceID, c.Param("id"), launcher.UpdateInstanceFileRequestInput{
		Status:     req.Status,
		ResultJSON: req.ResultJSON,
		ErrorCode:  req.ErrorCode,
	})
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "instance file request not found")
			return
		}
		mapInstanceBridgeError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

type ModUninstallRequestsHandler struct {
	Service *launcher.Service
}

func (h *ModUninstallRequestsHandler) Pending(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.FetchPendingModUninstall(c.Request.Context(), deviceID)
	if err != nil {
		mapInstanceBridgeError(c, err)
		return
	}
	if view == nil {
		c.JSON(http.StatusOK, gin.H{"item": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": view})
}

type patchModUninstallRequestBody struct {
	Status    string  `json:"status" binding:"required"`
	ErrorCode *string `json:"error_code"`
}

func (h *ModUninstallRequestsHandler) Update(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req patchModUninstallRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.UpdateModUninstallRequest(c.Request.Context(), deviceID, c.Param("id"), launcher.UpdateModUninstallRequestInput{
		Status:    req.Status,
		ErrorCode: req.ErrorCode,
	})
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "mod uninstall request not found")
			return
		}
		mapInstanceBridgeError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

type ResourceUploadRequestsHandler struct {
	Service *launcher.Service
}

func (h *ResourceUploadRequestsHandler) Pending(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.FetchPendingResourceUpload(c.Request.Context(), deviceID)
	if err != nil {
		mapInstanceBridgeError(c, err)
		return
	}
	if view == nil {
		c.JSON(http.StatusOK, gin.H{"item": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": view})
}

type patchResourceUploadRequestBody struct {
	Status    string  `json:"status" binding:"required"`
	ErrorCode *string `json:"error_code"`
}

func (h *ResourceUploadRequestsHandler) Update(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req patchResourceUploadRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.UpdateResourceUploadRequest(c.Request.Context(), deviceID, c.Param("id"), launcher.UpdateResourceUploadRequestInput{
		Status:    req.Status,
		ErrorCode: req.ErrorCode,
	})
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "resource upload request not found")
			return
		}
		mapInstanceBridgeError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

type ResourceExportRequestsHandler struct {
	Service *launcher.Service
}

func (h *ResourceExportRequestsHandler) Pending(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.FetchPendingResourceExport(c.Request.Context(), deviceID)
	if err != nil {
		mapInstanceBridgeError(c, err)
		return
	}
	if view == nil {
		c.JSON(http.StatusOK, gin.H{"item": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": view})
}

type patchResourceExportRequestBody struct {
	Status     string  `json:"status" binding:"required"`
	ContentB64 string  `json:"content_b64"`
	ErrorCode  *string `json:"error_code"`
}

func (h *ResourceExportRequestsHandler) Update(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req patchResourceExportRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.UpdateResourceExportRequest(c.Request.Context(), deviceID, c.Param("id"), launcher.UpdateResourceExportRequestInput{
		Status:     req.Status,
		ContentB64: req.ContentB64,
		ErrorCode:  req.ErrorCode,
	})
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "resource export request not found")
			return
		}
		mapInstanceBridgeError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}
