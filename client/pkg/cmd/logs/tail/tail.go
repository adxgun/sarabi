package tail

import (
	"context"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"sarabi/client/internal/config"
	"strings"
)

func NewTailLogsCmd(svc api.Service, cfg config.ApplicationConfig) *cobra.Command {
	var environment string
	cmd := &cobra.Command{
		Use:     "tail",
		Short:   "Tail logs",
		Long:    "Tail logs of an application, use --env flag to specify the environment",
		Example: "sarabi logs tail --env production",
		Run: func(cmd *cobra.Command, args []string) {
			if environment == "" {
				cmdutil.PrintE("environment is required")
				return
			}

			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
			defer cancel()

			resp, err := svc.TailLogs(ctx, cfg.ApplicationID, environment)
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			for {
				select {
				case ev := <-resp:
					handleLogEvent(ev, cancel)
				case <-ctx.Done():
					return
				}
			}
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "", "Environment in which to tail logs")
	return cmd
}

func handleLogEvent(ev api.Event, cancel context.CancelFunc) {
	switch ev.Type {
	case api.Info:
		cmdutil.Print(strings.Trim(ev.Message, "\n"))
	case api.Error:
		cancel()
		cmdutil.PrintE(strings.Trim(ev.Message, "\n"))
	}
}
