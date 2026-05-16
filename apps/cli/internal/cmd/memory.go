package cmd

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/crucible/cli/internal/client"
)

// `crucible memory` — Phase 5 memory layer surface for senior engineers
// reviewing per-tenant conventions, drifts, and supersession history.

func newMemoryCmd(f *flags) *cobra.Command {
	root := &cobra.Command{Use: "memory", Short: "Inspect per-tenant procedural memory."}
	root.AddCommand(memoryRecallCmd(f))
	root.AddCommand(memoryNoteCmd(f))
	root.AddCommand(memoryConventionsCmd(f))
	root.AddCommand(memoryDriftReviewCmd(f))
	return root
}

func memoryRecallCmd(f *flags) *cobra.Command {
	var query, scope string
	c := &cobra.Command{
		Use:   "recall",
		Short: "Run a multi-signal retrieval against memory.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if query == "" {
				return fmt.Errorf("--query is required")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			res, err := cli.MemoryRecall(ctx, query, scope)
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), res)
			}
			for _, m := range res {
				fmt.Fprintf(cmd.OutOrStdout(), "  • [%.2f] %s\n", m.Score, m.RuleNl)
			}
			return nil
		},
	}
	c.Flags().StringVar(&query, "query", "", "query string (required)")
	c.Flags().StringVar(&scope, "scope", "", "optional scope: repo:foo, file:bar/**, category:security")
	return c
}

func memoryNoteCmd(f *flags) *cobra.Command {
	var rule, category, scope string
	c := &cobra.Command{
		Use:   "note",
		Short: "Add an explicit convention. Subject to the gateway's deterministic + LLM-judge filters.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if rule == "" || category == "" {
				return fmt.Errorf("--rule and --category are required")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			if err := cli.MemoryNote(ctx, rule, category, scope); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Note submitted. Gateway will rule on admission within a minute.")
			return nil
		},
	}
	c.Flags().StringVar(&rule, "rule", "", "natural-language rule (required)")
	c.Flags().StringVar(&category, "category", "", "one of the 12 conventions taxonomy buckets (required)")
	c.Flags().StringVar(&scope, "scope", "", "optional scope")
	return c
}

func memoryConventionsCmd(f *flags) *cobra.Command {
	var status string
	c := &cobra.Command{
		Use:   "conventions",
		Short: "List per-tenant conventions.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			cs, err := cli.ListConventions(ctx, status)
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), cs)
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tSTATUS\tCONF\tCATEGORY\tRULE")
			for _, c := range cs {
				fmt.Fprintf(tw, "%s\t%s\t%.2f\t%s\t%s\n", shortID(c.ID), c.Status, c.Confidence, c.Category, trunc(c.RuleNl, 60))
			}
			tw.Flush()
			return nil
		},
	}
	c.Flags().StringVar(&status, "status", "", "filter by status (active, drifting, candidate, superseded)")
	return c
}

func memoryDriftReviewCmd(f *flags) *cobra.Command {
	return &cobra.Command{
		Use:   "drift-review",
		Short: "Show conventions whose positive/negative ratio fell below threshold in the last 30 days.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			cs, err := cli.ListConventions(ctx, "drifting")
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), cs)
			}
			if len(cs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(no drifting conventions)")
				return nil
			}
			for _, c := range cs {
				fmt.Fprintf(cmd.OutOrStdout(), "  ⚠ %s\n", c.RuleNl)
				fmt.Fprintf(cmd.OutOrStdout(), "    +30d %d  −30d %d  last violation %s\n",
					c.PositiveExamples30d, c.NegativeExamples30d, c.LastViolatedAt)
			}
			return nil
		},
	}
}
