package shell

import (
	"strings"

	"github.com/at-ishikawa/go-shell/internal/completion"
	"github.com/at-ishikawa/go-shell/internal/config"
	"github.com/at-ishikawa/go-shell/internal/plugin"
	"github.com/at-ishikawa/go-shell/internal/plugin/git"
	"github.com/at-ishikawa/go-shell/internal/plugin/kubectl"
)

type commandSuggester struct {
	history       *config.History
	plugins       map[string]plugin.Plugin
	defaultPlugin plugin.Plugin
	historyPlugin plugin.Plugin
}

func newCommandSuggester(history *config.History, homeDir string) (commandSuggester, error) {
	completionUi := completion.NewFzf()
	tcellCompletionUi, err := completion.NewTcellCompletion()
	if err != nil {
		return commandSuggester{}, err
	}
	pluginList := []plugin.Plugin{
		kubectl.NewKubeCtlPlugin(completionUi),
		git.NewGitPlugin(tcellCompletionUi),
	}
	plugins := make(map[string]plugin.Plugin, len(pluginList))
	for _, p := range pluginList {
		plugins[p.Command()] = p
	}

	return commandSuggester{
		history:       history,
		plugins:       plugins,
		defaultPlugin: plugin.NewFilePlugin(completionUi, homeDir),
		historyPlugin: plugin.NewHistoryPlugin(completionUi),
	}, nil
}

func (s commandSuggester) suggestHistory(args plugin.SuggestArg) ([]string, error) {
	return s.historyPlugin.Suggest(args)
}

func (s commandSuggester) suggestCommand(inputCommand string, pluginArgs plugin.SuggestArg) ([]string, error) {
	args := strings.Fields(inputCommand)
	if len(args) == 0 {
		return []string{inputCommand}, nil
	}

	suggestPlugin, ok := s.plugins[args[0]]
	if !ok {
		suggestPlugin = s.defaultPlugin
	}
	return suggestPlugin.Suggest(pluginArgs)
}
