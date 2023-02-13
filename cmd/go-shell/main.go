package main

import (
	"log"
	"os"

	"github.com/at-ishikawa/go-shell/internal/shell"
	"github.com/spf13/cobra"
)

func main() {
	rootCommand := &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {
			shell.Run()
		},
	}
	if err := rootCommand.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
