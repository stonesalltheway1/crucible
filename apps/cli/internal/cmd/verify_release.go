package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/crucible/cli/internal/client"
)

// `crucible verify-release` — the public-facing Tier-4 customer-side
// verifier. Any user can run this against any Crucible release to confirm
// hermetic rebuild + SLSA-L3 attestation + Sigstore inclusion.
//
// This command does NOT require authentication — the artefacts and their
// attestations are public.

func newVerifyReleaseCmd(_ *flags) *cobra.Command {
	var endpoint string
	c := &cobra.Command{
		Use:   "verify-release [version]",
		Short: "Verify a Crucible release end-to-end against its SLSA-L3 attestation.",
		Long: `Downloads the release's in-toto attestation from Rekor, re-builds the
release artefact hermetically (via Nix), and asserts that the rebuild's
SHA-256 matches the attestation's subject digest. The expected outcome
is a bit-identical rebuild — that is the load-bearing property of
Tier 4 verification.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()
			cli := client.NewPublic(endpoint)
			res, err := cli.VerifyRelease(ctx, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Crucible %s\n", args[0])
			fmt.Fprintf(cmd.OutOrStdout(), "  Attestation:        %s\n", res.RekorUUID)
			fmt.Fprintf(cmd.OutOrStdout(), "  Expected sha256:    %s\n", res.ExpectedDigest)
			fmt.Fprintf(cmd.OutOrStdout(), "  Rebuilt sha256:     %s\n", res.RebuildDigest)
			fmt.Fprintf(cmd.OutOrStdout(), "  Signer OIDC:        %s\n", res.SignerOIDC)
			fmt.Fprintf(cmd.OutOrStdout(), "  Match:              %v\n", res.Match)
			if !res.Match {
				os.Exit(2)
			}
			return nil
		},
	}
	c.Flags().StringVar(&endpoint, "endpoint", "https://attest.crucible.dev", "public attestation endpoint")
	return c
}
