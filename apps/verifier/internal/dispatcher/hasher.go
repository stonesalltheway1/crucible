package dispatcher

import (
	"crypto/sha256"
	"encoding/hex"
)

// diffHasher is a tiny wrapper to keep dispatcher.go readable.
type diffHasher struct {
	h interface {
		Write([]byte) (int, error)
		Sum([]byte) []byte
	}
}

func newDiffHasher() *diffHasher { return &diffHasher{h: sha256.New()} }
func (h *diffHasher) WriteString(s string) { _, _ = h.h.Write([]byte(s)) }
func (h *diffHasher) WriteByte(b byte)     { _, _ = h.h.Write([]byte{b}) }
func (h *diffHasher) Hex() string          { return hex.EncodeToString(h.h.Sum(nil)) }
