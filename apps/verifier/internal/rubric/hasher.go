package rubric

import (
	"crypto/sha256"
	"encoding/hex"
)

// newHasher is a tiny façade so we can swap to BLAKE3 later without
// touching call sites.
type hasher struct {
	h interface {
		Write(p []byte) (int, error)
		Sum(b []byte) []byte
	}
}

func newHasher() *hasher {
	return &hasher{h: sha256.New()}
}

func (h *hasher) Write(p []byte) { _, _ = h.h.Write(p) }

func (h *hasher) Hex() string { return hex.EncodeToString(h.h.Sum(nil)) }
