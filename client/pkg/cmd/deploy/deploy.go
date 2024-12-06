package deploy

import (
	"context"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"sarabi/bundler"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"sarabi/client/internal/config"
	"strings"
	"time"
)

func NewDeployCmd(svc api.Service, cfg config.Config) *cobra.Command {
	deployParams := &api.DeployParams{
		Instances:     1,
		ApplicationID: cfg.ApplicationID,
	}
	mValidator := validator.New(validator.WithRequiredStructEnabled())

	cmd := &cobra.Command{
		Use:     "deploy",
		Short:   "Deploy an application",
		Long:    "Deploy an application on your server via sarabi.",
		Example: "'sarabi deploy --env dev'",
		Run: func(cmd *cobra.Command, args []string) {
			if err := mValidator.Struct(deployParams); err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			cmdutil.StartLoading("Bundling...")
			tmpFePath := ""
			tmpBePath := ""
			if cfg.Frontend != "" {
				tmpFePath = filepath.Join(os.Getenv("HOME"), "tmp", "frontend.tar.gz")
				if err := bundler.Gzip(cfg.Frontend, tmpFePath); err != nil {
					cmdutil.PrintE("failed to bundle frontend: " + err.Error())
					return
				}
			}

			if cfg.Backend != "" {
				tmpBePath = filepath.Join(os.Getenv("HOME"), "tmp", "backend.tar.gz")
				if err := bundler.Gzip(cfg.Backend, tmpBePath); err != nil {
					cmdutil.PrintE("failed to bundle backend: " + err.Error())
					return
				}
			}

			var frontend io.Reader
			var backend io.Reader
			var err error
			if tmpFePath != "" {
				frontend, err = os.Open(tmpFePath)
				if err != nil {
					cmdutil.PrintE("failed to bundle frontend: " + err.Error())
					return
				}
			}

			if tmpBePath != "" {
				backend, err = os.Open(tmpBePath)
				if err != nil {
					cmdutil.PrintE("failed to bundle backend: " + err.Error())
					return
				}
			}

			defer func() {
				if tmpBePath != "" {
					_ = os.Remove(tmpBePath)
				}
				if tmpFePath != "" {
					_ = os.Remove(tmpBePath)
				}
			}()

			cmdutil.StopLoading()

			cmdutil.StartLoading("Deploying...")
			defer cmdutil.StopLoading()

			ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
			defer cancel()
			resp, err := svc.Deploy(ctx, frontend, backend, *deployParams)
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			cmdutil.PrintS("Deployment succeeded! Identifier: " + resp.Identifier)
			if len(resp.AccessURL.Backend) > 0 {
				cmdutil.Print("Backend: " + strings.Join(resp.AccessURL.Backend, " | "))
			}
			if len(resp.AccessURL.Frontend) > 0 {
				cmdutil.Print("Frontend: " + strings.Join(resp.AccessURL.Frontend, " | "))
			}
		},
	}

	cmd.Flags().StringVarP(&deployParams.Environment, "env", "e", "", "Environment you're targeting for deployment")
	cmd.Flags().IntVarP(&deployParams.Instances, "instances", "i", 1, "Total number of instances to run")
	return cmd
}
