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
	BuildInstanceManifest(ctx context.Context, instanceID, name, mcVersion, loader, loaderVersion string) (*mcmanifest.InstanceLaunchManifest, error)
}

type mcManifestProvider struct {
	client *mcmanifest.Client
}

func (p *mcManifestProvider) BuildInstanceManifest(ctx context.Context, instanceID, name, mcVersion, loader, loaderVersion string) (*mcmanifest.InstanceLaunchManifest, error) {
	return p.client.BuildInstanceManifest(ctx, instanceID, name, mcVersion, loader, loaderVersion)
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
	return &inst, nil
}

func (s *Service) InstanceManifest(ctx context.Context, owner Owner, instanceID string) (*mcmanifest.InstanceLaunchManifest, error) {
	inst, err := s.GetInstance(ctx, owner, instanceID)
	if err != nil {
		return nil, err
	}
	return s.manifestProvider().BuildInstanceManifest(ctx, inst.ID, inst.Name, inst.MCVersion, inst.Loader, instanceLoaderVersion(*inst))
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
