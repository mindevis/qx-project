package api

import (
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

func TestMonitoringList_PublicEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := testutil.OpenSQLiteDB(t)
	svc := servers.NewService(db, nil, nil, nil, servers.NoopDeployer{})
	h := &MonitoringHandler{Service: svc}

	ownerID := "owner-monitoring-http"
	vpsID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	addr := "play.example.com"
	now := time.Now().UTC()

	require.NoError(t, db.Create(&models.User{
		ID: ownerID, Email: "monitoring-http@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.Server{
		ID: vpsID, OwnerID: ownerID, Name: "Dedicated", Slug: "dedicated", Status: models.ServerStatusOnline,
		ConfigJSON: "{}", CreatedAt: now, UpdatedAt: now,
	}).Error)
	loader := "47.0.0"
	require.NoError(t, db.Create(&models.GameServer{
		ID: "gs-monitoring-http", ServerID: vpsID, Name: "Forge Public",
		ServerType: "forge", MCVersion: "1.21", LoaderVersion: &loader, Address: &addr, Port: 25565,
		Status: models.GameServerStatusRunning, ShowInMonitoring: true, CreatedAt: now, UpdatedAt: now,
	}).Error)

	r := gin.New()
	r.GET("/api/v1/monitoring/servers", h.List)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/servers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Items []struct {
			ID         string   `json:"id"`
			Name       string   `json:"name"`
			ServerType string   `json:"server_type"`
			IsOnline   bool     `json:"is_online"`
			Mods       []string `json:"mods"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Items, 1)
	require.Equal(t, "gs-monitoring-http", body.Items[0].ID)
	require.Equal(t, "forge", body.Items[0].ServerType)
	require.True(t, body.Items[0].IsOnline)
	require.Empty(t, body.Items[0].Mods)
}
