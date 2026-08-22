package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/agenthub"
	"github.com/qxproject/qx/services/qxapi/internal/servers"
)

type OllamaHandler struct {
	Service *servers.Service
}

type pullOllamaModelRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *OllamaHandler) Get(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.GetOllama(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		ollamaError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *OllamaHandler) Install(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.InstallOllama(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		ollamaError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, view)
}

func (h *OllamaHandler) Start(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.StartOllama(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		ollamaError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *OllamaHandler) Stop(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.StopOllama(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		ollamaError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *OllamaHandler) PullModel(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req pullOllamaModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.PullOllamaModel(c.Request.Context(), userID.(string), c.Param("id"), req.Name)
	if err != nil {
		ollamaError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, view)
}

func (h *OllamaHandler) DeleteModel(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	name := strings.TrimSpace(c.Query("name"))
	view, err := h.Service.DeleteOllamaModel(c.Request.Context(), userID.(string), c.Param("id"), name)
	if err != nil {
		ollamaError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func ollamaError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, servers.ErrNotFound):
		JSONError(c, http.StatusNotFound, "NOT_FOUND", "not found")
	case errors.Is(err, servers.ErrForbidden):
		JSONError(c, http.StatusForbidden, "FORBIDDEN", "forbidden")
	case errors.Is(err, servers.ErrNotDeployed):
		JSONError(c, http.StatusConflict, "NOT_DEPLOYED", "deploy agent first")
	case errors.Is(err, servers.ErrAgentOffline):
		JSONError(c, http.StatusConflict, "AGENT_OFFLINE", "agent is not connected")
	case errors.Is(err, servers.ErrOllamaBusy):
		JSONError(c, http.StatusConflict, "OLLAMA_BUSY", "ollama operation already in progress")
	case errors.Is(err, servers.ErrOllamaNotInstalled):
		JSONError(c, http.StatusConflict, "OLLAMA_NOT_INSTALLED", "install ollama first")
	case errors.Is(err, servers.ErrOllamaNotRunning):
		JSONError(c, http.StatusConflict, "OLLAMA_NOT_RUNNING", "start ollama first")
	case errors.Is(err, servers.ErrOllamaAlreadyRunning):
		JSONError(c, http.StatusConflict, "OLLAMA_ALREADY_RUNNING", "ollama is already running")
	case errors.Is(err, servers.ErrOllamaInvalidModel):
		JSONValidation(c, "invalid ollama model name")
	case errors.Is(err, agenthub.ErrTimeout):
		JSONError(c, http.StatusGatewayTimeout, "AGENT_TIMEOUT", "agent did not respond in time")
	default:
		JSONInternal(c)
	}
}
