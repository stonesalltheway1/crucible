package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/crucible/cli/internal/client"
)

// `crucible calibrate` — fit the per-tenant critical-path classifier weights
// against the tenant's incident corpus + path globs + customer overrides.
//
// Runs as a one-off; the resulting weights land in the tenant config
// document. The web console's Settings → Critical-path tab surfaces the
// fitted weights for editing.

func newCalibrateCmd(f *flags) *cobra.Command {
	var dryRun bool
	var sampleSize int
	c := &cobra.Command{
		Use:   "calibrate",
		Short: "Fit the per-tenant critical-path classifier weights.",
		Long: `Walks the tenant's incident history + PR-merge stream + ADR corpus and
fits the multinomial classifier weights that route tasks to Tier 3
verification + N-of-M approval cohorts.

This is the load-bearing per-tenant tuning step. Without it, the
classifier uses the global defaults from libs/policy — which work, but
miss tenant-specific signal like "all changes under billing/ are
critical because of INC-2231".`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			res, err := cli.Calibrate(ctx, sampleSize, dryRun)
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Calibration complete:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  samples:              %d\n", res.Samples)
			fmt.Fprintf(cmd.OutOrStdout(), "  per-fold AUC mean:    %.3f\n", res.AucMean)
			fmt.Fprintf(cmd.OutOrStdout(), "  dry-run:              %v\n", dryRun)
			fmt.Fprintln(cmd.OutOrStdout(), "  fitted weights:")
			for _, w := range res.Weights {
				fmt.Fprintf(cmd.OutOrStdout(), "    %-40s %.3f\n", w.Path, w.Weight)
			}
			if !dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), "Weights persisted to tenant config.")
			}
			return nil
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "fit but don't persist; just report")
	c.Flags().IntVar(&sampleSize, "samples", 2000, "max samples to draw from the PR-merge stream")
	return c
}
