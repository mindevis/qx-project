package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/agenthub"
	"github.com/qxproject/qx/services/qxapi/internal/servers"
)

type MySQLHandler struct {
	Service *servers.Service
}

type mysqlInstallRequest struct {
	Engine   string `json:"engine" binding:"required"`
	Version  string `json:"version" binding:"required"`
	Method   string `json:"method" binding:"required"`
	BindAddr string `json:"bind_addr"`
	Port     int    `json:"port"`
}

type mysqlDatabaseRequest struct {
	Name string `json:"name" binding:"required"`
}

type mysqlGrantBinding struct {
	Database   string   `json:"database" binding:"required"`
	Privileges []string `json:"privileges"`
}

type mysqlUserRequest struct {
	Username string              `json:"username" binding:"required"`
	Password string              `json:"password"`
	Host     string              `json:"host"`
	Grants   []mysqlGrantBinding `json:"grants"`
}

type mysqlGrantsRequest struct {
	Host   string              `json:"host"`
	Grants []mysqlGrantBinding `json:"grants"`
}

func (h *MySQLHandler) Get(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.GetMySQL(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		mysqlError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *MySQLHandler) Install(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req mysqlInstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.InstallMySQL(c.Request.Context(), userID.(string), c.Param("id"), servers.MySQLInstallInput{
		Engine:   req.Engine,
		Version:  req.Version,
		Method:   req.Method,
		BindAddr: req.BindAddr,
		Port:     req.Port,
	})
	if err != nil {
		mysqlError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, view)
}

func (h *MySQLHandler) Start(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.StartMySQL(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		mysqlError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *MySQLHandler) Stop(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.StopMySQL(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		mysqlError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *MySQLHandler) CreateDatabase(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req mysqlDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.CreateMySQLDatabase(c.Request.Context(), userID.(string), c.Param("id"), req.Name)
	if err != nil {
		mysqlError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *MySQLHandler) DropDatabase(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.DropMySQLDatabase(c.Request.Context(), userID.(string), c.Param("id"), c.Param("name"))
	if err != nil {
		mysqlError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *MySQLHandler) CreateUser(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req mysqlUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.CreateMySQLUser(c.Request.Context(), userID.(string), c.Param("id"), servers.MySQLUserInput{
		Username: req.Username,
		Password: req.Password,
		Host:     req.Host,
		Grants:   toGrantInputs(req.Grants),
	})
	if err != nil {
		mysqlError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *MySQLHandler) DropUser(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.DropMySQLUser(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("username"),
		strings.TrimSpace(c.Query("host")),
	)
	if err != nil {
		mysqlError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *MySQLHandler) SetUserGrants(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req mysqlGrantsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	host := strings.TrimSpace(req.Host)
	if host == "" {
		host = strings.TrimSpace(c.Query("host"))
	}
	view, err := h.Service.SetMySQLUserGrants(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("username"),
		host,
		toGrantInputs(req.Grants),
	)
	if err != nil {
		mysqlError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func toGrantInputs(items []mysqlGrantBinding) []servers.MySQLGrantInput {
	out := make([]servers.MySQLGrantInput, 0, len(items))
	for _, item := range items {
		out = append(out, servers.MySQLGrantInput{Database: item.Database, Privileges: item.Privileges})
	}
	return out
}

func mysqlError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, servers.ErrNotFound):
		JSONError(c, http.StatusNotFound, "NOT_FOUND", "not found")
	case errors.Is(err, servers.ErrForbidden):
		JSONError(c, http.StatusForbidden, "FORBIDDEN", "forbidden")
	case errors.Is(err, servers.ErrNotDeployed):
		JSONError(c, http.StatusConflict, "NOT_DEPLOYED", "deploy agent first")
	case errors.Is(err, servers.ErrAgentOffline):
		JSONError(c, http.StatusConflict, "AGENT_OFFLINE", "agent is not connected")
	case errors.Is(err, servers.ErrMySQLBusy):
		JSONError(c, http.StatusConflict, "MYSQL_BUSY", "mysql operation already in progress")
	case errors.Is(err, servers.ErrMySQLNotInstalled):
		JSONError(c, http.StatusConflict, "MYSQL_NOT_INSTALLED", "install mysql first")
	case errors.Is(err, servers.ErrMySQLNotRunning):
		JSONError(c, http.StatusConflict, "MYSQL_NOT_RUNNING", "start mysql first")
	case errors.Is(err, servers.ErrMySQLAlreadyRunning):
		JSONError(c, http.StatusConflict, "MYSQL_ALREADY_RUNNING", "mysql is already running")
	case errors.Is(err, servers.ErrMySQLInvalidIdent):
		JSONValidation(c, "invalid mysql name")
	case errors.Is(err, servers.ErrMySQLInvalidPrivilege):
		JSONValidation(c, "invalid mysql privilege")
	case errors.Is(err, servers.ErrMySQLInvalidEngine):
		JSONValidation(c, "invalid mysql engine, version, or install method")
	case errors.Is(err, agenthub.ErrTimeout):
		JSONError(c, http.StatusGatewayTimeout, "AGENT_TIMEOUT", "agent did not respond in time")
	default:
		JSONInternal(c)
	}
}
