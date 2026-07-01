package launcher

import (
	"context"
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

func resourceInstalledAt() string {
	return time.Now().UTC().Format(time.RFC3339)
}
