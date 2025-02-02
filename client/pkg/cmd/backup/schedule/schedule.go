package schedule

import (
	"context"
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"sarabi/client/internal/config"
)

func NewCreateBackupScheduleCmd(svc api.Service, cfg config.ApplicationConfig) *cobra.Command {
	var environment string
	var cronExpression string
	cmd := &cobra.Command{
		Use:     "schedule",
		Short:   "Create a backup schedule",
		Long:    "Schedule database backup for a specific environment.",
		Example: "sarabi backup schedule --env <environment> --expression <cron_expression>",
		Run: func(cmd *cobra.Command, args []string) {
			if environment == "" {
				cmdutil.PrintE("Please specify environment")
				return
			}

			if err := validateExpression(cronExpression); err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			cmdutil.StartLoading("Working...")
			defer cmdutil.StopLoading()

			params := api.CreateBackupParams{
				Environment:    environment,
				CronExpression: cronExpression,
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			err := svc.CreateBackupSchedule(ctx, cfg.ApplicationID, params)
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			cmdutil.PrintS("Backup schedule updated!")
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "", "Environment you want to schedule backup for")
	cmd.Flags().StringVarP(&cronExpression, "expression", "x", "", "The cron expression that defines how often the backup should run")
	return cmd
}

func validateExpression(value string) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(value)
	if err != nil {

		return fmt.Errorf("invalid cron expression: %w", err)
	}
	return nil
}
