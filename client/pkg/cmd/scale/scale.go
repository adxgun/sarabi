package scale

import (
	"context"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"sarabi/client/internal/config"
)

func NewScaleAppCmd(svc api.Service, cfg config.Config) *cobra.Command {
	var environment string
	var replicas int
	cmd := &cobra.Command{
		Use:     "scale",
		Short:   "Scale backend instances",
		Long:    "Use this command to increase/decrease the number of running backend instances for a specific app in your floki server",
		Example: "floki scale --env <environment> --replicas <number_of_replicas>",
		Run: func(cmd *cobra.Command, args []string) {
			if environment == "" {
				cmdutil.PrintE("Please specify environment")
				return
			}

			if replicas <= 0 {
				cmdutil.PrintE("Replicas must be > 0")
				return
			}

			cmdutil.StartLoading("Working...")
			defer cmdutil.StopLoading()

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			params := api.ScaleAppParams{
				Count:       replicas,
				Environment: environment,
			}
			err := svc.Scale(ctx, cfg.ApplicationID, params)
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			cmdutil.PrintS("Application updated!")
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "", "The name of the environment you want to scale")
	cmd.Flags().IntVarP(&replicas, "replicas", "i", 0, "The number of replicas")
	return cmd
}
