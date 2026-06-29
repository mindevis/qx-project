package cosmetics

import _ "embed"

//go:embed assets/qx-cape.png
var defaultQXCapePNG []byte

// DefaultQXCapePNG returns the built-in QX cape texture (64×32 PNG).
func DefaultQXCapePNG() []byte {
	if len(defaultQXCapePNG) == 0 {
		return nil
	}
	out := make([]byte, len(defaultQXCapePNG))
	copy(out, defaultQXCapePNG)
	return out
}
