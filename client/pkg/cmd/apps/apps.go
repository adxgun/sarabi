package apps

import (
	"github.com/spf13/cobra"
	"log"
	"sarabi/client/internal/api"
	"sarabi/client/pkg/cmd/apps/create"
	"sarabi/client/pkg/cmd/apps/list"
)

func NewAppsCmd(svc api.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "apps <command>",
		Aliases: []string{"a"},
		Short:   "Manage sarabi applications",
		Long:    "Create, view and delete applications",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("Run apps!")
		},
	}

	cmd.AddCommand(create.NewCreateAppCmd(svc))
	cmd.AddCommand(list.NewListAppsCmd(svc))
	return cmd
}
