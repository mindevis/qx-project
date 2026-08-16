package launcher

import (
	"context"
	"errors"

	"github.com/qxproject/qx/pkg/mcmanifest"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"gorm.io/gorm"
)

var (
	ErrDeviceMismatch = errors.New("device not linked to owner")
)

type ManifestProvider interface {
	BuildInstanceManifest(ctx context.Context, instanceID, name, mcVersion, loader, loaderVersion, targetOS string) (*mcmanifest.InstanceLaunchManifest, error)
}

type mcManifestProvider struct {
	client *mcmanifest.Client
}

func (p *mcManifestProvider) BuildInstanceManifest(ctx context.Context, instanceID, name, mcVersion, loader, loaderVersion, targetOS string) (*mcmanifest.InstanceLaunchManifest, error) {
	return p.client.BuildInstanceManifest(ctx, instanceID, name, mcVersion, loader, loaderVersion, targetOS)
}

func defaultManifestProvider() ManifestProvider {
	return &mcManifestProvider{client: mcmanifest.NewClient()}
}

func (s *Service) SetManifestProvider(p ManifestProvider) {
	s.manifest = p
}

func (s *Service) manifestProvider() ManifestProvider {
	if s.manifest != nil {
		return s.manifest
	}
	return defaultManifestProvider()
}

func (s *Service) GetInstance(ctx context.Context, owner Owner, instanceID string) (*models.LauncherInstance, error) {
	q := s.db.WithContext(ctx).Where("id = ?", instanceID)
	q = scopeOwner(q, owner)
	var inst models.LauncherInstance
	if err := q.First(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	annotated := s.AnnotateInstancesManaged(ctx, []models.LauncherInstance{inst})
	return &annotated[0], nil
}

func (s *Service) InstanceManifest(ctx context.Context, owner Owner, instanceID string) (*mcmanifest.InstanceLaunchManifest, error) {
	return s.InstanceManifestWithMemory(ctx, owner, instanceID)
}

func applyInstanceLaunchSettings(manifest *mcmanifest.InstanceLaunchManifest, inst *models.LauncherInstance) {
	if inst.MinMemoryMB != nil {
		mcmanifest.ApplyMinMemoryMB(manifest, *inst.MinMemoryMB)
	}
	if inst.MaxMemoryMB != nil {
		mcmanifest.ApplyMaxMemoryMB(manifest, *inst.MaxMemoryMB)
	}
	if len(inst.ExtraJVMArgs) > 0 {
		mcmanifest.ApplyExtraJVMArgs(manifest, inst.ExtraJVMArgs)
	}
	mcmanifest.ApplyWindowSize(manifest, inst.WindowWidth, inst.WindowHeight)
}

func (s *Service) InstanceManifestWithMemory(ctx context.Context, owner Owner, instanceID string) (*mcmanifest.InstanceLaunchManifest, error) {
	inst, err := s.GetInstance(ctx, owner, instanceID)
	if err != nil {
		return nil, err
	}
	manifest, err := s.manifestProvider().BuildInstanceManifest(ctx, inst.ID, inst.Name, inst.MCVersion, inst.Loader, instanceLoaderVersion(*inst), "")
	if err != nil {
		return nil, err
	}
	applyInstanceLaunchSettings(manifest, inst)
	return manifest, nil
}

func (s *Service) FindLinkedDevice(ctx context.Context, owner Owner) (string, error) {
	q := s.db.WithContext(ctx).Model(&models.LauncherDevice{}).
		Where("status = ? AND user_id = ?", models.DeviceStatusLinked, owner.UserID)
	var device models.LauncherDevice
	if err := q.Order("linked_at desc").First(&device).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}
	return device.DeviceID, nil
}

func (s *Service) UserLinkedDevice(ctx context.Context, userID string) (*DeviceMeResult, error) {
	deviceID, err := s.FindLinkedDevice(ctx, Owner{UserID: userID})
	if err != nil {
		return nil, err
	}
	device, err := s.getDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	out := &DeviceMeResult{
		DeviceID:        device.DeviceID,
		Status:          device.Status,
		LauncherVersion: device.LauncherVersion,
	}
	switch {
	case device.UserID != nil:
		out.OwnerType = "user"
		out.UserID = device.UserID
	default:
		out.OwnerType = "none"
	}
	return out, nil
}

func (s *Service) ListInstancesForDevice(ctx context.Context, deviceID string) ([]models.LauncherInstance, error) {
	device, err := s.getDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if device.Status != models.DeviceStatusLinked {
		return nil, ErrDeviceNotLinked
	}
	if device.UserID == nil {
		return nil, ErrDeviceNotLinked
	}
	return s.ListInstances(ctx, Owner{UserID: *device.UserID})
}

// deliveryDeviceIDs returns every device_id that belongs to the same owner as
// the polling device. Web-created requests (launch, mod install, …) are bound to
// a single device chosen at creation time; if the owner has more than one linked
// device that request could otherwise be delivered to a device that is not
// running, hanging forever in "queued". Fetching against the whole owner set lets
// whichever launcher is actually polling claim and process the work.
func (s *Service) deliveryDeviceIDs(ctx context.Context, deviceID string) ([]string, error) {
	device, err := s.getDevice(ctx, deviceID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Unknown device (e.g. never registered): it can only claim its own work.
			return []string{deviceID}, nil
		}
		return nil, err
	}
	q := s.db.WithContext(ctx).Model(&models.LauncherDevice{}).
		Where("status = ?", models.DeviceStatusLinked)
	switch {
	case device.UserID != nil:
		q = q.Where("user_id = ?", *device.UserID)
	case device.GuestSessionID != nil:
		q = q.Where("guest_session_id = ?", *device.GuestSessionID)
	default:
		// Unlinked device: it can only claim work addressed to itself.
		return []string{deviceID}, nil
	}
	var ids []string
	if err := q.Pluck("device_id", &ids).Error; err != nil {
		return nil, err
	}
	seen := false
	for _, id := range ids {
		if id == deviceID {
			seen = true
			break
		}
	}
	if !seen {
		ids = append(ids, deviceID)
	}
	return ids, nil
}

func (s *Service) ValidateDeviceForOwner(ctx context.Context, owner Owner, deviceID string) error {
	device, err := s.getDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	if device.Status != models.DeviceStatusLinked {
		return ErrDeviceNotLinked
	}
	if device.UserID == nil || *device.UserID != owner.UserID {
		return ErrDeviceMismatch
	}
	return nil
}
