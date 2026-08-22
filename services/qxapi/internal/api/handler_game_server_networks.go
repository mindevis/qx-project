package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/servers"
)

type GameServerNetworksHandler struct {
	Service *servers.Service
}

type createGameServerNetworkRequest struct {
	Name string `json:"name" binding:"required"`
}

type updateGameServerNetworkRequest struct {
	Name      string                                 `json:"name"`
	Members   []servers.GameServerNetworkMemberInput `json:"members"`
	Apply     *bool                                  `json:"apply"`
	Overwrite bool                                   `json:"overwrite"`
}

type applyGameServerNetworkRequest struct {
	Overwrite bool `json:"overwrite"`
}

func (h *GameServerNetworksHandler) List(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	items, err := h.Service.ListGameServerNetworks(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *GameServerNetworksHandler) Create(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req createGameServerNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.CreateGameServerNetwork(c.Request.Context(), userID.(string), c.Param("id"), strings.TrimSpace(req.Name))
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusCreated, view)
}

func (h *GameServerNetworksHandler) Update(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req updateGameServerNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	apply := true
	if req.Apply != nil {
		apply = *req.Apply
	}
	view, err := h.Service.UpdateGameServerNetwork(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("networkId"),
		req.Name,
		req.Members,
		apply,
		req.Overwrite,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *GameServerNetworksHandler) Apply(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req applyGameServerNetworkRequest
	_ = c.ShouldBindJSON(&req)
	view, err := h.Service.ApplyGameServerNetwork(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("networkId"),
		req.Overwrite,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *GameServerNetworksHandler) Delete(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	if err := h.Service.DeleteGameServerNetwork(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("networkId"),
	); err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
