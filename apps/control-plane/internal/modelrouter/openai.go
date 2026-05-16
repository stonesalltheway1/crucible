package modelrouter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

// OpenAIClient wraps github.com/openai/openai-go/v3 v3.35+ using the new
// Responses API (Chat Completions still works but is no longer the recommended
// path for new builds per OpenAI's migration guide).
type OpenAIClient struct {
	api openai.Client
}

// NewOpenAIClient returns a client backed by OPENAI_API_KEY.
func NewOpenAIClient() (*OpenAIClient, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, errors.New("OPENAI_API_KEY not set")
	}
	return &OpenAIClient{api: openai.NewClient(option.WithAPIKey(key))}, nil
}

// NewOpenAIClientFromEnv returns nil if the env var is unset.
func NewOpenAIClientFromEnv() *OpenAIClient {
	c, err := NewOpenAIClient()
	if err != nil {
		return nil
	}
	return c
}

func (c *OpenAIClient) Vendor() Vendor { return VendorOpenAI }

func (c *OpenAIClient) Call(ctx context.Context, req Request) (*Response, error) {
	start := time.Now()

	// Responses API takes `input` (string or array of content), not `messages`.
	// For Phase 1, concatenate the chat into structured input items.
	items := []responses.ResponseInputItemUnionParam{}
	for _, m := range req.Messages {
		var role responses.EasyInputMessageRole
		switch m.Role {
		case RoleUser:
			role = responses.EasyInputMessageRoleUser
		case RoleAssistant:
			role = responses.EasyInputMessageRoleAssistant
		case RoleSystem:
			role = responses.EasyInputMessageRoleSystem
		default:
			return nil, fmt.Errorf("openai: unsupported role %q", m.Role)
		}
		items = append(items, responses.ResponseInputItemParamOfMessage(m.Content, role))
	}
	if req.System != "" {
		items = append([]responses.ResponseInputItemUnionParam{
			responses.ResponseInputItemParamOfMessage(req.System, responses.EasyInputMessageRoleSystem),
		}, items...)
	}

	params := responses.ResponseNewParams{
		Model: openai.ResponsesModel(req.Model),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: items,
		},
		MaxOutputTokens: openai.Int(int64(orDefault(req.MaxOutput, 1024))),
		// Phase 1: keep storage off so we don't accumulate state by default.
		Store: openai.Bool(false),
	}
	if req.Temperature > 0 {
		params.Temperature = openai.Float(req.Temperature)
	}
	if req.ThinkingLevel != "" {
		// OpenAI Responses uses "reasoning.effort".
		params.Reasoning = openai.ReasoningParam{
			Effort: openai.ReasoningEffort(req.ThinkingLevel),
		}
	}

	resp, err := c.api.Responses.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openai: responses.new: %w", err)
	}

	out := &Response{
		Model:         req.Model,
		Content:       resp.OutputText(),
		Latency:       time.Since(start),
		RawProviderID: resp.ID,
		Usage: Usage{
			InputTokensFresh: int(resp.Usage.InputTokens),
			OutputTokens:     int(resp.Usage.OutputTokens),
		},
	}
	if resp.Usage.InputTokensDetails.CachedTokens > 0 {
		out.Usage.CacheReadTokens = int(resp.Usage.InputTokensDetails.CachedTokens)
		// Responses API counts cached against fresh; subtract so totals add up.
		out.Usage.InputTokensFresh -= out.Usage.CacheReadTokens
		out.CacheHit = true
	}
	return out, nil
}
