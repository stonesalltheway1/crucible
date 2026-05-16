package cmd

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/crucible/cli/internal/client"
)

func newWebhookCmd(f *flags) *cobra.Command {
	root := &cobra.Command{Use: "webhook", Aliases: []string{"webhooks"}, Short: "Manage webhook subscriptions."}
	root.AddCommand(webhookCreateCmd(f))
	root.AddCommand(webhookListCmd(f))
	root.AddCommand(webhookRedeliverCmd(f))
	return root
}

func webhookCreateCmd(f *flags) *cobra.Command {
	var url, eventsCsv, description string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a webhook subscription.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if url == "" {
				return fmt.Errorf("--url is required")
			}
			if !strings.HasPrefix(url, "https://") {
				return fmt.Errorf("--url must be https://")
			}
			events := strings.Split(eventsCsv, ",")
			for i := range events {
				events[i] = strings.TrimSpace(events[i])
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			sub, err := cli.CreateWebhook(ctx, url, events, description)
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), sub)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", sub.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Signing secret (shown once): %s\n", sub.SigningSecret)
			return nil
		},
	}
	c.Flags().StringVar(&url, "url", "", "https endpoint (required)")
	c.Flags().StringVar(&eventsCsv, "events", "task.*", "comma-separated event globs")
	c.Flags().StringVar(&description, "description", "", "human description")
	return c
}

func webhookListCmd(f *flags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List active subscriptions.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			subs, err := cli.ListWebhooks(ctx)
			if err != nil {
				return err
			}
			if f.json {
				return writeJSON(cmd.OutOrStdout(), subs)
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tACTIVE\tEVENTS\tURL")
			for _, s := range subs {
				fmt.Fprintf(tw, "%s\t%v\t%s\t%s\n", shortID(s.ID), s.Active, strings.Join(s.Events, ","), s.URL)
			}
			tw.Flush()
			return nil
		},
	}
}

func webhookRedeliverCmd(f *flags) *cobra.Command {
	var sub, event string
	c := &cobra.Command{
		Use:   "redeliver",
		Short: "Replay an event to a subscription.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if sub == "" || event == "" {
				return fmt.Errorf("--sub and --event are required")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cli := client.New(f.endpoint, f.tenant)
			if err := cli.RedeliverWebhook(ctx, sub, event); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Queued.")
			return nil
		},
	}
	c.Flags().StringVar(&sub, "sub", "", "subscription id")
	c.Flags().StringVar(&event, "event", "", "event id")
	return c
}
