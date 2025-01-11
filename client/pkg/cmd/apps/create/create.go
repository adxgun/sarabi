package create

import (
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"sarabi/client/internal/api"
	"sarabi/client/internal/cmdutil"
	"sarabi/client/internal/config"
	"sarabi/internal/misc"
	"time"
)

var (
	storageEngines = []string{
		"None", "Postgresql", "Mysql", "MongoDB", "Redis", "Done",
	}
	cfgFileName = ".sarabi.yml"
	engineAlias = map[string]string{
		"Postgresql": "postgres",
		"Mysql":      "mysql",
		"MongoDB":    "mongo",
		"redis":      "redis",
	}
)

func NewCreateAppCmd(svc api.Service) *cobra.Command {
	mValidator := validator.New(validator.WithRequiredStructEnabled())
	selectedEngines := map[int]bool{}

	return &cobra.Command{
		Use:   "create",
		Short: "Create a new application",
		Long:  "Create a new application on sarabi, name of the application must be unique. An error will be returned if application with same name already exists",
		Run: func(cmd *cobra.Command, args []string) {
			namePrompt := createNamePrompt()
			name, err := namePrompt.Run()
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			domainNamePrompt := createDomainNamePrompt(mValidator)
			domainName, err := domainNamePrompt.Run()
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			idx := -1
			var result []string
			for {
				enginePrompt := createStorageEnginePrompt(selectedEngines)
				idx, _, err = enginePrompt.Run()
				if err != nil {
					return
				}

				// selects None or Done
				if idx == 0 || idx == len(storageEngines)-1 {
					break
				}
				if _, ok := selectedEngines[idx]; ok {
					delete(selectedEngines, idx)
				} else {
					selectedEngines[idx] = true
				}
			}

			for i, _ := range selectedEngines {
				value := storageEngines[i]
				if misc.StrContains(value, []string{
					"None", "Done",
				}) {
					continue
				}

				result = append(result, engineAlias[storageEngines[i]])
			}

			bePathPrompt := createBackendFilesPathPrompt()
			bePath, err := bePathPrompt.Run()
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			fePathPrompt := createFrontendFilesPathPrompt()
			fePath, err := fePathPrompt.Run()
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			param := &api.CreateApplicationParams{
				Name:          name,
				Domain:        domainName,
				StorageEngine: result,
				FePath:        fePath,
				BePath:        bePath,
			}

			if err := runValidator(mValidator, *param); err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			cmdutil.StartLoading(fmt.Sprintf("creating app: %s...", param.Name))
			defer cmdutil.StopLoading()

			app, err := createApp(svc, *param)
			if err != nil {
				cmdutil.PrintE(err.Error())
				return
			}

			cmdutil.PrintS("application created! " + app.ID.String())
			cmdutil.PrintS("run 'sarabi deploy -e dev' to deploy!")
		},
	}
}

func runValidator(v *validator.Validate, param api.CreateApplicationParams) error {
	err := v.Struct(param)
	if err != nil {
		var validationError validator.ValidationErrors
		if errors.As(err, &validationError) {
			for _, nextErr := range validationError {
				return fmt.Errorf("invalid value provided for: %s", nextErr.Field())
			}
		}
	}
	return nil
}

func createNamePrompt() promptui.Prompt {
	return promptui.Prompt{
		Label: "What's your application name?",
		Validate: func(s string) error {
			if len(s) <= 0 {
				return errors.New("please enter a valid name")
			}
			return nil
		},
	}
}

func createDomainNamePrompt(v *validator.Validate) promptui.Prompt {
	return promptui.Prompt{
		Label: "What's your application domain name?",
		Validate: func(s string) error {
			var validateError validator.ValidationErrors
			if err := v.Var(s, "fqdn"); err != nil {
				if errors.As(err, &validateError) {
					return fmt.Errorf("%s is not a fully qualified domain name", s)
				}
			}
			return nil
		},
	}
}

func createStorageEnginePrompt(selectedEngines map[int]bool) promptui.Select {
	return promptui.Select{
		Label: fmt.Sprintf("Select items (Currently selected: %d)", len(selectedEngines)),
		Items: storageEngines,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "▸ {{ . | cyan }}",
			Inactive: "  {{ . }}",
			Selected: "✔ {{ . | green }}",
			Details: `
				--------- Selected Items ---------
				{{ range $index, $item := . }}
				  - {{ $item }}
				{{ end }}`,
		},
	}
}

func createFrontendFilesPathPrompt() promptui.Prompt {
	return promptui.Prompt{
		Label: "Enter your frontend file location",
		Validate: func(s string) error {
			if s == "" {
				return nil
			}

			_, err := os.Open(s)
			if err != nil {
				return fmt.Errorf("specified directory returned error: %s: %s", s, err.Error())
			}
			return nil
		},
	}
}

func createBackendFilesPathPrompt() promptui.Prompt {
	return promptui.Prompt{
		Label: "Enter your backend file location",
		Validate: func(s string) error {
			if s == "" {
				return nil
			}

			_, err := os.Open(s)
			if err != nil {
				return fmt.Errorf("specified directory returned error: %s: %s", s, err.Error())
			}

			dockerFilePath := filepath.Join(s, "Dockerfile")
			_, err = os.Stat(dockerFilePath)
			if err != nil {
				return fmt.Errorf("Dockerfile not found in the directory specified: %s", s)
			}

			return nil
		},
	}
}

func createApp(svc api.Service, param api.CreateApplicationParams) (api.Application, error) {
	cfgFile, err := os.Create(cfgFileName)
	if err != nil {
		return api.Application{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
	defer cancel()
	app, err := svc.CreateApplication(ctx, param)
	if err != nil {
		return api.Application{}, err
	}

	cfg := config.Config{
		ApplicationID: app.ID,
		Frontend:      param.FePath,
		Backend:       param.BePath,
	}

	value, err := yaml.Marshal(cfg)
	if err != nil {
		return api.Application{}, err
	}

	_, err = io.WriteString(cfgFile, string(value))
	if err != nil {
		return api.Application{}, fmt.Errorf("failed to initialize application: %v", err)
	}

	return app, nil
}
