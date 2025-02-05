package use

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"sarabi/client/internal/config"
)

func NewUseAppCmd(svc api.Service) *cobra.Command {
	var applicationID string
	cmd := &cobra.Command{
		Use:   "use",
		Short: "Use an application",
		Long:  "Use this command to notify to sarabi download the configuration for the specified application. The configuration will be stored in the current directory (.sarabi.yml), subsequent deployment from this directory will use this configuration",
		Run: func(cmd *cobra.Command, args []string) {
			appID, err := uuid.Parse(applicationID)
			if err != nil {
				cmdutil.Print(fmt.Sprintf("Invalid applicationID: %s", color.RedString(applicationID)))
				return
			}

			cmdutil.StartLoading("Working...")
			defer cmdutil.StopLoading()

			application, err := svc.GetApplication(cmd.Context(), appID)
			if err != nil {
				cmdutil.Print(fmt.Sprintf("failed to fetch application: %s", color.RedString(err.Error())))
				return
			}

			if err := createConfig(application); err != nil {
				cmdutil.Print(fmt.Sprintf("failed to create application config: %s", color.RedString(err.Error())))
				return
			}

			println()
			cmdutil.Print(fmt.Sprintf("Created application config: %s", color.GreenString(".sarabi.yml")))
		},
	}
	cmd.Flags().StringVarP(&applicationID, "id", "i", "", "The ID of the application you want to use for this directory")
	return cmd
}

func createConfig(application api.Application) error {
	cfgFile, err := os.Create(".sarabi.yml")
	if err != nil {
		return err
	}

	cfg := config.ApplicationConfig{
		ApplicationID:  application.ID,
		Domain:         application.Domain,
		StorageEngines: application.StorageEngines,
		Frontend:       application.Frontend,
		Backend:        application.Backend,
	}

	value, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	_, err = io.WriteString(cfgFile, string(value))
	if err != nil {
		return err
	}

	return nil
}
