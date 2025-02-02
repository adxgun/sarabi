package initcmd

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"net/url"
	"os"
	"sarabi/client/internal/api"
	"sarabi/client/internal/auth"
	"sarabi/client/internal/cmdutil"
	"sarabi/client/internal/config"
	"strings"
)

func NewConfigInitCmd(svc api.Pinger) *cobra.Command {
	var host, accessKey string
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Set sarabi configuration",
		Long:    "Set sarabi configuration",
		Example: "sarabi config init --host <https://sarabi.dev> --access-key <key>",
		Run: func(cmd *cobra.Command, args []string) {
			uri, err := url.Parse(host)
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			// TODO: validate length
			if len(accessKey) == 0 {
				cmdutil.PrintE("Invalid access key: Length must be at least 32")
				return
			}

			cmdutil.StartLoading("Running test...")
			defer cmdutil.StopLoading()

			serverUrl := toURL(uri)
			err = svc.Ping(cmd.Context(), serverUrl, accessKey)
			if err != nil {
				color.Cyan(err.Error())
				return
			}

			if err := config.SaveConfig(config.Config{Host: serverUrl}); err != nil {
				cmdutil.Print(fmt.Sprintf("Failed to save config: %s", color.RedString(err.Error())))
				return
			}

			if err := auth.Save(accessKey); err != nil {
				cmdutil.Print(fmt.Sprintf("Failed to save access key: %s", color.RedString(err.Error())))
				return
			}

			_, _ = fmt.Fprintln(os.Stdout, fmt.Sprintf("\n%s: Configuration set successfully", color.GreenString("Test passed")))
		},
	}
	cmd.Flags().StringVarP(&host, "host", "i", "", "sarabi server host url")
	cmd.Flags().StringVarP(&accessKey, "access-key", "a", "", "sarabi server access key")
	return cmd
}

func toURL(u *url.URL) string {
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	return u.String()
}
