package plugin

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/at-ishikawa/go-shell/internal/config"
	"go.uber.org/zap"

	"github.com/at-ishikawa/go-shell/internal/completion"
)

type HistoryPlugin struct {
	completionUi completion.Completion
	plugins      map[string]Plugin
	logger       *zap.Logger
}

var _ Plugin = (*HistoryPlugin)(nil)

func NewHistoryPlugin(plugins map[string]Plugin, completionUi completion.Completion, logger *zap.Logger) *HistoryPlugin {
	return &HistoryPlugin{
		plugins:      plugins,
		completionUi: completionUi,
		logger:       logger,
	}
}

func (h HistoryPlugin) Command() string {
	return ""
}

func (h HistoryPlugin) GetContext(_ string) (map[string]string, error) {
	return nil, nil
}

func (h HistoryPlugin) filterHistoryList(historyList []config.HistoryItem, query string) []config.HistoryItem {
	allContexts := map[string]map[string]string{}
	for _, p := range h.plugins {
		context, err := p.GetContext(query)
		if err != nil {
			h.logger.Error("failed p.GetContext: %w", zap.Error(err))
			continue
		}
		if len(context) == 0 {
			continue
		}
		allContexts[p.Command()] = context
	}

	result := make([]config.HistoryItem, 0, len(historyList))
	for _, item := range historyList {
		var zeroTime time.Time
		if item.LastSucceededAt == zeroTime {
			continue
		}

		contexts, ok := allContexts[strings.Fields(item.Command)[0]]
		if !ok {
			result = append(result, item)
			continue
		}
		if len(contexts) == 0 {
			result = append(result, item)
			continue
		}
		if reflect.DeepEqual(contexts, item.Context) {
			result = append(result, item)
			continue
		}
	}
	if len(result) == 0 {
		return nil
	}
	sort.Slice(result, func(i, j int) bool {
		return i > j
	})
	return result
}

func (h HistoryPlugin) Suggest(arg SuggestArg) ([]string, error) {
	var query string
	if len(arg.Args) > 0 {
		// invoke by a short cut key
		query = strings.Join(arg.Args, " ")
	}

	historyList := h.filterHistoryList(arg.History.Get(), query)
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
