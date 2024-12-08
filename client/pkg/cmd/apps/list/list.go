package list

import (
	"context"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"strings"
	"time"
)

func NewListAppsCmd(svc api.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List applications",
		Long:  "List all the applications on your sarabi server",
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.StartLoading("Working...")
			defer cmdutil.StopLoading()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			apps, err := svc.ListApplications(ctx)
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			header := table.Row{"ID", "Name", "Domain", "Storage Engines", "Time Created"}
			tw := table.NewWriter()
			tw.AppendHeader(header)
			for _, next := range apps {
				row := table.Row{
					next.ID.String(),
					next.Name,
					next.Domain,
					strings.Join(next.StorageEngines, ", "),
					next.CreatedAt.Format("02-01-2006"),
				}
				tw.AppendRow(row)
				tw.AppendSeparator()
			}
			cmdutil.Print("")
			cmdutil.Print(tw.Render())
		},
	}
}
