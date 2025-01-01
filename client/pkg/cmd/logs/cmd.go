package logs

import (
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/internal/config"
	"sarabi/client/pkg/cmd/logs/tail"
)

func NewLogsCmd(svc api.Service, cfg config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "logs <command>",
		Aliases: []string{"l"},
		Short:   "Manage floki applications logs",
		Long:    "Tail logs, view logs",
		Run:     func(cmd *cobra.Command, args []string) {},
	}

	cmd.AddCommand(tail.NewTailLogsCmd(svc, cfg))
	return cmd
}
