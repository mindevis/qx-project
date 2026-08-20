package launcher

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func cloneDisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Instance"
	}
	suffix := " (copy)"
	if len(name)+len(suffix) > 128 {
		name = strings.TrimSpace(name[:128-len(suffix)])
	}
	return name + suffix
}

func cloneResourceList(list models.InstanceResourceList) models.InstanceResourceList {
	if len(list) == 0 {
		return models.InstanceResourceList{}
	}
	out := make(models.InstanceResourceList, len(list))
	copy(out, list)
	return out
}

func cloneIntPtr(v *int) *int {
	if v == nil {
		return nil
	}
	n := *v
	return &n
}

func cloneStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	s := *v
	return &s
}

func (s *Service) CloneInstance(ctx context.Context, owner Owner, instanceID string) (*CreateInstanceResult, error) {
	src, err := s.GetInstance(ctx, owner, instanceID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireLinkedDevice(ctx, owner); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	dest := models.LauncherInstance{
		ID:             uuid.NewString(),
		UserID:         src.UserID,
		GuestSessionID: src.GuestSessionID,
		Name:           cloneDisplayName(src.Name),
		MCVersion:      src.MCVersion,
		Loader:         src.Loader,
		LoaderVersion:  cloneStringPtr(src.LoaderVersion),
		MaxMemoryMB:    cloneIntPtr(src.MaxMemoryMB),
		MinMemoryMB:    cloneIntPtr(src.MinMemoryMB),
		ExtraJVMArgs:   append(models.StringList(nil), src.ExtraJVMArgs...),
		WindowWidth:    cloneIntPtr(src.WindowWidth),
		WindowHeight:   cloneIntPtr(src.WindowHeight),
		Mods:           cloneResourceList(src.Mods),
		ResourcePacks:  cloneResourceList(src.ResourcePacks),
		Shaders:        cloneResourceList(src.Shaders),
		Datapacks:      cloneResourceList(src.Datapacks),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.db.WithContext(ctx).Create(&dest).Error; err != nil {
		return nil, err
	}

	copied, err := s.cloneInstanceFiles(ctx, owner, dest.ID, src.ID)
	if err != nil {
		_ = s.DeleteInstance(context.Background(), owner, dest.ID)
		return nil, err
	}
	var prepareID *string
	if !copied {
		prepareID, err = s.enqueuePrepareForInstance(ctx, owner, dest.ID)
		if err != nil {
			_ = s.DeleteInstance(context.Background(), owner, dest.ID)
			return nil, err
		}
	}
	return &CreateInstanceResult{Instance: &dest, PrepareRequestID: prepareID}, nil
}
