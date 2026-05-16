package modelrouter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

// Client is the vendor-neutral interface every LLM call goes through.
// Each Vendor has its own concrete implementation (anthropic.go, google.go,
// openai.go) — Router.Call(...) dispatches.
type Client interface {
	Vendor() Vendor
	Call(ctx context.Context, req Request) (*Response, error)
}

// Request is a vendor-neutral chat-completion request.
type Request struct {
	Model         string
	System        string           // system prompt (cached 1h slot)
	Tools         []ToolDefinition // (cached 1h slot)
	Messages      []Message
	MaxOutput     int
	Temperature   float64
	ThinkingLevel string // "low" | "medium" | "high" | "" (off)
	JSONMode      bool
	JSONSchema    map[string]any // structured-output schema
	CacheSystem   bool           // emit cache_control on the system slot (Anthropic)
	CacheTools    bool           // emit cache_control on the tools slot (Anthropic)
	Stream        bool
}

// Role enumerates message roles.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is one turn in the chat. Content is a plain string in Phase 1; the
// vendor-specific structured-content shapes are constructed inside each adapter.
type Message struct {
	Role    Role
	Content string
	Name    string // for tool-result messages, the tool's name
}

// ToolDefinition is a vendor-neutral function-calling tool spec.
type ToolDefinition struct {
	Name        string
	Description string
	JSONSchema  map[string]any
}

// Response is the vendor-neutral response shape.
type Response struct {
	Model        string
	Content      string         // primary text content
	StopReason   string         // "end_turn" | "tool_use" | "max_tokens" | ...
	Usage        Usage
	ToolCalls    []ToolCall
	Latency      time.Duration
	RawProviderID string         // vendor's request/response ID for tracing
	CacheHit     bool
}

type Usage struct {
	InputTokensFresh   int
	InputTokensCached  int
	OutputTokens       int
	ThinkingTokens     int
	CacheReadTokens    int
	CacheCreationTokens int
}

type ToolCall struct {
	ID         string
	Name       string
	ArgumentsJSON string
}

// Router dispatches a vendor-neutral Request to the correct Client.
type Router struct {
	clients map[Vendor]Client
}

// NewRouter constructs a Router with the provided clients. Vendors not in the
// map cause Route(...) to fail.
func NewRouter(clients ...Client) *Router {
	r := &Router{clients: make(map[Vendor]Client, len(clients))}
	for _, c := range clients {
		if c == nil {
			continue
		}
		r.clients[c.Vendor()] = c
	}
	return r
}

// Vendors returns the set of registered vendors.
func (r *Router) Vendors() []Vendor {
	out := make([]Vendor, 0, len(r.clients))
	for v := range r.clients {
		out = append(out, v)
	}
	return out
}

// Call routes by model id → vendor lookup, then dispatches to the right Client.
func (r *Router) Call(ctx context.Context, req Request) (*Response, error) {
	if req.Model == "" {
		return nil, errors.New("modelrouter: request missing Model")
	}
	spec, err := Lookup(req.Model)
	if err != nil {
		return nil, err
	}
	cli, ok := r.clients[spec.Vendor]
	if !ok {
		return nil, fmt.Errorf("modelrouter: no client registered for vendor %q (model %q). "+
			"Did you forget to set the %s env var?", spec.Vendor, req.Model, envVarFor(spec.Vendor))
	}
	return cli.Call(ctx, req)
}

// EstimateCostUSD returns the projected USD cost of a response based on usage.
func EstimateCostUSD(model string, u Usage) float64 {
	spec, err := Lookup(model)
	if err != nil {
		return 0
	}
	cost := float64(u.InputTokensFresh)*spec.InputUSDPerMillion/1_000_000 +
		float64(u.OutputTokens+u.ThinkingTokens)*spec.OutputUSDPerMillion/1_000_000 +
		float64(u.CacheReadTokens)*spec.CacheReadUSDPerMillion/1_000_000 +
		float64(u.CacheCreationTokens)*spec.CacheWrite1hPerMillion/1_000_000
	return cost
}

func envVarFor(v Vendor) string {
	switch v {
	case VendorAnthropic:
		return "ANTHROPIC_API_KEY"
	case VendorGoogle:
		return "GOOGLE_API_KEY (or GEMINI_API_KEY)"
	case VendorOpenAI:
		return "OPENAI_API_KEY"
	case VendorXai:
		return "XAI_API_KEY"
	case VendorDeepSeek:
		return "DEEPSEEK_API_KEY"
	default:
		return string(v) + "_API_KEY"
	}
}

// HasEnv returns true if the appropriate env var for the vendor is set.
func HasEnv(v Vendor) bool {
	switch v {
	case VendorAnthropic:
		return os.Getenv("ANTHROPIC_API_KEY") != ""
	case VendorGoogle:
		return os.Getenv("GOOGLE_API_KEY") != "" || os.Getenv("GEMINI_API_KEY") != ""
	case VendorOpenAI:
		return os.Getenv("OPENAI_API_KEY") != ""
	default:
		return false
	}
}
