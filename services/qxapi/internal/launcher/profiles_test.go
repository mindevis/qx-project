package launcher

import (
	"context"
	"errors"
	"testing"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func TestCreateProfileModel(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()
	owner := Owner{UserID: "user-1"}

	profile, err := svc.CreateProfile(ctx, owner, CreateProfileInput{
		Username: "Alexa",
		Model:    models.ProfileModelAlex,
	})
	if err != nil {
		t.Fatalf("create alex: %v", err)
	}
	if profile.Model != models.ProfileModelAlex {
		t.Fatalf("model: %q", profile.Model)
	}

	profile, err = svc.CreateProfile(ctx, owner, CreateProfileInput{Username: "Steve"})
	if err != nil {
		t.Fatalf("create default: %v", err)
	}
	if profile.Model != models.ProfileModelSteve {
		t.Fatalf("default model: %q", profile.Model)
	}

	_, err = svc.CreateProfile(ctx, owner, CreateProfileInput{
		Username: "BadModel",
		Model:    "herobrine",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation, got %v", err)
	}
}
