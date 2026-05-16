package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/crucible/cli/internal/client"
)

// `crucible attestation` — Phase 6 audit-trail surface for compliance auditors,
// senior engineers, and the public Tier-4 customer-side verifier.

func newAttestationCmd(f *flags) *cobra.Command {
	root := &cobra.Command{Use: "attestation", Aliases: []string{"att", "attest"}, Short: "Inspect and verify attestations."}
	root.AddCommand(attGetCmd(f))
	root.AddCommand(attVerifyCmd(f))
	root.AddCommand(attChainCmd(f))
	root.AddCommand(attExportCmd(f))
	return root
}

func attGetCmd(f *flags) *cobra.Command {
	return &cobra.Command{
		Use:   "get [rekor_uuid]",
		Short: "Fetch the canonical attestation document.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			a, err := cli.GetAttestation(ctx, args[0])
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), a)
		},
	}
}

func attVerifyCmd(f *flags) *cobra.Command {
	return &cobra.Command{
		Use:   "verify [rekor_uuid]",
		Short: "End-to-end verify: certificate chain + Merkle inclusion proof + subject digest.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			res, err := cli.VerifyAttestation(ctx, args[0])
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Attestation %s\n", args[0])
			fmt.Fprintf(cmd.OutOrStdout(), "  verified:                %v\n", res.Verified)
			fmt.Fprintf(cmd.OutOrStdout(), "  inclusion_proof_valid:   %v\n", res.Details.InclusionProofValid)
			fmt.Fprintf(cmd.OutOrStdout(), "  cert_chain_valid:        %v\n", res.Details.CertChainValid)
			fmt.Fprintf(cmd.OutOrStdout(), "  subject_digest_matches:  %v\n", res.Details.SubjectDigestMatches)
			fmt.Fprintf(cmd.OutOrStdout(), "  self_hosted_rekor:       %v\n", res.Details.SelfHosted)
			if !res.Verified {
				os.Exit(2)
			}
			return nil
		},
	}
}

func attChainCmd(f *flags) *cobra.Command {
	return &cobra.Command{
		Use:   "chain [task_id]",
		Short: "Show the full attestation chain for a task: plan → writes → tests → verifier → promotion → outcome.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			ch, err := cli.GetAttestationChain(ctx, args[0])
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), ch)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Chain for task %s — %d nodes:\n\n", args[0], len(ch.Nodes))
			for i, n := range ch.Nodes {
				fmt.Fprintf(cmd.OutOrStdout(), "  %2d. %s  %s\n", i+1, n.PredicateType, n.Label)
				fmt.Fprintf(cmd.OutOrStdout(), "      %s  signed %s\n", n.RekorUUID, n.SignedAt)
			}
			return nil
		},
	}
}

func attExportCmd(f *flags) *cobra.Command {
	var outPath string
	c := &cobra.Command{
		Use:   "export [task_id]",
		Short: "Export the complete attestation bundle for an auditor: all nodes + inclusion proofs + Sigstore root.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			data, err := cli.ExportAttestationBundle(ctx, args[0])
			if err != nil {
				return err
			}
			if outPath == "" {
				_, err := cmd.OutOrStdout().Write(data)
				return err
			}
			if err := os.WriteFile(outPath, data, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Wrote %d bytes to %s\n", len(data), outPath)
			return nil
		},
	}
	c.Flags().StringVar(&outPath, "out", "", "output file path (defaults to stdout)")
	return c
}
