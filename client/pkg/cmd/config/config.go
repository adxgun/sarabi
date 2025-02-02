package configcmd

import (
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	initcmd "sarabi/client/pkg/cmd/config/init"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config <command>",
		Aliases: []string{"c"},
		Short:   "Manage sarabi client configuration",
	}

	cmd.AddCommand(initcmd.NewConfigInitCmd(api.NewPinger()))
	return cmd
}
