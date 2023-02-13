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
			if err := shell.Run(os.Stdin); err != nil {
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
