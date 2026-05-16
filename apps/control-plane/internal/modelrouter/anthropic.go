package modelrouter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicClient wraps github.com/anthropics/anthropic-sdk-go v1.43+.
type AnthropicClient struct {
	api anthropic.Client
}

// NewAnthropicClient returns a client backed by ANTHROPIC_API_KEY. Returns an
// error if the env var is unset, so callers fail fast rather than receive a
// 401 on the first call.
func NewAnthropicClient() (*AnthropicClient, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil, errors.New("ANTHROPIC_API_KEY not set")
	}
	return &AnthropicClient{api: anthropic.NewClient(option.WithAPIKey(key))}, nil
}

// NewAnthropicClientFromEnv is identical to NewAnthropicClient but returns nil
// (no error) when the env var is unset — useful at process start where the
// vendor may be optional.
func NewAnthropicClientFromEnv() *AnthropicClient {
	c, err := NewAnthropicClient()
	if err != nil {
		return nil
	}
	return c
}

func (c *AnthropicClient) Vendor() Vendor { return VendorAnthropic }

func (c *AnthropicClient) Call(ctx context.Context, req Request) (*Response, error) {
	start := time.Now()

	// Build the Messages API request.
	msgs := make([]anthropic.MessageParam, 0, len(req.Messages))
	for _, m := range req.Messages {
		switch m.Role {
		case RoleUser:
			msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		case RoleAssistant:
			msgs = append(msgs, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		case RoleSystem:
			// Anthropic system goes in its own slot; collapse below.
		case RoleTool:
			// Phase 1 control plane doesn't issue tool_use; ignore.
		default:
			return nil, fmt.Errorf("anthropic: unsupported role %q", m.Role)
		}
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		Messages:  msgs,
		MaxTokens: int64(orDefault(req.MaxOutput, 1024)),
	}

	if req.Temperature > 0 {
		params.Temperature = anthropic.Float(req.Temperature)
	}

	if req.System != "" {
		// IMPORTANT: pass ttl="1h" explicitly. Anthropic flipped the default
		// cache TTL from 1h to 5m on 2026-03-06 (silently).
		systemBlock := anthropic.TextBlockParam{
			Text: req.System,
		}
		if req.CacheSystem {
			systemBlock.CacheControl = anthropic.CacheControlEphemeralParam{
				TTL: "1h",
			}
		}
		params.System = []anthropic.TextBlockParam{systemBlock}
	}

	if req.ThinkingLevel != "" {
		// Anthropic uses "extended_thinking" with a token budget. Map our
		// portable level to a reasonable budget.
		params.Thinking = anthropic.ThinkingConfigParamOfEnabled(budgetForThinkingLevel(req.ThinkingLevel))
	}

	resp, err := c.api.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic: messages.new: %w", err)
	}

	out := &Response{
		Model:         req.Model,
		StopReason:    string(resp.StopReason),
		Latency:       time.Since(start),
		RawProviderID: resp.ID,
		Usage: Usage{
			InputTokensFresh:    int(resp.Usage.InputTokens),
			OutputTokens:        int(resp.Usage.OutputTokens),
			CacheReadTokens:     int(resp.Usage.CacheReadInputTokens),
			CacheCreationTokens: int(resp.Usage.CacheCreationInputTokens),
		},
	}
	out.CacheHit = out.Usage.CacheReadTokens > 0

	for _, block := range resp.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			out.Content += b.Text
		case anthropic.ThinkingBlock:
			out.Usage.ThinkingTokens += len(b.Thinking) // approximate; SDK reports separately
		case anthropic.ToolUseBlock:
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:           b.ID,
				Name:         b.Name,
				ArgumentsJSON: string(b.Input),
			})
		}
	}

	return out, nil
}

func budgetForThinkingLevel(level string) int64 {
	switch level {
	case "low":
		return 2048
	case "medium":
		return 8192
	case "high":
		return 16384
	default:
		return 0
	}
}

func orDefault[T comparable](v, def T) T {
	var zero T
	if v == zero {
		return def
	}
	return v
}
