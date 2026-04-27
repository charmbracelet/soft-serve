package cmd

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/jobs"
	"github.com/spf13/cobra"
)

// CronJobCommand returns a command for manually triggering cronjobs
func CronJobCommand(ctx context.Context) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:               "cronjob",
		Short:             "Run cron job task",
		PersistentPreRunE: checkIfAdmin,
	}

	for name, j := range jobs.List() {
		cfg, err := j.Runner.Config(ctx)
		if err != nil {
			return nil, err
		}

		cronCmd := &cobra.Command{
			Use:   name,
			Short: j.Runner.Description(),
			Run: func(cmd *cobra.Command, args []string) {
				cfg.SetOut(cmd.OutOrStdout())
				cfg.SetErr(cmd.OutOrStderr())

				j.Runner.Func(cmd.Context(), cfg)()
			},
		}
		cronCmd.Flags().AddFlagSet(cfg.FlagSet())

		cmd.AddCommand(cronCmd)
	}

	return cmd, nil
}
