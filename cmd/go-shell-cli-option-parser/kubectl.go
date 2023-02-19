package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"

	"github.com/at-ishikawa/go-shell/internal/plugin/kubectl/kubectloptions"

	"github.com/spf13/cobra"
)

const kubeCtlCommand = "kubectl"

type kubectlOptionTemplateType struct {
	GlobalCommandOptions []kubectloptions.CLIOption
	SubCommandOptions    map[string][]kubectloptions.CLIOption
}

var kubectlOptionTemplate = `
// Code generated by go-shell-cli-option-parser DO NOT EDIT.

package kubectloptions

var KubeCtlGlobalOptions = []CLIOption{
{{ range .GlobalCommandOptions }}
	{
		ShortOption: "{{ .ShortOption }}",
		LongOption: "{{ .LongOption }}",
		HasDefaultValue: {{ .HasDefaultValue }},
	},
{{ end }}
}

var KubeCtlOptions = map[string][]CLIOption{
{{ range $k, $v := .SubCommandOptions }}
	"{{ $k }}": {
		{{ range $v }}
		{
			ShortOption: "{{ .ShortOption }}",
			LongOption: "{{ .LongOption }}",
			HasDefaultValue: {{ .HasDefaultValue }},
		},
		{{ end }}
	},
{{ end }}
}
`

func newOption(output []byte) []kubectloptions.CLIOption {
	var commandOptions []kubectloptions.CLIOption
	regExp := regexp.MustCompile(`(-(?P<ShortOption>[a-zA-Z-_]+), )?--(?P<LongOption>[a-zA-Z\-]+)=(?P<Value>.+):$`)
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] != '-' {
			continue
		}

		paramMap := extractParametersFromRegexp(regExp, line)
		hasDefaultValue := false
		if paramMap["Value"] == "true" || paramMap["Value"] == "false" {
			hasDefaultValue = true
		}

		commandOptions = append(commandOptions, kubectloptions.CLIOption{
			ShortOption:     paramMap["ShortOption"],
			LongOption:      paramMap["LongOption"],
			HasDefaultValue: hasDefaultValue,
		})
	}
	return commandOptions
}

func newKubeCtlCommand() *cobra.Command {
	kubeCtlCmd := &cobra.Command{
		Use: "kubectl",
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := exec.Command(kubeCtlCommand, "help").CombinedOutput()
			if err != nil {
				return fmt.Errorf("%s options error: %w", kubeCtlCommand, err)
			}
			var subCommands []string

			regExp := regexp.MustCompile(`^  (?P<subCommand>[a-zA-Z\-_]+)`)
			for _, line := range strings.Split(string(output), "\n") {
				if len(line) == 0 {
					continue
				}

				paramMap := extractParametersFromRegexp(regExp, line)
				subCommand, ok := paramMap["subCommand"]
				if !ok || subCommand == "kubectl" {
					continue
				}

				subCommands = append(subCommands, subCommand)
			}

			output, err = exec.Command(kubeCtlCommand, "options").CombinedOutput()
			if err != nil {
				return fmt.Errorf("%s options error: %w", kubeCtlCommand, err)
			}
			globalCommandOptions := newOption(output)

			allSubCommandOptions := map[string][]kubectloptions.CLIOption{}
			for _, subCommand := range subCommands {
				output, err := exec.Command(kubeCtlCommand, subCommand, "--help").CombinedOutput()
				if err != nil {
					return fmt.Errorf("%s %s --help error: %w", kubeCtlCommand, subCommand, err)
				}

				allSubCommandOptions[subCommand] = newOption(output)
			}

			t := template.Must(template.New("kubectl").Parse(kubectlOptionTemplate))

			var fileWriter io.Writer
			if outputFile == "" {
				fileWriter = os.Stdout
			} else {
				file, err := os.Create(outputFile)
				if err != nil {
					return fmt.Errorf("failed to create an output file: %w", err)
				}
				fileWriter = file
			}

			return t.Execute(fileWriter, kubectlOptionTemplateType{
				GlobalCommandOptions: globalCommandOptions,
				SubCommandOptions:    allSubCommandOptions,
			})
		},
	}
	return kubeCtlCmd
}
