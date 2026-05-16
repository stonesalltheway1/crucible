package console

import (
	"strings"
	"testing"

	"github.com/crucible/apps/cartographer/internal/types"
)

func TestLinesProducesCanonicalOutput(t *testing.T) {
	r := &types.CartographyResult{
		FilesIndexed:                1247,
		Directories:                 38,
		StackPrimary:                "Next.js 14",
		StackSecondary:              []string{"FastAPI 0.110", "PostgreSQL 16"},
		ConventionsFromConfigs:      120,
		ConventionsFromAgentsMD:     50,
		ConventionsFromADRs:         14,
		ConventionsFromOSSDefaults:  312,
		ConventionsFromPRReview:     35,
		ConventionsFromIncidents:    12,
		HighConfidenceCount:         12,
		MediumConfidenceCount:       23,
		LowConfidenceCount:          12,
		HasCustomerOverride:         false,
		WallClockSeconds:            187.4,
		UsdSpent:                    3.42,
	}
	lines := Lines(r)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"Indexed 1,247 files",
		"Detected stack: Next.js 14",
		"FastAPI 0.110, PostgreSQL 16",
		"312 OSS-derived defaults",
		"high-confidence",
		"inferred draft for review",
		"https://app.crucible.dev/memory",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in output:\n%s", want, joined)
		}
	}
}

func TestCommasFormats(t *testing.T) {
	cases := map[int]string{
		0: "0", 5: "5", 999: "999", 1000: "1,000", 12345: "12,345", 1234567: "1,234,567",
	}
	for in, want := range cases {
		if got := commas(in); got != want {
			t.Errorf("commas(%d)=%q want %q", in, got, want)
		}
	}
}
