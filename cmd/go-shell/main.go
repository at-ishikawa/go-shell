package main

import (
	"fmt"
	"os"

	"github.com/at-ishikawa/go-shell/internal/shell"
	"github.com/spf13/cobra"
)

func main() {
	var commandLineOptions shell.Options

	rootCommand := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			tty, err := os.Open("/dev/tty")
			if err != nil {
				return err
			}

			s, err := shell.NewShell(
				tty,
				os.Stdout,
				os.Stderr,
				commandLineOptions,
			)
			if err != nil {
				return err
			}
			return s.Run()
		},
	}

	rootCommand.PersistentFlags().BoolVarP(&commandLineOptions.IsDebug, "debug", "", false, "Enable to a debug mode")
	if err := rootCommand.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
