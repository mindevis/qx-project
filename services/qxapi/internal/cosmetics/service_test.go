package cosmetics

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func validSkinPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestValidateSkinPNG(t *testing.T) {
	if err := ValidateSkinPNG(validSkinPNG(64, 64)); err != nil {
		t.Fatalf("64x64: %v", err)
	}
	if err := ValidateSkinPNG(validSkinPNG(64, 32)); err != nil {
		t.Fatalf("64x32: %v", err)
	}
	if err := ValidateSkinPNG([]byte("not png")); err == nil {
		t.Fatal("expected invalid format")
	}
	if err := ValidateSkinPNG(validSkinPNG(32, 32)); err == nil {
		t.Fatal("expected invalid size")
	}
}

func TestServiceCRUD(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()
	svc := NewService(db, Config{DataDir: dir, PublicAPIURL: "http://localhost:3000"})

	view, err := svc.UploadSkin(t.Context(), "user-1", validSkinPNG(64, 64))
	if err != nil {
		t.Fatalf("upload skin: %v", err)
	}
	if !view.HasSkin || view.SkinURL == "" {
		t.Fatalf("unexpected view: %+v", view)
	}

	view, err = svc.Equip(t.Context(), "user-1", EquipInput{
		SkinModel: strPtr("alex"),
	})
	if err != nil {
		t.Fatalf("equip: %v", err)
	}
	if view.SkinModel != "alex" {
		t.Fatalf("equip mismatch: %+v", view)
	}

	if _, err := svc.ReadSkinPNG("user-1"); err != nil {
		t.Fatalf("read skin: %v", err)
	}

	view, err = svc.DeleteSkin(t.Context(), "user-1")
	if err != nil || view.HasSkin {
		t.Fatalf("delete skin: %+v %v", view, err)
	}
}

func TestServiceCapeUploadLegacy(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()
	svc := NewService(db, Config{DataDir: dir, PublicAPIURL: "http://localhost:3000"})

	view, err := svc.UploadCape(t.Context(), "user-2", validSkinPNG(64, 32))
	if err != nil {
		t.Fatalf("upload cape: %v", err)
	}
	if view.HasSkin && view.SkinURL != "" {
		t.Fatalf("cape upload must not expose cape in view: %+v", view)
	}
	if _, err := svc.ReadCapePNG("user-2"); err != nil {
		t.Fatalf("cape file still stored: %v", err)
	}
}

func strPtr(s string) *string { return &s }
