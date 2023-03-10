package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/at-ishikawa/go-shell/internal/completion"
)

type FilePlugin struct {
	completionUi *completion.Fzf
	homeDir      string
}

var _ Plugin = (*FilePlugin)(nil)

func NewFilePlugin(completionUi *completion.Fzf, homeDir string) Plugin {
	return &FilePlugin{
		completionUi: completionUi,
		homeDir:      homeDir,
	}
}

func (f FilePlugin) Command() string {
	return ""
}

func (f FilePlugin) Suggest(arg SuggestArg) ([]string, error) {
	pathSeparator := string(os.PathSeparator)
	query := arg.CurrentArgToken
	directories := strings.Split(arg.CurrentArgToken, pathSeparator)
	if len(directories) > 1 {
		// Directory except the last part
		query = directories[len(directories)-1]
	} else if arg.CurrentArgToken == ".." {
		query = ""
	}

	currentDirectory := filepath.Dir(arg.CurrentArgToken)
	entries, err := os.ReadDir(strings.ReplaceAll(currentDirectory, "~", f.homeDir))
	if err != nil {
		return []string{}, fmt.Errorf("os.ReadDir failed: %w", err)
	}
	var filePaths []string
	for _, e := range entries {
		filePath := currentDirectory + pathSeparator + e.Name()
		if e.IsDir() {
			filePath = filePath + "/"
		}
		filePaths = append(filePaths, filePath)
	}

	return f.completionUi.CompleteMulti(filePaths, completion.FzfOption{
		Query: query,
	})
}
