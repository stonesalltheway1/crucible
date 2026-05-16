package modelrouter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/genai"
)

// GoogleClient wraps google.golang.org/genai v1.57+ (the GA unified SDK).
// IMPORTANT: cloud.google.com/go/vertexai/genai is being removed 2026-06-24;
// do not use it.
type GoogleClient struct {
	api *genai.Client
}

// NewGoogleClient returns a client backed by GOOGLE_API_KEY (or GEMINI_API_KEY).
func NewGoogleClient(ctx context.Context) (*GoogleClient, error) {
	key := os.Getenv("GOOGLE_API_KEY")
	if key == "" {
		key = os.Getenv("GEMINI_API_KEY")
	}
	if key == "" {
		return nil, errors.New("GOOGLE_API_KEY (or GEMINI_API_KEY) not set")
	}
	cli, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  key,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("google: new client: %w", err)
	}
	return &GoogleClient{api: cli}, nil
}

// NewGoogleClientFromEnv returns nil (no error) if the env var is unset.
func NewGoogleClientFromEnv(ctx context.Context) *GoogleClient {
	c, err := NewGoogleClient(ctx)
	if err != nil {
		return nil
	}
	return c
}

func (c *GoogleClient) Vendor() Vendor { return VendorGoogle }

func (c *GoogleClient) Call(ctx context.Context, req Request) (*Response, error) {
	start := time.Now()

	contents := []*genai.Content{}
	for _, m := range req.Messages {
		var role string
		switch m.Role {
		case RoleUser:
			role = "user"
		case RoleAssistant:
			role = "model"
		case RoleSystem:
			continue // collapsed into systemInstruction below
		default:
			return nil, fmt.Errorf("google: unsupported role %q", m.Role)
		}
		contents = append(contents, &genai.Content{
			Role:  role,
			Parts: []*genai.Part{{Text: m.Content}},
		})
	}

	cfg := &genai.GenerateContentConfig{
		MaxOutputTokens: int32(orDefault(req.MaxOutput, 1024)),
	}
	if req.Temperature > 0 {
		t := float32(req.Temperature)
		cfg.Temperature = &t
	}
	if req.System != "" {
		cfg.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: req.System}},
		}
	}
	if req.ThinkingLevel != "" {
		// Gemini 3+: prefer thinking_level over thinking_budget.
		cfg.ThinkingConfig = &genai.ThinkingConfig{
			ThinkingLevel: strings.ToUpper(req.ThinkingLevel), // SDK accepts "LOW"|"MEDIUM"|"HIGH"
		}
	}
	if req.JSONMode || req.JSONSchema != nil {
		cfg.ResponseMIMEType = "application/json"
		if req.JSONSchema != nil {
			cfg.ResponseSchema = jsonSchemaToGenAISchema(req.JSONSchema)
		}
	}

	resp, err := c.api.Models.GenerateContent(ctx, req.Model, contents, cfg)
	if err != nil {
		return nil, fmt.Errorf("google: generate-content: %w", err)
	}

	out := &Response{
		Model:   req.Model,
		Latency: time.Since(start),
	}
	if resp.UsageMetadata != nil {
		out.Usage.InputTokensFresh = int(resp.UsageMetadata.PromptTokenCount)
		out.Usage.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
		out.Usage.ThinkingTokens = int(resp.UsageMetadata.ThoughtsTokenCount)
		out.Usage.CacheReadTokens = int(resp.UsageMetadata.CachedContentTokenCount)
	}
	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		for _, p := range resp.Candidates[0].Content.Parts {
			if p.Text != "" {
				out.Content += p.Text
			}
		}
		out.StopReason = string(resp.Candidates[0].FinishReason)
	}
	out.CacheHit = out.Usage.CacheReadTokens > 0
	return out, nil
}

// jsonSchemaToGenAISchema converts a generic JSON-Schema map to genai.Schema.
// Phase 1 supports the small subset we actually use: object/string/number/array
// plus required + properties. Phase 2 will extend coverage.
func jsonSchemaToGenAISchema(s map[string]any) *genai.Schema {
	if s == nil {
		return nil
	}
	out := &genai.Schema{}
	if t, ok := s["type"].(string); ok {
		switch t {
		case "object":
			out.Type = genai.TypeObject
		case "string":
			out.Type = genai.TypeString
		case "number":
			out.Type = genai.TypeNumber
		case "integer":
			out.Type = genai.TypeInteger
		case "boolean":
			out.Type = genai.TypeBoolean
		case "array":
			out.Type = genai.TypeArray
		}
	}
	if desc, ok := s["description"].(string); ok {
		out.Description = desc
	}
	if req, ok := s["required"].([]any); ok {
		for _, r := range req {
			if str, ok := r.(string); ok {
				out.Required = append(out.Required, str)
			}
		}
	}
	if props, ok := s["properties"].(map[string]any); ok {
		out.Properties = make(map[string]*genai.Schema, len(props))
		for k, v := range props {
			if m, ok := v.(map[string]any); ok {
				out.Properties[k] = jsonSchemaToGenAISchema(m)
			}
		}
	}
	if items, ok := s["items"].(map[string]any); ok {
		out.Items = jsonSchemaToGenAISchema(items)
	}
	if enum, ok := s["enum"].([]any); ok {
		for _, e := range enum {
			if str, ok := e.(string); ok {
				out.Enum = append(out.Enum, str)
			}
		}
	}
	return out
}
