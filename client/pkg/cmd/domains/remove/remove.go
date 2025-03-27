package remove

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"sarabi/client/internal/config"
)

func NewRemoveDomainCmd(svc api.Service, cfg config.ApplicationConfig) *cobra.Command {
	mValidator := validator.New(validator.WithRequiredStructEnabled())
	param := &api.RemoveDomainParam{}

	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "Remove domain from an application",
		Long:    "Remove the specified domain name from the application.",
		Example: "sarabi domains remove <fqdn>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) <= 0 {
				cmdutil.PrintE("please input your FQDN")
				return
			}

			param.FQDN = args[0]
			if err := mValidator.Var(param.FQDN, "fqdn"); err != nil {
				cmdutil.PrintE(fmt.Sprintf("%s is not a fully qualified domain name", param.FQDN))
				return
			}

			cmdutil.StartLoading("Working...")
			defer cmdutil.StopLoading()

			err := svc.RemoveDomain(cmd.Context(), cfg.ApplicationID, param.FQDN)
			if err != nil {
				cmdutil.Print(fmt.Sprintf("cannot remove domain: %s", color.RedString(err.Error())))
				return
			}

			cmdutil.PrintS("Operation succeeded")
		},
	}

	cmd.Flags().StringVarP(&param.FQDN, "fqdn", "n", "", "Domain name to remove")
	return cmd
}
