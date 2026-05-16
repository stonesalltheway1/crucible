// Package synth handles the "miss-in-spec" branches of the tape decision
// tree:
//
//   3. Miss, in OpenAPI, READ-ONLY  → SYNTHESIZE (Faker / Microcks-style)
//   4. Miss, in OpenAPI, MUTATING   → DETERMINISTIC STUB + state-journal entry
//
// Both produce a candidate tape entry tagged
// `X-Crucible-Tape: synth-readonly` or `synth-mutation`. Candidate tapes are
// NOT auto-promoted; the operator reviews them via the shadow recorder
// dashboard before they become first-class.
//
// Phase 3 currency check picks (May 2026):
//   - Microcks 1.11.1 AI Copilot pattern (OpenAPI + LLM-augmented Faker) is
//     the primary synthesis recipe. We embed the pattern, not the engine,
//     so we don't fork the project.
//   - Stoplight Prism 5.x is the deterministic-example fallback when the
//     LLM is unavailable.
//   - Faker (Python) / @faker-js/faker — field-level filler called below
//     the schema walker.
//
// The package is library-pure: it has no hard dependency on the control
// plane's model router; the caller injects an LLMAugmenter via Generator
// options. The default Generator runs SchemaOnly mode without an LLM and
// satisfies all tests without external service calls.
package synth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Disposition mirrors the X-Crucible-Tape header values that classify a
// synth response.
type Disposition string

const (
	DispoSynthReadOnly  Disposition = "synth-readonly"
	DispoSynthMutation  Disposition = "synth-mutation"
	DispoCandidate      Disposition = "synth-candidate"  // CANDIDATE not yet promoted
)

// Method classifies a request as read or write for the decision tree.
// Anything other than the canonical read verbs is treated as mutating.
type Method string

const (
	MethodGET     Method = "GET"
	MethodHEAD    Method = "HEAD"
	MethodOPTIONS Method = "OPTIONS"
	MethodPOST    Method = "POST"
	MethodPUT     Method = "PUT"
	MethodPATCH   Method = "PATCH"
	MethodDELETE  Method = "DELETE"
)

// IsReadOnly returns true for canonical read verbs.
func (m Method) IsReadOnly() bool {
	switch m {
	case MethodGET, MethodHEAD, MethodOPTIONS:
		return true
	}
	return false
}

// Request is the synth-side projection of a tape Request — only the
// fields the synthesizer needs.
type Request struct {
	Service  string
	Endpoint string  // canonical endpoint path WITHOUT query (e.g., /v1/charges/{id})
	Method   Method
	Headers  map[string]string
	Body     []byte
	// PathParams is the resolved path-parameter map (e.g., {"id": "ch_123"}).
	PathParams map[string]string
}

// Response is the synth output.
type Response struct {
	Status      int
	Headers     map[string]string
	Body        []byte
	Disposition Disposition
	// Provenance lets the verifier weight the response.
	Provenance Provenance
}

// Provenance describes how a response was constructed.
type Provenance struct {
	Engine       string    // "schema" | "schema+llm" | "state-journal" | "openapi-example"
	LLMModel     string    // populated when LLM augmentation fired
	GeneratedAt  time.Time
	SchemaHash   string    // sha256 of the schema definition used
	CandidateID  string    // stable ID for human review queue
	StateJournal bool      // true when reads were served from the in-memory journal
}

// EndpointSpec describes one OpenAPI/proto endpoint the synthesizer can
// produce responses for. Crucible's tape importer transforms the customer's
// OpenAPI/proto into a list of EndpointSpec at recorder boot.
type EndpointSpec struct {
	Service        string
	Endpoint       string
	Method         Method
	ResponseSchema SchemaNode
	SuccessStatus  int
	// SuccessExample, if non-empty, is used verbatim for synth-mutation.
	SuccessExample json.RawMessage
}

// SchemaNode is a minimal subset of JSON-Schema sufficient for Faker-style
// generation. Implementations are exclusive — exactly one of Type / Object
// Properties / Array Items / Enum / Const / Ref is populated per node.
type SchemaNode struct {
	Type        string                 `json:"type,omitempty"`
	Format      string                 `json:"format,omitempty"`
	Properties  map[string]SchemaNode  `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Items       *SchemaNode            `json:"items,omitempty"`
	Enum        []json.RawMessage      `json:"enum,omitempty"`
	Const       json.RawMessage        `json:"const,omitempty"`
	Example     json.RawMessage        `json:"example,omitempty"`
	Minimum     *float64               `json:"minimum,omitempty"`
	Maximum     *float64               `json:"maximum,omitempty"`
	MinLength   *int                   `json:"minLength,omitempty"`
	MaxLength   *int                   `json:"maxLength,omitempty"`
	Pattern     string                 `json:"pattern,omitempty"`
	Nullable    bool                   `json:"nullable,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// LLMAugmenter wraps an optional LLM call that returns realistic field
// values from a schema + description pair. Callers inject the control
// plane's model router (Haiku 4.5 by default) or a stub.
type LLMAugmenter interface {
	// Augment returns a JSON value (any) for the schema. ctx may cancel.
	// Implementations MUST treat the schema as authoritative — outputs that
	// don't match the schema are discarded by the Generator.
	Augment(ctx context.Context, hint AugmentationHint) (json.RawMessage, error)
}

// AugmentationHint is the LLM prompt shape.
type AugmentationHint struct {
	Service     string
	Endpoint    string
	Method      Method
	FieldPath   string
	Schema      SchemaNode
	Description string
}

// Generator is the package's primary entry point.
type Generator struct {
	specs      map[string]EndpointSpec // keyed by routeKey(service, method, endpoint)
	augmenter  LLMAugmenter
	stateJrnl  *StateJournal
	clock      func() time.Time
	fakerSeed  uint64
	mu         sync.RWMutex
}

// Options shapes Generator construction.
type Options struct {
	// Specs is the OpenAPI/proto-derived endpoint list.
	Specs []EndpointSpec
	// Augmenter, if non-nil, is called per response field to upgrade Faker
	// output with realistic values. The default Generator runs schema-only.
	Augmenter LLMAugmenter
	// FakerSeed makes Faker output deterministic across runs. Zero means
	// "seed from time" — tests should always pass an explicit seed.
	FakerSeed uint64
	// Clock, if non-nil, overrides time.Now (test hook).
	Clock func() time.Time
}

// New constructs a Generator. The returned generator has no LLM augmenter
// and an empty state journal — call WithStateJournal / WithAugmenter.
func New(opts Options) *Generator {
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	g := &Generator{
		specs:     make(map[string]EndpointSpec, len(opts.Specs)),
		augmenter: opts.Augmenter,
		stateJrnl: NewStateJournal(opts.Clock),
		clock:     opts.Clock,
		fakerSeed: opts.FakerSeed,
	}
	for _, s := range opts.Specs {
		g.specs[routeKey(s.Service, s.Method, s.Endpoint)] = s
	}
	return g
}

// HasSpec reports whether an endpoint is in the registry.
func (g *Generator) HasSpec(service string, method Method, endpoint string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, ok := g.specs[routeKey(service, method, endpoint)]
	return ok
}

// Synthesize returns a response per the tape decision tree's miss-in-spec
// branches. The disposition tells the caller whether the result is a
// CANDIDATE awaiting human promotion or a determined synth-readonly /
// synth-mutation response.
func (g *Generator) Synthesize(ctx context.Context, req Request) (Response, error) {
	g.mu.RLock()
	spec, ok := g.specs[routeKey(req.Service, req.Method, req.Endpoint)]
	g.mu.RUnlock()
	if !ok {
		return Response{}, ErrNoSpec
	}
	if req.Method.IsReadOnly() {
		return g.synthReadOnly(ctx, req, spec)
	}
	return g.synthMutation(ctx, req, spec)
}

func (g *Generator) synthReadOnly(
	ctx context.Context, req Request, spec EndpointSpec,
) (Response, error) {
	// If the journal has a remembered write side for this resource, prefer
	// that — it's the closest thing to "what production would have done."
	if jbody, ok := g.stateJrnl.LookupRead(req); ok {
		return Response{
			Status:      200,
			Headers:     map[string]string{"X-Crucible-Tape": string(DispoSynthReadOnly), "Content-Type": "application/json"},
			Body:        jbody,
			Disposition: DispoSynthReadOnly,
			Provenance: Provenance{
				Engine:       "state-journal",
				GeneratedAt:  g.clock(),
				SchemaHash:   schemaHash(spec.ResponseSchema),
				StateJournal: true,
				CandidateID:  candidateID(req, jbody),
			},
		}, nil
	}
	body, engine, hash, err := g.generateFromSchema(ctx, req, spec)
	if err != nil {
		return Response{}, err
	}
	return Response{
		Status:      ifZero(spec.SuccessStatus, 200),
		Headers:     map[string]string{"X-Crucible-Tape": string(DispoSynthReadOnly), "Content-Type": "application/json"},
		Body:        body,
		Disposition: DispoSynthReadOnly,
		Provenance: Provenance{
			Engine:      engine,
			GeneratedAt: g.clock(),
			SchemaHash:  hash,
			CandidateID: candidateID(req, body),
		},
	}, nil
}

func (g *Generator) synthMutation(
	ctx context.Context, req Request, spec EndpointSpec,
) (Response, error) {
	// Mutation responses come from the spec's success example if available
	// (Microcks AI Copilot prefers the example over Faker output); else
	// generate from schema. The mutation is recorded in the journal so
	// subsequent reads from the same agent see consistent state.
	var body json.RawMessage
	engine := "openapi-example"
	if len(spec.SuccessExample) > 0 {
		body = spec.SuccessExample
	} else {
		generated, gengine, _, err := g.generateFromSchema(ctx, req, spec)
		if err != nil {
			return Response{}, err
		}
		body = generated
		engine = gengine
	}
	g.stateJrnl.RecordWrite(req, body)
	return Response{
		Status:      ifZero(spec.SuccessStatus, 200),
		Headers:     map[string]string{"X-Crucible-Tape": string(DispoSynthMutation), "Content-Type": "application/json"},
		Body:        body,
		Disposition: DispoSynthMutation,
		Provenance: Provenance{
			Engine:      engine,
			GeneratedAt: g.clock(),
			SchemaHash:  schemaHash(spec.ResponseSchema),
			CandidateID: candidateID(req, body),
		},
	}, nil
}

// generateFromSchema runs the schema walker. The walker uses Faker-style
// presets for known formats; if an LLMAugmenter is configured, the walker
// calls it for free-text fields (description+name combinations that lack
// machine-readable shape).
func (g *Generator) generateFromSchema(
	ctx context.Context, req Request, spec EndpointSpec,
) (json.RawMessage, string, string, error) {
	w := &walker{
		augmenter: g.augmenter,
		req:       req,
		seed:      g.fakerSeed,
	}
	value, err := w.walk(ctx, "", spec.ResponseSchema)
	if err != nil {
		return nil, "", "", err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, "", "", fmt.Errorf("marshal generated value: %w", err)
	}
	engine := "schema"
	if w.augmenterFired {
		engine = "schema+llm"
	}
	return raw, engine, schemaHash(spec.ResponseSchema), nil
}

// StateJournal exposes the per-task in-memory mutation journal so callers
// can pre-seed or inspect it from tests.
func (g *Generator) StateJournal() *StateJournal { return g.stateJrnl }

// ──────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────

// ErrNoSpec is returned when the request's endpoint has no registered spec.
// The tape driver translates this into the next branch of the decision tree.
var ErrNoSpec = errors.New("synth: no OpenAPI/proto spec for endpoint")

func routeKey(service string, method Method, endpoint string) string {
	return strings.ToLower(service) + "|" + string(method) + "|" + endpoint
}

func schemaHash(s SchemaNode) string {
	raw, _ := json.Marshal(s)
	h := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(h[:])
}

func candidateID(req Request, body []byte) string {
	h := sha256.New()
	h.Write([]byte(req.Service))
	h.Write([]byte{0})
	h.Write([]byte(req.Method))
	h.Write([]byte{0})
	h.Write([]byte(req.Endpoint))
	h.Write([]byte{0})
	h.Write(body)
	sum := h.Sum(nil)
	return "cand_" + hex.EncodeToString(sum[:8])
}

func ifZero(v, dflt int) int {
	if v == 0 {
		return dflt
	}
	return v
}

// CompareEndpointSpecs orders specs deterministically; useful in tests.
func CompareEndpointSpecs(a, b EndpointSpec) bool {
	if a.Service != b.Service {
		return a.Service < b.Service
	}
	if a.Method != b.Method {
		return a.Method < b.Method
	}
	return a.Endpoint < b.Endpoint
}

// SortSpecs sorts a slice in place for deterministic iteration.
func SortSpecs(specs []EndpointSpec) {
	sort.Slice(specs, func(i, j int) bool { return CompareEndpointSpecs(specs[i], specs[j]) })
}
