package main

import (
	"fmt"
	"os"

	"github.com/at-ishikawa/go-shell/internal/shell"
	"github.com/spf13/cobra"
)

func main() {
	rootCommand := &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {
			s, err := shell.NewShell(os.Stdin, os.Stdout)
			if err != nil {
				fmt.Println(err)
			}
			if err := s.Run(); err != nil {
				fmt.Println(err)
			}
		},
	}

	if err := rootCommand.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
