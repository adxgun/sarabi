package list

import (
	"context"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"sarabi/client/internal/config"
	"time"
)

func NewListVarsCmd(svc api.Service, cfg config.Config) *cobra.Command {
	var environment string
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List application variables/secrets",
		Long:    "List all application variables/secrets, you can specify '--env' flag to list vars for a specific environment",
		Example: "sarabi vars list --env <environment>",
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.StartLoading("Working...")
			defer cmdutil.StopLoading()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vars, err := svc.ListVariables(ctx, cfg.ApplicationID, environment)
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			tw := table.NewWriter()
			header := table.Row{"ID", "Name", "Value", "Environment"}
			tw.AppendHeader(header)

			for _, v := range vars {
				row := table.Row{
					v.ID,
					v.Name,
					v.Value,
					v.Environment,
				}
				tw.AppendRow(row)
				tw.AppendSeparator()
			}
			cmdutil.Print(tw.Render())
		},
	}
	cmd.Flags().StringVarP(&environment, "env", "e", "", "Environment you want to list its variables/secrets, empty value will return variables/secrets for all environments")
	return cmd
}
