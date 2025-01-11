package destroy

import (
	"context"
	"github.com/google/uuid"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"sarabi/client/internal/config"
	"sarabi/internal/misc"
	"time"
)

func NewDestroyDeploymentCmd(svc api.Service, cfg config.Config) *cobra.Command {
	var environment string
	cmd := &cobra.Command{
		Use:     "destroy",
		Short:   "Destroy/Take down a deployment",
		Long:    "Destroy/Take down a deployment on your sarabi server. If you specify the 'environment' flag, the deploy for this specific environment will be taken down or else all the deployment associated with this app will be destroyed/taken down",
		Example: "sarabi deploy destroy --env <environment>",
		Run: func(cmd *cobra.Command, args []string) {
			if environment == "" {
				p := promptui.Prompt{
					Label:     "Are you sure you want to take down all the deployment for this application?",
					IsConfirm: true,
				}
				result, err := p.Run()
				if err != nil {
					cmdutil.PrintE(err.Error())
					return
				}

				if misc.StrContains(result, []string{"Yes", "yes", "y"}) {
					if err := runDestroy(svc, cfg.ApplicationID, ""); err != nil {
						cmdutil.PrintE(err.Error())
						return
					} else {
						cmdutil.PrintS("Operation succeeded!")
					}
				}
			} else {
				err := runDestroy(svc, cfg.ApplicationID, environment)
				if err != nil {
					cmdutil.PrintE(err.Error())
				} else {
					cmdutil.PrintS("Operation succeeded!")
				}
			}
		},
	}
	cmd.Flags().StringVarP(&environment, "env", "e", "", "The environment you want to destroy its resources. WARNING: empty value means all deployment in all environments will be destroyed")
	return cmd
}

func runDestroy(svc api.Service, applicationID uuid.UUID, environment string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return svc.Destroy(ctx, applicationID, environment)
}
