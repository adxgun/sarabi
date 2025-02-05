package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/internal/auth"
	"sarabi/client/internal/config"
	"sarabi/client/pkg/cmd/apps"
	"sarabi/client/pkg/cmd/backup"
	configcmd "sarabi/client/pkg/cmd/config"
	"sarabi/client/pkg/cmd/deploy"
	"sarabi/client/pkg/cmd/deployments"
	"sarabi/client/pkg/cmd/destroy"
	"sarabi/client/pkg/cmd/domains"
	"sarabi/client/pkg/cmd/logs"
	"sarabi/client/pkg/cmd/rollback"
	"sarabi/client/pkg/cmd/scale"
	"sarabi/client/pkg/cmd/vars"
)

func New() (*cobra.Command, error) {
	apiClient, appConfig, err := validateConfig()
	cfg, apiAccessKey, initerr := validateInitConfig()

	cmd := &cobra.Command{
		Use:   "sarabi",
		Short: "sarabi - the fullstack application deployment tool",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "init" {
				return nil
			}

			if cmd.Name() == "create" {
				return initerr
			}

			return err
		},
	}

	if apiClient == nil {
		apiClient = api.NewClient(api.Config{
			Host:      cfg.Host,
			AccessKey: apiAccessKey,
		})
	}

	svc := api.NewService(apiClient)
	cmd.AddCommand(apps.NewAppsCmd(svc))
	cmd.AddCommand(deploy.NewDeployCmd(svc, appConfig))
	cmd.AddCommand(vars.NewVarsCmd(svc, appConfig))
	cmd.AddCommand(destroy.NewDestroyDeploymentCmd(svc, appConfig))
	cmd.AddCommand(domains.NewDomainsCmd(svc, appConfig))
	cmd.AddCommand(deployments.NewDeploymentsCmd(svc, appConfig))
	cmd.AddCommand(scale.NewScaleAppCmd(svc, appConfig))
	cmd.AddCommand(rollback.NewRollbackCmd(svc))
	cmd.AddCommand(backup.NewBackupCmd(svc, appConfig))
	cmd.AddCommand(logs.NewLogsCmd(svc, appConfig))
	cmd.AddCommand(configcmd.NewConfigCmd())
	return cmd, nil
}

func validateInitConfig() (config.Config, string, error) {
	cfg, err := config.Parse()
	if err != nil {
		return config.Config{}, "", fmt.Errorf("failed to parse sarabi configuration: Did you call <sarabi config init>?")
	}

	apiAccessKey, err := auth.Get()
	if err != nil {
		return config.Config{}, "", fmt.Errorf("failed to parse sarabi configuration: Did you call <sarabi config init>?")
	}

	return cfg, apiAccessKey, nil
}

func validateConfig() (api.Client, config.ApplicationConfig, error) {
	cfg, apiAccessKey, err := validateInitConfig()
	if err != nil {
		return nil, config.ApplicationConfig{}, err
	}

	appConfig, err := config.ParseApplicationConfig()
	if err != nil {
		return nil, config.ApplicationConfig{}, fmt.Errorf("application configuration not found. are you sure you are in your app root directory?")
	}

	apiClient := api.NewClient(api.Config{
		Host:      cfg.Host,
		AccessKey: apiAccessKey,
	})
	return apiClient, appConfig, nil
}
