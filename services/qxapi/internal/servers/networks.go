package servers

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/qxproject/qx/pkg/mcproxy"
	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

type GameServerNetworkMemberInput struct {
	GameServerID string `json:"game_server_id"`
	Role         string `json:"role"`
	Alias        string `json:"alias"`
	SortOrder    int    `json:"sort_order"`
}

type GameServerNetworkMemberView struct {
	ID           string `json:"id"`
	GameServerID string `json:"game_server_id"`
	Role         string `json:"role"`
	Alias        string `json:"alias"`
	SortOrder    int    `json:"sort_order"`
	Name         string `json:"name"`
	ServerType   string `json:"server_type"`
	Port         int    `json:"port"`
	Address      string `json:"address,omitempty"`
	Status       string `json:"status"`
}

type GameServerNetworkView struct {
	ID         string                        `json:"id"`
	Name       string                        `json:"name"`
	Members    []GameServerNetworkMemberView `json:"members"`
	Applied    bool                          `json:"applied"`
	ApplyError string                        `json:"apply_error,omitempty"`
	CreatedAt  time.Time                     `json:"created_at"`
	UpdatedAt  time.Time                     `json:"updated_at"`
}

func isProxyGameServerType(serverType string) bool {
	return strings.EqualFold(strings.TrimSpace(serverType), "velocity")
}

func (s *Service) ListGameServerNetworks(ctx context.Context, ownerID, vpsID string) ([]GameServerNetworkView, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	var networks []models.GameServerNetwork
	if err := s.db.WithContext(ctx).Where("server_id = ?", vpsID).Order("created_at ASC").Find(&networks).Error; err != nil {
		return nil, err
	}
	out := make([]GameServerNetworkView, 0, len(networks))
	for i := range networks {
		view, err := s.networkView(ctx, &networks[i])
		if err != nil {
			return nil, err
		}
		out = append(out, view)
	}
	return out, nil
}

func (s *Service) CreateGameServerNetwork(ctx context.Context, ownerID, vpsID, name string) (*GameServerNetworkView, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrValidation
	}
	secret, err := mcproxy.GenerateForwardingSecret()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	item := models.GameServerNetwork{
		ID:               uuid.NewString(),
		ServerID:         vpsID,
		Name:             name,
		ForwardingSecret: secret,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.db.WithContext(ctx).Create(&item).Error; err != nil {
		return nil, err
	}
	view := GameServerNetworkView{
		ID:        item.ID,
		Name:      item.Name,
		Members:   []GameServerNetworkMemberView{},
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
	return &view, nil
}

func (s *Service) UpdateGameServerNetwork(
	ctx context.Context,
	ownerID, vpsID, networkID, name string,
	members []GameServerNetworkMemberInput,
	apply bool,
) (*GameServerNetworkView, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	network, err := s.getOwnedNetwork(ctx, vpsID, networkID)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = network.Name
	}
	normalized, err := s.normalizeNetworkMembers(ctx, vpsID, networkID, members)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.GameServerNetwork{}).Where("id = ?", network.ID).Updates(map[string]any{
			"name":       name,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("network_id = ?", network.ID).Delete(&models.GameServerNetworkMember{}).Error; err != nil {
			return err
		}
		if len(normalized) == 0 {
			return nil
		}
		return tx.Create(&normalized).Error
	})
	if err != nil {
		return nil, err
	}
	network.Name = name
	network.UpdatedAt = now
	view, err := s.networkView(ctx, network)
	if err != nil {
		return nil, err
	}
	if apply {
		if applyErr := s.applyGameServerNetwork(ctx, ownerID, vpsID, network, view.Members); applyErr != nil {
			view.ApplyError = applyErr.Error()
		} else {
			view.Applied = true
		}
	}
	return &view, nil
}

func (s *Service) ApplyGameServerNetwork(ctx context.Context, ownerID, vpsID, networkID string) (*GameServerNetworkView, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	network, err := s.getOwnedNetwork(ctx, vpsID, networkID)
	if err != nil {
		return nil, err
	}
	view, err := s.networkView(ctx, network)
	if err != nil {
		return nil, err
	}
	if err := s.applyGameServerNetwork(ctx, ownerID, vpsID, network, view.Members); err != nil {
		view.ApplyError = err.Error()
		return &view, nil
	}
	view.Applied = true
	return &view, nil
}

func (s *Service) DeleteGameServerNetwork(ctx context.Context, ownerID, vpsID, networkID string) error {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return err
	}
	network, err := s.getOwnedNetwork(ctx, vpsID, networkID)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("network_id = ?", network.ID).Delete(&models.GameServerNetworkMember{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.GameServerNetwork{}, "id = ?", network.ID).Error
	})
}

func (s *Service) syncNetworksForGameServer(ctx context.Context, vpsID, gameServerID string) {
	var member models.GameServerNetworkMember
	if err := s.db.WithContext(ctx).Where("game_server_id = ?", gameServerID).First(&member).Error; err != nil {
		return
	}
	var network models.GameServerNetwork
	if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", member.NetworkID, vpsID).First(&network).Error; err != nil {
		return
	}
	view, err := s.networkView(ctx, &network)
	if err != nil {
		return
	}
	_ = s.applyGameServerNetwork(ctx, "", vpsID, &network, view.Members)
}

func (s *Service) getOwnedNetwork(ctx context.Context, vpsID, networkID string) (*models.GameServerNetwork, error) {
	var network models.GameServerNetwork
	if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", networkID, vpsID).First(&network).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &network, nil
}

func (s *Service) networkView(ctx context.Context, network *models.GameServerNetwork) (GameServerNetworkView, error) {
	var members []models.GameServerNetworkMember
	if err := s.db.WithContext(ctx).Where("network_id = ?", network.ID).Order("sort_order ASC, created_at ASC").Find(&members).Error; err != nil {
		return GameServerNetworkView{}, err
	}
	ids := make([]string, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.GameServerID)
	}
	games := map[string]models.GameServer{}
	if len(ids) > 0 {
		var rows []models.GameServer
		if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
			return GameServerNetworkView{}, err
		}
		for _, row := range rows {
			games[row.ID] = row
		}
	}
	outMembers := make([]GameServerNetworkMemberView, 0, len(members))
	for _, member := range members {
		game := games[member.GameServerID]
		address := ""
		if game.Address != nil {
			address = *game.Address
		}
		outMembers = append(outMembers, GameServerNetworkMemberView{
			ID:           member.ID,
			GameServerID: member.GameServerID,
			Role:         member.Role,
			Alias:        member.Alias,
			SortOrder:    member.SortOrder,
			Name:         game.Name,
			ServerType:   game.ServerType,
			Port:         game.Port,
			Address:      address,
			Status:       game.Status,
		})
	}
	return GameServerNetworkView{
		ID:        network.ID,
		Name:      network.Name,
		Members:   outMembers,
		CreatedAt: network.CreatedAt,
		UpdatedAt: network.UpdatedAt,
	}, nil
}

func (s *Service) normalizeNetworkMembers(
	ctx context.Context,
	vpsID, networkID string,
	in []GameServerNetworkMemberInput,
) ([]models.GameServerNetworkMember, error) {
	if len(in) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(in))
	seenServers := map[string]struct{}{}
	seenAliases := map[string]struct{}{}
	proxyCount := 0
	lobbyCount := 0
	now := time.Now().UTC()
	out := make([]models.GameServerNetworkMember, 0, len(in))
	for i, item := range in {
		gameServerID := strings.TrimSpace(item.GameServerID)
		if gameServerID == "" {
			return nil, ErrValidation
		}
		if _, ok := seenServers[gameServerID]; ok {
			return nil, ErrValidation
		}
		seenServers[gameServerID] = struct{}{}
		ids = append(ids, gameServerID)
		role := strings.ToLower(strings.TrimSpace(item.Role))
		switch role {
		case models.GameServerNetworkRoleProxy:
			proxyCount++
		case models.GameServerNetworkRoleLobby:
			lobbyCount++
		case models.GameServerNetworkRoleBackend:
		default:
			return nil, ErrValidation
		}
		alias := strings.ToLower(strings.TrimSpace(item.Alias))
		if alias == "" {
			alias = mcproxy.AliasFromName(gameServerID)
		}
		if !mcproxy.ValidAlias(alias) {
			return nil, ErrValidation
		}
		if _, ok := seenAliases[alias]; ok {
			return nil, ErrValidation
		}
		seenAliases[alias] = struct{}{}
		sortOrder := item.SortOrder
		if sortOrder == 0 {
			sortOrder = i
		}
		out = append(out, models.GameServerNetworkMember{
			ID:           uuid.NewString(),
			NetworkID:    networkID,
			GameServerID: gameServerID,
			Role:         role,
			Alias:        alias,
			SortOrder:    sortOrder,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}
	if proxyCount > 1 || lobbyCount > 1 {
		return nil, ErrValidation
	}
	var games []models.GameServer
	if err := s.db.WithContext(ctx).Where("id IN ? AND server_id = ?", ids, vpsID).Find(&games).Error; err != nil {
		return nil, err
	}
	if len(games) != len(ids) {
		return nil, ErrValidation
	}
	byID := map[string]models.GameServer{}
	for _, game := range games {
		byID[game.ID] = game
	}
	var other []models.GameServerNetworkMember
	if err := s.db.WithContext(ctx).Where("game_server_id IN ? AND network_id <> ?", ids, networkID).Find(&other).Error; err != nil {
		return nil, err
	}
	if len(other) > 0 {
		return nil, ErrValidation
	}
	for i := range out {
		game := byID[out[i].GameServerID]
		if out[i].Role == models.GameServerNetworkRoleProxy && !isProxyGameServerType(game.ServerType) {
			return nil, ErrValidation
		}
		if out[i].Role != models.GameServerNetworkRoleProxy && isProxyGameServerType(game.ServerType) {
			out[i].Role = models.GameServerNetworkRoleProxy
		}
		if out[i].Alias == mcproxy.AliasFromName(out[i].GameServerID) {
			out[i].Alias = uniqueAlias(mcproxy.AliasFromName(game.Name), seenAliases, out[i].Alias)
		}
	}
	proxyRecount := 0
	for _, member := range out {
		if member.Role == models.GameServerNetworkRoleProxy {
			proxyRecount++
		}
	}
	if proxyRecount > 1 {
		return nil, ErrValidation
	}
	return out, nil
}

func uniqueAlias(base string, seen map[string]struct{}, previous string) string {
	delete(seen, previous)
	alias := base
	if alias == "" || !mcproxy.ValidAlias(alias) {
		alias = "server"
	}
	if _, ok := seen[alias]; !ok {
		seen[alias] = struct{}{}
		return alias
	}
	for n := 2; n < 100; n++ {
		next := alias + "-" + strconv.Itoa(n)
		if _, ok := seen[next]; !ok {
			seen[next] = struct{}{}
			return next
		}
	}
	seen[alias] = struct{}{}
	return alias
}

func (s *Service) applyGameServerNetwork(
	ctx context.Context,
	ownerID, vpsID string,
	network *models.GameServerNetwork,
	members []GameServerNetworkMemberView,
) error {
	if s.hub == nil || !s.hub.IsOnline(vpsID) {
		return ErrAgentOffline
	}
	var proxy *GameServerNetworkMemberView
	var backends []GameServerNetworkMemberView
	var tryList []string
	for i := range members {
		member := members[i]
		if member.Role == models.GameServerNetworkRoleProxy {
			proxy = &members[i]
			continue
		}
		backends = append(backends, member)
		if member.Role == models.GameServerNetworkRoleLobby {
			tryList = append([]string{member.Alias}, tryList...)
		}
	}
	if proxy == nil {
		return nil
	}
	if len(tryList) == 0 && len(backends) > 0 {
		sorted := append([]GameServerNetworkMemberView{}, backends...)
		sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].SortOrder < sorted[j].SortOrder })
		tryList = []string{sorted[0].Alias}
	}
	var proxyRow models.GameServer
	if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", proxy.GameServerID, vpsID).First(&proxyRow).Error; err != nil {
		return err
	}
	if !gameServerIsInstalled(&proxyRow) {
		return ErrGameServerNotInstalled
	}
	tomlBackends := make([]mcproxy.Backend, 0, len(backends))
	for _, backend := range backends {
		tomlBackends = append(tomlBackends, mcproxy.Backend{
			Alias:   backend.Alias,
			Address: networkBackendAddress(backend),
		})
	}
	bindHost := "0.0.0.0"
	if ip := parsePublicIP(proxy.Address); ip != "" {
		bindHost = ip
	}
	port := proxy.Port
	if port <= 0 {
		port = 25565
	}
	toml := mcproxy.VelocityToml(bindHost+":"+strconv.Itoa(port), strings.TrimSpace(proxy.Name), tomlBackends, tryList)
	if err := s.writeInstalledGameServerFile(ctx, vpsID, &proxyRow, "velocity.toml", toml); err != nil {
		return err
	}
	if err := s.writeInstalledGameServerFile(ctx, vpsID, &proxyRow, "forwarding.secret", network.ForwardingSecret+"\n"); err != nil {
		return err
	}
	_ = ownerID
	var applyErr error
	for _, backend := range backends {
		var row models.GameServer
		if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", backend.GameServerID, vpsID).First(&row).Error; err != nil {
			continue
		}
		if !gameServerIsInstalled(&row) {
			continue
		}
		if err := s.patchInstalledGameServerProperties(ctx, vpsID, &row, map[string]string{"online-mode": "false"}); err != nil {
			applyErr = err
		}
		if !supportsPaperVelocityForwarding(row.ServerType) {
			continue
		}
		existing := ""
		if file, err := s.readInstalledGameServerFile(ctx, vpsID, &row, "config/paper-global.yml"); err == nil {
			existing = file
		}
		yaml := mcproxy.PatchPaperVelocityForwarding(existing, network.ForwardingSecret)
		if err := s.writeInstalledGameServerFile(ctx, vpsID, &row, "config/paper-global.yml", yaml); err != nil {
			applyErr = err
		}
	}
	return applyErr
}

func networkBackendAddress(member GameServerNetworkMemberView) string {
	host := "127.0.0.1"
	if ip := parsePublicIP(member.Address); ip != "" && !isLoopbackIP(ip) {
		host = ip
	}
	port := member.Port
	if port <= 0 {
		port = 25565
	}
	return host + ":" + strconv.Itoa(port)
}

func parsePublicIP(address string) string {
	address = strings.TrimSpace(address)
	if address == "" {
		return ""
	}
	if ip := net.ParseIP(address); ip != nil {
		return ip.String()
	}
	return ""
}

func isLoopbackIP(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.IsLoopback()
}

func supportsPaperVelocityForwarding(serverType string) bool {
	switch strings.ToLower(strings.TrimSpace(serverType)) {
	case "paper", "purpur":
		return true
	default:
		return false
	}
}

func (s *Service) writeInstalledGameServerFile(ctx context.Context, vpsID string, item *models.GameServer, path, content string) error {
	payload, err := json.Marshal(protocol.ServerFilesWritePayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
		Path:         path,
		Content:      content,
	})
	if err != nil {
		return err
	}
	_, err = s.agentRPC(ctx, vpsID, protocol.TypeCmdServerFilesWrite, protocol.TypeResServerFilesWrite, payload)
	return err
}

func (s *Service) readInstalledGameServerFile(ctx context.Context, vpsID string, item *models.GameServer, path string) (string, error) {
	payload, err := json.Marshal(protocol.ServerFilesPathPayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
		Path:         path,
	})
	if err != nil {
		return "", err
	}
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdServerFilesRead, protocol.TypeResServerFilesRead, payload)
	if err != nil {
		return "", err
	}
	var result protocol.ServerFilesReadResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	return result.Content, nil
}

func (s *Service) patchInstalledGameServerProperties(ctx context.Context, vpsID string, item *models.GameServer, updates map[string]string) error {
	payload, err := json.Marshal(protocol.ServerPropertiesPatchPayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
		Updates:      updates,
	})
	if err != nil {
		return err
	}
	_, err = s.agentRPC(ctx, vpsID, protocol.TypeCmdServerPropertiesPatch, protocol.TypeResServerPropertiesPatch, payload)
	return err
}
