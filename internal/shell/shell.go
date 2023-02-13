package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
)

// https://hackernoon.com/today-i-learned-making-a-simple-interactive-shell-application-in-golang-aa83adcb266a

func Run() {
	// https://github.com/c-bata/kube-prompt/blob/fa7ef4fe0000bbeef8a2b0e022d23f44d83af5b7/main.go#L33-L41
	prompt.New(
		func(command string) {
			if err := runCommand(command); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		},
		createCompleter().complete,
		prompt.OptionTitle("go-shell: interactive shell"),
		prompt.OptionPrefix("$ "),
		prompt.OptionInputTextColor(prompt.Yellow),
		prompt.OptionCompletionWordSeparator(completer.FilePathCompletionSeparator),
	).Run()
}

func runCommand(commandStr string) error {
	commandStr = strings.TrimSuffix(commandStr, "\n")
	arrCommandStr := strings.Fields(commandStr)
	if len(arrCommandStr) == 0 {
		return nil
	}

	switch arrCommandStr[0] {
	case "exit":
		os.Exit(0)
		// add another case here for custom commands.
	}

	cmd := exec.Command(arrCommandStr[0], arrCommandStr[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
