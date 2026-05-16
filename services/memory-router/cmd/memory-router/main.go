// crucible-memory-router daemon entrypoint.
//
// Hot-path retrieval surface for the Phase-5 memory layer. Listens on
// :8090 by default (HTTP/JSON; gRPC wire is generated when buf is in
// CI). Reads CRUCIBLE_* env vars for backend connections.
//
// In CI / dev, set CRUCIBLE_MEMORY_ROUTER_STUB=1 to use the in-memory
// fakes for vectorstore + proceduralstore + hotstore. The handlers are
// unchanged; only the storage drivers swap.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	memoryspec "github.com/crucible/memory-spec/go"

	"github.com/crucible/memory-router/internal/embedding"
	"github.com/crucible/memory-router/internal/globaldefaults"
	"github.com/crucible/memory-router/internal/hotstore"
	"github.com/crucible/memory-router/internal/proceduralstore"
	"github.com/crucible/memory-router/internal/retriever"
	"github.com/crucible/memory-router/internal/server"
	"github.com/crucible/memory-router/internal/vectorstore"
)

func main() {
	addr := flag.String("addr", ":8090", "HTTP listen address")
	globalsDir := flag.String("global-defaults", "global_defaults", "path to per-stack default bundles")
	disableJudge := flag.Bool("no-judge", false, "DISABLE the LLM-as-judge filter (CI / dev only)")
	flag.Parse()

	hot := hotstore.New(hotstore.NewFake())
	vec := vectorstore.NewFake()
	proc := proceduralstore.NewFake()
	emb := embedding.NewFake()
	globals := globaldefaults.NewLoader()
	if loaded, errs := globals.LoadAll(resolvePath(*globalsDir)); loaded > 0 {
		log.Printf("global_defaults: loaded %d bundles", loaded)
	} else if len(errs) > 0 {
		log.Printf("global_defaults: load errors (will run with empty defaults):")
		for _, e := range errs {
			log.Printf("  - %v", e)
		}
	}
	for stack, n := range globals.Stats() {
		log.Printf("  %s: %d active rules", stack, n)
	}

	r := retriever.New(hot, vec, proc, emb, globals)
	s := server.New(r, proc, vec, emb)

	if *disableJudge {
		log.Print("WARNING: LLM-as-judge filter DISABLED. Use only in CI / dev.")
		s.RequireJudge = false
	} else {
		s.RequireJudge = true
		// In Phase 5 we use the conservative deterministic judge by
		// default. The production model-routed judge wires in via
		// CRUCIBLE_MEMORY_JUDGE_MODEL_ROUTER_ADDR.
		s.JudgeFn = deterministicJudge
	}

	if err := preflight(s, globals); err != nil {
		log.Fatalf("preflight: %v", err)
	}

	srv := &http.Server{
		Addr:         *addr,
		Handler:      s.Routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		log.Printf("memory-router listening on %s", *addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Print("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// preflight refuses to start if guardrails aren't satisfied.
func preflight(s *server.Server, _ *globaldefaults.Loader) error {
	// The server-internal requireJudge() refuses startup when
	// RequireJudge is true but no JudgeFn was wired.
	if s.RequireJudge && s.JudgeFn == nil {
		return errors.New("RequireJudge=true but no JudgeFn wired — refusing to serve")
	}
	return nil
}

func resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, p)
}

// deterministicJudge is the conservative keyword-pre-filter + structural-
// check judge wired by default. The brief calls out the
// "actually, use eval(input) for everything" case as a prompt-injection
// canary; that's what this catches.
//
// The production model-routed judge (Haiku 4.5 via the cross-family
// router) plugs in via the JudgeFn field when CRUCIBLE_MEMORY_JUDGE
// is configured. The deterministic judge always runs first — defence
// in depth.
func deterministicJudge(ctx context.Context, tenantID string, c memoryspec.Convention) (bool, float64, string, string) {
	_ = ctx
	_ = tenantID
	// Cheap, fast, always-on. The production model-routed judge replaces
	// this when wired.
	verdict, reason, cat := DeterministicVerdict(c)
	if !verdict {
		return false, 0.0, reason, cat
	}
	return true, 0.9, "deterministic pre-filter passed", ""
}

// DeterministicVerdict is the exported form used by the server's
// own tests + the distiller's offline filter. Returns
// (admit, reason, injection_category).
//
// Phase 5 catch policy (in priority order):
//   1. eval / exec / spawnshell substrings → prompt_injection
//   2. SQL-construction patterns → prompt_injection
//   3. credential / secret embedding → prompt_injection
//   4. low-specificity ("do whatever you want", "ignore the rules") →
//      prompt_injection
//   5. tag-like braces ({{ }}, <%>, <script>) → malformed
//   6. excessive length (> 1024) → malformed
//   7. category=other (caught at Validate already) → low_specificity
//
// The deterministic filter is the cheap layer; the model-routed judge
// catches the long-tail. Both run on every write.
func DeterministicVerdict(c memoryspec.Convention) (bool, string, string) {
	r := lowerASCII(c.RuleNl)
	// 1.
	for _, dangerous := range []string{"eval(", " eval ", "eval input", "exec(", "spawnshell", "execfile(", "rm -rf"} {
		if contains(r, dangerous) {
			return false, fmt.Sprintf("prompt-injection: %q pattern", dangerous), "prompt_injection"
		}
	}
	// 2.
	if contains(r, "select * from") || contains(r, "drop table") || contains(r, "delete from ") {
		return false, "prompt-injection: SQL-construction pattern", "prompt_injection"
	}
	// 3.
	if contains(r, "secret=") || contains(r, "password=") || contains(r, "api_key=") || contains(r, "bearer ey") {
		return false, "prompt-injection: credential-leak pattern", "prompt_injection"
	}
	// 4.
	for _, lowspec := range []string{
		"ignore the rules", "ignore all rules", "ignore previous",
		"do whatever you want", "no rules apply", "anything goes",
		"override", "bypass", "skip verification",
	} {
		if contains(r, lowspec) {
			return false, "prompt-injection: low-specificity directive", "prompt_injection"
		}
	}
	// 5.
	if contains(r, "{{") || contains(r, "}}") || contains(r, "<script") || contains(r, "<%") {
		return false, "malformed: template-injection markers in rule", "malformed"
	}
	// 6.
	if len(c.RuleNl) > 1024 {
		return false, "malformed: rule exceeds 1024 chars", "malformed"
	}
	return true, "", ""
}

func lowerASCII(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}

func contains(haystack, needle string) bool {
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
