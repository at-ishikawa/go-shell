package plugin

import (
	"fmt"
	"os"
	"strings"

	"github.com/at-ishikawa/go-shell/internal/completion"
)

type HistoryPlugin struct {
	completionUi completion.Completion
}

var _ Plugin = (*HistoryPlugin)(nil)

func NewHistoryPlugin(completionUi completion.Completion) *HistoryPlugin {
	return &HistoryPlugin{
		completionUi: completionUi,
	}
}

func (h HistoryPlugin) Command() string {
	return ""
}

func (h HistoryPlugin) Suggest(arg SuggestArg) ([]string, error) {
	var query string
	if len(arg.Args) > 0 {
		// invoke by a short cut key
		query = strings.Join(arg.Args, " ")
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return []string{}, err
	}
	historyList := arg.History.FilterByDirectory(currentDir)

	lines := make([]string, 0, len(historyList)+1)
	lines = append(lines, fmt.Sprintf("%-50s %20s", "command", "status"))
	for _, historyItem := range historyList {
		lines = append(lines, fmt.Sprintf("%-50s %20d",
			historyItem.Command,
			historyItem.Status))
	}
	// todo: show a preview like
	//     item := s.history.list[index]
	//     return fmt.Sprintf("status: %d\nRunning at: %s", item.Status, item.RunAt.Format(time.RFC3339))
	result, err := h.completionUi.Complete(lines[1:], completion.CompleteOptions{
		Header:       lines[0],
		InitialQuery: query,
	})
	if err != nil {
		return []string{""}, err
	} else if result != "" {
		selectedCommand := strings.Fields(result)
		return []string{
			strings.Join(selectedCommand[:len(selectedCommand)-1], " "),
		}, nil
	}
	return []string{}, nil
}
