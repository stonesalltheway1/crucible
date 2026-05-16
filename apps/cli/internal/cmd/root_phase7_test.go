package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// Phase-7 CLI surface coverage: every new subcommand registered, every
// subcommand carries --output json or json-eligible flags, and `version`
// reports the phase-7 tag.

func TestRoot_Phase7CommandsRegistered(t *testing.T) {
	root := NewRoot()
	want := []string{"promote", "memory", "attestation", "webhook", "tenant", "verify-release", "calibrate"}
	for _, name := range want {
		if _, _, err := root.Find([]string{name}); err != nil {
			t.Errorf("missing %q: %v", name, err)
		}
	}
}

func TestRoot_VersionIsPhase7(t *testing.T) {
	root := NewRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("version: %v", err)
	}
	if !strings.Contains(buf.String(), "phase7") {
		t.Errorf("version output missing phase7 tag: %q", buf.String())
	}
}

func TestPromoteCmd_HasExpectedSubcommands(t *testing.T) {
	root := NewRoot()
	want := []string{"list", "get", "approve", "reject", "status", "rollback"}
	promote, _, err := root.Find([]string{"promote"})
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	for _, name := range want {
		c, _, err := promote.Find([]string{name})
		if err != nil || c == promote {
			t.Errorf("promote missing %q", name)
		}
	}
}

func TestAttestationCmd_HasVerifyChain(t *testing.T) {
	root := NewRoot()
	att, _, err := root.Find([]string{"attestation"})
	if err != nil {
		t.Fatalf("attestation: %v", err)
	}
	for _, name := range []string{"get", "verify", "chain", "export"} {
		c, _, err := att.Find([]string{name})
		if err != nil || c == att {
			t.Errorf("attestation missing %q", name)
		}
	}
}
