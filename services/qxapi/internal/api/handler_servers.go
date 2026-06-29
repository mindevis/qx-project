package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/deploy"
	"github.com/qxproject/qx/services/qxapi/internal/servers"
)

type ServersHandler struct {
	Service *servers.Service
}

type sshInputRequest struct {
	Host                 string `json:"host" binding:"required"`
	Port                 int    `json:"port"`
	Username             string `json:"username" binding:"required"`
	PrivateKey           string `json:"private_key" binding:"required"`
	PrivateKeyPassphrase string `json:"private_key_passphrase"`
}

type serverConfigRequest struct {
	JarPath   string   `json:"jar_path"`
	JVMArgs   []string `json:"jvm_args"`
	ExtraArgs []string `json:"extra_args"`
}

type createServerRequest struct {
	Name       string              `json:"name" binding:"required"`
	ServerType string              `json:"server_type"`
	MCVersion  string              `json:"mc_version"`
	SSH        sshInputRequest     `json:"ssh" binding:"required"`
	Config     serverConfigRequest `json:"config"`
}

func (h *ServersHandler) List(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	items, err := h.Service.List(c.Request.Context(), userID.(string))
	if err != nil {
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *ServersHandler) Create(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req createServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.Create(c.Request.Context(), userID.(string), servers.CreateServerInput{
		Name:       req.Name,
		ServerType: req.ServerType,
		MCVersion:  req.MCVersion,
		SSH: servers.SSHInput{
			Host:                 req.SSH.Host,
			Port:                 req.SSH.Port,
			Username:             req.SSH.Username,
			PrivateKey:           req.SSH.PrivateKey,
			PrivateKeyPassphrase: req.SSH.PrivateKeyPassphrase,
		},
		Config: servers.ServerConfig{
			JarPath:   req.Config.JarPath,
			JVMArgs:   req.Config.JVMArgs,
			ExtraArgs: req.Config.ExtraArgs,
		},
	})
	if err != nil {
		if errors.Is(err, servers.ErrValidation) {
			JSONValidation(c, "invalid server data")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusCreated, view)
}

func (h *ServersHandler) Get(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.Get(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		if errors.Is(err, servers.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "server not found")
			return
		}
		if errors.Is(err, servers.ErrForbidden) {
			JSONError(c, http.StatusForbidden, "FORBIDDEN", "forbidden")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *ServersHandler) Delete(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	if err := h.Service.Delete(c.Request.Context(), userID.(string), c.Param("id")); err != nil {
		if errors.Is(err, servers.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "server not found")
			return
		}
		if errors.Is(err, servers.ErrForbidden) {
			JSONError(c, http.StatusForbidden, "FORBIDDEN", "forbidden")
			return
		}
		JSONInternal(c)
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

func (h *ServersHandler) Deploy(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	out, err := h.Service.Deploy(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, servers.ErrNotFound):
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "server not found")
		case errors.Is(err, servers.ErrForbidden):
			JSONError(c, http.StatusForbidden, "FORBIDDEN", "forbidden")
		case errors.Is(err, deploy.ErrNonLinuxHost):
			JSONError(c, http.StatusUnprocessableEntity, "HOST_NOT_LINUX", "QX agent requires a Linux dedicated server")
		case errors.Is(err, deploy.ErrInvalidSSHKey):
			JSONValidation(c, "invalid ssh private key")
		case errors.Is(err, deploy.ErrBinaryNotConfigured):
			JSONError(c, http.StatusFailedDependency, "AGENT_BINARY_MISSING", "configure agent_binary_path in qxapi.toml (see qxapi.toml.example)")
		default:
			slog.Error("deploy failed", "error", err, "server_id", c.Param("id"))
			msg := "deploy failed"
			if err != nil {
				msg = err.Error()
			}
			JSONError(c, http.StatusBadGateway, "DEPLOY_FAILED", msg)
		}
		return
	}
	c.JSON(http.StatusOK, out.View)
}

func (h *ServersHandler) Start(c *gin.Context) {
	h.lifecycle(c, h.Service.Start)
}

func (h *ServersHandler) Stop(c *gin.Context) {
	h.lifecycle(c, h.Service.Stop)
}

func (h *ServersHandler) Restart(c *gin.Context) {
	h.lifecycle(c, h.Service.Restart)
}

func (h *ServersHandler) lifecycle(c *gin.Context, fn func(context.Context, string, string) error) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	err := fn(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, servers.ErrNotFound):
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "server not found")
		case errors.Is(err, servers.ErrForbidden):
			JSONError(c, http.StatusForbidden, "FORBIDDEN", "forbidden")
		case errors.Is(err, servers.ErrNotDeployed):
			JSONError(c, http.StatusConflict, "NOT_DEPLOYED", "deploy agent first")
		case errors.Is(err, servers.ErrAgentOffline):
			JSONError(c, http.StatusConflict, "AGENT_OFFLINE", "agent is not connected")
		default:
			JSONInternal(c)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
