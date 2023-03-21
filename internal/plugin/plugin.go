package plugin

//go:generate mockgen -destination=../mocks/mock_plugin/mock_plugin.go -source=./plugin.go Plugin

import (
	"fmt"
	"os"

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

func (arg SuggestArg) Suggest(completionUi completion.Completion) ([]string, error) {
	values, err := arg.GetSuggestedValues()
	if err != nil {
		return nil, fmt.Errorf("arg.GetSuggestedValues failed: %w", err)
	}

	result, err := completionUi.Complete(values, completion.CompleteOptions{
		InitialQuery: arg.CurrentArgToken,
	})
	return []string{result}, err
}

func (arg SuggestArg) GetSuggestedValues() ([]string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("os.Getwd failed: %w", err)
	}
	historyCommandStats := getHistoryCommandStats(arg.History.FilterByDirectory(currentDir))
	return historyCommandStats.getSuggestedValues(arg.Args, arg.CurrentArgToken), nil
}
