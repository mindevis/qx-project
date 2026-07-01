package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/servers"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func TestMonitoringBindings_HTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := testutil.OpenSQLiteDB(t)
	svc := servers.NewService(db, nil, nil, nil, servers.NoopDeployer{})
	h := &MonitoringHandler{Service: svc}

	ownerID := "binding-http-user"
	gameServerID := "gs-binding-http"
	vpsID := "vps-binding-http"
	now := time.Now().UTC()
	addr := "play.example.com"

	require.NoError(t, db.Create(&models.User{
		ID: "server-owner-http", Email: "owner-http@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.Server{
		ID: vpsID, OwnerID: "server-owner-http", Name: "VPS", Status: models.ServerStatusOnline, CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.GameServer{
		ID: gameServerID, ServerID: vpsID, Name: "Public",
		ServerType: models.ServerTypeVanilla, MCVersion: "1.21", Address: &addr, Port: 25565,
		Status: models.GameServerStatusRunning, ShowInMonitoring: true, CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.User{
		ID: ownerID, Email: "binding-http@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)
	instID := "inst-binding-http"
	require.NoError(t, db.Create(&models.LauncherInstance{
		ID: instID, UserID: &ownerID, Name: "Survival Client", MCVersion: "1.21", Loader: models.LoaderVanilla,
		CreatedAt: now, UpdatedAt: now,
	}).Error)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(UserIDKey, ownerID)
		c.Next()
	})
	r.GET("/api/v1/monitoring/bindings", h.ListBindings)
	r.PUT("/api/v1/monitoring/servers/:id/binding", h.SetBinding)
	r.DELETE("/api/v1/monitoring/servers/:id/binding", h.ClearBinding)

	body := `{"instance_id":"` + instID + `"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/monitoring/servers/"+gameServerID+"/binding", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var binding struct {
		GameServerID string `json:"game_server_id"`
		InstanceID   string `json:"instance_id"`
		InstanceName string `json:"instance_name"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &binding))
	require.Equal(t, gameServerID, binding.GameServerID)
	require.Equal(t, instID, binding.InstanceID)
	require.Equal(t, "Survival Client", binding.InstanceName)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/bindings", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var list struct {
		Items []struct {
			GameServerID string `json:"game_server_id"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	require.Len(t, list.Items, 1)
	require.Equal(t, gameServerID, list.Items[0].GameServerID)

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/monitoring/servers/"+gameServerID+"/binding", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}
