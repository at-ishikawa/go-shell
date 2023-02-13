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
	// suggests := getCommandSuggests()
	suggests := []prompt.Suggest{}

	// https://github.com/c-bata/kube-prompt/blob/fa7ef4fe0000bbeef8a2b0e022d23f44d83af5b7/main.go#L33-L41
	prompt.New(
		func(command string) {
			if err := runCommand(command); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		},
		func(d prompt.Document) []prompt.Suggest {
			currentWord := d.GetWordBeforeCursor()

			return prompt.FilterHasPrefix(suggests, currentWord, true)
		},
		prompt.OptionTitle("go-shell: interactive shell"),
		prompt.OptionPrefix("$ "),
		prompt.OptionInputTextColor(prompt.Yellow),
		prompt.OptionCompletionWordSeparator(completer.FilePathCompletionSeparator),
	).Run()
}

func getCommandSuggests() []prompt.Suggest {
	paths := strings.Split(os.Getenv("PATH"), ":")
	commands := []string{}
	for _, path := range paths {
		entries, err := os.ReadDir(path)
		if err != nil {
			fmt.Print("debug: failed to read a dir: " + path + ". error: ")
			fmt.Println(err)
			continue
		}
		for _, entry := range entries {
			commands = append(commands, entry.Name())
		}
	}
	suggests := make([]prompt.Suggest, len(commands))
	for index, command := range commands {
		suggests[index] = prompt.Suggest{
			Text: command,
			// Description: "",
		}
	}

	return suggests
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
