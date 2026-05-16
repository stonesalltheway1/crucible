package attestation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// Publisher persists a signed DSSE envelope to a transparency log and returns
// a RekorEntry receipt.
type Publisher interface {
	Publish(ctx context.Context, env *cruciblev1.DsseEnvelope) (*cruciblev1.RekorEntry, error)
	Fetch(ctx context.Context, uuid string) (*cruciblev1.DsseEnvelope, error)
}

// LocalJournalPublisher writes each envelope as a JSON line to a hash-chained
// append-only file. Each entry's hash includes the previous entry's hash,
// giving us a tamper-evident audit log that survives until Phase 2 wires the
// real Sigstore Rekor v2 client.
//
// The journal file format is one JSON object per line:
//
//	{
//	  "uuid":     "<sha256 of envelope+prev_hash>",
//	  "prev":     "<sha256 of prior entry, or 64 zeros>",
//	  "ts":       "<RFC3339 timestamp>",
//	  "index":    "<monotonic counter>",
//	  "envelope": { ... }
//	}
type LocalJournalPublisher struct {
	path string
	mu   sync.Mutex
}

// NewLocalJournalPublisher opens (creating if needed) a journal file at path.
// If path is empty, ~/.crucible/attestations/journal.jsonl is used.
func NewLocalJournalPublisher(path string) (*LocalJournalPublisher, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("attestation: locate home dir: %w", err)
		}
		path = filepath.Join(home, ".crucible", "attestations", "journal.jsonl")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("attestation: mkdir journal: %w", err)
	}
	return &LocalJournalPublisher{path: path}, nil
}

// Path returns the journal file path.
func (p *LocalJournalPublisher) Path() string { return p.path }

type journalEntry struct {
	UUID     string                  `json:"uuid"`
	Prev     string                  `json:"prev"`
	Ts       string                  `json:"ts"`
	Index    uint64                  `json:"index"`
	Envelope *cruciblev1.DsseEnvelope `json:"envelope"`
}

// Publish appends an entry to the journal and returns a RekorEntry receipt
// flagged with LocalJournalFallback=true.
func (p *LocalJournalPublisher) Publish(_ context.Context, env *cruciblev1.DsseEnvelope) (*cruciblev1.RekorEntry, error) {
	if env == nil {
		return nil, errors.New("attestation: nil envelope")
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	prev, index, err := p.tailLocked()
	if err != nil {
		return nil, err
	}

	envBytes, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("attestation: marshal envelope: %w", err)
	}

	// UUID = sha256(prev || envBytes) — content-addressed and hash-chained.
	h := sha256.New()
	h.Write([]byte(prev))
	h.Write(envBytes)
	uuid := hex.EncodeToString(h.Sum(nil))

	entry := journalEntry{
		UUID:     uuid,
		Prev:     prev,
		Ts:       Now().Format(time.RFC3339Nano),
		Index:    index + 1,
		Envelope: env,
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("attestation: marshal journal entry: %w", err)
	}

	f, err := os.OpenFile(p.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("attestation: open journal: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return nil, fmt.Errorf("attestation: append journal: %w", err)
	}

	return &cruciblev1.RekorEntry{
		UUID:                  uuid,
		LogIndex:              strconv.FormatUint(entry.Index, 10),
		LogID:                 "local-journal",
		IntegratedTime:        entry.Ts,
		URL:                   "file://" + p.path + "#" + uuid,
		LocalJournalFallback:  true,
	}, nil
}

// tailLocked returns (prevHash, lastIndex). prevHash is sixty-four zeros if
// the journal is empty.
func (p *LocalJournalPublisher) tailLocked() (string, uint64, error) {
	const zero = "0000000000000000000000000000000000000000000000000000000000000000"
	f, err := os.Open(p.path)
	if errors.Is(err, os.ErrNotExist) {
		return zero, 0, nil
	}
	if err != nil {
		return "", 0, fmt.Errorf("attestation: open journal for tail: %w", err)
	}
	defer f.Close()
	// Read the whole file (Phase 1's journals stay small; Phase 2 swaps in a
	// real backend).
	data, err := io.ReadAll(f)
	if err != nil {
		return "", 0, fmt.Errorf("attestation: read journal: %w", err)
	}
	if len(data) == 0 {
		return zero, 0, nil
	}
	var last journalEntry
	// Walk the JSONL file from the start; this is O(n) per write — fine for
	// Phase 1 demo loads. The cap (a few thousand attestations per local run)
	// is well within bounds.
	dec := json.NewDecoder(bytesReader(data))
	for dec.More() {
		var e journalEntry
		if err := dec.Decode(&e); err != nil {
			return "", 0, fmt.Errorf("attestation: decode journal: %w", err)
		}
		last = e
	}
	return last.UUID, last.Index, nil
}

// Fetch reads an envelope by UUID. Returns os.ErrNotExist if not present.
func (p *LocalJournalPublisher) Fetch(_ context.Context, uuid string) (*cruciblev1.DsseEnvelope, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	f, err := os.Open(p.path)
	if err != nil {
		return nil, fmt.Errorf("attestation: open journal: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("attestation: read journal: %w", err)
	}
	dec := json.NewDecoder(bytesReader(data))
	for dec.More() {
		var e journalEntry
		if err := dec.Decode(&e); err != nil {
			return nil, fmt.Errorf("attestation: decode journal: %w", err)
		}
		if e.UUID == uuid {
			return e.Envelope, nil
		}
	}
	return nil, os.ErrNotExist
}

// RekorV2Publisher is the Phase-2 publisher backed by the real Sigstore Rekor
// v2 client. Phase 1 stubs it out and returns a structured error if you ask
// for it without CRUCIBLE_REKOR_PUBLISH=1 *and* a wired Rekor URL.
type RekorV2Publisher struct {
	URL    string
	Local  *LocalJournalPublisher // mirror to local for offline verification
}

// NewRekorV2Publisher returns the Phase-2 publisher. As of May 2026 Sigstore
// Rekor v2 has not yet GA'd; this constructor returns an error unless the
// caller explicitly opts in via CRUCIBLE_REKOR_PUBLISH=1.
func NewRekorV2Publisher(rekorURL string, local *LocalJournalPublisher) (Publisher, error) {
	if os.Getenv("CRUCIBLE_REKOR_PUBLISH") != "1" {
		return nil, errors.New(
			"STUB: RekorV2Publisher requires CRUCIBLE_REKOR_PUBLISH=1 and a wired Rekor v2 client " +
				"(Rekor v2 had not GA'd as of May 2026 — see docs/PHASE-1-REPORT.md). " +
				"Phase 1 uses LocalJournalPublisher by default.",
		)
	}
	return nil, errors.New("STUB: RekorV2Publisher impl ships with Phase 6 (Block 6, Provenance Plumbing)")
}

// bytesReader avoids importing bytes just for one Reader.
type byteSliceReader struct {
	b []byte
	i int
}

func bytesReader(b []byte) *byteSliceReader { return &byteSliceReader{b: b} }

func (r *byteSliceReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}
