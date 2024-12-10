package list

import (
	"context"
	"github.com/google/uuid"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"sarabi/client/internal/config"
)

func NewListDeploymentsCmd(svc api.Service, cfg config.Config) *cobra.Command {
	instance := ""
	environment := ""

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all deployments for an application",
		Long:    "List all the deployments for the specified application. This command will show you all the running instances for backend, all the databases and the frontend of your application",
		Example: "'sarabi deployments list --instance database --env dev --app app_name'",
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.StartLoading("Working...")
			defer cmdutil.StopLoading()

			deps, err := listDeployments(svc, cfg.ApplicationID, instance, environment)
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			writer := table.NewWriter()
			writer.AppendHeader(table.Row{"Identifier", "Name", "Environment", "Instance Type", "Instances", "Status", "Created Time"})
			for _, dep := range deps {
				row := table.Row{
					dep.Identifier,
					dep.Name,
					dep.Environment,
					dep.InstanceType,
					dep.Instances,
					dep.Status,
					dep.CreatedAt.Format("2006-02-01"),
				}
				writer.AppendRow(row)
				writer.AppendSeparator()
			}

			cmdutil.Print("")
			cmdutil.Print(writer.Render())
		},
	}

	cmd.Flags().StringVarP(&instance, "type", "t", "", "The instance type you want listed: accepted values are backend, frontend, database")
	cmd.Flags().StringVarP(&environment, "env", "e", "", "The environment you want to see it deployments")
	return cmd
}

func listDeployments(svc api.Service, applicationID uuid.UUID, instance, environment string) ([]api.Deployment, error) {
	result := make([]api.Deployment, 0)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	deps, err := svc.ListDeployments(ctx, applicationID)
	if err != nil {
		return nil, err
	}

	if instance != "" {
		for _, next := range deps {
			if next.InstanceType == instance {
				result = append(result, next)
			}
		}
		return result, nil
	}

	if environment != "" {
		for _, next := range deps {
			if next.Environment == environment {
				result = append(result, next)
			}
		}
		return result, nil
	}

	return deps, nil
}
