package add

import (
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sarabi"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"sarabi/client/internal/config"
	"time"
)

func NewAddDomainCmd(svc api.Service, cfg config.Config) *cobra.Command {
	mValidator := validator.New(validator.WithRequiredStructEnabled())
	param := &api.AddDomainParam{}

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a domain to an application",
		Long:  "Add the specified domain to an application. You must specify the environment and the instance you want to apply the domain to, e.g 'dev.app.io dev backend'",
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

			if err := mValidator.Struct(param); err != nil {
				var vError validator.ValidationErrors
				if errors.As(err, &vError) {
					for _, nextErr := range vError {
						cmdutil.PrintE(fmt.Sprintf("Invalid value input for: %s", nextErr.Field()))
					}
				}
				return
			}

			if !sarabi.StrContains(param.Instance, []string{"backend", "frontend"}) {
				cmdutil.PrintE(fmt.Sprintf("Instance type must be backend or frontend: received unknown value %s", param.Instance))
				return
			}

			cmdutil.StartLoading("Working...")
			defer cmdutil.StopLoading()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			err := svc.AddDomain(ctx, cfg.ApplicationID, *param)
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			cmdutil.PrintS("Operation succeeded!")
		},
	}

	cmd.Flags().StringVarP(&param.FQDN, "fqdn", "n", "", "The fully qualified domain name(e.g dev.mysuperapp.io) you wish to add e.g --fqdn dev.mysuperapp.io")
	cmd.Flags().StringVarP(&param.Environment, "env", "e", "", "The environment you wish to add this domain to. e.g --env staging")
	cmd.Flags().StringVarP(&param.Instance, "instance", "i", "", "The instance type this domain belongs to. e.g --instance backend or --instance frontend")
	return cmd
}
