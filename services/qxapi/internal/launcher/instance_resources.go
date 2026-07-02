package launcher

import (
	"context"
	"strings"
	"time"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

type InstanceResourceView struct {
	Source        string `json:"source"`
	ProjectID     string `json:"project_id,omitempty"`
	ProjectName   string `json:"project_name"`
	VersionID     string `json:"version_id,omitempty"`
	VersionNumber string `json:"version_number,omitempty"`
	Filename      string `json:"filename"`
	ResourceType  string `json:"resource_type"`
	IconURL       string `json:"icon_url,omitempty"`
	Downloads     int64  `json:"downloads,omitempty"`
	FileSize      int64  `json:"file_size,omitempty"`
	InstalledAt   string `json:"installed_at"`
	SideOverride  string `json:"side_override,omitempty"`
}

func resourceViewFromEntry(entry models.InstanceResourceEntry) InstanceResourceView {
	return InstanceResourceView{
		Source:        entry.Source,
		ProjectID:     entry.ProjectID,
		ProjectName:   entry.ProjectName,
		VersionID:     entry.VersionID,
		VersionNumber: entry.VersionNumber,
		Filename:      entry.Filename,
		ResourceType:  entry.ResourceType,
		IconURL:       entry.IconURL,
		Downloads:     entry.Downloads,
		FileSize:      entry.FileSize,
		InstalledAt:   entry.InstalledAt,
		SideOverride:  entry.SideOverride,
	}
}

func (s *Service) ListInstanceResources(ctx context.Context, owner Owner, instanceID string) ([]InstanceResourceView, error) {
	inst, err := s.GetInstance(ctx, owner, instanceID)
	if err != nil {
		return nil, err
	}
	out := make([]InstanceResourceView, 0, len(inst.Mods)+len(inst.ResourcePacks)+len(inst.Shaders)+len(inst.Datapacks))
	for _, entry := range inst.Mods {
		out = append(out, resourceViewFromEntry(entry))
	}
	for _, entry := range inst.ResourcePacks {
		out = append(out, resourceViewFromEntry(entry))
	}
	for _, entry := range inst.Shaders {
		out = append(out, resourceViewFromEntry(entry))
	}
	for _, entry := range inst.Datapacks {
		out = append(out, resourceViewFromEntry(entry))
	}
	return out, nil
}

func appendInstanceResource(inst *models.LauncherInstance, entry models.InstanceResourceEntry) {
	switch entry.ResourceType {
	case "resourcepack":
		inst.ResourcePacks = appendUniqueResource(inst.ResourcePacks, entry)
	case "shader":
		inst.Shaders = appendUniqueResource(inst.Shaders, entry)
	case "datapack":
		inst.Datapacks = appendUniqueResource(inst.Datapacks, entry)
	default:
		inst.Mods = appendUniqueResource(inst.Mods, entry)
	}
}

func appendUniqueResource(list models.InstanceResourceList, entry models.InstanceResourceEntry) models.InstanceResourceList {
	for i, existing := range list {
		if existing.Filename == entry.Filename && existing.ProjectID == entry.ProjectID {
			list[i] = entry
			return list
		}
	}
	return append(list, entry)
}

type DeleteInstanceResourceInput struct {
	Source       string
	ProjectID    string
	Filename     string
	ResourceType string
}

func resourceEntryMatches(entry models.InstanceResourceEntry, in DeleteInstanceResourceInput) bool {
	if entry.Source != in.Source {
		return false
	}
	if in.ProjectID != "" && entry.ProjectID != in.ProjectID {
		return false
	}
	if in.Filename != "" && entry.Filename != in.Filename {
		return false
	}
	return in.ProjectID != "" || in.Filename != ""
}

func removeInstanceResource(inst *models.LauncherInstance, in DeleteInstanceResourceInput) bool {
	switch normalizeResourceType(in.ResourceType) {
	case "resourcepack":
		return removeFromResourceList(&inst.ResourcePacks, in)
	case "shader":
		return removeFromResourceList(&inst.Shaders, in)
	case "datapack":
		return removeFromResourceList(&inst.Datapacks, in)
	default:
		return removeFromResourceList(&inst.Mods, in)
	}
}

func removeFromResourceList(list *models.InstanceResourceList, in DeleteInstanceResourceInput) bool {
	next := (*list)[:0]
	removed := false
	for _, entry := range *list {
		if resourceEntryMatches(entry, in) {
			removed = true
			continue
		}
		next = append(next, entry)
	}
	*list = next
	return removed
}

func (s *Service) DeleteInstanceResource(ctx context.Context, owner Owner, instanceID string, in DeleteInstanceResourceInput) error {
	if in.Source == "" || (in.ProjectID == "" && in.Filename == "") {
		return ErrValidation
	}
	inst, err := s.GetInstance(ctx, owner, instanceID)
	if err != nil {
		return err
	}
	if !removeInstanceResource(inst, in) {
		return ErrNotFound
	}
	return s.db.WithContext(ctx).Model(inst).Updates(map[string]any{
		"mods":           inst.Mods,
		"resource_packs": inst.ResourcePacks,
		"shaders":        inst.Shaders,
		"datapacks":      inst.Datapacks,
	}).Error
}

func resourceInstalledAt() string {
	return time.Now().UTC().Format(time.RFC3339)
}

type UpdateInstanceResourceSideInput struct {
	Source       string
	ProjectID    string
	Filename     string
	ResourceType string
	SideOverride string
}

func updateResourceSide(list *models.InstanceResourceList, in UpdateInstanceResourceSideInput) bool {
	match := DeleteInstanceResourceInput{
		Source:       in.Source,
		ProjectID:    in.ProjectID,
		Filename:     in.Filename,
		ResourceType: in.ResourceType,
	}
	for i, entry := range *list {
		if resourceEntryMatches(entry, match) {
			(*list)[i].SideOverride = in.SideOverride
			return true
		}
	}
	return false
}

func (s *Service) UpdateInstanceResourceSide(ctx context.Context, owner Owner, instanceID string, in UpdateInstanceResourceSideInput) error {
	side := strings.TrimSpace(in.SideOverride)
	if in.Source == "" || (in.ProjectID == "" && in.Filename == "") {
		return ErrValidation
	}
	switch side {
	case "", "client", "server", "both":
	default:
		return ErrValidation
	}
	inst, err := s.GetInstance(ctx, owner, instanceID)
	if err != nil {
		return err
	}
	updated := false
	switch normalizeResourceType(in.ResourceType) {
	case "resourcepack":
		updated = updateResourceSide(&inst.ResourcePacks, in)
	case "shader":
		updated = updateResourceSide(&inst.Shaders, in)
	case "datapack":
		updated = updateResourceSide(&inst.Datapacks, in)
	default:
		updated = updateResourceSide(&inst.Mods, in)
	}
	if !updated {
		return ErrNotFound
	}
	return s.db.WithContext(ctx).Model(inst).Updates(map[string]any{
		"mods":           inst.Mods,
		"resource_packs": inst.ResourcePacks,
		"shaders":        inst.Shaders,
		"datapacks":      inst.Datapacks,
	}).Error
}
