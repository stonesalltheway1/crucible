package events

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestInMemory_FanOutToMultipleSubscribers(t *testing.T) {
	p := NewInMemory()
	a := p.Subscribe(4)
	b := p.Subscribe(4)
	if p.Subscribers() != 2 {
		t.Fatalf("expected 2 subscribers, got %d", p.Subscribers())
	}
	ev := Event{Type: EventTaskSubmitted, TaskID: "task_1"}
	if err := p.Publish(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	for name, ch := range map[string]<-chan Event{"a": a, "b": b} {
		select {
		case got := <-ch:
			if got.TaskID != "task_1" {
				t.Errorf("%s: bad payload %+v", name, got)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("%s: didn't receive event", name)
		}
	}
}

func TestInMemory_DropsOnSlowSubscriber(t *testing.T) {
	p := NewInMemory()
	ch := p.Subscribe(1)
	for i := 0; i < 100; i++ {
		_ = p.Publish(context.Background(), Event{Type: EventTaskSubmitted, TaskID: "t"})
	}
	// The subscriber's buffer should be full; no panic.
	if len(ch) > 1 {
		t.Fatalf("buffer overflow: %d", len(ch))
	}
}

func TestWebhook_RetryOn5xx(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		if atomic.LoadInt32(&attempts) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	w := NewWebhook(srv.URL, logger)
	err := w.Publish(context.Background(), Event{Type: EventTaskSubmitted, TaskID: "t"})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
}

func TestWebhook_DropsOn4xxAfterLog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	w := NewWebhook(srv.URL, slog.New(slog.NewTextHandler(io.Discard, nil)))
	err := w.Publish(context.Background(), Event{Type: EventTaskSubmitted, TaskID: "t"})
	if err == nil {
		t.Fatal("expected error on 4xx")
	}
}

func TestWebhook_PostsJSON(t *testing.T) {
	var got Event
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ev Event
		if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
			t.Errorf("decode: %v", err)
			w.WriteHeader(500)
			return
		}
		mu.Lock()
		got = ev
		mu.Unlock()
		w.WriteHeader(200)
	}))
	defer srv.Close()
	w := NewWebhook(srv.URL, slog.New(slog.NewTextHandler(io.Discard, nil)))
	in := Event{Type: EventTaskPlanApproved, TaskID: "task_X", TenantID: "t1"}
	if err := w.Publish(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if got.TaskID != "task_X" || got.Type != EventTaskPlanApproved {
		t.Fatalf("payload mismatch: %+v", got)
	}
}

func TestWebhook_RejectsEmptyURL(t *testing.T) {
	w := NewWebhook("", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := w.Publish(context.Background(), Event{}); err == nil {
		t.Fatal("expected error on empty URL")
	}
}

func TestMulti_FansOut(t *testing.T) {
	p1 := NewInMemory()
	p2 := NewInMemory()
	ch1 := p1.Subscribe(2)
	ch2 := p2.Subscribe(2)
	m := NewMulti(slog.New(slog.NewTextHandler(io.Discard, nil)), p1, p2)
	if err := m.Publish(context.Background(), Event{Type: EventTaskSubmitted, TaskID: "t"}); err != nil {
		t.Fatal(err)
	}
	for name, ch := range map[string]<-chan Event{"p1": ch1, "p2": ch2} {
		select {
		case <-ch:
		case <-time.After(100 * time.Millisecond):
			t.Errorf("%s: missed event", name)
		}
	}
}

func TestMulti_AutoStampsAt(t *testing.T) {
	p := NewInMemory()
	ch := p.Subscribe(1)
	m := NewMulti(slog.New(slog.NewTextHandler(io.Discard, nil)), p)
	_ = m.Publish(context.Background(), Event{Type: EventTaskSubmitted, TaskID: "t"})
	ev := <-ch
	if ev.At.IsZero() {
		t.Fatal("expected At to be stamped by MultiPublisher")
	}
}
