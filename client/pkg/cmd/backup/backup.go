package backup

import (
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/internal/config"
	"sarabi/client/pkg/cmd/backup/schedule"
)

func NewBackupCmd(svc api.Service, cfg config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "backup <command>",
		Aliases: []string{"bc"},
		Short:   "Manage floki applications backup",
		Long:    "Create view, delete applications database backup settings. Download a specific backup file",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	cmd.AddCommand(schedule.NewCreateBackupScheduleCmd(svc, cfg))
	return cmd
}
