package cmd

import (
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/pkg/cmd/apps"
)

func New() (*cobra.Command, error) {
	apiClient, err := api.NewClient()
	if err != nil {
		return nil, err
	}

	svc := api.NewService(apiClient)

	cmd := &cobra.Command{
		Use:   "sarabi",
		Short: "sarabi - the fullstack application deployment tool",
	}

	cmd.AddCommand(apps.NewAppsCmd(svc))
	return cmd, nil
}
