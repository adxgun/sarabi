package apps

import (
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/pkg/cmd/apps/create"
	"sarabi/client/pkg/cmd/apps/list"
	"sarabi/client/pkg/cmd/apps/use"
)

func NewAppsCmd(svc api.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "apps <command>",
		Aliases: []string{"a"},
		Short:   "Manage sarabi applications",
		Long:    "Create, view and delete applications",
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: validate initial configuration is set
		},
	}

	cmd.AddCommand(create.NewCreateAppCmd(svc))
	cmd.AddCommand(list.NewListAppsCmd(svc))
	cmd.AddCommand(use.NewUseAppCmd(svc))
	return cmd
}
