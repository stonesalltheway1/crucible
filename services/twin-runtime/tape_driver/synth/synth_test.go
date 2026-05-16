package synth

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func ptrFloat(v float64) *float64 { return &v }
func ptrInt(v int) *int           { return &v }

func newGen(t *testing.T, specs ...EndpointSpec) *Generator {
	t.Helper()
	return New(Options{Specs: specs, FakerSeed: 0xC0DE})
}

func TestSynthesizeReadOnlyReturnsBodyMatchingSchema(t *testing.T) {
	spec := EndpointSpec{
		Service:  "stripe",
		Endpoint: "/v1/customers/{id}",
		Method:   MethodGET,
		ResponseSchema: SchemaNode{
			Type: "object",
			Properties: map[string]SchemaNode{
				"id":      {Type: "string", Pattern: "^cus_"},
				"email":   {Type: "string", Format: "email"},
				"created": {Type: "integer", Minimum: ptrFloat(1500000000), Maximum: ptrFloat(2000000000)},
			},
			Required: []string{"id", "email"},
		},
	}
	g := newGen(t, spec)
	r, err := g.Synthesize(context.Background(), Request{
		Service:  "stripe",
		Method:   MethodGET,
		Endpoint: "/v1/customers/{id}",
		PathParams: map[string]string{"id": "cus_abc"},
	})
	require.NoError(t, err)
	require.Equal(t, DispoSynthReadOnly, r.Disposition)
	require.Equal(t, "synth-readonly", r.Headers["X-Crucible-Tape"])
	require.Equal(t, "schema", r.Provenance.Engine)
	var body map[string]any
	require.NoError(t, json.Unmarshal(r.Body, &body))
	require.NotEmpty(t, body["id"])
	require.Equal(t, "synth.user@example.com", body["email"])
}

func TestSynthesizeMutationStoresAndReads(t *testing.T) {
	spec := EndpointSpec{
		Service:  "stripe",
		Endpoint: "/v1/customers",
		Method:   MethodPOST,
		ResponseSchema: SchemaNode{
			Type: "object",
			Properties: map[string]SchemaNode{
				"id":    {Type: "string", Pattern: "^cus_"},
				"email": {Type: "string", Format: "email"},
			},
			Required: []string{"id", "email"},
		},
		SuccessExample: json.RawMessage(`{"id":"cus_synth1","email":"e@example.com"}`),
	}
	getSpec := EndpointSpec{
		Service:  "stripe",
		Endpoint: "/v1/customers/{id}",
		Method:   MethodGET,
		ResponseSchema: SchemaNode{
			Type: "object",
			Properties: map[string]SchemaNode{
				"id":    {Type: "string"},
				"email": {Type: "string", Format: "email"},
			},
		},
	}
	g := newGen(t, spec, getSpec)
	// Write
	w, err := g.Synthesize(context.Background(), Request{
		Service:  "stripe",
		Method:   MethodPOST,
		Endpoint: "/v1/customers",
	})
	require.NoError(t, err)
	require.Equal(t, DispoSynthMutation, w.Disposition)
	require.Equal(t, "openapi-example", w.Provenance.Engine)
	require.Contains(t, string(w.Body), `"cus_synth1"`)

	// Read by id from journal
	r, err := g.Synthesize(context.Background(), Request{
		Service:    "stripe",
		Method:     MethodGET,
		Endpoint:   "/v1/customers/{id}",
		PathParams: map[string]string{"id": "cus_synth1"},
	})
	require.NoError(t, err)
	require.Equal(t, DispoSynthReadOnly, r.Disposition)
	require.True(t, r.Provenance.StateJournal)
	require.Contains(t, string(r.Body), `"cus_synth1"`)
}

func TestSynthesizeReturnsErrNoSpec(t *testing.T) {
	g := newGen(t)
	_, err := g.Synthesize(context.Background(), Request{
		Service:  "unknown",
		Endpoint: "/x",
		Method:   MethodGET,
	})
	require.ErrorIs(t, err, ErrNoSpec)
}

func TestSynthesizeListEndpointReturnsJournalledEntries(t *testing.T) {
	postSpec := EndpointSpec{
		Service:  "stripe",
		Endpoint: "/v1/charges",
		Method:   MethodPOST,
		ResponseSchema: SchemaNode{Type: "object"},
		SuccessExample: json.RawMessage(`{"id":"ch_a","amount":100}`),
	}
	listSpec := EndpointSpec{
		Service:        "stripe",
		Endpoint:       "/v1/charges",
		Method:         MethodGET,
		ResponseSchema: SchemaNode{Type: "object"},
	}
	g := newGen(t, postSpec, listSpec)
	for _, id := range []string{"ch_a", "ch_b", "ch_c"} {
		req := Request{
			Service:  "stripe",
			Method:   MethodPOST,
			Endpoint: "/v1/charges",
		}
		postSpec.SuccessExample = json.RawMessage(`{"id":"` + id + `","amount":100}`)
		g.specs[routeKey(postSpec.Service, postSpec.Method, postSpec.Endpoint)] = postSpec
		_, err := g.Synthesize(context.Background(), req)
		require.NoError(t, err)
	}
	r, err := g.Synthesize(context.Background(), Request{
		Service:  "stripe",
		Method:   MethodGET,
		Endpoint: "/v1/charges",
	})
	require.NoError(t, err)
	require.True(t, r.Provenance.StateJournal)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(r.Body, &parsed))
	data := parsed["data"].([]any)
	require.Len(t, data, 3)
}

func TestStateJournalRecordAndLookup(t *testing.T) {
	j := NewStateJournal(nil)
	j.RecordWrite(Request{
		Service: "s",
		Method:  MethodPOST,
		Endpoint: "/v1/things",
	}, json.RawMessage(`{"id":"thing_1","name":"a"}`))
	body, ok := j.LookupRead(Request{
		Service:    "s",
		Method:     MethodGET,
		Endpoint:   "/v1/things/{id}",
		PathParams: map[string]string{"id": "thing_1"},
	})
	require.True(t, ok)
	require.JSONEq(t, `{"id":"thing_1","name":"a"}`, string(body))
}

func TestSchemaWalkerDeterministicWithSeed(t *testing.T) {
	spec := EndpointSpec{
		Service:  "s",
		Method:   MethodGET,
		Endpoint: "/x",
		ResponseSchema: SchemaNode{
			Type: "object",
			Properties: map[string]SchemaNode{
				"name":  {Type: "string", MinLength: ptrInt(5), MaxLength: ptrInt(8)},
				"age":   {Type: "integer", Minimum: ptrFloat(0), Maximum: ptrFloat(120)},
				"score": {Type: "number", Minimum: ptrFloat(0), Maximum: ptrFloat(1)},
			},
			Required: []string{"name", "age", "score"},
		},
	}
	a := newGen(t, spec)
	b := newGen(t, spec)
	r1, _ := a.Synthesize(context.Background(), Request{Service: "s", Method: MethodGET, Endpoint: "/x"})
	r2, _ := b.Synthesize(context.Background(), Request{Service: "s", Method: MethodGET, Endpoint: "/x"})
	require.Equal(t, string(r1.Body), string(r2.Body))
}

type fakeLLM struct{ value string }

func (f *fakeLLM) Augment(ctx context.Context, h AugmentationHint) (json.RawMessage, error) {
	raw, _ := json.Marshal(f.value)
	return raw, nil
}

func TestLLMAugmenterEnrichesFreeText(t *testing.T) {
	spec := EndpointSpec{
		Service:  "s",
		Method:   MethodGET,
		Endpoint: "/x",
		ResponseSchema: SchemaNode{
			Type: "object",
			Properties: map[string]SchemaNode{
				"description": {Type: "string", Description: "a one-line description"},
			},
			Required: []string{"description"},
		},
	}
	g := New(Options{
		Specs:     []EndpointSpec{spec},
		Augmenter: &fakeLLM{value: "synthetic free-text from LLM"},
		FakerSeed: 1,
	})
	r, _ := g.Synthesize(context.Background(), Request{Service: "s", Method: MethodGET, Endpoint: "/x"})
	require.Equal(t, "schema+llm", r.Provenance.Engine)
	require.Contains(t, string(r.Body), "synthetic free-text from LLM")
}

func TestOpenAPIParseProducesEndpointSpecs(t *testing.T) {
	doc := `{
      "openapi": "3.1.0",
      "info": {"title": "stripe-sim", "version": "1.0.0"},
      "paths": {
        "/v1/customers/{id}": {
          "get": {
            "operationId": "getCustomer",
            "responses": {
              "200": {
                "description": "ok",
                "content": {
                  "application/json": {
                    "schema": {"$ref": "#/components/schemas/Customer"}
                  }
                }
              }
            }
          }
        }
      },
      "components": {
        "schemas": {
          "Customer": {
            "type": "object",
            "properties": {
              "id": {"type": "string"},
              "email": {"type": "string", "format": "email"}
            }
          }
        }
      }
    }`
	specs, err := ParseOpenAPI([]byte(doc), "")
	require.NoError(t, err)
	require.Len(t, specs, 1)
	require.Equal(t, "stripe-sim", specs[0].Service)
	require.Equal(t, MethodGET, specs[0].Method)
	require.Equal(t, "object", specs[0].ResponseSchema.Type)
	require.Contains(t, specs[0].ResponseSchema.Properties, "email")
}

func TestEndpointSpecsSortDeterministically(t *testing.T) {
	specs := []EndpointSpec{
		{Service: "b", Method: MethodGET, Endpoint: "/x"},
		{Service: "a", Method: MethodGET, Endpoint: "/y"},
		{Service: "a", Method: MethodGET, Endpoint: "/x"},
	}
	SortSpecs(specs)
	require.Equal(t, "a", specs[0].Service)
	require.Equal(t, "/x", specs[0].Endpoint)
	require.Equal(t, "a", specs[1].Service)
	require.Equal(t, "b", specs[2].Service)
}
