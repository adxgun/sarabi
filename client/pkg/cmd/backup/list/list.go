package list

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"sarabi/client/internal/config"
)

func NewListBackupsCmd(svc api.Service, cfg config.Config) *cobra.Command {
	var environment string
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all backups",
		Long:    "List all the backups that has been created for this application. You can filter by environment.",
		Example: "floki backup list --env <environment>",
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.StartLoading("Working...")
			defer cmdutil.StopLoading()

			backups, err := svc.ListBackups(cmd.Context(), cfg.ApplicationID, environment)
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			tw := table.NewWriter()
			header := table.Row{"ID", "Environment", "Location", "Database Engine", "Created At", "Size"}
			tw.AppendHeader(header)
			tw.SetStyle(table.StyleLight)
			tw.AppendSeparator()

			for _, backup := range backups {
				row := table.Row{
					backup.ID,
					backup.Environment,
					backup.StorageTypeString(),
					backup.StorageEngine,
					backup.CreatedAt.Format("2006-01-02 15:04:05"),
					"100 MB",
				}
				tw.AppendRow(row)
				tw.AppendSeparator()
			}

			cmdutil.Print("")
			cmdutil.Print(tw.Render())
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "", "Name of the environment you want to list backups for")
	return cmd
}
