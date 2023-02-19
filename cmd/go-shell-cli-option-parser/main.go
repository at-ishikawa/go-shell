package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/cobra"
)

// persistent flags
var outputFile string

func extractParametersFromRegexp(regExp *regexp.Regexp, value string) (params map[string]string) {
	match := regExp.FindStringSubmatch(value)

	params = make(map[string]string)
	for i, name := range regExp.SubexpNames() {
		if i > 0 && i <= len(match) {
			params[name] = match[i]
		}
	}
	return params
}

func main() {
	rootCmd := cobra.Command{}
	flags := rootCmd.PersistentFlags()
	flags.StringVarP(&outputFile, "output", "o", "", "the output file path")

	rootCmd.AddCommand(newKubeCtlCommand())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
