package plugin

import "github.com/at-ishikawa/go-shell/internal/config"

type Plugin interface {
	Command() string
	Suggest(arg SuggestArg) ([]string, error)
}

type SuggestArg struct {
	Input   string
	Cursor  int
	Args    []string
	History *config.History
}
