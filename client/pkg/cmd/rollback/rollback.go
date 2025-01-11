package rollback

import (
	"context"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"strings"
)

func NewRollbackCmd(svc api.Service) *cobra.Command {
	var identifier string
	cmd := &cobra.Command{
		Use:     "rollback",
		Short:   "Rollback to a previous deployment",
		Long:    "Rollback to a previous deployment using the deployment 'identifier'",
		Example: "sarabi rollback --identifier <deployment_identifier>",
		Run: func(cmd *cobra.Command, args []string) {
			identifier = strings.TrimSpace(identifier)
			if len(identifier) != 10 {
				cmdutil.PrintE("invalid identifier: " + identifier)
				return
			}

			cmdutil.StartLoading("Working...")
			defer cmdutil.StopLoading()

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			err := svc.Rollback(ctx, identifier)
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			cmdutil.PrintS("Rollback succeeded!")
		},
	}

	cmd.Flags().StringVarP(&identifier, "identifier", "i", "", "Identifier of the deployment to rollback")
	return cmd
}
