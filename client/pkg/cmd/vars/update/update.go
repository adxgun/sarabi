package update

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"sarabi/client/internal/config"
	"strings"
)

func NewUpdateVarsCmd(svc api.Service, cfg config.ApplicationConfig) *cobra.Command {
	var environment, varFile string
	var vars []string
	params := &api.UpdateVariablesParams{}

	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Update environment variables for backend instance(s)",
		Long:    "Update environment variables. You can update these values via a .env file or you pass the values directly to the stdin. Note: This command restarts the application container",
		Example: "sarabi vars update --env <environment> --file <path_to_env_file> or sarabi vars update --env <environment> --var KEY1=value KEY2=value ...",
		Run: func(cmd *cobra.Command, args []string) {
			if environment == "" {
				cmdutil.PrintE("Please specify environment --env")
				return
			}
			params.Environment = environment

			var err error
			if varFile != "" {
				params.Vars, err = parseVarFile(varFile)
				if err != nil {
					cmdutil.PrintE(err.Error())
					return
				}
			}

			if len(vars) > 0 {
				params.Vars, err = parseVars(vars)
				if err != nil {
					cmdutil.PrintE(err.Error())
					return
				}
			}

			cmdutil.StartLoading(fmt.Sprintf("updating variables...(%d)", len(params.Vars)))
			defer cmdutil.StopLoading()

			err = svc.UpdateVariables(cmd.Context(), cfg.ApplicationID, *params)
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			cmdutil.PrintS("variables updated!")
		},
	}

	cmd.Flags().StringArrayVarP(&vars, "var", "v", []string{}, "specify variable you will like to add or update")
	cmd.Flags().StringVarP(&environment, "env", "e", "", "Environment you want the update to reflect")
	cmd.Flags().StringVarP(&varFile, "file", "f", "", "Specify a .env file, sarabi will parse the values and update your application")
	return cmd
}

func parseVarFile(f string) ([]api.KV, error) {
	values, err := godotenv.Read(f)
	if err != nil {
		return nil, err
	}

	result := make([]api.KV, 0)
	for k, v := range values {
		result = append(result, api.KV{
			Key:   k,
			Value: v,
		})
	}
	return result, nil
}

func parseVars(vars []string) ([]api.KV, error) {
	result := make([]api.KV, 0)
	for _, next := range vars {
		v := strings.SplitN(next, "=", 2)
		if len(v) != 2 {
			return nil, fmt.Errorf("invalid variable format: %s, expected key=value", next)
		}
		result = append(result, api.KV{
			Key:   v[0],
			Value: v[1],
		})
	}
	return result, nil
}
