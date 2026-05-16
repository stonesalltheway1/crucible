// Package relay is the gate-side HTTP client for the Rust attestation
// relay. It implements bundle_validator.Verifier + outcome_watcher.OutcomeSink.
package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/crucible/promotion-gate/internal/bundle_validator"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// Client is the HTTP client.
type Client struct {
	base   string
	http   *http.Client
	cache  sync.Map // uuid → *bundle_validator.FetchedStatement
}

// New builds a Client.
func New(baseURL string) (*Client, error) {
	if baseURL == "" {
		return nil, errors.New("relay: empty base URL")
	}
	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("relay: invalid base URL: %w", err)
	}
	return &Client{
		base: baseURL,
		http: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// FetchStatement implements bundle_validator.Verifier.
func (c *Client) FetchStatement(ctx context.Context, uuid string) (*bundle_validator.FetchedStatement, error) {
	if v, ok := c.cache.Load(uuid); ok {
		return v.(*bundle_validator.FetchedStatement), nil
	}
	u := c.base + "/v1/attestations/" + url.PathEscape(uuid)
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("relay: fetch %s: %w", uuid, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("relay: fetch %s: status=%d body=%s", uuid, resp.StatusCode, string(body))
	}
	var env struct {
		Payload string `json:"payload"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("relay: decode envelope: %w", err)
	}
	payload, err := decodeB64(env.Payload)
	if err != nil {
		return nil, err
	}
	var stmt struct {
		PredicateType string                          `json:"predicateType"`
		Subject       []bundle_validator.StatementSubject `json:"subject"`
		Predicate     map[string]any                  `json:"predicate"`
	}
	if err := json.Unmarshal(payload, &stmt); err != nil {
		return nil, fmt.Errorf("relay: decode statement: %w", err)
	}
	out := &bundle_validator.FetchedStatement{
		UUID:          uuid,
		PredicateType: stmt.PredicateType,
		Subject:       stmt.Subject,
		Predicate:     stmt.Predicate,
	}
	c.cache.Store(uuid, out)
	return out, nil
}

// Emit posts to /v1/attestations and returns the relay receipt.
func (c *Client) Emit(ctx context.Context, predicateType, subjectName string, subjectContent []byte, predicate any) (*EmitResponse, error) {
	body := map[string]any{
		"predicate_type":   predicateType,
		"subject_name":     subjectName,
		"subject_content_b64": encodeB64(subjectContent),
		"predicate":        predicate,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	u := c.base + "/v1/attestations"
	req, _ := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("relay: emit: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		bb, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("relay: emit status=%d body=%s", resp.StatusCode, string(bb))
	}
	var er EmitResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		return nil, err
	}
	return &er, nil
}

// EmitResponse mirrors the relay's response.
type EmitResponse struct {
	Receipt cruciblev1.RekorEntry `json:"receipt"`
}

// EmitOutcome implements outcome_watcher.OutcomeSink — emits a
// PromotionOutcome/v1 attestation through the relay.
func (c *Client) EmitOutcome(ctx context.Context, predicate cruciblev1.PromotionOutcomeAttestation) (string, error) {
	er, err := c.Emit(ctx, cruciblev1.PredicatePromotionOutcome, predicate.PromotionID, []byte(predicate.PromotionID), predicate)
	if err != nil {
		return "", err
	}
	return er.Receipt.UUID, nil
}

// ── encoders without pulling in encoding/base64 here in many places ────

func encodeB64(b []byte) string {
	return base64StdEncoding.EncodeToString(b)
}

func decodeB64(s string) ([]byte, error) {
	return base64StdEncoding.DecodeString(s)
}
