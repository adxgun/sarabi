package list

import (
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/internal/config"
)

func NewListBackupsCmd(svc api.Service, cfg config.Config) *cobra.Command {
	return &cobra.Command{}
}
