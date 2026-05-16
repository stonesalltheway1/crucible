package cruciblev1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// Scope serializes as either the literal string "all" or as a {repo,file_glob,category}
// object. UnmarshalJSON accepts both shapes.

func (s Scope) MarshalJSON() ([]byte, error) {
	if s.All {
		return []byte(`"all"`), nil
	}
	if s.Filter == nil {
		return []byte(`{}`), nil
	}
	return json.Marshal(s.Filter)
}

func (s *Scope) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return errors.New("scope: empty input")
	}
	if data[0] == '"' {
		var lit string
		if err := json.Unmarshal(data, &lit); err != nil {
			return err
		}
		if lit != "all" {
			return fmt.Errorf("scope: unexpected string literal %q (only \"all\" is supported)", lit)
		}
		s.All = true
		s.Filter = nil
		return nil
	}
	if data[0] == '{' {
		var f ScopeFilter
		if err := json.Unmarshal(data, &f); err != nil {
			return err
		}
		s.All = false
		s.Filter = &f
		return nil
	}
	return fmt.Errorf("scope: unexpected JSON token %q", string(data[0]))
}

// CrucibleError is the structured-error contract returned by the control plane.
// Mirrors the SDK reference doc's `class CrucibleError`.
type CrucibleError struct {
	Code     ErrorCode       `json:"code"`
	Message  string          `json:"message"`
	Retry    bool            `json:"retryable"`
	Hint     string          `json:"hint,omitempty"`
	Details  json.RawMessage `json:"details,omitempty"`
}

func (e *CrucibleError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s: %s (hint: %s)", e.Code, e.Message, e.Hint)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *CrucibleError) Retryable() bool { return e.Retry }

func NewError(code ErrorCode, msg, hint string, retryable bool) *CrucibleError {
	return &CrucibleError{
		Code:    code,
		Message: msg,
		Hint:    hint,
		Retry:   retryable,
	}
}
