package plugin

//go:generate mockgen -destination=../mocks/mock_plugin/mock_plugin.go -source=./plugin.go Plugin

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
