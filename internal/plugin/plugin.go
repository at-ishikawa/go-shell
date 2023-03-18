package plugin

//go:generate mockgen -destination=../mocks/mock_plugin/mock_plugin.go -source=./plugin.go Plugin

import (
	"github.com/at-ishikawa/go-shell/internal/completion"
	"github.com/at-ishikawa/go-shell/internal/config"
)

type Plugin interface {
	Command() string
	Suggest(arg SuggestArg) ([]string, error)
}

type SuggestArg struct {
	CurrentArgToken string
	Args            []string
	History         *config.History
}

func (arg SuggestArg) Suggest(completionUi *completion.Fzf) ([]string, error) {
	result, err := completionUi.Complete(arg.GetSuggestedValues(), completion.FzfOption{})
	return []string{result}, err
}

func (arg SuggestArg) GetSuggestedValues() []string {
	historyCommandStats := getHistoryCommandStats(arg.History.Get())
	return historyCommandStats.getSuggestedValues(arg.Args, arg.CurrentArgToken)
}
