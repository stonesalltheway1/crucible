package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/crucible/cli/internal/client"
)

func newTenantCmd(f *flags) *cobra.Command {
	root := &cobra.Command{Use: "tenant", Short: "Tenant configuration."}
	root.AddCommand(&cobra.Command{
		Use:   "config",
		Short: "Get / set tenant config.",
	})
	root.AddCommand(tenantConfigGetCmd(f))
	root.AddCommand(tenantConfigSetCmd(f))
	return root
}

func tenantConfigGetCmd(f *flags) *cobra.Command {
	return &cobra.Command{
		Use:   "config-get",
		Short: "Fetch the tenant's current config document.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			cfg, err := cli.GetTenantConfig(ctx)
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), cfg)
		},
	}
}

func tenantConfigSetCmd(f *flags) *cobra.Command {
	var path string
	c := &cobra.Command{
		Use:   "config-set",
		Short: "Update the tenant config. Reads JSON from --file or stdin.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var body []byte
			var err error
			if path == "" || path == "-" {
				body, err = io.ReadAll(cmd.InOrStdin())
			} else {
				body, err = os.ReadFile(path)
			}
			if err != nil {
				return err
			}
			var parsed any
			if err := json.Unmarshal(body, &parsed); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 20*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			if err := cli.SetTenantConfig(ctx, parsed); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Config updated. New tasks pick up the change immediately.")
			return nil
		},
	}
	c.Flags().StringVar(&path, "file", "", "JSON config file (or - for stdin)")
	return c
}
