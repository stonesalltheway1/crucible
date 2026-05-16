package synth

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// OpenAPIDocument is the minimal subset of OpenAPI 3.x we parse to drive
// the synthesizer. Customers upload their full spec to Crucible on
// onboarding; we parse it once into a list of EndpointSpec at recorder
// boot.
//
// Phase 3 does NOT implement a complete OpenAPI 3.1 parser — that's a
// vendored library's job. We accept only what we need to generate
// responses and leave the rest to follow-up work. The parser is forgiving:
// unknown keys are ignored, $ref is resolved one hop deep, and components
// are inlined.
type OpenAPIDocument struct {
	OpenAPI    string                            `json:"openapi"`
	Info       OpenAPIInfo                       `json:"info"`
	Servers    []OpenAPIServer                   `json:"servers"`
	Paths      map[string]map[string]OpenAPIOp   `json:"paths"`
	Components OpenAPIComponents                 `json:"components"`
}

// OpenAPIInfo is the spec's info block. Crucible uses Info.Title as the
// service name unless the document declares an `x-crucible-service` field.
type OpenAPIInfo struct {
	Title      string          `json:"title"`
	Version    string          `json:"version"`
	XCrucible  json.RawMessage `json:"x-crucible-service,omitempty"`
}

// OpenAPIServer is the bound base URL. Only the first server is used; the
// host slug becomes part of the service name if Info.Title is empty.
type OpenAPIServer struct {
	URL string `json:"url"`
}

// OpenAPIOp is one method on a path.
type OpenAPIOp struct {
	Summary     string                       `json:"summary"`
	Description string                       `json:"description"`
	OperationID string                       `json:"operationId"`
	Responses   map[string]OpenAPIResponse   `json:"responses"`
}

// OpenAPIResponse is one status-code keyed response.
type OpenAPIResponse struct {
	Description string                          `json:"description"`
	Content     map[string]OpenAPIMediaType     `json:"content"`
}

// OpenAPIMediaType is a content-type keyed response body.
type OpenAPIMediaType struct {
	Schema  json.RawMessage   `json:"schema"`
	Example json.RawMessage   `json:"example,omitempty"`
}

// OpenAPIComponents holds reusable schemas.
type OpenAPIComponents struct {
	Schemas map[string]json.RawMessage `json:"schemas"`
}

// ParseOpenAPI returns the list of EndpointSpec extracted from the document.
// The serviceName argument is the explicit service name to assign; if empty,
// we use Info.Title.
func ParseOpenAPI(raw []byte, serviceName string) ([]EndpointSpec, error) {
	var doc OpenAPIDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("openapi: parse: %w", err)
	}
	if doc.OpenAPI == "" {
		return nil, errors.New("openapi: missing openapi version field")
	}
	if serviceName == "" {
		serviceName = doc.Info.Title
	}
	if serviceName == "" && len(doc.Servers) > 0 {
		serviceName = doc.Servers[0].URL
	}
	if serviceName == "" {
		return nil, errors.New("openapi: cannot determine service name")
	}
	resolver := newRefResolver(doc.Components.Schemas)
	out := make([]EndpointSpec, 0, len(doc.Paths)*3)
	for path, ops := range doc.Paths {
		for methodStr, op := range ops {
			method := Method(strings.ToUpper(methodStr))
			status, resp, ok := pickSuccessResponse(op.Responses)
			if !ok {
				continue
			}
			schemaRaw, exampleRaw, ok := pickJSONContent(resp.Content)
			if !ok {
				continue
			}
			node, err := resolver.parseSchema(schemaRaw)
			if err != nil {
				return nil, fmt.Errorf("openapi: %s %s: %w", methodStr, path, err)
			}
			out = append(out, EndpointSpec{
				Service:        serviceName,
				Endpoint:       path,
				Method:         method,
				ResponseSchema: node,
				SuccessStatus:  status,
				SuccessExample: exampleRaw,
			})
		}
	}
	SortSpecs(out)
	return out, nil
}

func pickSuccessResponse(
	responses map[string]OpenAPIResponse,
) (int, OpenAPIResponse, bool) {
	if r, ok := responses["200"]; ok {
		return 200, r, true
	}
	if r, ok := responses["201"]; ok {
		return 201, r, true
	}
	if r, ok := responses["204"]; ok {
		return 204, r, true
	}
	if r, ok := responses["default"]; ok {
		return 200, r, true
	}
	for code, r := range responses {
		if n, err := strconv.Atoi(code); err == nil && n >= 200 && n < 300 {
			return n, r, true
		}
	}
	return 0, OpenAPIResponse{}, false
}

func pickJSONContent(
	content map[string]OpenAPIMediaType,
) (json.RawMessage, json.RawMessage, bool) {
	if c, ok := content["application/json"]; ok && len(c.Schema) > 0 {
		return c.Schema, c.Example, true
	}
	for ctype, c := range content {
		if strings.Contains(ctype, "json") && len(c.Schema) > 0 {
			return c.Schema, c.Example, true
		}
	}
	return nil, nil, false
}

// ──────────────────────────────────────────────────────────────────────
// $ref resolution
// ──────────────────────────────────────────────────────────────────────

type refResolver struct {
	components map[string]json.RawMessage
	cache      map[string]SchemaNode
}

func newRefResolver(components map[string]json.RawMessage) *refResolver {
	return &refResolver{components: components, cache: make(map[string]SchemaNode)}
}

func (r *refResolver) parseSchema(raw json.RawMessage) (SchemaNode, error) {
	if len(raw) == 0 {
		return SchemaNode{}, nil
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return SchemaNode{}, err
	}
	if refRaw, ok := probe["$ref"]; ok {
		var ref string
		if err := json.Unmarshal(refRaw, &ref); err == nil {
			return r.resolveRef(ref)
		}
	}
	var node SchemaNode
	if err := json.Unmarshal(raw, &node); err != nil {
		return SchemaNode{}, err
	}
	// Recursively resolve $ref within properties / items.
	if node.Items != nil {
		// If the items block has a ref, resolve to a new node.
		var itemsProbe map[string]json.RawMessage
		if itemsRaw, ok := probe["items"]; ok {
			if err := json.Unmarshal(itemsRaw, &itemsProbe); err == nil {
				if refRaw, ok := itemsProbe["$ref"]; ok {
					var ref string
					if err := json.Unmarshal(refRaw, &ref); err == nil {
						resolved, err := r.resolveRef(ref)
						if err == nil {
							node.Items = &resolved
						}
					}
				}
			}
		}
	}
	if len(node.Properties) > 0 {
		// Resolve property refs.
		var propsRaw map[string]json.RawMessage
		if rawProps, ok := probe["properties"]; ok {
			if err := json.Unmarshal(rawProps, &propsRaw); err == nil {
				for name, propRaw := range propsRaw {
					child, err := r.parseSchema(propRaw)
					if err != nil {
						continue
					}
					node.Properties[name] = child
				}
			}
		}
	}
	return node, nil
}

func (r *refResolver) resolveRef(ref string) (SchemaNode, error) {
	if cached, ok := r.cache[ref]; ok {
		return cached, nil
	}
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return SchemaNode{}, fmt.Errorf("openapi: unresolvable $ref %q", ref)
	}
	name := strings.TrimPrefix(ref, prefix)
	raw, ok := r.components[name]
	if !ok {
		return SchemaNode{}, fmt.Errorf("openapi: missing component %q", name)
	}
	node, err := r.parseSchema(raw)
	if err != nil {
		return SchemaNode{}, err
	}
	r.cache[ref] = node
	return node, nil
}
