// Package types defines the wire and internal types for cartographer.
//
// We keep these structs free of dependencies on the rest of the
// service so they're trivially serialisable and reusable from
// callers (control-plane onboarding, web console, CLI).
package types

import "time"

// CartographyJob is what the control plane submits.
type CartographyJob struct {
	JobID                string    `json:"job_id"`
	TenantID             string    `json:"tenant_id"`
	Repo                 string    `json:"repo"`
	RepoLocalPath        string    `json:"repo_local_path"`
	StackHint            string    `json:"stack_hint,omitempty"`
	IncludePRHistory     bool      `json:"include_pr_history"`
	PRHistoryMonths      int       `json:"pr_history_months,omitempty"` // default 24
	PRHistoryMaxComments int       `json:"pr_history_max_comments,omitempty"` // default 1000
	GitHubTokenSecretRef string    `json:"github_token_secret_ref,omitempty"`
	IncidentSourceRefs   []string  `json:"incident_source_refs,omitempty"` // Linear / Jira / Slack URL prefixes
	WallClockBudget      string    `json:"wall_clock_budget,omitempty"`    // RFC 3339 duration; default 30m
	EnqueuedAt           time.Time `json:"enqueued_at"`
}

// CartographyResult is the structured output the control plane consumes.
type CartographyResult struct {
	JobID                       string                  `json:"job_id"`
	TenantID                    string                  `json:"tenant_id"`
	Repo                        string                  `json:"repo"`
	FilesIndexed                int                     `json:"files_indexed"`
	Directories                 int                     `json:"directories"`
	StackPrimary                string                  `json:"stack_primary"`
	StackSecondary              []string                `json:"stack_secondary"`
	SymbolCount                 int                     `json:"symbol_count"`
	ConventionsFromConfigs      int                     `json:"conventions_from_configs"`
	ConventionsFromAgentsMD     int                     `json:"conventions_from_agents_md"`
	ConventionsFromContributing int                     `json:"conventions_from_contributing"`
	ConventionsFromADRs         int                     `json:"conventions_from_adrs"`
	ConventionsFromPRReview     int                     `json:"conventions_from_pr_review"`
	ConventionsFromIncidents    int                     `json:"conventions_from_incidents"`
	ConventionsFromOSSDefaults  int                     `json:"conventions_from_oss_defaults"`
	HighConfidenceCount         int                     `json:"high_confidence_count"`
	MediumConfidenceCount       int                     `json:"medium_confidence_count"`
	LowConfidenceCount          int                     `json:"low_confidence_count"`
	Sample                      []ConventionCandidate   `json:"sample"`
	InferredAgentsMDMarkdown    string                  `json:"inferred_agents_md_markdown"`
	HasCustomerOverride         bool                    `json:"has_customer_override"`
	CustomerOverridePath        string                  `json:"customer_override_path"`
	FirstTaskSuggestions        []FirstTaskSuggestion   `json:"first_task_suggestions"`
	ConsoleOutputLines          []string                `json:"console_output_lines"`
	StartedAt                   time.Time               `json:"started_at"`
	CompletedAt                 time.Time               `json:"completed_at"`
	WallClockSeconds            float64                 `json:"wall_clock_seconds"`
	TokensSpent                 int                     `json:"tokens_spent"`
	UsdSpent                    float64                 `json:"usd_spent"`
}

// ConventionCandidate mirrors the Phase-5 distiller schema; we keep
// the JSON tag names compatible so memory-router admission consumes
// either side without translation.
type ConventionCandidate struct {
	ID            string    `json:"id"`
	Category      string    `json:"category"`
	RuleNL        string    `json:"rule_nl"`
	FileGlob      string    `json:"file_glob"`
	Rationale     string    `json:"rationale"`
	EvidenceQuote string    `json:"evidence_quote"`
	SourceChannel string    `json:"source_channel"`
	SourcePath    string    `json:"source_path"`
	Confidence    float64   `json:"confidence"`
	Status        string    `json:"status"`
	Stack         string    `json:"stack,omitempty"`
	FirstSeen     time.Time `json:"first_seen"`
}

// FirstTaskSuggestion is what the onboarding flow surfaces after
// Cartography completes, per docs/04-operations/onboarding.md §Stage 3.
type FirstTaskSuggestion struct {
	Title       string   `json:"title"`
	Rationale   string   `json:"rationale"` // why we picked this for THIS repo
	Touches     []string `json:"touches"`   // file globs the task would modify
	EstUSD      float64  `json:"est_usd"`
	EstWallMin  int      `json:"est_wall_min"`
	Complexity  string   `json:"complexity"` // small | medium | large
	WhySafeFirst string  `json:"why_safe_first"`
}

// JobStatus is what the control plane polls.
type JobStatus struct {
	JobID         string    `json:"job_id"`
	State         string    `json:"state"` // queued | running | done | error
	Stage         string    `json:"stage"`
	StageProgress float64   `json:"stage_progress"` // 0..1
	UpdatedAt     time.Time `json:"updated_at"`
	Error         string    `json:"error,omitempty"`
}

// SymbolEntry is one row of the per-file symbol index.
type SymbolEntry struct {
	Path     string   `json:"path"`
	Language string   `json:"language"`
	Kind     string   `json:"kind"` // func | class | const | type
	Name     string   `json:"name"`
	Line     int      `json:"line"`
	Calls    []string `json:"calls,omitempty"`
}
