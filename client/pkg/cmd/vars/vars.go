package vars

import (
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/internal/config"
	"sarabi/client/pkg/cmd/vars/update"
)

func NewVarsCmd(svc api.Service, cfg config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vars <command>",
		Aliases: []string{"a"},
		Short:   "Manage sarabi applications variables",
		Long:    "Create, update, view and delete variables",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	cmd.AddCommand(update.NewUpdateVarsCmd(svc, cfg))
	return cmd
}
