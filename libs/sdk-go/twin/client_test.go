package twin

import (
	"context"
	"strings"
	"testing"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func TestNewClientRequiresTaskID(t *testing.T) {
	_, err := NewClient(Config{})
	if err == nil {
		t.Fatal("expected error for empty TaskID")
	}
}

func TestStubClientWriteThenRead(t *testing.T) {
	c := NewStubClient("task_test")
	ctx := context.Background()
	att, err := c.FsWrite(ctx, "src/main.rs", "fn main() {}", "step_1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(att.AttestationID, "stub:") {
		t.Fatalf("attestation id missing prefix: %s", att.AttestationID)
	}
	got, err := c.FsRead(ctx, "src/main.rs")
	if err != nil {
		t.Fatal(err)
	}
	if got != "fn main() {}" {
		t.Fatalf("read got %q", got)
	}
}

func TestStubClientDeleteRemoves(t *testing.T) {
	c := NewStubClient("task_test")
	ctx := context.Background()
	_, _ = c.FsWrite(ctx, "x.txt", "hello", "")
	_, err := c.FsDelete(ctx, "x.txt", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.FsRead(ctx, "x.txt")
	if err == nil {
		t.Fatal("expected not-found after delete")
	}
}

func TestStubShellExecReturnsResult(t *testing.T) {
	c := NewStubClient("task_test")
	out, err := c.ShellExec(context.Background(), ShellExecOpts{Cmd: "echo hi"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Result == nil || out.Result.ExitCode != 0 {
		t.Fatalf("unexpected: %+v", out)
	}
}

func TestStubSecretGetReturnsHandleNotValue(t *testing.T) {
	c := NewStubClient("task_test")
	ref, err := c.SecretGet(context.Background(), "stripe")
	if err != nil {
		t.Fatal(err)
	}
	// The stub returns "stub-handle" — not a value. The invariant is that
	// no SDK call surfaces a raw value; the egress proxy substitutes.
	if ref.Handle == "" {
		t.Fatal("expected non-empty handle")
	}
	if strings.Contains(ref.Handle, "actual-value") {
		t.Fatal("handle should not contain a real secret value")
	}
	// Sanity: SecretRef shape is preserved.
	_ = cruciblev1.SecretRef{}
}

func TestStubSvcCallSetsTapeHeader() {
	// Compile-only sanity — see TestStubSvcCallSetsTapeHeaderT below.
}

func TestStubSvcCallSetsTapeHeaderT(t *testing.T) {
	c := NewStubClient("task_test")
	resp, err := c.SvcCall(context.Background(), SvcCallRequest{Service: "stripe", Endpoint: "/v1/charges"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TapeDispoHdr == "" {
		t.Fatal("expected X-Crucible-Tape header on every response")
	}
}
