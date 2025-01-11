package tail

import (
	"bufio"
	"encoding/json"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"sarabi/client/internal/config"
)

func NewTailLogsCmd(svc api.Service, cfg config.Config) *cobra.Command {
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

			defer func() {
				_ = resp.Close()
			}()

			scanner := bufio.NewScanner(resp)
			for scanner.Scan() {
				entry := api.LogEntry{}
				if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
					continue
				}

				cmdutil.Print(entry.Owner + " " + entry.Log)
			}
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "", "Environment in which to tail logs")
	return cmd
}
