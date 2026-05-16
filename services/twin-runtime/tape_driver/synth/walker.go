package synth

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math/rand"
	"regexp"
	"strings"
)

// walker turns a SchemaNode into a concrete Go value following Faker-style
// presets for known JSON-Schema formats and inviting the optional LLM
// augmenter to fill free-text fields.
//
// The walker is deterministic when the Generator's FakerSeed is set: the
// per-field RNG is derived from FakerSeed + the JSON-pointer path. Same
// schema + seed → same output.
type walker struct {
	augmenter      LLMAugmenter
	req            Request
	seed           uint64
	augmenterFired bool
}

func (w *walker) walk(ctx context.Context, path string, node SchemaNode) (any, error) {
	if node.Nullable && len(node.Properties) == 0 && node.Type == "" {
		return nil, nil
	}
	if len(node.Const) > 0 {
		var v any
		if err := json.Unmarshal(node.Const, &v); err == nil {
			return v, nil
		}
	}
	if len(node.Enum) > 0 {
		rng := w.rng(path)
		idx := rng.Intn(len(node.Enum))
		var v any
		if err := json.Unmarshal(node.Enum[idx], &v); err == nil {
			return v, nil
		}
	}
	if len(node.Example) > 0 {
		var v any
		if err := json.Unmarshal(node.Example, &v); err == nil {
			return v, nil
		}
	}
	switch node.Type {
	case "object":
		return w.walkObject(ctx, path, node)
	case "array":
		return w.walkArray(ctx, path, node)
	case "string":
		return w.walkString(ctx, path, node)
	case "integer":
		return w.walkInteger(path, node), nil
	case "number":
		return w.walkNumber(path, node), nil
	case "boolean":
		return w.walkBool(path), nil
	}
	if len(node.Properties) > 0 {
		return w.walkObject(ctx, path, node)
	}
	if node.Items != nil {
		return w.walkArray(ctx, path, node)
	}
	return nil, nil
}

func (w *walker) walkObject(ctx context.Context, path string, node SchemaNode) (any, error) {
	required := make(map[string]struct{}, len(node.Required))
	for _, r := range node.Required {
		required[r] = struct{}{}
	}
	out := make(map[string]any, len(node.Properties))
	for name, child := range node.Properties {
		// Skip optional fields ~30% of the time for shape variety unless
		// the seed says otherwise.
		if _, ok := required[name]; !ok {
			if w.rng(path+"/"+name+"/skip").Intn(10) < 3 {
				continue
			}
		}
		val, err := w.walk(ctx, path+"/"+name, child)
		if err != nil {
			return nil, err
		}
		out[name] = val
	}
	return out, nil
}

func (w *walker) walkArray(ctx context.Context, path string, node SchemaNode) (any, error) {
	if node.Items == nil {
		return []any{}, nil
	}
	n := 1 + w.rng(path+"/len").Intn(3) // 1..3 items by default
	out := make([]any, 0, n)
	for i := 0; i < n; i++ {
		v, err := w.walk(ctx, fmt.Sprintf("%s[%d]", path, i), *node.Items)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (w *walker) walkString(ctx context.Context, path string, node SchemaNode) (string, error) {
	// Format presets — these mirror Faker's common locales for shape, not
	// for value realism. The LLM augmenter is the realism layer.
	switch node.Format {
	case "email":
		return "synth.user@example.com", nil
	case "uri", "uri-reference", "url":
		return "https://synth.example.com/" + tokenFromPath(path), nil
	case "uuid":
		return uuidFromPath(path, w.seed), nil
	case "date":
		return "2026-05-15", nil
	case "date-time":
		return "2026-05-15T12:34:56Z", nil
	case "ipv4":
		return "192.0.2.1", nil
	case "ipv6":
		return "2001:db8::1", nil
	case "hostname":
		return "synth.example.com", nil
	case "byte":
		return "c3ludGg=", nil // "synth"
	case "binary":
		return "synth", nil
	}
	// Pattern fields: if the schema specifies a regex, we attempt a one-of-
	// fallback for common shapes. Otherwise produce a hint string.
	if node.Pattern != "" {
		if v, ok := patternStub(node.Pattern); ok {
			return v, nil
		}
	}
	// If an LLMAugmenter is configured, ask it for a realistic value for
	// free-text fields (those without format / pattern). The schema is
	// still authoritative; we discard outputs that don't fit.
	if w.augmenter != nil && node.Format == "" && node.Pattern == "" {
		raw, err := w.augmenter.Augment(ctx, AugmentationHint{
			Service:     w.req.Service,
			Endpoint:    w.req.Endpoint,
			Method:      w.req.Method,
			FieldPath:   path,
			Schema:      node,
			Description: node.Description,
		})
		if err == nil && len(raw) > 0 {
			var v string
			if err := json.Unmarshal(raw, &v); err == nil {
				if !violatesShape(v, node) {
					w.augmenterFired = true
					return v, nil
				}
			}
		}
	}
	// Last resort: shape-only Faker output.
	min, max := 1, 12
	if node.MinLength != nil {
		min = *node.MinLength
	}
	if node.MaxLength != nil {
		max = *node.MaxLength
	}
	length := min
	if max > min {
		length = min + w.rng(path).Intn(max-min+1)
	}
	if length < 1 {
		length = 1
	}
	return synthAlpha(length, w.rng(path)), nil
}

func (w *walker) walkInteger(path string, node SchemaNode) int {
	rng := w.rng(path)
	min, max := 0, 1000
	if node.Minimum != nil {
		min = int(*node.Minimum)
	}
	if node.Maximum != nil {
		max = int(*node.Maximum)
	}
	if max <= min {
		return min
	}
	return min + rng.Intn(max-min+1)
}

func (w *walker) walkNumber(path string, node SchemaNode) float64 {
	rng := w.rng(path)
	min, max := 0.0, 1000.0
	if node.Minimum != nil {
		min = *node.Minimum
	}
	if node.Maximum != nil {
		max = *node.Maximum
	}
	if max <= min {
		return min
	}
	return min + rng.Float64()*(max-min)
}

func (w *walker) walkBool(path string) bool {
	return w.rng(path).Intn(2) == 1
}

func (w *walker) rng(path string) *rand.Rand {
	h := fnv.New64a()
	h.Write([]byte(path))
	src := rand.NewSource(int64(w.seed ^ h.Sum64()))
	return rand.New(src)
}

// ──────────────────────────────────────────────────────────────────────
// Token helpers
// ──────────────────────────────────────────────────────────────────────

func tokenFromPath(path string) string {
	h := fnv.New64a()
	h.Write([]byte(path))
	return fmt.Sprintf("%016x", h.Sum64())[:8]
}

func uuidFromPath(path string, seed uint64) string {
	h := fnv.New64a()
	h.Write([]byte(path))
	hashLow := h.Sum64()
	hashHigh := seed ^ hashLow
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		uint32(hashLow),
		uint16(hashLow>>32),
		uint16((hashLow>>48)&0x0FFF|0x4000),
		uint16(hashHigh&0x3FFF|0x8000),
		hashHigh,
	)
}

const synthAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

func synthAlpha(length int, rng *rand.Rand) string {
	out := make([]byte, length)
	for i := range out {
		out[i] = synthAlphabet[rng.Intn(len(synthAlphabet))]
	}
	return string(out)
}

// patternStub provides one-of stubs for common regex shapes. We DO NOT
// attempt to invert arbitrary regexes — that's a different package's
// problem.
func patternStub(pattern string) (string, bool) {
	switch {
	case strings.Contains(pattern, "[A-Z]{3}"):
		return "USD", true
	case strings.Contains(pattern, "^cus_"):
		return "cus_synth0001", true
	case strings.Contains(pattern, "^ch_"):
		return "ch_synth0001", true
	case strings.Contains(pattern, "^evt_"):
		return "evt_synth0001", true
	}
	return "", false
}

var (
	digitsOnly = regexp.MustCompile(`^\d+$`)
)

func violatesShape(v string, node SchemaNode) bool {
	if node.MinLength != nil && len(v) < *node.MinLength {
		return true
	}
	if node.MaxLength != nil && len(v) > *node.MaxLength {
		return true
	}
	if node.Pattern != "" {
		// Only enforce a pattern we KNOW how to evaluate cheaply; if the
		// regex is exotic we accept the value (LLM output is best-effort).
		if pattern, err := regexp.Compile(node.Pattern); err == nil {
			if !pattern.MatchString(v) {
				return true
			}
		}
	}
	return false
}
