// Package cmd builds the crucible CLI command tree.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/crucible/cli/internal/client"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

const cliVersion = "2026.06.0-phase7"

type flags struct {
	endpoint string
	tenant   string
	json     bool
}

// NewRoot returns the root cobra command.
func NewRoot() *cobra.Command {
	f := &flags{}

	root := &cobra.Command{
		Use:           "crucible",
		Short:         "Crucible CLI — submit tasks, review plans, watch budgets.",
		Long:          "Crucible is the AI engineer that tests every change in a digital twin before touching real code.\n\nThis CLI is the Phase-1 entry point. See docs/PHASE-1-REPORT.md for what's wired today.",
		Version:       cliVersion,
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	root.PersistentFlags().StringVar(&f.endpoint, "endpoint", envOr("CRUCIBLE_ENDPOINT", "http://localhost:8080"), "Control-plane endpoint")
	root.PersistentFlags().StringVar(&f.tenant, "tenant", envOr("CRUCIBLE_TENANT", "single-tenant"), "Tenant ID")
	root.PersistentFlags().BoolVar(&f.json, "json", false, "Emit JSON instead of human-readable text")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newHealthCmd(f))
	root.AddCommand(newTaskCmd(f))
	root.AddCommand(newPlanCmd(f))
	root.AddCommand(newBudgetCmd(f))
	root.AddCommand(newPromoteCmd(f))
	root.AddCommand(newMemoryCmd(f))
	root.AddCommand(newAttestationCmd(f))
	root.AddCommand(newWebhookCmd(f))
	root.AddCommand(newTenantCmd(f))
	root.AddCommand(newVerifyReleaseCmd(f))
	root.AddCommand(newCalibrateCmd(f))

	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version.",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), cliVersion)
		},
	}
}

func newHealthCmd(f *flags) *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Ping the control plane's /healthz endpoint.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			c := client.New(f.endpoint, f.tenant)
			h, err := c.Health(ctx)
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), h)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "status:  %v\nversion: %v\nnow:     %v\nstubs:   twin=%v verifier=%v promotion=%v\n",
				h["status"], h["version"], h["now"],
				h["stub_twin_runtime"], h["stub_verifier"], h["stub_promotion"])
			return nil
		},
	}
}

// ── task ──────────────────────────────────────────────────────────────────

func newTaskCmd(f *flags) *cobra.Command {
	task := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks.",
	}

	var description, repo, baseSha string
	var costCap float64
	var wallMin uint32

	taskNew := &cobra.Command{
		Use:   "new",
		Short: "Submit a new task.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if description == "" {
				return fmt.Errorf("--description is required")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 120*time.Second)
			defer cancel()
			c := client.New(f.endpoint, f.tenant)
			t, err := c.SubmitTask(ctx, client.SubmitTaskRequest{
				Description:     description,
				Repo:            repo,
				BaseSha:         baseSha,
				CostCapUSD:      costCap,
				WallClockCapMin: wallMin,
			})
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), t)
			}
			renderTaskShort(cmd.OutOrStdout(), t)
			return nil
		},
	}
	taskNew.Flags().StringVar(&description, "description", "", "Task description (required)")
	taskNew.Flags().StringVar(&repo, "repo", "", "Repository (e.g. github.com/acme/payments)")
	taskNew.Flags().StringVar(&baseSha, "base-sha", "", "Base SHA to plan against")
	taskNew.Flags().Float64Var(&costCap, "cost-cap-usd", 0, "Override the cost cap (0 → estimate × 1.5)")
	taskNew.Flags().Uint32Var(&wallMin, "wall-clock-cap-min", 0, "Override the wall-clock cap")

	taskList := &cobra.Command{
		Use:   "list",
		Short: "List recent tasks.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			c := client.New(f.endpoint, f.tenant)
			ts, err := c.ListTasks(ctx, 50)
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), ts)
			}
			if len(ts) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(no tasks)")
				return nil
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tSTATUS\tEXECUTOR\tCOST CAP\tCREATED\tDESCRIPTION")
			for _, t := range ts {
				cap := "-"
				if t.Budget != nil {
					cap = "$" + strconv.FormatFloat(t.Budget.CapUsd, 'f', 2, 64)
				}
				exec := "-"
				if t.Routing != nil {
					exec = t.Routing.ExecutorModel
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
					shortID(t.ID), t.Status, exec, cap,
					t.CreatedAt.Format(time.RFC3339), trunc(t.Description, 50))
			}
			tw.Flush()
			return nil
		},
	}

	taskGet := &cobra.Command{
		Use:   "get [task_id]",
		Args:  cobra.ExactArgs(1),
		Short: "Show a task with its plan + budget.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			c := client.New(f.endpoint, f.tenant)
			t, err := c.GetTask(ctx, args[0])
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), t)
			}
			renderTaskFull(cmd.OutOrStdout(), t)
			return nil
		},
	}

	task.AddCommand(taskNew, taskList, taskGet)
	return task
}

// ── plan ──────────────────────────────────────────────────────────────────

func newPlanCmd(f *flags) *cobra.Command {
	plan := &cobra.Command{
		Use:   "plan",
		Short: "Inspect and approve task plans.",
	}

	planShow := &cobra.Command{
		Use:   "show [task_id]",
		Args:  cobra.ExactArgs(1),
		Short: "Show the plan for a task.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			c := client.New(f.endpoint, f.tenant)
			t, err := c.GetTask(ctx, args[0])
			if err != nil {
				return err
			}
			if t.Plan == nil {
				return fmt.Errorf("task has no plan yet (status: %s)", t.Status)
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), t.Plan)
			}
			renderPlan(cmd.OutOrStdout(), t)
			return nil
		},
	}

	var planHash string
	var costCap float64
	var wallMin, retryCap uint32

	planApprove := &cobra.Command{
		Use:   "approve [task_id]",
		Args:  cobra.ExactArgs(1),
		Short: "Approve the plan attached to a task.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			c := client.New(f.endpoint, f.tenant)
			t, ap, err := c.ApprovePlan(ctx, args[0], client.ApprovePlanRequest{
				PlanHash:           planHash,
				CostCapUSD:         costCap,
				WallClockCapMin:    wallMin,
				RetryCapPerSubgoal: retryCap,
			})
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), map[string]any{"task": t, "approval": ap})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Approved %s (plan %s).\n", shortID(t.ID), shortHash(t.Plan.PlanHash))
			if ap != nil && ap.AttestationID != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Attestation: %s\n", ap.AttestationID)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", t.Status)
			return nil
		},
	}
	planApprove.Flags().StringVar(&planHash, "plan-hash", "", "Optimistic-concurrency guard (optional)")
	planApprove.Flags().Float64Var(&costCap, "cost-cap-usd", 0, "Override the cost cap")
	planApprove.Flags().Uint32Var(&wallMin, "wall-clock-cap-min", 0, "Override the wall-clock cap")
	planApprove.Flags().Uint32Var(&retryCap, "retry-cap", 0, "Override the retry cap per subgoal")

	var reason string
	planReject := &cobra.Command{
		Use:   "reject [task_id]",
		Args:  cobra.ExactArgs(1),
		Short: "Reject the plan attached to a task.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if reason == "" {
				return fmt.Errorf("--reason is required")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			c := client.New(f.endpoint, f.tenant)
			t, _, err := c.RejectPlan(ctx, args[0], client.RejectPlanRequest{Reason: reason})
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), t)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Rejected %s (status: %s).\n", shortID(t.ID), t.Status)
			return nil
		},
	}
	planReject.Flags().StringVar(&reason, "reason", "", "Reason for rejection (required)")

	plan.AddCommand(planShow, planApprove, planReject)
	return plan
}

// ── budget ────────────────────────────────────────────────────────────────

func newBudgetCmd(f *flags) *cobra.Command {
	budget := &cobra.Command{
		Use:   "budget",
		Short: "Inspect live task budgets.",
	}
	budget.AddCommand(&cobra.Command{
		Use:   "show [task_id]",
		Args:  cobra.ExactArgs(1),
		Short: "Show the live Budget snapshot.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			c := client.New(f.endpoint, f.tenant)
			b, err := c.GetBudget(ctx, args[0])
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), b)
			}
			renderBudget(cmd.OutOrStdout(), b)
			return nil
		},
	})
	return budget
}

// ── rendering helpers ─────────────────────────────────────────────────────

func renderTaskShort(w io.Writer, t *cruciblev1.Task) {
	if t == nil {
		fmt.Fprintln(w, "(nil task)")
		return
	}
	fmt.Fprintf(w, "Task %s submitted.\n", t.ID)
	fmt.Fprintf(w, "  Status:      %s\n", t.Status)
	if t.Routing != nil {
		fmt.Fprintf(w, "  Executor:    %s (%s, tier %d)\n", t.Routing.ExecutorModel, t.Routing.ExecutorVendor, t.Routing.ExecutorTier)
		fmt.Fprintf(w, "  Verifier:    %s (%s)\n", t.Routing.VerifierModel, t.Routing.VerifierVendor)
		if t.Routing.IsCritical {
			fmt.Fprintf(w, "  Critical:    yes (score %.0f)\n", t.Routing.CriticalScore)
		}
	}
	if t.Plan != nil {
		fmt.Fprintf(w, "  Plan:        %d steps, est $%.2f over %d min\n",
			len(t.Plan.Steps), t.Plan.EstimatedCostUsd, t.Plan.EstimatedDurationMin)
		fmt.Fprintf(w, "  Plan hash:   %s\n", t.Plan.PlanHash)
	}
	if t.Budget != nil {
		fmt.Fprintf(w, "  Cost cap:    $%.2f   (wall-clock %ds, retries %d)\n",
			t.Budget.CapUsd, t.Budget.WallClockCapSeconds, t.Budget.RetryCap)
	}
	fmt.Fprintf(w, "\nReview plan:    crucible plan show %s\n", t.ID)
	fmt.Fprintf(w, "Approve plan:   crucible plan approve %s\n", t.ID)
}

func renderTaskFull(w io.Writer, t *cruciblev1.Task) {
	renderTaskShort(w, t)
	if t.Plan != nil {
		fmt.Fprintln(w)
		renderPlan(w, t)
	}
}

func renderPlan(w io.Writer, t *cruciblev1.Task) {
	p := t.Plan
	fmt.Fprintf(w, "Plan for %s\n", t.ID)
	fmt.Fprintln(w, strings.Repeat("─", 64))
	fmt.Fprintf(w, "Description:  %s\n", p.Description)
	fmt.Fprintf(w, "Complexity:   %s\n", p.Complexity)
	fmt.Fprintf(w, "Estimate:     $%.2f over %d min\n", p.EstimatedCostUsd, p.EstimatedDurationMin)
	fmt.Fprintf(w, "Caps:         retry %d/step, wall-clock %d min\n", p.RetryBudgetPerStep, p.WallClockBudgetMin)
	fmt.Fprintf(w, "Plan hash:    %s\n", p.PlanHash)
	if len(p.FilesToTouch) > 0 {
		fmt.Fprintf(w, "Files:        %s\n", strings.Join(p.FilesToTouch, ", "))
	}
	if p.DbMigrations > 0 {
		fmt.Fprintf(w, "Migrations:   %d\n", p.DbMigrations)
	}
	if len(p.ExternalEffects) > 0 {
		fmt.Fprintln(w, "External:")
		for _, e := range p.ExternalEffects {
			fmt.Fprintf(w, "  - %s %v  (live=%v)\n", e.Service, e.Endpoints, e.Live)
		}
	}
	fmt.Fprintln(w, "Steps:")
	for _, s := range p.Steps {
		fmt.Fprintf(w, "  %2d. %s   (retry %d/%d)\n", s.Ordinal, s.Description, s.RetriesUsed, s.RetryBudget)
	}
	if len(p.TopRisks) > 0 {
		fmt.Fprintln(w, "Risks:")
		for _, r := range p.TopRisks {
			fmt.Fprintf(w, "  - [%s] %s\n", r.Impact, r.Description)
		}
	}
}

func renderBudget(w io.Writer, b *cruciblev1.Budget) {
	if b == nil {
		fmt.Fprintln(w, "(no budget snapshot — task may have completed)")
		return
	}
	fmt.Fprintf(w, "Spent:        $%.4f / $%.4f\n", b.SpentUsd, b.CapUsd)
	fmt.Fprintf(w, "Retries:      %d / %d\n", b.RetriesUsed, b.RetryCap)
	fmt.Fprintf(w, "Wall-clock:   %ds / %ds\n", b.WallClockUsedSeconds, b.WallClockCapSeconds)
	if b.StepsCap > 0 {
		fmt.Fprintf(w, "Steps:        %d / %d\n", b.StepsUsed, b.StepsCap)
	}
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func shortID(id string) string {
	if len(id) <= 14 {
		return id
	}
	return id[:14]
}

func shortHash(h string) string {
	if len(h) <= 12 {
		return h
	}
	return h[:12]
}

func trunc(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
