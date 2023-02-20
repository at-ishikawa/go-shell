package plugin

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/at-ishikawa/go-shell/internal/completion"
)

type FilePlugin struct {
	completionUi *completion.Fzf
	maxDepth     int
	skipList     map[string]struct{}
}

var _ Plugin = (*FilePlugin)(nil)

func NewFilePlugin(completionUi *completion.Fzf) Plugin {
	return &FilePlugin{
		completionUi: completionUi,
		maxDepth:     3,
		skipList: map[string]struct{}{
			".git":   {},
			"vendor": {},
		},
	}
}

func (f FilePlugin) Command() string {
	return ""
}

func (f FilePlugin) Suggest(arg SuggestArg) ([]string, error) {
	inputCommand := arg.Input
	cursor := arg.Cursor
	previousChar := inputCommand[len(inputCommand)+cursor-1]
	if previousChar == ' ' {
		// TODO: fix not only the new argument
		// todo: fix max depth configuration
		// todo: fix a skip list

		// var files []fs.DirEntry
		var filePaths []string
		if err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if _, ok := f.skipList[d.Name()]; ok {
				return fs.SkipDir
			}
			if d.IsDir() && strings.Count(path, string(os.PathSeparator)) > f.maxDepth {
				return fs.SkipDir
			}
			filePaths = append(filePaths, path)
			return nil
		}); err != nil {
			return []string{""}, err
		}

		return f.completionUi.CompleteMulti(filePaths, completion.FzfOption{})
	}
	return []string{}, nil
}
