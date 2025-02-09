package stream

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

func NewStreamLogsCmd(svc api.Service, cfg config.ApplicationConfig) *cobra.Command {
	var environment, since, start, end string

	cmd := &cobra.Command{
		Use:     "stream",
		Short:   "Stream application logs",
		Long:    steamDocs,
		Example: "sarabi logs stream --env production --since 1h",
		Run: func(cmd *cobra.Command, args []string) {
			if environment == "" {
				cmdutil.PrintE("environment is required")
				return
			}

			filter := api.LogFilterParams{
				Environment: environment,
				Since:       since,
				Start:       start,
				End:         end,
			}

			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
			defer cancel()

			resp, err := svc.StreamLogs(ctx, cfg.ApplicationID, filter)
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

	cmd.Flags().StringVarP(&environment, "env", "e", "", "Environment where you want to stream logs from")
	cmd.Flags().StringVarP(&since, "since", "", "", "Duration with respect to current time")
	cmd.Flags().StringVarP(&start, "start", "", "", "Date from when the logs should start: Format accepted are YYYY-mm-dd hh:mm:ss or YYYY-mm-dd")
	cmd.Flags().StringVarP(&end, "end", "", "", "Date from when the logs should stop: Format accepted are YYYY-mm-dd hh:mm:ss or YYYY-mm-dd")
	return cmd
}

func handleLogEvent(ev api.Event, cancel context.CancelFunc) {
	switch ev.Type {
	case api.Info:
		cmdutil.Print(strings.Trim(ev.Message, "\n"))
	case api.Error:
		cancel()
		cmdutil.PrintE(strings.Trim(ev.Message, "\n"))
	case api.Complete:
		cancel()
	}
}

const steamDocs = `
Stream application logs. You can use parameters such as 'since', 'start', 'end' to request for specific logs with respect to date. e.g 
'sarabi logs stream --env dev --since 10m' which will return all application logs from 10 minutes ago. 
Another example 'sarabi logs stream --env dev --start 2025-02-08 03:00:00 --end 2025-02-08 06:00:00' which will return all application logs from 3pm to 6pm on the specified date 
`
