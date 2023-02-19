package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"

	"github.com/at-ishikawa/go-shell/internal/kubectl/kubectloptions"
	"github.com/spf13/cobra"
)

type kubectlOptionTemplateType struct {
	Name    string
	Options []kubectloptions.CLIOption
}

var kubectlOptionTemplate = `
package kubectloptions

var KubeCtl{{ .Name }}CommandOptions = []CLIOption{
{{ range .Options }}
	{
		ShortOption: "{{ .ShortOption }}",
		LongOption: "{{ .LongOption }}",
		HasDefaultValue: {{ .HasDefaultValue }},
	},
{{ end }}
}
`

func newKubeCtlCommand() *cobra.Command {
	kubeCtlCmd := &cobra.Command{
		Use: "kubectl",
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := exec.Command("kubectl", "get", "--help").CombinedOutput()
			if err != nil {
				return fmt.Errorf("kubectl get --help error: %w", err)
			}

			commandOptions := []kubectloptions.CLIOption{}
			// r := regexp.MustCompile(`(-(?P<ShortOption>[a-zA-Z-_]+), )?--(?P<LongOption>[a-zA-Z]+)=(?P<Value>[!:]+):`)
			r := regexp.MustCompile(`(-(?P<ShortOption>[a-zA-Z-_]+), )?--(?P<LongOption>[a-zA-Z\-]+)=(?P<Value>.+):$`)
			for _, line := range strings.Split(string(output), "\n") {
				line = strings.TrimSpace(line)
				if len(line) == 0 || line[0] != '-' {
					continue
				}

				paramMap := extractParametersFromRegexp(r, line)
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
				Name:    "Get",
				Options: commandOptions,
			})
		},
	}
	return kubeCtlCmd
}
