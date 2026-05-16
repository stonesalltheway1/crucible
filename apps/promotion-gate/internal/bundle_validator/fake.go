package bundle_validator

import (
	"context"
	"errors"
	"sync"
)

// FakeVerifier is the in-memory test double. Test code seeds it with
// (uuid → FetchedStatement) pairs.
type FakeVerifier struct {
	mu   sync.Mutex
	data map[string]*FetchedStatement
}

// NewFakeVerifier constructs an empty FakeVerifier.
func NewFakeVerifier() *FakeVerifier {
	return &FakeVerifier{data: map[string]*FetchedStatement{}}
}

// Put inserts an entry.
func (f *FakeVerifier) Put(s *FetchedStatement) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[s.UUID] = s
}

// FetchStatement implements Verifier.
func (f *FakeVerifier) FetchStatement(_ context.Context, uuid string) (*FetchedStatement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.data[uuid]; ok {
		return s, nil
	}
	return nil, errors.New("fake: uuid not found: " + uuid)
}
