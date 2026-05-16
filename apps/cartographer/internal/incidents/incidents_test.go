package incidents

import (
	"strings"
	"testing"
)

func TestDetectFindsLinearAndINC(t *testing.T) {
	body := "fix from https://linear.app/acme/issue/PAY-99 and INC-1234"
	got := Detect("repo", "ten", "pr/1", body)
	if len(got) != 2 {
		t.Fatalf("got %d", len(got))
	}
	combined := got[0].EvidenceQuote + "|" + got[1].EvidenceQuote
	if !strings.Contains(combined, "PAY-99") || !strings.Contains(combined, "INC-1234") {
		t.Errorf("missing refs: %v", combined)
	}
}

func TestDetectEmpty(t *testing.T) {
	got := Detect("r", "t", "p", "no incidents here")
	if len(got) != 0 {
		t.Errorf("got %d", len(got))
	}
}
