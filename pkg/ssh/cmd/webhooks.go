package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/webhook"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func webhookCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "webhook",
		Aliases: []string{"webhooks"},
		Short:   "Manage repository webhooks",
	}

	cmd.AddCommand(
		webhookListCommand(),
		webhookCreateCommand(),
		webhookDeleteCommand(),
		webhookUpdateCommand(),
		webhookDeliveriesCommand(),
	)

	return cmd
}

var webhookEvents []string

func init() {
	events := webhook.Events()
	webhookEvents = make([]string, len(events))
	for i, e := range events {
		webhookEvents[i] = e.String()
	}
}

func webhookListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "list REPOSITORY",
		Short:             "List repository webhooks",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo, err := be.Repository(ctx, args[0])
			if err != nil {
				return err
			}

			webhooks, err := be.ListWebhooks(ctx, repo)
			if err != nil {
				return err
			}

			table := table.New().Headers("ID", "URL", "Events", "Active", "Created At", "Updated At")
			for _, h := range webhooks {
				events := make([]string, len(h.Events))
				for i, e := range h.Events {
					events[i] = e.String()
				}

				table = table.Row(
					strconv.FormatInt(h.ID, 10),
					h.URL,
					strings.Join(events, ","),
					strconv.FormatBool(h.Active),
					humanize.Time(h.CreatedAt),
					humanize.Time(h.UpdatedAt),
				)
			}
			cmd.Println(table)
			return nil
		},
	}

	return cmd
}

func webhookCreateCommand() *cobra.Command {
	var events []string
	var secret string
	var active bool
	var contentType string
	cmd := &cobra.Command{
		Use:               "create REPOSITORY URL",
		Short:             "Create a repository webhook",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo, err := be.Repository(ctx, args[0])
			if err != nil {
				return err
			}

			var evs []webhook.Event
			for _, e := range events {
				ev, err := webhook.ParseEvent(e)
				if err != nil {
					return fmt.Errorf("invalid event: %w", err)
				}

				evs = append(evs, ev)
			}

			var ct webhook.ContentType
			switch strings.ToLower(strings.TrimSpace(contentType)) {
			case "json":
				ct = webhook.ContentTypeJSON
			case "form":
				ct = webhook.ContentTypeForm
			default:
				return webhook.ErrInvalidContentType
			}

			return be.CreateWebhook(ctx, repo, strings.TrimSpace(args[1]), ct, secret, evs, active)
		},
	}

	cmd.Flags().StringSliceVarP(&events, "events", "e", nil, fmt.Sprintf("events to trigger the webhook, available events are (%s)", strings.Join(webhookEvents, ", ")))
	cmd.Flags().StringVarP(&secret, "secret", "s", "", "secret to sign the webhook payload")
	cmd.Flags().BoolVarP(&active, "active", "a", true, "whether the webhook is active")
	cmd.Flags().StringVarP(&contentType, "content-type", "c", "json", "content type of the webhook payload, can be either `json` or `form`")

	return cmd
}

func webhookDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete REPOSITORY WEBHOOK_ID",
		Short:             "Delete a repository webhook",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo, err := be.Repository(ctx, args[0])
			if err != nil {
				return err
			}

			id, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid webhook ID: %w", err)
			}

			return be.DeleteWebhook(ctx, repo, id)
		},
	}

	return cmd
}

func webhookUpdateCommand() *cobra.Command {
	var events []string
	var secret string
	var active string
	var contentType string
	var url string
	cmd := &cobra.Command{
		Use:               "update REPOSITORY WEBHOOK_ID",
		Short:             "Update a repository webhook",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo, err := be.Repository(ctx, args[0])
			if err != nil {
				return err
			}

			id, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid webhook ID: %w", err)
			}

			wh, err := be.Webhook(ctx, repo, id)
			if err != nil {
				return err
			}

			newURL := wh.URL
			if url != "" {
				newURL = url
			}

			newSecret := wh.Secret
			if secret != "" {
				newSecret = secret
			}

			newActive := wh.Active
			if active != "" {
				active, err := strconv.ParseBool(active)
				if err != nil {
					return fmt.Errorf("invalid active value: %w", err)
				}

				newActive = active
			}

			newContentType := wh.ContentType
			if contentType != "" {
				var ct webhook.ContentType
				switch strings.ToLower(strings.TrimSpace(contentType)) {
				case "json":
					ct = webhook.ContentTypeJSON
				case "form":
					ct = webhook.ContentTypeForm
				default:
					return webhook.ErrInvalidContentType
				}
				newContentType = ct
			}

			newEvents := wh.Events
			if len(events) > 0 {
				var evs []webhook.Event
				for _, e := range events {
					ev, err := webhook.ParseEvent(e)
					if err != nil {
						return fmt.Errorf("invalid event: %w", err)
					}

					evs = append(evs, ev)
				}

				newEvents = evs
			}

			return be.UpdateWebhook(ctx, repo, id, newURL, newContentType, newSecret, newEvents, newActive)
		},
	}

	cmd.Flags().StringSliceVarP(&events, "events", "e", nil, fmt.Sprintf("events to trigger the webhook, available events are (%s)", strings.Join(webhookEvents, ", ")))
	cmd.Flags().StringVarP(&secret, "secret", "s", "", "secret to sign the webhook payload")
	cmd.Flags().StringVarP(&active, "active", "a", "", "whether the webhook is active")
	cmd.Flags().StringVarP(&contentType, "content-type", "c", "", "content type of the webhook payload, can be either `json` or `form`")
	cmd.Flags().StringVarP(&url, "url", "u", "", "webhook URL")

	return cmd
}

func webhookDeliveriesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deliveries",
		Short:   "Manage webhook deliveries",
		Aliases: []string{"delivery", "deliver"},
	}

	cmd.AddCommand(
		webhookDeliveriesListCommand(),
		webhookDeliveriesRedeliverCommand(),
		webhookDeliveriesGetCommand(),
	)

	return cmd
}

func webhookDeliveriesListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "list REPOSITORY WEBHOOK_ID",
		Short:             "List webhook deliveries",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			id, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid webhook ID: %w", err)
			}

			dels, err := be.ListWebhookDeliveries(ctx, id)
			if err != nil {
				return err
			}

			table := table.New().Headers("Status", "ID", "Event", "Created At")
			for _, d := range dels {
				status := "❌"
				if d.ResponseStatus >= 200 && d.ResponseStatus < 300 {
					status = "✅"
				}
				table = table.Row(
					status,
					d.ID.String(),
					d.Event.String(),
					humanize.Time(d.CreatedAt),
				)
			}
			cmd.Println(table)
			return nil
		},
	}

	return cmd
}

func webhookDeliveriesRedeliverCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "redeliver REPOSITORY WEBHOOK_ID DELIVERY_ID",
		Short:             "Redeliver a webhook delivery",
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			repo, err := be.Repository(ctx, args[0])
			if err != nil {
				return err
			}

			id, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid webhook ID: %w", err)
			}

			delID, err := uuid.Parse(args[2])
			if err != nil {
				return fmt.Errorf("invalid delivery ID: %w", err)
			}

			return be.RedeliverWebhookDelivery(ctx, repo, id, delID)
		},
	}

	return cmd
}

func webhookDeliveriesGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "get REPOSITORY WEBHOOK_ID DELIVERY_ID",
		Short:             "Get a webhook delivery",
		PersistentPreRunE: checkIfAdmin,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			be := backend.FromContext(ctx)
			id, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid webhook ID: %w", err)
			}

			delID, err := uuid.Parse(args[2])
			if err != nil {
				return fmt.Errorf("invalid delivery ID: %w", err)
			}

			del, err := be.WebhookDelivery(ctx, id, delID)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "ID: %s\n", del.ID)                             //nolint:errcheck
			fmt.Fprintf(out, "Event: %s\n", del.Event)                       //nolint:errcheck
			fmt.Fprintf(out, "Request URL: %s\n", del.RequestURL)            //nolint:errcheck
			fmt.Fprintf(out, "Request Method: %s\n", del.RequestMethod)      //nolint:errcheck
			fmt.Fprintf(out, "Request Error: %s\n", del.RequestError.String) //nolint:errcheck
			fmt.Fprintf(out, "Request Headers:\n")                           //nolint:errcheck
			reqHeaders := strings.Split(del.RequestHeaders, "\n")
			for _, h := range reqHeaders {
				fmt.Fprintf(out, "  %s\n", h) //nolint:errcheck
			}

			fmt.Fprintf(out, "Request Body:\n") //nolint:errcheck
			reqBody := strings.Split(del.RequestBody, "\n")
			for _, b := range reqBody {
				fmt.Fprintf(out, "  %s\n", b) //nolint:errcheck
			}

			fmt.Fprintf(out, "Response Status: %d\n", del.ResponseStatus) //nolint:errcheck
			fmt.Fprintf(out, "Response Headers:\n")                       //nolint:errcheck
			resHeaders := strings.Split(del.ResponseHeaders, "\n")
			for _, h := range resHeaders {
				fmt.Fprintf(out, "  %s\n", h) //nolint:errcheck
			}

			fmt.Fprintf(out, "Response Body:\n") //nolint:errcheck
			resBody := strings.Split(del.ResponseBody, "\n")
			for _, b := range resBody {
				fmt.Fprintf(out, "  %s\n", b) //nolint:errcheck
			}

			return nil
		},
	}

	return cmd
}
