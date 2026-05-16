package embedding

import (
	"context"
	"errors"
	"testing"
)

func TestEnforceSingleTenant_RefusesCrossTenant(t *testing.T) {
	err := EnforceSingleTenant([]Request{
		{TenantID: "ten_a", Content: "x"},
		{TenantID: "ten_b", Content: "y"},
	})
	if !errors.Is(err, ErrCrossTenantBatch) {
		t.Fatalf("want ErrCrossTenantBatch; got %v", err)
	}
}

func TestEnforceSingleTenant_RefusesEmpty(t *testing.T) {
	err := EnforceSingleTenant([]Request{{Content: "x"}})
	if !errors.Is(err, ErrEmptyTenant) {
		t.Fatalf("want ErrEmptyTenant; got %v", err)
	}
}

func TestEnforceSingleTenant_OkSingleTenant(t *testing.T) {
	err := EnforceSingleTenant([]Request{
		{TenantID: "ten_a", Content: "x"},
		{TenantID: "ten_a", Content: "y"},
	})
	if err != nil {
		t.Fatalf("single tenant batch should pass; got %v", err)
	}
}

func TestEnforceSingleTenant_EmptyBatchOk(t *testing.T) {
	if err := EnforceSingleTenant(nil); err != nil {
		t.Fatalf("empty batch must be ok; got %v", err)
	}
}

func TestHashContent_TenantFoldedIn(t *testing.T) {
	a := HashContent("ten_a", "hello")
	b := HashContent("ten_b", "hello")
	if a == b {
		t.Fatal("same content across tenants must yield distinct embed-cache keys")
	}
}

func TestFakeClient_RefusesCrossTenant(t *testing.T) {
	c := NewFake()
	_, err := c.Embed(context.Background(), []Request{
		{TenantID: "ten_a", Content: "x"},
		{TenantID: "ten_b", Content: "y"},
	})
	if !errors.Is(err, ErrCrossTenantBatch) {
		t.Fatalf("fake must refuse cross-tenant; got %v", err)
	}
}

func TestFakeClient_DeterministicOutput(t *testing.T) {
	c := NewFake()
	out1, _ := c.Embed(context.Background(), []Request{{TenantID: "ten_a", Content: "hello"}})
	out2, _ := c.Embed(context.Background(), []Request{{TenantID: "ten_a", Content: "hello"}})
	if len(out1) != 1 || len(out2) != 1 {
		t.Fatal("expected one vector each")
	}
	if out1[0][0] != out2[0][0] || out1[0][1024] != out2[0][1024] {
		t.Fatal("fake should be deterministic per (tenant, content)")
	}
}

func TestFakeClient_TenantChangesOutput(t *testing.T) {
	c := NewFake()
	a, _ := c.Embed(context.Background(), []Request{{TenantID: "ten_a", Content: "x"}})
	b, _ := c.Embed(context.Background(), []Request{{TenantID: "ten_b", Content: "x"}})
	if a[0][0] == b[0][0] && a[0][512] == b[0][512] && a[0][2048] == b[0][2048] {
		t.Fatal("different tenants should produce different fake vectors for the same content")
	}
}

func TestFakeClient_Dim3072(t *testing.T) {
	c := NewFake()
	if c.Dimension() != 3072 {
		t.Fatalf("dim must be 3072 (text-embedding-3-large); got %d", c.Dimension())
	}
	out, _ := c.Embed(context.Background(), []Request{{TenantID: "ten_a", Content: "x"}})
	if len(out[0]) != Dim {
		t.Fatalf("vec length must equal Dim; got %d", len(out[0]))
	}
}
