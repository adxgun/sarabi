package domains

import (
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/internal/config"
	"sarabi/client/pkg/cmd/domains/add"
)

func NewDomainsCmd(svc api.Service, cfg config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains <command>",
		Short: "Manage sarabi application domains",
		Long:  "Add, remove and view application domains",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	cmd.AddCommand(add.NewAddDomainCmd(svc, cfg))
	return cmd
}
