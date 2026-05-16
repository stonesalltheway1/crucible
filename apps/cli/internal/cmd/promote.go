package cmd

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/crucible/cli/internal/client"
)

// `crucible promote` — flesh out the Phase 6 promotion gate surface for
// scripts and operators. Mirrors the web console's Promotions tab.

func newPromoteCmd(f *flags) *cobra.Command {
	root := &cobra.Command{
		Use:   "promote",
		Short: "Manage promotion bundles (Phase 6 gate).",
	}

	root.AddCommand(promoteListCmd(f))
	root.AddCommand(promoteGetCmd(f))
	root.AddCommand(promoteApproveCmd(f))
	root.AddCommand(promoteRejectCmd(f))
	root.AddCommand(promoteStatusCmd(f))
	root.AddCommand(promoteRollbackCmd(f))
	return root
}

func promoteListCmd(f *flags) *cobra.Command {
	var status string
	c := &cobra.Command{
		Use:   "list",
		Short: "List recent promotions for the tenant.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			ps, err := cli.ListPromotions(ctx, status)
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), ps)
			}
			if len(ps) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(no promotions)")
				return nil
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tTASK\tSTATUS\tDIFF HASH")
			for _, p := range ps {
				diff := "-"
				if p.Bundle != nil {
					diff = shortHash(p.Bundle.DiffHash)
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", shortID(p.ID), shortID(p.TaskID), p.Status, diff)
			}
			tw.Flush()
			return nil
		},
	}
	c.Flags().StringVar(&status, "status", "", "filter by status (pending_approval, canary_dwell, landed, rolled_back, ...)")
	return c
}

func promoteGetCmd(f *flags) *cobra.Command {
	return &cobra.Command{
		Use:   "get [promotion_id]",
		Short: "Show a promotion's full state.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			p, err := cli.GetPromotion(ctx, args[0])
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), p)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Promotion %s\n", p.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "  Task:        %s\n", p.TaskID)
			fmt.Fprintf(cmd.OutOrStdout(), "  Status:      %s\n", p.Status)
			if p.Decision != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "  Decision:    allow=%v needs_human=%v trace=%s\n",
					p.Decision.Allow, p.Decision.NeedsHuman, p.Decision.Trace.Path)
			}
			if p.Bundle != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "  Diff hash:   %s\n", p.Bundle.DiffHash)
				fmt.Fprintf(cmd.OutOrStdout(), "  Files:       %d\n", len(p.Bundle.FilesChanged))
			}
			if p.Canary != nil {
				for i, s := range p.Canary.Steps {
					marker := " "
					if i == p.Canary.CurrentStep {
						marker = "→"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "  %s %3d%% (dwell %ds, slo=%s)\n",
						marker, s.Weight, s.DwellSeconds, s.SloCheck)
				}
			}
			return nil
		},
	}
}

func promoteApproveCmd(f *flags) *cobra.Command {
	var group, bundleHash string
	c := &cobra.Command{
		Use:   "approve [promotion_id]",
		Short: "Approve a promotion. Sigstore keyless OIDC binds your signature to the bundle hash.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if group == "" {
				return fmt.Errorf("--group is required (e.g. @payments-leads)")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 20*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			p, err := cli.ApprovePromotion(ctx, args[0], group, bundleHash)
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), p)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Approved %s (status now %s).\n", shortID(p.ID), p.Status)
			return nil
		},
	}
	c.Flags().StringVar(&group, "group", "", "approver group (must match Rego decision)")
	c.Flags().StringVar(&bundleHash, "bundle-hash", "", "diff hash to bind the approval to (defaults to current)")
	return c
}

func promoteRejectCmd(f *flags) *cobra.Command {
	var reason string
	c := &cobra.Command{
		Use:   "reject [promotion_id]",
		Short: "Reject a promotion.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if reason == "" {
				return fmt.Errorf("--reason is required")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			if err := cli.RejectPromotion(ctx, args[0], reason); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Rejected %s.\n", shortID(args[0]))
			return nil
		},
	}
	c.Flags().StringVar(&reason, "reason", "", "reason for rejection (required)")
	return c
}

func promoteStatusCmd(f *flags) *cobra.Command {
	return &cobra.Command{
		Use:   "status [promotion_id]",
		Short: "Short-form status with rollout progression.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			p, err := cli.GetPromotion(ctx, args[0])
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), map[string]any{
					"id":     p.ID,
					"status": p.Status,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s · %s\n", shortID(p.ID), p.Status)
			return nil
		},
	}
}

func promoteRollbackCmd(f *flags) *cobra.Command {
	var reason string
	c := &cobra.Command{
		Use:   "rollback [promotion_id]",
		Short: "Force-rollback an in-flight promotion. Bypasses the SLO-watcher cadence.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if reason == "" {
				return fmt.Errorf("--reason is required")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			if err := cli.RollbackPromotion(ctx, args[0], reason); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Rollback initiated for %s.\n", shortID(args[0]))
			return nil
		},
	}
	c.Flags().StringVar(&reason, "reason", "", "reason for rollback (required)")
	return c
}
