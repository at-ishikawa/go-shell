package plugin

import (
	"strings"

	"github.com/at-ishikawa/go-shell/internal/config"
)

type optionStats struct {
	noValue int
	values  map[string]int
}

type commandStats struct {
	count   int
	options map[string]optionStats
	args    map[string]commandStats
}

type HistoryCommandStats map[string]commandStats

func (h HistoryCommandStats) getSuggestedValues(args []string, currentToken string) []string {
	if len(args) == 0 ||
		len(args) == 1 && currentToken != "" {
		result := make([]string, 0, len(h))
		for command, _ := range h {
			result = append(result, command)
		}
		return result
	}
	if currentToken != "" {
		args = args[:len(args)-1]
	}

	suggests, ok := h[args[0]]
	if !ok {
		return []string{}
	}
	var suggestsOptions []optionStats
	var optionNames []string
	var isLastOptionRecognized bool
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			optionName := arg
			opt, ok := suggests.options[optionName]
			if ok {
				optionNames = append(optionNames, optionName)
				suggestsOptions = append(suggestsOptions, opt)
				isLastOptionRecognized = true
			} else {
				isLastOptionRecognized = false
			}
			continue
		}
		optionNames = []string{}
		suggestsOptions = []optionStats{}
		suggests, ok = suggests.args[arg]
		if !ok {
			return []string{}
		}
	}

	var result []string
	for _, suggestOpt := range suggestsOptions {
		for arg, _ := range suggestOpt.values {
			result = append(result, arg)
		}
	}
	var lastOption optionStats
	if isLastOptionRecognized {
		lastOption = suggestsOptions[len(suggestsOptions)-1]
	}
	if lastOption.noValue == 0 && len(lastOption.values) > 0 {
		return result
	}

L:
	for name, _ := range suggests.options {
		for _, optionName := range optionNames {
			if optionName == name {
				continue L
			}
		}
		result = append(result, name)
	}
	for arg, _ := range suggests.args {
		result = append(result, arg)
	}
	return result
}

func getSubCommandStats(result map[string]commandStats, args []string, currentArgIndex int) map[string]commandStats {
	if currentArgIndex >= len(args) {
		return result
	}

	command := args[currentArgIndex]
	commandStat, ok := result[command]
	if !ok {
		commandStat = commandStats{
			args:    make(map[string]commandStats),
			options: make(map[string]optionStats),
		}
	}
	commandStat.count++

	for i := currentArgIndex + 1; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") || strings.HasPrefix(args[i], "-") {
			optionName := args[i]
			optionStat, ok := commandStat.options[optionName]
			if !ok {
				optionStat = optionStats{
					values: make(map[string]int),
				}
			}
			if i+1 == len(args) {
				optionStat.noValue++
			} else {
				optionValue := args[i+1]
				if strings.HasPrefix(optionValue, "--") || strings.HasPrefix(optionValue, "-") {
					optionStat.noValue++
				} else {
					if _, ok := optionStat.values[optionValue]; !ok {
						optionStat.values[optionValue] = 0
					}
					optionStat.values[optionValue]++
					i++
				}
			}
			commandStat.options[optionName] = optionStat
		} else {
			commandStat.args = getSubCommandStats(commandStat.args, args, i)
			break
		}
	}

	result[command] = commandStat
	return result
}

func getHistoryCommandStats(historyList []config.HistoryItem) HistoryCommandStats {
	result := make(HistoryCommandStats)
	for _, item := range historyList {
		if item.Status != 0 {
			continue
		}

		args := strings.Fields(item.Command)
		result = getSubCommandStats(result, args, 0)
	}
	return result
}
