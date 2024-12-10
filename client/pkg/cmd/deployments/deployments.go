package deployments

import (
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/internal/config"
	"sarabi/client/pkg/cmd/deployments/list"
)

func NewDeploymentsCmd(svc api.Service, cfg config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployments <command>",
		Aliases: []string{"d"},
		Short:   "Manage sarabi deployments",
		Long:    "Create, view and delete applications",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
	cmd.AddCommand(list.NewListDeploymentsCmd(svc, cfg))
	return cmd
}
