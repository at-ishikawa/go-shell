package plugin

type Plugin interface {
	Command() string
	Suggest(args []string) ([]string, error)
}
