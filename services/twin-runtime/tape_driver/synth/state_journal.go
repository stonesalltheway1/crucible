package synth

import (
	"encoding/json"
	"strings"
	"sync"
	"time"
)

// StateJournal is the in-memory mutation journal that the synth-mutation
// branch of the tape decision tree writes to. Subsequent reads in the same
// task consult the journal BEFORE falling through to schema generation.
//
// Design point per docs/06-research/tape-coverage-strategy.md "State-mutating
// calls":
//
//   "Mutations are NEVER replayed as having had effect on real systems.
//    The mutation is written to the twin's in-memory state journal:
//      {charges: [{id: ch_synth_1, amount: 1234}]}
//    Subsequent reads consult the state journal first, then fall through
//    to the tape."
//
// Phase 3 ships a simple per-resource journal keyed by (service, resource_path).
// Path parameters (the part after the resource collection name) identify the
// resource; e.g., POST /v1/charges → /v1/charges + journal entry tagged
// `{id: ch_synth_1}`; subsequent GET /v1/charges/ch_synth_1 → journal hit.
type StateJournal struct {
	mu     sync.RWMutex
	clock  func() time.Time
	byKey  map[string]journalEntry
	idsByCollection map[string][]string // collection → ordered ids
}

type journalEntry struct {
	Body     json.RawMessage
	WrittenAt time.Time
	Method    Method
}

// NewStateJournal constructs an empty journal.
func NewStateJournal(clock func() time.Time) *StateJournal {
	if clock == nil {
		clock = time.Now
	}
	return &StateJournal{
		clock:           clock,
		byKey:           make(map[string]journalEntry),
		idsByCollection: make(map[string][]string),
	}
}

// RecordWrite stores a write-side response so subsequent reads can return it.
// For non-GET methods on resource-collection endpoints (POST /v1/charges),
// we parse the response body for an "id" field; if present, we register
// /v1/charges/{id} as a journal hit for subsequent GETs.
func (j *StateJournal) RecordWrite(req Request, body json.RawMessage) {
	j.mu.Lock()
	defer j.mu.Unlock()
	key := j.canonicalKey(req.Service, req.Endpoint, req.PathParams)
	j.byKey[key] = journalEntry{Body: body, WrittenAt: j.clock(), Method: req.Method}
	// Collection bookkeeping: POST /v1/charges → also register /v1/charges/{id}
	// for the id present in the response.
	if id := extractID(body); id != "" {
		collectionKey := strings.TrimSuffix(req.Endpoint, "/")
		j.idsByCollection[collectionKey] = append(j.idsByCollection[collectionKey], id)
		// Synthesize the canonical resource path so a GET-by-id finds it.
		resPath := collectionKey + "/" + id
		j.byKey[j.canonicalKey(req.Service, resPath, nil)] = journalEntry{
			Body:      body,
			WrittenAt: j.clock(),
			Method:    req.Method,
		}
	}
}

// LookupRead reports whether the request matches a previously-journalled
// write-side response. We match the canonical path AND any single-id
// substitution variant — e.g., a GET /v1/charges/{id} request with path
// param id=ch_synth_1 matches a write to /v1/charges that produced
// {"id": "ch_synth_1"}.
func (j *StateJournal) LookupRead(req Request) (json.RawMessage, bool) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	key := j.canonicalKey(req.Service, req.Endpoint, req.PathParams)
	if e, ok := j.byKey[key]; ok {
		return e.Body, true
	}
	// List endpoint: GET /v1/charges → return all journalled entries.
	if len(req.PathParams) == 0 {
		ids, ok := j.idsByCollection[strings.TrimSuffix(req.Endpoint, "/")]
		if ok && len(ids) > 0 {
			out := make([]json.RawMessage, 0, len(ids))
			for _, id := range ids {
				resPath := strings.TrimSuffix(req.Endpoint, "/") + "/" + id
				if e, ok := j.byKey[j.canonicalKey(req.Service, resPath, nil)]; ok {
					out = append(out, e.Body)
				}
			}
			if len(out) > 0 {
				raw, err := json.Marshal(map[string]any{"data": out, "has_more": false})
				if err == nil {
					return raw, true
				}
			}
		}
	}
	return nil, false
}

// Reset drops all entries (used between tasks).
func (j *StateJournal) Reset() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.byKey = make(map[string]journalEntry)
	j.idsByCollection = make(map[string][]string)
}

// Len returns the number of journal entries (sum of resource entries; the
// list-endpoint bookkeeping is not counted).
func (j *StateJournal) Len() int {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return len(j.byKey)
}

func (j *StateJournal) canonicalKey(service, endpoint string, params map[string]string) string {
	// We substitute path-parameter placeholders so GET /v1/charges/{id}
	// with params {id: ch_X} canonicalises to /v1/charges/ch_X.
	canon := endpoint
	for k, v := range params {
		canon = strings.ReplaceAll(canon, "{"+k+"}", v)
	}
	return strings.ToLower(service) + "|" + canon
}

// extractID looks for a top-level "id" field (the dominant pattern in the
// REST-API ecosystem we target). Returns "" when no id is present.
func extractID(body json.RawMessage) string {
	var probe struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &probe); err == nil && probe.ID != "" {
		return probe.ID
	}
	return ""
}
