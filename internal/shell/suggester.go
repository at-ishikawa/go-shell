package shell

import (
	"strings"

	"github.com/at-ishikawa/go-shell/internal/completion"
	"github.com/at-ishikawa/go-shell/internal/config"
	"github.com/at-ishikawa/go-shell/internal/plugin"
	"github.com/at-ishikawa/go-shell/internal/plugin/git"
	"github.com/at-ishikawa/go-shell/internal/plugin/kubectl"
)

type suggester struct {
	terminal      *terminal
	history       *config.History
	plugins       map[string]plugin.Plugin
	defaultPlugin plugin.Plugin
	historyPlugin plugin.Plugin
}

func newSuggester(terminal *terminal, completionUi *completion.Fzf, history *config.History, homeDir string) suggester {
	pluginList := []plugin.Plugin{
		kubectl.NewKubeCtlPlugin(completionUi),
		git.NewGitPlugin(completionUi),
	}
	plugins := make(map[string]plugin.Plugin, len(pluginList))
	for _, p := range pluginList {
		plugins[p.Command()] = p
	}

	return suggester{
		terminal:      terminal,
		history:       history,
		plugins:       plugins,
		defaultPlugin: plugin.NewFilePlugin(completionUi, homeDir),
		historyPlugin: plugin.NewHistoryPlugin(completionUi),
	}
}

func (s suggester) suggestHistory(args []string, inputCommand string) (string, error) {
	return s.suggest(s.historyPlugin, args, inputCommand)
}

func (s suggester) suggestCommand(inputCommand string) (string, error) {
	args := strings.Fields(inputCommand)
	if len(args) == 0 {
		return inputCommand, nil
	}

	suggestPlugin, ok := s.plugins[args[0]]
	if !ok {
		suggestPlugin = s.defaultPlugin
	}
	var err error
	inputCommand, err = s.suggest(suggestPlugin, args, inputCommand)
	if err != nil {
		return inputCommand, err
	}
	return inputCommand, nil
}

func (s suggester) suggest(p plugin.Plugin, args []string, inputCommand string) (string, error) {
	// move these logics to terminal
	var currentArgToken string
	var previousArgs string
	if len(inputCommand) > 1 {
		previousChar := inputCommand[len(inputCommand)+s.terminal.out.cursor-1]
		if previousChar != ' ' {
			lastSpaceIndex := strings.LastIndex(inputCommand, " ")
			if lastSpaceIndex != -1 {
				currentArgToken = inputCommand[lastSpaceIndex:]
				previousArgs = inputCommand[:lastSpaceIndex]
			} else {
				currentArgToken = inputCommand
			}
		}
	}

	arg := plugin.SuggestArg{
		Args:            args,
		History:         s.history,
		CurrentArgToken: strings.TrimSpace(currentArgToken),
	}
	var suggested []string
	var err error
	suggested, err = p.Suggest(arg)
	if err != nil {
		return inputCommand, err
	}
	if len(suggested) > 0 {
		if previousArgs != "" {
			inputCommand = previousArgs + " " + strings.Join(suggested, " ")
		} else {
			inputCommand = inputCommand + strings.Join(suggested, " ")
		}
	}
	return inputCommand, nil
}
