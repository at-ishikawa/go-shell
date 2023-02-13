package shell

import (
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
)

type shellCompleter struct {
	commandSuggests   []prompt.Suggest
	filePathCompleter completer.FilePathCompleter
}

func createCompleter() shellCompleter {
	return shellCompleter{
		commandSuggests:   getCommandSuggests(),
		filePathCompleter: completer.FilePathCompleter{},
	}
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

func (c shellCompleter) complete(d prompt.Document) []prompt.Suggest {
	if d.TextBeforeCursor() == "" {
		return []prompt.Suggest{}
	}
	args := strings.Split(d.TextBeforeCursor(), " ")
	word := d.GetWordBeforeCursor()

	// The first input must be a command
	if len(args) == 1 {
		return prompt.FilterHasPrefix(c.commandSuggests, word, true)
	}

	// TODO: Use file candidates after a command as default
	return c.filePathCompleter.Complete(d)
}
