package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crucible/apps/cartographer/internal/distill"
	"github.com/crucible/apps/cartographer/internal/oss"
	"github.com/crucible/apps/cartographer/internal/types"
)

func writeFile(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunEndToEndOnTinyRepo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), `{"dependencies": {"next": "14.0.0"}}`)
	writeFile(t, filepath.Join(root, "tsconfig.json"), `{"compilerOptions": {"strict": true}}`)
	writeFile(t, filepath.Join(root, "src", "webhook.ts"), "export function handle_webhook() {}")
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# Naming\n- Use snake_case for db column names\n")
	writeFile(t, filepath.Join(root, ".editorconfig"), "[*]\nindent_style=space\nmax_line_length=120\n")

	stages := []string{}
	job := types.CartographyJob{
		JobID: "j1", TenantID: "ten_x", Repo: "acme/payments",
		RepoLocalPath: root,
	}
	res, err := Run(context.Background(), job, Deps{
		LLM: distill.NewClient(distill.Config{}),
		OSS: nil,
	}, func(stage string, frac float64) { stages = append(stages, stage) })
	if err != nil {
		t.Fatal(err)
	}
	if res.FilesIndexed == 0 {
		t.Error("no files indexed")
	}
	if res.StackPrimary != "nextjs" {
		t.Errorf("stack=%q want nextjs", res.StackPrimary)
	}
	if res.ConventionsFromConfigs == 0 {
		t.Error("no config conventions")
	}
	if !res.HasCustomerOverride {
		t.Error("override not detected")
	}
	if res.CustomerOverridePath != "AGENTS.md" {
		t.Errorf("override path %q", res.CustomerOverridePath)
	}
	// inferred AGENTS.md should NOT be generated since one exists.
	if res.InferredAgentsMDMarkdown != "" {
		t.Error("inferred should be empty when override exists")
	}
	if len(res.FirstTaskSuggestions) == 0 {
		t.Error("no first-task suggestions")
	}
	if len(res.ConsoleOutputLines) == 0 {
		t.Error("no console lines")
	}
	if len(stages) == 0 || stages[len(stages)-1] != "done" {
		t.Errorf("stages did not end with done: %v", stages)
	}
}

func TestRunGeneratesInferredWhenNoOverride(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/x\n")
	writeFile(t, filepath.Join(root, ".editorconfig"), "[*]\nindent_style=tab\n")

	job := types.CartographyJob{
		JobID: "j2", TenantID: "ten_y", Repo: "acme/svc",
		RepoLocalPath: root,
	}
	res, err := Run(context.Background(), job, Deps{LLM: distill.NewClient(distill.Config{})}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.HasCustomerOverride {
		t.Error("override should not be detected")
	}
	if !strings.Contains(res.InferredAgentsMDMarkdown, "AGENTS.md") {
		t.Error("inferred AGENTS.md not generated")
	}
}

func TestRunWiresOSSDefaults(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module x")

	ossRoot := t.TempDir()
	_ = os.MkdirAll(filepath.Join(ossRoot, "go-services"), 0o755)
	body := `[{"category":"Logging","rule_nl":"Use slog with structured key-value pairs","file_glob":"**/*.go","confidence":0.85,"status":"active"}]`
	if err := os.WriteFile(filepath.Join(ossRoot, "go-services", "conventions.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	loader, _ := oss.NewLoader(ossRoot)

	job := types.CartographyJob{
		JobID: "j3", TenantID: "ten_z", Repo: "acme/api",
		RepoLocalPath: root,
	}
	res, err := Run(context.Background(), job, Deps{
		OSS: loader,
		LLM: distill.NewClient(distill.Config{}),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.ConventionsFromOSSDefaults == 0 {
		t.Errorf("expected OSS defaults loaded, got %d", res.ConventionsFromOSSDefaults)
	}
}
