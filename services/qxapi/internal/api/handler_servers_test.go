package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/crypto"
	"github.com/qxproject/qx/services/qxapi/internal/deploy"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/servers"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

type stubDeployer struct {
	err error
}

func (s stubDeployer) Deploy(context.Context, string, models.SSHCredential, string) error {
	return s.err
}

const testSSHKey = `-----BEGIN OPENSSH PRIVATE KEY-----
test-key-content
-----END OPENSSH PRIVATE KEY-----`

func newServersHandler(t *testing.T) (*ServersHandler, *auth.Service) {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	authSvc := auth.NewService(db, tokens)
	enc, _ := crypto.NewEncryptor(devSSHMasterKey())
	svc := servers.NewService(db, tokens, enc, nil, servers.NoopDeployer{})
	return &ServersHandler{Service: svc}, authSvc
}

func registerUserToken(t *testing.T, authSvc *auth.Service) (userID, accessToken string) {
	t.Helper()
	user, tokens, err := authSvc.Register(context.Background(), auth.RegisterInput{
		Email:    "servers@test.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return user.ID, tokens.AccessToken
}

func TestServersHandlerCRUD(t *testing.T) {
	h, authSvc := newServersHandler(t)
	userID, _ := registerUserToken(t, authSvc)

	createBody := map[string]any{
		"name": "My dedicated server",
		"ssh": map[string]any{
			"host":        "10.0.0.2",
			"port":        22,
			"username":    "root",
			"private_key": testSSHKey,
		},
		"config": map[string]any{"jar_path": "/opt/qxsystem/server.jar"},
	}
	b, _ := json.Marshal(createBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/servers", bytes.NewReader(b))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, userID)
	h.Create(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.ID == "" {
		t.Fatalf("created: %v body=%s", err, w.Body.String())
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/servers", nil)
	c.Set(UserIDKey, userID)
	h.List(c)
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d", w.Code)
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/servers/"+created.ID, nil)
	c.Params = gin.Params{{Key: "id", Value: created.ID}}
	c.Set(UserIDKey, userID)
	h.Get(c)
	if w.Code != http.StatusOK {
		t.Fatalf("get: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/servers/"+created.ID, nil)
	c.Params = gin.Params{{Key: "id", Value: created.ID}}
	c.Set(UserIDKey, userID)
	h.Delete(c)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", w.Code)
	}
}

func TestServersHandlerUnauthorized(t *testing.T) {
	h, _ := newServersHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/servers", nil)
	h.List(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("list: %d", w.Code)
	}
}

func TestServersHandlerValidation(t *testing.T) {
	h, authSvc := newServersHandler(t)
	_, _ = registerUserToken(t, authSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/servers", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, "user-1")
	h.Create(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("validation: %d", w.Code)
	}
}

func TestServersHandlerDeployNonLinux(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	authSvc := auth.NewService(db, tokens)
	enc, _ := crypto.NewEncryptor(devSSHMasterKey())
	svc := servers.NewService(db, tokens, enc, nil, stubDeployer{
		err: fmt.Errorf("%w: Darwin", deploy.ErrNonLinuxHost),
	})
	h := &ServersHandler{Service: svc}
	userID, _ := registerUserToken(t, authSvc)

	createBody, _ := json.Marshal(map[string]any{
		"name": "Win dedicated server",
		"ssh": map[string]any{
			"host": "1.2.3.4", "username": "admin", "private_key": testSSHKey,
		},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/servers", bytes.NewReader(createBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, userID)
	h.Create(c)
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/servers/"+created.ID+"/deploy", nil)
	c.Params = gin.Params{{Key: "id", Value: created.ID}}
	c.Set(UserIDKey, userID)
	h.Deploy(c)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("deploy non-linux: %d %s", w.Code, w.Body.String())
	}
}

func TestServersHandlerDeployInvalidSSHKey(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	authSvc := auth.NewService(db, tokens)
	enc, _ := crypto.NewEncryptor(devSSHMasterKey())
	svc := servers.NewService(db, tokens, enc, nil, stubDeployer{
		err: fmt.Errorf("%w: parse", deploy.ErrInvalidSSHKey),
	})
	h := &ServersHandler{Service: svc}
	userID, _ := registerUserToken(t, authSvc)

	createBody, _ := json.Marshal(map[string]any{
		"name": "dedicated server",
		"ssh": map[string]any{
			"host": "1.2.3.4", "username": "root", "private_key": testSSHKey,
		},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/servers", bytes.NewReader(createBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, userID)
	h.Create(c)
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/servers/"+created.ID+"/deploy", nil)
	c.Params = gin.Params{{Key: "id", Value: created.ID}}
	c.Set(UserIDKey, userID)
	h.Deploy(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("deploy invalid key: %d %s", w.Code, w.Body.String())
	}
}

func TestServersHandlerDeployAndLifecycle(t *testing.T) {
	h, authSvc := newServersHandler(t)
	userID, _ := registerUserToken(t, authSvc)

	createBody, _ := json.Marshal(map[string]any{
		"name": "dedicated server",
		"ssh": map[string]any{
			"host": "1.2.3.4", "username": "root", "private_key": testSSHKey,
		},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/servers", bytes.NewReader(createBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, userID)
	h.Create(c)
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/servers/"+created.ID+"/deploy", nil)
	c.Params = gin.Params{{Key: "id", Value: created.ID}}
	c.Set(UserIDKey, userID)
	h.Deploy(c)
	if w.Code != http.StatusOK {
		t.Fatalf("deploy: %d %s", w.Code, w.Body.String())
	}
	var deployResp struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &deployResp); err != nil {
		t.Fatalf("decode deploy: %v", err)
	}
	if deployResp.Status != "offline" {
		t.Fatalf("status: %q", deployResp.Status)
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/servers/"+created.ID+"/start", nil)
	c.Params = gin.Params{{Key: "id", Value: created.ID}}
	c.Set(UserIDKey, userID)
	h.Start(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("start offline: %d %s", w.Code, w.Body.String())
	}
}

func TestServersHandlerNotFound(t *testing.T) {
	h, authSvc := newServersHandler(t)
	userID, _ := registerUserToken(t, authSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/servers/missing", nil)
	c.Params = gin.Params{{Key: "id", Value: "00000000-0000-0000-0000-000000000099"}}
	c.Set(UserIDKey, userID)
	h.Get(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("not found: %d", w.Code)
	}
}
