package cosmetics

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/qxproject/qx/pkg/safepath"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation error")
	ErrNoSkin     = errors.New("no custom skin")
	ErrNoCape     = errors.New("no custom cape")
)

type Config struct {
	DataDir            string
	PublicAPIURL       string
	SkinServerPublicURL string
}

type Service struct {
	db              *gorm.DB
	dataDir         string
	apiURL          string
	skinServerURL   string
}

type View struct {
	SkinModel string `json:"skin_model"`
	HasSkin   bool   `json:"has_skin"`
	SkinURL   string `json:"skin_url,omitempty"`
	HasCape   bool   `json:"has_cape"`
	CapeType  string `json:"cape_type,omitempty"`
	CapeURL   string `json:"cape_url,omitempty"`
	UpdatedAt string `json:"updated_at"`
}

// LaunchView extends View with fields QXLauncher needs for the skin server.
type LaunchView struct {
	View
	SkinServerHost string `json:"skin_server_host,omitempty"`
	GameUUID       string `json:"game_uuid,omitempty"`
	UseSkinServer  bool   `json:"use_skin_server,omitempty"`
}

type EquipInput struct {
	SkinModel *string
	CapeType  *string
}

func NewService(db *gorm.DB, cfg Config) *Service {
	apiURL := strings.TrimRight(strings.TrimSpace(cfg.PublicAPIURL), "/")
	skinServer := strings.TrimRight(strings.TrimSpace(cfg.SkinServerPublicURL), "/")
	if skinServer == "" {
		skinServer = apiURL
	}
	return &Service{
		db:            db,
		dataDir:       strings.TrimSpace(cfg.DataDir),
		apiURL:        apiURL,
		skinServerURL: skinServer,
	}
}

func (s *Service) SkinServerHost() string {
	return s.skinServerURL
}

func (s *Service) Get(ctx context.Context, userID string) (*View, error) {
	row, err := s.ensureRow(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.viewFromRow(row, ""), nil
}

func (s *Service) Equip(ctx context.Context, userID string, in EquipInput) (*View, error) {
	row, err := s.ensureRow(ctx, userID)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_at": time.Now().UTC()}
	if in.SkinModel != nil {
		model := normalizeSkinModel(*in.SkinModel)
		updates["skin_model"] = model
		row.SkinModel = model
	}
	if in.CapeType != nil {
		cape := normalizeCapeType(*in.CapeType)
		updates["cape_type"] = cape
		row.CapeType = cape
		switch cape {
		case models.CosmeticsCapeNone:
			updates["has_cape"] = false
			row.HasCape = false
		case models.CosmeticsCapeQX:
			updates["has_cape"] = true
			row.HasCape = true
		case models.CosmeticsCapeCustom:
			// has_cape unchanged — set by upload
		}
	}
	if err := s.db.WithContext(ctx).Model(row).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(row).Error; err != nil {
		return nil, err
	}
	return s.viewFromRow(row, ""), nil
}

func (s *Service) UploadSkin(ctx context.Context, userID string, png []byte) (*View, error) {
	if err := ValidateSkinPNG(png); err != nil {
		return nil, err
	}
	path, err := s.skinPath(userID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, png, 0o644); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	row, err := s.ensureRow(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(row).Updates(map[string]any{
		"has_skin":   true,
		"updated_at": now,
	}).Error; err != nil {
		return nil, err
	}
	row.HasSkin = true
	row.UpdatedAt = now
	return s.viewFromRow(row, ""), nil
}

func (s *Service) DeleteSkin(ctx context.Context, userID string) (*View, error) {
	path, err := s.skinPath(userID)
	if err != nil {
		return nil, err
	}
	_ = os.Remove(path)
	now := time.Now().UTC()
	row, err := s.ensureRow(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(row).Updates(map[string]any{
		"has_skin":   false,
		"updated_at": now,
	}).Error; err != nil {
		return nil, err
	}
	row.HasSkin = false
	row.UpdatedAt = now
	return s.viewFromRow(row, ""), nil
}

func (s *Service) UploadCape(ctx context.Context, userID string, png []byte) (*View, error) {
	if err := ValidateSkinPNG(png); err != nil {
		return nil, err
	}
	path, err := s.capePath(userID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, png, 0o644); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	row, err := s.ensureRow(ctx, userID)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"has_cape":   true,
		"cape_type":  models.CosmeticsCapeCustom,
		"updated_at": now,
	}
	if err := s.db.WithContext(ctx).Model(row).Updates(updates).Error; err != nil {
		return nil, err
	}
	row.HasCape = true
	row.CapeType = models.CosmeticsCapeCustom
	row.UpdatedAt = now
	return s.viewFromRow(row, ""), nil
}

func (s *Service) DeleteCape(ctx context.Context, userID string) (*View, error) {
	path, err := s.capePath(userID)
	if err != nil {
		return nil, err
	}
	_ = os.Remove(path)
	now := time.Now().UTC()
	row, err := s.ensureRow(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(row).Updates(map[string]any{
		"has_cape":   false,
		"cape_type":  models.CosmeticsCapeNone,
		"updated_at": now,
	}).Error; err != nil {
		return nil, err
	}
	row.HasCape = false
	row.CapeType = models.CosmeticsCapeNone
	row.UpdatedAt = now
	return s.viewFromRow(row, ""), nil
}

func (s *Service) ReadSkinPNG(userID string) ([]byte, error) {
	path, err := s.skinPath(userID)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSkin
		}
		return nil, err
	}
	return data, nil
}

func (s *Service) ReadCapePNG(userID string) ([]byte, error) {
	row, err := s.readRowByUserID(userID)
	if err != nil {
		return nil, err
	}
	return s.readCapeForRow(row)
}

func (s *Service) ReadSkinPNGForGameUUID(ctx context.Context, gameUUID string) ([]byte, error) {
	userID, err := s.ResolveUserByGameUUID(ctx, gameUUID)
	if err != nil {
		return nil, err
	}
	return s.ReadSkinPNG(userID)
}

func (s *Service) ReadCapePNGForGameUUID(ctx context.Context, gameUUID string) ([]byte, error) {
	userID, err := s.ResolveUserByGameUUID(ctx, gameUUID)
	if err != nil {
		return nil, err
	}
	row, err := s.readRowByUserID(userID)
	if err != nil {
		return nil, err
	}
	return s.readCapeForRow(row)
}

func (s *Service) readCapeForRow(row *models.UserCosmetics) ([]byte, error) {
	if !row.HasCape {
		return nil, ErrNoCape
	}
	switch row.CapeType {
	case models.CosmeticsCapeQX:
		if data := DefaultQXCapePNG(); len(data) > 0 {
			return data, nil
		}
		return nil, ErrNoCape
	case models.CosmeticsCapeCustom:
		path, err := s.capePath(row.UserID)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, ErrNoCape
			}
			return nil, err
		}
		return data, nil
	default:
		return nil, ErrNoCape
	}
}

func (s *Service) LaunchView(ctx context.Context, userID string) (*View, error) {
	if userID == "" {
		return nil, nil
	}
	return s.Get(ctx, userID)
}

func (s *Service) LaunchViewForGame(ctx context.Context, userID, gameUUID string) (*LaunchView, error) {
	if userID == "" {
		return nil, nil
	}
	row, err := s.ensureRow(ctx, userID)
	if err != nil {
		return nil, err
	}
	gameUUID = NormalizeGameUUID(gameUUID)
	view := s.viewFromRow(row, gameUUID)
	lv := &LaunchView{View: *view, GameUUID: gameUUID}
	if gameUUID != "" && s.hasEquippedCosmetics(row) {
		lv.UseSkinServer = true
		lv.SkinServerHost = s.skinServerURL
	}
	return lv, nil
}

func (s *Service) SessionProfile(ctx context.Context, gameUUID, username string) ([]byte, error) {
	userID, err := s.ResolveUserByGameUUID(ctx, gameUUID)
	if err != nil {
		return nil, err
	}
	row, err := s.ensureRow(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !s.hasEquippedCosmetics(row) {
		return nil, ErrNotFound
	}
	gameUUID = NormalizeGameUUID(gameUUID)
	if strings.TrimSpace(username) == "" {
		username = s.profileUsername(ctx, userID, gameUUID)
	}
	tex := ProfileTextures{
		Username:  username,
		GameUUID:  gameUUID,
		SkinModel: normalizeSkinModel(row.SkinModel),
	}
	if row.HasSkin {
		tex.SkinURL = s.skinTextureURL(gameUUID)
	}
	if row.HasCape {
		tex.CapeURL = s.capeTextureURL(gameUUID)
	}
	return BuildSessionProfile(tex)
}

func (s *Service) profileUsername(ctx context.Context, userID, gameUUID string) string {
	var link models.MojangLink
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&link).Error; err == nil {
		return link.Username
	}
	var profile models.OfflineProfile
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND offline_uuid = ?", userID, gameUUID).
		First(&profile).Error; err == nil {
		return profile.Username
	}
	return "Player"
}

func (s *Service) ResolveUserByGameUUID(ctx context.Context, gameUUID string) (string, error) {
	normalized := NormalizeGameUUID(gameUUID)
	if normalized == "" {
		return "", ErrNotFound
	}
	var link models.MojangLink
	err := s.db.WithContext(ctx).Where("minecraft_uuid = ?", normalized).First(&link).Error
	if err == nil {
		return link.UserID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	var profile models.OfflineProfile
	err = s.db.WithContext(ctx).Where("offline_uuid = ?", normalized).First(&profile).Error
	if err == nil && profile.UserID != nil {
		return *profile.UserID, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrNotFound
	}
	return "", err
}

func (s *Service) ResolveTextureOwner(ctx context.Context, id string) (string, error) {
	id = strings.TrimSuffix(strings.TrimSpace(id), ".png")
	if id == "" {
		return "", ErrNotFound
	}
	if userID, err := s.ResolveUserByGameUUID(ctx, id); err == nil {
		return userID, nil
	} else if !errors.Is(err, ErrNotFound) {
		return "", err
	}
	// Web preview URLs may still use QX user_id.
	if _, err := s.readRowByUserID(id); err == nil {
		return id, nil
	}
	return "", ErrNotFound
}

func (s *Service) SkinDomains() []string {
	host := strings.TrimPrefix(strings.TrimPrefix(s.skinServerURL, "https://"), "http://")
	if host == "" {
		return nil
	}
	if i := strings.Index(host, "/"); i >= 0 {
		host = host[:i]
	}
	if host == "" {
		return nil
	}
	return []string{host}
}

func (s *Service) ensureRow(ctx context.Context, userID string) (*models.UserCosmetics, error) {
	if userID == "" {
		return nil, ErrValidation
	}
	var row models.UserCosmetics
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		now := time.Now().UTC()
		row = models.UserCosmetics{
			UserID:    userID,
			SkinModel: models.CosmeticsSkinModelSteve,
			CapeType:  models.CosmeticsCapeNone,
			WingsType: models.CosmeticsWingsNone,
			UpdatedAt: now,
		}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
		return &row, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) readRowByUserID(userID string) (*models.UserCosmetics, error) {
	var row models.UserCosmetics
	err := s.db.Where("user_id = ?", userID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) viewFromRow(row *models.UserCosmetics, gameUUID string) *View {
	v := &View{
		SkinModel: normalizeSkinModel(row.SkinModel),
		HasSkin:   row.HasSkin,
		HasCape:   row.HasCape,
		CapeType:  row.CapeType,
		UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if s.apiURL == "" {
		return v
	}
	if row.HasSkin {
		if gameUUID != "" {
			v.SkinURL = s.skinTextureURL(gameUUID)
		} else {
			v.SkinURL = s.apiURL + "/api/v1/cosmetics/skins/" + row.UserID + ".png"
		}
	}
	if row.HasCape {
		if gameUUID != "" {
			v.CapeURL = s.capeTextureURL(gameUUID)
		} else {
			v.CapeURL = s.apiURL + "/api/v1/cosmetics/capes/" + row.UserID + ".png"
		}
	}
	return v
}

func (s *Service) hasEquippedCosmetics(row *models.UserCosmetics) bool {
	return row.HasSkin || row.HasCape
}

func (s *Service) skinTextureURL(gameUUID string) string {
	if s.apiURL == "" || gameUUID == "" {
		return ""
	}
	return s.apiURL + "/api/v1/cosmetics/skins/" + gameUUID + ".png"
}

func (s *Service) capeTextureURL(gameUUID string) string {
	if s.apiURL == "" || gameUUID == "" {
		return ""
	}
	return s.apiURL + "/api/v1/cosmetics/capes/" + gameUUID + ".png"
}

func (s *Service) skinPath(userID string) (string, error) {
	return s.assetPath("skins", userID+".png")
}

func (s *Service) capePath(userID string) (string, error) {
	return s.assetPath("capes", userID+".png")
}

func (s *Service) assetPath(subdir, filename string) (string, error) {
	root, err := safepath.ResolveRoot(s.dataDir)
	if err != nil {
		return "", err
	}
	return safepath.Join(root, subdir, filename)
}

func normalizeSkinModel(model string) string {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case models.CosmeticsSkinModelAlex:
		return models.CosmeticsSkinModelAlex
	default:
		return models.CosmeticsSkinModelSteve
	}
}

func normalizeCapeType(cape string) string {
	switch strings.ToLower(strings.TrimSpace(cape)) {
	case models.CosmeticsCapeQX:
		return models.CosmeticsCapeQX
	case models.CosmeticsCapeCustom:
		return models.CosmeticsCapeCustom
	default:
		return models.CosmeticsCapeNone
	}
}

func normalizeWingsType(wings string) string {
	switch strings.ToLower(strings.TrimSpace(wings)) {
	case models.CosmeticsWingsAngel:
		return models.CosmeticsWingsAngel
	case models.CosmeticsWingsDemon:
		return models.CosmeticsWingsDemon
	case models.CosmeticsWingsDragon:
		return models.CosmeticsWingsDragon
	default:
		return models.CosmeticsWingsNone
	}
}
