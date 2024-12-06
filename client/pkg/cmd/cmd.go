package cmd

import (
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/internal/config"
	"sarabi/client/pkg/cmd/apps"
	"sarabi/client/pkg/cmd/deploy"
)

func New() (*cobra.Command, error) {
	apiClient, err := api.NewClient()
	if err != nil {
		return nil, err
	}

	svc := api.NewService(apiClient)
	cfg, err := config.Parse()
	if err != nil {
		return nil, err
	}

	cmd := &cobra.Command{
		Use:   "sarabi",
		Short: "sarabi - the fullstack application deployment tool",
	}

	cmd.AddCommand(apps.NewAppsCmd(svc))
	cmd.AddCommand(deploy.NewDeployCmd(svc, cfg))
	return cmd, nil
}
