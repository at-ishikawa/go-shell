package plugin

import "github.com/at-ishikawa/go-shell/internal/config"

type Plugin interface {
	Command() string
	Suggest(arg SuggestArg) ([]string, error)
}

type SuggestArg struct {
	CurrentArgToken string
	Args            []string
	History         *config.History
}
