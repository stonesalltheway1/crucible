// Package audit enforces the ADR-002 invariant at the runner ingress
// boundary: the executor's reasoning trace must never reach the
// verifier model — or the verifier tools. The daemon already runs
// this guard on its side; we re-run it here because (a) the daemon
// could change without the runner noticing, (b) defence in depth is
// cheaper than the brand-existential cost of a leaked
// chain-of-thought, and (c) the runner is the last place we control
// before the diff hits real tooling.
//
// The guard works on a generic decoded JSON map (the same blob the
// runner just unmarshalled from stdin). Any field whose lower-case
// name contains a denylist substring causes a Refusal. Recursive on
// nested maps and arrays-of-maps.
package audit

import (
	"fmt"
	"sort"
	"strings"
)

// Denylist is the lower-case substring set the guard checks against
// each JSON field name. Kept in sync with
// apps/verifier/internal/verification/verification.go.
//
// We bias toward false-positives. A legitimate field that trips this
// guard must be renamed, not whitelisted: the cost of one rename is
// trivial vs. the cost of accidentally piping reasoning through.
var Denylist = []string{
	"reasoning",
	"chain_of_thought",
	"chain-of-thought",
	"cot",
	"thinking_trace",
	"thinking-trace",
	"thoughts",
	"scratchpad",
	"internal_monologue",
	"hidden_state",
	"agent_trace",
	"executor_trace",
	"trajectory",
	"plan_critique",
	"reflection",
}

// LeakageError reports the first offending field and the substring it
// matched. The runner surfaces this as a tool_unavailable TestReport
// and refuses to run any tier.
type LeakageError struct {
	OffendingField string
	Pattern        string
}

func (e *LeakageError) Error() string {
	return fmt.Sprintf("audit: executor-reasoning leak — field %q matched denylist pattern %q (ADR-002 invariant)",
		e.OffendingField, e.Pattern)
}

// NoLeakage walks an arbitrary decoded-JSON value and returns a
// LeakageError on the first denylisted field. Pass the parsed
// stdin map[string]any directly.
func NoLeakage(payload map[string]any) error {
	return walkMap(payload, "")
}

func walkMap(m map[string]any, prefix string) error {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Sort so the offending field reported is deterministic across runs.
	sort.Strings(keys)
	for _, k := range keys {
		full := k
		if prefix != "" {
			full = prefix + "." + k
		}
		lk := strings.ToLower(k)
		for _, deny := range Denylist {
			if strings.Contains(lk, deny) {
				return &LeakageError{OffendingField: full, Pattern: deny}
			}
		}
		switch v := m[k].(type) {
		case map[string]any:
			if err := walkMap(v, full); err != nil {
				return err
			}
		case []any:
			for i, elem := range v {
				if mm, ok := elem.(map[string]any); ok {
					if err := walkMap(mm, fmt.Sprintf("%s[%d]", full, i)); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
