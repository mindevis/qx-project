package cosmetics

import (
	"encoding/json"
	"testing"
)

func TestBuildSessionProfile(t *testing.T) {
	gameUUID := "550e8400-e29b-41d4-a716-446655440000"
	body, err := BuildSessionProfile(ProfileTextures{
		Username:  "Steve",
		GameUUID:  gameUUID,
		SkinModel: "alex",
		SkinURL:   "http://localhost:3000/api/v1/cosmetics/skins/" + gameUUID + ".png",
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var profile map[string]any
	if err := json.Unmarshal(body, &profile); err != nil {
		t.Fatalf("json: %v", err)
	}
	if profile["id"] != CompactGameUUID(gameUUID) {
		t.Fatalf("id: %v", profile["id"])
	}
}

func TestServiceLaunchViewForGame(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()
	svc := NewService(db, Config{DataDir: dir, PublicAPIURL: "http://localhost:3000"})
	gameUUID := "550e8400-e29b-41d4-a716-446655440000"
	if _, err := svc.UploadSkin(t.Context(), "user-1", validSkinPNG(64, 64)); err != nil {
		t.Fatalf("upload: %v", err)
	}
	lv, err := svc.LaunchViewForGame(t.Context(), "user-1", gameUUID)
	if err != nil {
		t.Fatalf("launch view: %v", err)
	}
	if !lv.UseSkinServer || lv.SkinServerHost == "" {
		t.Fatalf("expected skin server enabled: %+v", lv)
	}
	if lv.SkinURL == "" || !lv.HasSkin {
		t.Fatalf("skin url: %+v", lv)
	}
}

func TestServiceCapeEquipLegacyDB(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()
	svc := NewService(db, Config{DataDir: dir, PublicAPIURL: "http://localhost:3000"})

	_, err := svc.UploadCape(t.Context(), "user-2", validSkinPNG(64, 32))
	if err != nil {
		t.Fatalf("upload cape: %v", err)
	}
	view, err := svc.Get(t.Context(), "user-2")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if view.HasSkin {
		t.Fatalf("cape upload must not expose cape in API view: %+v", view)
	}
	if _, err := svc.ReadCapePNG("user-2"); err != nil {
		t.Fatalf("cape still stored in DB/files: %v", err)
	}
}
