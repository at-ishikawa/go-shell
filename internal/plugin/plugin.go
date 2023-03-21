package plugin

//go:generate mockgen -destination=./mock_plugin.go -source=./plugin.go -package plugin Plugin

import (
	"fmt"

	"github.com/at-ishikawa/go-shell/internal/completion"
	"github.com/at-ishikawa/go-shell/internal/config"
)

type Plugin interface {
	// Command should return the command for completion of arguments, options and so on
	Command() string

	// GetContext is the metadata for the command which can be different on each runtime
	// The metadata can be stored in a history file and used for suggestions,
	// for example, for filtering some candidates
	// The example of this metadata is the origin of a git repository or the context of kubectl
	GetContext(command string) (map[string]string, error)

	// Suggest returns arguments or options depending on the current input
	Suggest(arg SuggestArg) ([]string, error)
}

type SuggestArg struct {
	Command         string
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
	historyCommandStats := getHistoryCommandStats(arg.History.Get())
	return historyCommandStats.getSuggestedValues(arg.Args, arg.CurrentArgToken), nil
}
