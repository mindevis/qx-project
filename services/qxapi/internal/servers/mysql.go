package servers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/qxproject/qx/pkg/mysqlutil"
	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

const mysqlStartTimeout = 2 * time.Minute

type MySQLInstallInput struct {
	Engine   string
	Version  string
	Method   string
	BindAddr string
	Port     int
}

type MySQLGrantInput struct {
	Database   string
	Privileges []string
}

type MySQLUserInput struct {
	Username string
	Password string
	Host     string
	Grants   []MySQLGrantInput
}

type MySQLDatabaseView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type MySQLGrantView struct {
	Database   string   `json:"database"`
	Privileges []string `json:"privileges"`
}

type MySQLUserView struct {
	ID       string           `json:"id"`
	Username string           `json:"username"`
	Host     string           `json:"host"`
	Password string           `json:"password"`
	Grants   []MySQLGrantView `json:"grants"`
	JDBC     string           `json:"jdbc,omitempty"`
	DSN      string           `json:"dsn,omitempty"`
}

type MySQLView struct {
	Status           string              `json:"status"`
	Engine           string              `json:"engine,omitempty"`
	Version          string              `json:"version,omitempty"`
	PackageVersion   string              `json:"package_version,omitempty"`
	Method           string              `json:"method,omitempty"`
	BindAddr         string              `json:"bind_addr,omitempty"`
	Port             int                 `json:"port,omitempty"`
	Image            string              `json:"image,omitempty"`
	HostLocal        string              `json:"host_local,omitempty"`
	HostPublic       string              `json:"host_public,omitempty"`
	RootUser         string              `json:"root_user,omitempty"`
	RootPassword     string              `json:"root_password,omitempty"`
	JDBC             string              `json:"jdbc,omitempty"`
	DSN              string              `json:"dsn,omitempty"`
	LastError        string              `json:"last_error,omitempty"`
	Databases        []MySQLDatabaseView `json:"databases"`
	Users            []MySQLUserView     `json:"users"`
	PrivilegeCatalog []string            `json:"privilege_catalog"`
}

func (s *Service) GetMySQL(ctx context.Context, ownerID, vpsID string) (*MySQLView, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	row, err := s.getMySQLRow(ctx, vpsID)
	if err != nil {
		return nil, err
	}
	if s.hub != nil && s.hub.IsOnline(vpsID) {
		if raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdMySQLStatus, protocol.TypeResMySQLStatus, nil); err == nil {
			s.mergeMySQLStatus(ctx, vpsID, raw)
			if row, err = s.getMySQLRow(ctx, vpsID); err != nil {
				return nil, err
			}
		}
	}
	return s.mysqlView(ctx, vpsID, row)
}

func (s *Service) InstallMySQL(ctx context.Context, ownerID, vpsID string, in MySQLInstallInput) (*MySQLView, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	engine, version, err := mysqlutil.NormalizeEngineVersion(in.Engine, in.Version)
	if err != nil {
		return nil, ErrMySQLInvalidEngine
	}
	method, err := mysqlutil.NormalizeMethod(in.Method)
	if err != nil {
		return nil, ErrMySQLInvalidEngine
	}
	bind, err := mysqlutil.NormalizeBind(in.BindAddr)
	if err != nil {
		return nil, ErrMySQLInvalidEngine
	}
	port, err := mysqlutil.NormalizePort(in.Port)
	if err != nil {
		return nil, ErrMySQLInvalidEngine
	}
	if err := s.requireAgentOnline(ctx, vpsID); err != nil {
		return nil, err
	}
	row, err := s.getOrCreateMySQLRow(ctx, vpsID)
	if err != nil {
		return nil, err
	}
	if mysqlBusy(row.Status) {
		return nil, ErrMySQLBusy
	}
	password := strings.TrimSpace(row.RootPassword)
	if password == "" {
		password, err = generateRconPassword()
		if err != nil {
			return nil, err
		}
	}
	image, err := mysqlutil.DockerImage(engine, version)
	if err != nil {
		return nil, ErrMySQLInvalidEngine
	}
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(row).Updates(map[string]any{
		"status":        models.MySQLStatusInstalling,
		"engine":        engine,
		"version":       version,
		"method":        method,
		"bind_addr":     bind,
		"port":          port,
		"image":         image,
		"root_password": password,
		"last_error":    "",
		"updated_at":    now,
	}).Error; err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(protocol.MySQLInstallPayload{
		Engine:       engine,
		Version:      version,
		Method:       method,
		BindAddr:     bind,
		Port:         port,
		RootPassword: password,
	})
	if err := s.hub.SendCommand(ctx, vpsID, protocol.Envelope{
		V:         protocol.Version,
		Type:      protocol.TypeCmdMySQLInstall,
		RequestID: uuid.NewString(),
		TS:        now.Format(time.RFC3339),
		Payload:   payload,
	}); err != nil {
		_ = s.setMySQLError(ctx, vpsID, err.Error())
		return nil, err
	}
	row, err = s.getMySQLRow(ctx, vpsID)
	if err != nil {
		return nil, err
	}
	return s.mysqlView(ctx, vpsID, row)
}

func (s *Service) StartMySQL(ctx context.Context, ownerID, vpsID string) (*MySQLView, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	if err := s.requireAgentOnline(ctx, vpsID); err != nil {
		return nil, err
	}
	row, err := s.getMySQLRow(ctx, vpsID)
	if err != nil {
		return nil, err
	}
	if row.Status == models.MySQLStatusNotInstalled {
		return nil, ErrMySQLNotInstalled
	}
	if row.Status == models.MySQLStatusRunning {
		return nil, ErrMySQLAlreadyRunning
	}
	if mysqlBusy(row.Status) {
		return nil, ErrMySQLBusy
	}
	now := time.Now().UTC()
	_ = s.db.WithContext(ctx).Model(&models.ServerMySQL{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":     models.MySQLStatusStarting,
		"last_error": "",
		"updated_at": now,
	}).Error
	raw, err := s.agentRPCWait(ctx, vpsID, protocol.TypeCmdMySQLStart, protocol.TypeResMySQLStart, nil, mysqlStartTimeout)
	if err != nil {
		_ = s.setMySQLError(ctx, vpsID, err.Error())
		return nil, err
	}
	if err := agentPayloadError(raw); err != nil {
		_ = s.setMySQLError(ctx, vpsID, err.Error())
		return nil, err
	}
	_ = s.db.WithContext(ctx).Model(&models.ServerMySQL{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":     models.MySQLStatusRunning,
		"updated_at": time.Now().UTC(),
	}).Error
	return s.GetMySQL(ctx, ownerID, vpsID)
}

func (s *Service) StopMySQL(ctx context.Context, ownerID, vpsID string) (*MySQLView, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	if err := s.requireAgentOnline(ctx, vpsID); err != nil {
		return nil, err
	}
	row, err := s.getMySQLRow(ctx, vpsID)
	if err != nil {
		return nil, err
	}
	if row.Status == models.MySQLStatusNotInstalled {
		return nil, ErrMySQLNotInstalled
	}
	now := time.Now().UTC()
	_ = s.db.WithContext(ctx).Model(&models.ServerMySQL{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":     models.MySQLStatusStopping,
		"updated_at": now,
	}).Error
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdMySQLStop, protocol.TypeResMySQLStop, nil)
	if err != nil {
		_ = s.setMySQLError(ctx, vpsID, err.Error())
		return nil, err
	}
	if err := agentPayloadError(raw); err != nil {
		_ = s.setMySQLError(ctx, vpsID, err.Error())
		return nil, err
	}
	_ = s.db.WithContext(ctx).Model(&models.ServerMySQL{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":     models.MySQLStatusInstalled,
		"updated_at": time.Now().UTC(),
	}).Error
	return s.GetMySQL(ctx, ownerID, vpsID)
}

func (s *Service) CreateMySQLDatabase(ctx context.Context, ownerID, vpsID, name string) (*MySQLView, error) {
	if err := s.prepareMySQLSQL(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if !mysqlutil.ValidIdent(name) {
		return nil, ErrMySQLInvalidIdent
	}
	payload, _ := json.Marshal(protocol.MySQLIdentPayload{Name: name})
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdMySQLDatabaseCreate, protocol.TypeResMySQLDatabaseCreate, payload)
	if err != nil {
		return nil, err
	}
	if err := agentPayloadError(raw); err != nil {
		return nil, err
	}
	var existing models.ServerMySQLDatabase
	err = s.db.WithContext(ctx).Where("server_id = ? AND name = ?", vpsID, name).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := s.db.WithContext(ctx).Create(&models.ServerMySQLDatabase{
			ID:       uuid.NewString(),
			ServerID: vpsID,
			Name:     name,
		}).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return s.GetMySQL(ctx, ownerID, vpsID)
}

func (s *Service) DropMySQLDatabase(ctx context.Context, ownerID, vpsID, name string) (*MySQLView, error) {
	if err := s.prepareMySQLSQL(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if !mysqlutil.ValidIdent(name) {
		return nil, ErrMySQLInvalidIdent
	}
	payload, _ := json.Marshal(protocol.MySQLIdentPayload{Name: name})
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdMySQLDatabaseDrop, protocol.TypeResMySQLDatabaseDrop, payload)
	if err != nil {
		return nil, err
	}
	if err := agentPayloadError(raw); err != nil {
		return nil, err
	}
	var dbRow models.ServerMySQLDatabase
	if err := s.db.WithContext(ctx).Where("server_id = ? AND name = ?", vpsID, name).First(&dbRow).Error; err == nil {
		_ = s.db.WithContext(ctx).Where("database_id = ?", dbRow.ID).Delete(&models.ServerMySQLGrant{}).Error
		_ = s.db.WithContext(ctx).Delete(&dbRow).Error
	}
	return s.GetMySQL(ctx, ownerID, vpsID)
}

func (s *Service) CreateMySQLUser(ctx context.Context, ownerID, vpsID string, in MySQLUserInput) (*MySQLView, error) {
	row, err := s.runningMySQL(ctx, ownerID, vpsID)
	if err != nil {
		return nil, err
	}
	username := strings.TrimSpace(in.Username)
	if !mysqlutil.ValidUsername(username, row.Version) {
		return nil, ErrMySQLInvalidIdent
	}
	host := strings.TrimSpace(in.Host)
	if host == "" {
		host = "%"
	}
	if !mysqlutil.ValidHost(host) {
		return nil, ErrMySQLInvalidIdent
	}
	password := strings.TrimSpace(in.Password)
	if password == "" {
		password, err = generateRconPassword()
		if err != nil {
			return nil, err
		}
	}
	for _, grant := range in.Grants {
		if !mysqlutil.ValidIdent(strings.TrimSpace(grant.Database)) {
			return nil, ErrMySQLInvalidIdent
		}
		if _, err := mysqlutil.NormalizePrivileges(grant.Privileges); err != nil {
			return nil, ErrMySQLInvalidPrivilege
		}
		var dbRow models.ServerMySQLDatabase
		if err := s.db.WithContext(ctx).Where("server_id = ? AND name = ?", vpsID, strings.TrimSpace(grant.Database)).First(&dbRow).Error; err != nil {
			return nil, ErrMySQLInvalidIdent
		}
	}
	payload, _ := json.Marshal(protocol.MySQLUserPayload{Username: username, Password: password, Host: host})
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdMySQLUserCreate, protocol.TypeResMySQLUserCreate, payload)
	if err != nil {
		return nil, err
	}
	if err := agentPayloadError(raw); err != nil {
		return nil, err
	}
	user := models.ServerMySQLUser{
		ID:       uuid.NewString(),
		ServerID: vpsID,
		Username: username,
		Host:     host,
		Password: password,
	}
	if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
		return nil, err
	}
	if err := s.applyMySQLGrants(ctx, vpsID, &user, in.Grants); err != nil {
		return nil, err
	}
	return s.GetMySQL(ctx, ownerID, vpsID)
}

func (s *Service) DropMySQLUser(ctx context.Context, ownerID, vpsID, username, host string) (*MySQLView, error) {
	if err := s.prepareMySQLSQL(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	username = strings.TrimSpace(username)
	host = strings.TrimSpace(host)
	if host == "" {
		host = "%"
	}
	if !mysqlutil.ValidUsername(username, mysqlutil.Version80) || !mysqlutil.ValidHost(host) {
		return nil, ErrMySQLInvalidIdent
	}
	payload, _ := json.Marshal(protocol.MySQLUserPayload{Username: username, Host: host})
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdMySQLUserDrop, protocol.TypeResMySQLUserDrop, payload)
	if err != nil {
		return nil, err
	}
	if err := agentPayloadError(raw); err != nil {
		return nil, err
	}
	var user models.ServerMySQLUser
	if err := s.db.WithContext(ctx).Where("server_id = ? AND username = ? AND host = ?", vpsID, username, host).First(&user).Error; err == nil {
		_ = s.db.WithContext(ctx).Where("user_id = ?", user.ID).Delete(&models.ServerMySQLGrant{}).Error
		_ = s.db.WithContext(ctx).Delete(&user).Error
	}
	return s.GetMySQL(ctx, ownerID, vpsID)
}

func (s *Service) SetMySQLUserGrants(ctx context.Context, ownerID, vpsID, username, host string, grants []MySQLGrantInput) (*MySQLView, error) {
	if err := s.prepareMySQLSQL(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	username = strings.TrimSpace(username)
	host = strings.TrimSpace(host)
	if host == "" {
		host = "%"
	}
	var user models.ServerMySQLUser
	if err := s.db.WithContext(ctx).Where("server_id = ? AND username = ? AND host = ?", vpsID, username, host).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := s.replaceMySQLGrants(ctx, vpsID, &user, grants); err != nil {
		return nil, err
	}
	return s.GetMySQL(ctx, ownerID, vpsID)
}

func (s *Service) applyMySQLInstallResult(ctx context.Context, vpsID string, payload []byte) {
	if err := agentPayloadError(payload); err != nil {
		_ = s.setMySQLError(ctx, vpsID, err.Error())
		return
	}
	var result protocol.MySQLInstallResult
	if err := json.Unmarshal(payload, &result); err != nil {
		_ = s.setMySQLError(ctx, vpsID, "invalid mysql install result")
		return
	}
	now := time.Now().UTC()
	updates := map[string]any{
		"status":     models.MySQLStatusStarting,
		"last_error": "",
		"updated_at": now,
	}
	if result.Engine != "" {
		updates["engine"] = result.Engine
	}
	if result.Version != "" {
		updates["version"] = result.Version
	}
	if result.Method != "" {
		updates["method"] = result.Method
	}
	if result.BindAddr != "" {
		updates["bind_addr"] = result.BindAddr
	}
	if result.Port > 0 {
		updates["port"] = result.Port
	}
	if result.Image != "" {
		updates["image"] = result.Image
	}
	_ = s.db.WithContext(ctx).Model(&models.ServerMySQL{}).Where("server_id = ?", vpsID).Updates(updates).Error
	_ = s.hub.SendCommand(ctx, vpsID, protocol.Envelope{
		V:         protocol.Version,
		Type:      protocol.TypeCmdMySQLStart,
		RequestID: uuid.NewString(),
		TS:        now.Format(time.RFC3339),
	})
}

func (s *Service) applyMySQLStartResult(ctx context.Context, vpsID string, payload []byte) {
	if err := agentPayloadError(payload); err != nil {
		_ = s.setMySQLError(ctx, vpsID, err.Error())
		return
	}
	s.mergeMySQLStatus(ctx, vpsID, payload)
	_ = s.db.WithContext(ctx).Model(&models.ServerMySQL{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":     models.MySQLStatusRunning,
		"last_error": "",
		"updated_at": time.Now().UTC(),
	}).Error
}

func (s *Service) applyMySQLStopResult(ctx context.Context, vpsID string, payload []byte) {
	if err := agentPayloadError(payload); err != nil {
		_ = s.setMySQLError(ctx, vpsID, err.Error())
		return
	}
	_ = s.db.WithContext(ctx).Model(&models.ServerMySQL{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":     models.MySQLStatusInstalled,
		"last_error": "",
		"updated_at": time.Now().UTC(),
	}).Error
}

func (s *Service) mergeMySQLStatus(ctx context.Context, vpsID string, raw []byte) {
	var st protocol.MySQLStatusResult
	if json.Unmarshal(raw, &st) != nil {
		return
	}
	row, err := s.getMySQLRow(ctx, vpsID)
	if err != nil || mysqlBusy(row.Status) {
		return
	}
	if row.Status == models.MySQLStatusRunning && !st.Running {
		return
	}
	status := models.MySQLStatusNotInstalled
	if st.Running {
		status = models.MySQLStatusRunning
	} else if st.Installed {
		status = models.MySQLStatusInstalled
	}
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now().UTC(),
	}
	if st.Engine != "" {
		updates["engine"] = st.Engine
	}
	if st.Version != "" {
		updates["version"] = st.Version
	}
	if st.Method != "" {
		updates["method"] = st.Method
	}
	if st.BindAddr != "" {
		updates["bind_addr"] = st.BindAddr
	}
	if st.Port > 0 {
		updates["port"] = st.Port
	}
	if st.Image != "" {
		updates["image"] = st.Image
	}
	_ = s.db.WithContext(ctx).Model(&models.ServerMySQL{}).Where("server_id = ?", vpsID).Updates(updates).Error
}

func (s *Service) applyMySQLGrants(ctx context.Context, vpsID string, user *models.ServerMySQLUser, grants []MySQLGrantInput) error {
	for _, grant := range grants {
		if err := s.upsertMySQLGrant(ctx, vpsID, user, grant); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) replaceMySQLGrants(ctx context.Context, vpsID string, user *models.ServerMySQLUser, grants []MySQLGrantInput) error {
	wanted := map[string]MySQLGrantInput{}
	for _, grant := range grants {
		name := strings.TrimSpace(grant.Database)
		if !mysqlutil.ValidIdent(name) {
			return ErrMySQLInvalidIdent
		}
		wanted[name] = grant
	}
	var existing []models.ServerMySQLGrant
	if err := s.db.WithContext(ctx).Where("user_id = ?", user.ID).Find(&existing).Error; err != nil {
		return err
	}
	var dbs []models.ServerMySQLDatabase
	if err := s.db.WithContext(ctx).Where("server_id = ?", vpsID).Find(&dbs).Error; err != nil {
		return err
	}
	dbByID := map[string]models.ServerMySQLDatabase{}
	for _, dbRow := range dbs {
		dbByID[dbRow.ID] = dbRow
	}
	for _, row := range existing {
		dbRow, ok := dbByID[row.DatabaseID]
		if !ok {
			_ = s.db.WithContext(ctx).Delete(&row).Error
			continue
		}
		if _, keep := wanted[dbRow.Name]; keep {
			continue
		}
		if err := s.grantOnAgent(ctx, vpsID, user.Username, user.Host, dbRow.Name, nil); err != nil {
			return err
		}
		_ = s.db.WithContext(ctx).Delete(&row).Error
	}
	for _, grant := range wanted {
		if err := s.upsertMySQLGrant(ctx, vpsID, user, grant); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) upsertMySQLGrant(ctx context.Context, vpsID string, user *models.ServerMySQLUser, grant MySQLGrantInput) error {
	name := strings.TrimSpace(grant.Database)
	if !mysqlutil.ValidIdent(name) {
		return ErrMySQLInvalidIdent
	}
	privs, err := mysqlutil.NormalizePrivileges(grant.Privileges)
	if err != nil {
		return ErrMySQLInvalidPrivilege
	}
	var dbRow models.ServerMySQLDatabase
	if err := s.db.WithContext(ctx).Where("server_id = ? AND name = ?", vpsID, name).First(&dbRow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMySQLInvalidIdent
		}
		return err
	}
	if err := s.grantOnAgent(ctx, vpsID, user.Username, user.Host, name, privs); err != nil {
		return err
	}
	var existing models.ServerMySQLGrant
	err = s.db.WithContext(ctx).Where("user_id = ? AND database_id = ?", user.ID, dbRow.ID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.db.WithContext(ctx).Create(&models.ServerMySQLGrant{
			ID:         uuid.NewString(),
			UserID:     user.ID,
			DatabaseID: dbRow.ID,
			Privileges: models.StringList(privs),
		}).Error
	}
	if err != nil {
		return err
	}
	existing.Privileges = models.StringList(privs)
	return s.db.WithContext(ctx).Save(&existing).Error
}

func (s *Service) grantOnAgent(ctx context.Context, vpsID, username, host, database string, privs []string) error {
	payload, _ := json.Marshal(protocol.MySQLGrantPayload{
		Username:   username,
		Host:       host,
		Database:   database,
		Privileges: privs,
	})
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdMySQLUserGrant, protocol.TypeResMySQLUserGrant, payload)
	if err != nil {
		return err
	}
	return agentPayloadError(raw)
}

func (s *Service) prepareMySQLSQL(ctx context.Context, ownerID, vpsID string) error {
	_, err := s.runningMySQL(ctx, ownerID, vpsID)
	return err
}

func (s *Service) runningMySQL(ctx context.Context, ownerID, vpsID string) (*models.ServerMySQL, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	if err := s.requireAgentOnline(ctx, vpsID); err != nil {
		return nil, err
	}
	row, err := s.getMySQLRow(ctx, vpsID)
	if err != nil {
		return nil, err
	}
	if row.Status == models.MySQLStatusNotInstalled {
		return nil, ErrMySQLNotInstalled
	}
	if row.Status != models.MySQLStatusRunning {
		return nil, ErrMySQLNotRunning
	}
	return row, nil
}

func (s *Service) mysqlView(ctx context.Context, vpsID string, row *models.ServerMySQL) (*MySQLView, error) {
	view := mysqlViewFromRow(row)
	view.HostPublic = s.mysqlPublicHost(ctx, vpsID)
	if row != nil && row.Status != models.MySQLStatusNotInstalled && row.Port > 0 {
		host := view.HostLocal
		view.JDBC = fmt.Sprintf("jdbc:mysql://%s:%d/", host, row.Port)
		if row.RootPassword != "" {
			view.DSN = fmt.Sprintf("%s:%s@tcp(%s:%d)/", "root", row.RootPassword, host, row.Port)
		}
	}
	var databases []models.ServerMySQLDatabase
	if err := s.db.WithContext(ctx).Where("server_id = ?", vpsID).Order("name").Find(&databases).Error; err != nil {
		return nil, err
	}
	dbByID := map[string]string{}
	view.Databases = make([]MySQLDatabaseView, 0, len(databases))
	for _, dbRow := range databases {
		dbByID[dbRow.ID] = dbRow.Name
		view.Databases = append(view.Databases, MySQLDatabaseView{ID: dbRow.ID, Name: dbRow.Name})
	}
	var users []models.ServerMySQLUser
	if err := s.db.WithContext(ctx).Where("server_id = ?", vpsID).Order("username").Find(&users).Error; err != nil {
		return nil, err
	}
	view.Users = make([]MySQLUserView, 0, len(users))
	for _, user := range users {
		item := MySQLUserView{
			ID:       user.ID,
			Username: user.Username,
			Host:     user.Host,
			Password: user.Password,
			Grants:   []MySQLGrantView{},
		}
		if row != nil && row.Port > 0 {
			item.JDBC = fmt.Sprintf("jdbc:mysql://%s:%d/", view.HostLocal, row.Port)
			item.DSN = fmt.Sprintf("%s:%s@tcp(%s:%d)/", user.Username, user.Password, view.HostLocal, row.Port)
		}
		var grants []models.ServerMySQLGrant
		if err := s.db.WithContext(ctx).Where("user_id = ?", user.ID).Find(&grants).Error; err != nil {
			return nil, err
		}
		for _, grant := range grants {
			dbName := dbByID[grant.DatabaseID]
			if dbName == "" {
				continue
			}
			privs := []string(grant.Privileges)
			if privs == nil {
				privs = []string{}
			}
			item.Grants = append(item.Grants, MySQLGrantView{Database: dbName, Privileges: privs})
		}
		view.Users = append(view.Users, item)
	}
	return view, nil
}

func (s *Service) mysqlPublicHost(ctx context.Context, vpsID string) string {
	var cred models.SSHCredential
	if err := s.db.WithContext(ctx).Where("server_id = ?", vpsID).First(&cred).Error; err != nil {
		return ""
	}
	return cred.Host
}

func (s *Service) getMySQLRow(ctx context.Context, vpsID string) (*models.ServerMySQL, error) {
	var row models.ServerMySQL
	err := s.db.WithContext(ctx).Where("server_id = ?", vpsID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &models.ServerMySQL{
			ServerID: vpsID,
			Status:   models.MySQLStatusNotInstalled,
			BindAddr: mysqlutil.DefaultBind,
			Port:     mysqlutil.DefaultPort,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) getOrCreateMySQLRow(ctx context.Context, vpsID string) (*models.ServerMySQL, error) {
	row, err := s.getMySQLRow(ctx, vpsID)
	if err != nil {
		return nil, err
	}
	if row.CreatedAt.IsZero() && row.Status == models.MySQLStatusNotInstalled {
		now := time.Now().UTC()
		row.CreatedAt = now
		row.UpdatedAt = now
		if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
			existing, loadErr := s.getMySQLRow(ctx, vpsID)
			if loadErr == nil && !existing.CreatedAt.IsZero() {
				return existing, nil
			}
			return nil, err
		}
	}
	return row, nil
}

func (s *Service) setMySQLError(ctx context.Context, vpsID, message string) error {
	return s.db.WithContext(ctx).Model(&models.ServerMySQL{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":     models.MySQLStatusError,
		"last_error": message,
		"updated_at": time.Now().UTC(),
	}).Error
}

func (s *Service) deleteMySQLForVPS(ctx context.Context, serverID string) {
	var users []models.ServerMySQLUser
	if err := s.db.WithContext(ctx).Where("server_id = ?", serverID).Find(&users).Error; err == nil {
		for _, user := range users {
			_ = s.db.WithContext(ctx).Where("user_id = ?", user.ID).Delete(&models.ServerMySQLGrant{}).Error
		}
	}
	_ = s.db.WithContext(ctx).Where("server_id = ?", serverID).Delete(&models.ServerMySQLUser{}).Error
	_ = s.db.WithContext(ctx).Where("server_id = ?", serverID).Delete(&models.ServerMySQLDatabase{}).Error
	_ = s.db.WithContext(ctx).Where("server_id = ?", serverID).Delete(&models.ServerMySQL{}).Error
}

func mysqlViewFromRow(row *models.ServerMySQL) *MySQLView {
	view := &MySQLView{
		Status:           models.MySQLStatusNotInstalled,
		HostLocal:        mysqlutil.DefaultBind,
		PrivilegeCatalog: append([]string{}, mysqlutil.PrivilegeCatalog...),
		Databases:        []MySQLDatabaseView{},
		Users:            []MySQLUserView{},
	}
	if row == nil {
		return view
	}
	view.Status = row.Status
	view.Engine = row.Engine
	view.Version = row.Version
	view.PackageVersion = mysqlutil.PackageVersion(row.Engine, row.Version)
	view.Method = row.Method
	view.BindAddr = row.BindAddr
	view.Port = row.Port
	view.Image = row.Image
	view.LastError = row.LastError
	if row.BindAddr != "" && row.BindAddr != "0.0.0.0" {
		view.HostLocal = row.BindAddr
	}
	if row.Status != models.MySQLStatusNotInstalled {
		view.RootUser = "root"
		view.RootPassword = row.RootPassword
	}
	return view
}

func mysqlBusy(status string) bool {
	switch status {
	case models.MySQLStatusInstalling, models.MySQLStatusStarting, models.MySQLStatusStopping:
		return true
	default:
		return false
	}
}
