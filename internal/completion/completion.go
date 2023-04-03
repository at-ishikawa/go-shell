package completion

//go:generate mockgen -destination=./mock_completion.go -source=./completion.go -package completion Completion

type Completion interface {
	Complete(rows []string, options CompleteOptions) (string, error)
	CompleteMulti(rows []string, options CompleteOptions) ([]string, error)
}

type PreviewCommandType func(row int) (string, error)
type LiveReloading func(row int, query string) ([]string, error)

type CompleteOptions struct {
	PreviewCommand PreviewCommandType
	Header         string
	InitialQuery   string
	IsAnsiColor    bool
	LiveReloading  LiveReloading
}
