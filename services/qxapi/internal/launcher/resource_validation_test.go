package launcher

import "testing"

func TestValidateResourceFilename(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr bool
	}{
		{name: "jar ok", file: "mod.jar", wantErr: false},
		{name: "apostrophe ok", file: "Mowzie's Mobs.jar", wantErr: false},
		{name: "zip ok", file: "pack.zip", wantErr: false},
		{name: "mrpack ok", file: "pack.mrpack", wantErr: false},
		{name: "bad ext", file: "readme.txt", wantErr: true},
		{name: "traversal", file: "../evil.jar", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResourceFilename(tt.file)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateResourceFilename(%q) err=%v wantErr=%v", tt.file, err, tt.wantErr)
			}
		})
	}
}

func TestValidateResourceUploadSize(t *testing.T) {
	if err := ValidateResourceUploadSize(1024); err != nil {
		t.Fatalf("expected ok size: %v", err)
	}
	if err := ValidateResourceUploadSize(MaxResourceUploadBytes + 1); err == nil {
		t.Fatal("expected oversize error")
	}
}
