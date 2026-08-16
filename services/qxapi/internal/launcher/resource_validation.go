package launcher

import (
	"path/filepath"
	"strings"

	"github.com/qxproject/qx/pkg/protocol"
)

const MaxResourceUploadBytes = protocol.MaxContentFileBytes

var allowedResourceExtensions = map[string]struct{}{
	".jar":  {},
	".zip":  {},
	".mrpack": {},
}

func ValidateResourceFilename(filename string) error {
	filename = strings.TrimSpace(filename)
	if filename == "" || strings.Contains(filename, "..") || strings.ContainsAny(filename, `/\`) {
		return ErrValidation
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if _, ok := allowedResourceExtensions[ext]; !ok {
		return ErrValidation
	}
	return nil
}

func ValidateResourceUploadSize(size int64) error {
	if size <= 0 || size > MaxResourceUploadBytes {
		return ErrValidation
	}
	return nil
}
