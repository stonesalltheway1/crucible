// Package events publishes structured webhook + SSE events for task lifecycle
// transitions. The event types match docs/03-sdk/event-spec.md (Phase-2 file).
//
// Phase 1 ships two backends:
//   - InMemoryPublisher    — fan-out to in-process subscribers; used by tests
//                            and the CLI's "task watch" subcommand (Phase 2).
//   - WebhookPublisher     — POSTs JSON to a configured URL with retry.
//
// All publishers implement Publisher; Server.publish() iterates a slice so a
// single event can fan out to multiple sinks.
package events

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// EventType enumerates the lifecycle events Phase 1 emits.
type EventType string

const (
	EventTaskSubmitted        EventType = "task.submitted"
	EventTaskPlanReady        EventType = "task.plan_ready"
	EventTaskPlanApproved     EventType = "task.plan_approved"
	EventTaskPlanRejected     EventType = "task.plan_rejected"
	EventTaskReplanned        EventType = "task.replanned"
	EventTaskBudgetWarn80     EventType = "task.budget_warn_80"
	EventTaskBudgetExceeded   EventType = "task.budget_exceeded"
	EventTaskRetryLimit       EventType = "task.retry_limit_exceeded"
	EventTaskWallClockExceeded EventType = "task.wall_clock_exceeded"
)

// Event is one structured task lifecycle event.
type Event struct {
	Type      EventType      `json:"type"`
	TaskID    string         `json:"task_id"`
	TenantID  string         `json:"tenant_id,omitempty"`
	At        time.Time      `json:"at"`
	Detail    map[string]any `json:"detail,omitempty"`
}

// Publisher is the fan-out interface every backend implements.
type Publisher interface {
	Publish(ctx context.Context, ev Event) error
}

// MultiPublisher fans out a single event to multiple downstreams. Errors are
// logged but do not abort fan-out (one bad webhook can't block the others).
type MultiPublisher struct {
	logger *slog.Logger
	subs   []Publisher
}

// NewMulti returns a fan-out publisher.
func NewMulti(logger *slog.Logger, subs ...Publisher) *MultiPublisher {
	return &MultiPublisher{logger: logger, subs: subs}
}

// Publish fans out concurrently with a 5-second deadline per subscriber.
func (m *MultiPublisher) Publish(ctx context.Context, ev Event) error {
	if len(m.subs) == 0 {
		return nil
	}
	if ev.At.IsZero() {
		ev.At = time.Now().UTC()
	}
	var wg sync.WaitGroup
	for _, sub := range m.subs {
		wg.Add(1)
		go func(p Publisher) {
			defer wg.Done()
			sctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err := p.Publish(sctx, ev); err != nil {
				m.logger.WarnContext(ctx, "events: subscriber failed", "type", ev.Type, "err", err)
			}
		}(sub)
	}
	wg.Wait()
	return nil
}

// InMemoryPublisher fan-outs to a slice of channels. Callers subscribe via
// Subscribe(); slow subscribers drop events (non-blocking send).
type InMemoryPublisher struct {
	mu   sync.RWMutex
	subs []chan Event
}

// NewInMemory returns a publisher with no subscribers.
func NewInMemory() *InMemoryPublisher { return &InMemoryPublisher{} }

// Subscribe returns a channel that receives every event from now on. Buffer
// size is the caller's slack; we never block to push.
func (p *InMemoryPublisher) Subscribe(buffer int) <-chan Event {
	if buffer <= 0 {
		buffer = 16
	}
	ch := make(chan Event, buffer)
	p.mu.Lock()
	p.subs = append(p.subs, ch)
	p.mu.Unlock()
	return ch
}

// Publish broadcasts to all live subscribers, dropping on overflow.
func (p *InMemoryPublisher) Publish(_ context.Context, ev Event) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, ch := range p.subs {
		select {
		case ch <- ev:
		default:
			// subscriber slow; drop
		}
	}
	return nil
}

// Subscribers returns the current subscriber count.
func (p *InMemoryPublisher) Subscribers() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.subs)
}

// WebhookPublisher POSTs JSON to a URL with a configurable timeout. One simple
// retry on transient HTTP 5xx; permanent 4xx are dropped after the log.
type WebhookPublisher struct {
	URL    string
	Client *http.Client
	Logger *slog.Logger
}

// NewWebhook constructs a WebhookPublisher with a 5s timeout.
func NewWebhook(url string, logger *slog.Logger) *WebhookPublisher {
	return &WebhookPublisher{
		URL:    url,
		Client: &http.Client{Timeout: 5 * time.Second},
		Logger: logger,
	}
}

// Publish issues the POST.
func (w *WebhookPublisher) Publish(ctx context.Context, ev Event) error {
	if w.URL == "" {
		return errors.New("events: webhook URL not configured")
	}
	body, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("events: marshal: %w", err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("events: build request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "crucible-control-plane/2026.06.0-phase1")

		resp, err := w.Client.Do(req)
		if err != nil {
			if attempt == 0 {
				continue
			}
			return fmt.Errorf("events: do request: %w", err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		if resp.StatusCode >= 500 && attempt == 0 {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		return fmt.Errorf("events: webhook returned status %d", resp.StatusCode)
	}
	return errors.New("events: exhausted retries")
}
