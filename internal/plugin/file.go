package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/at-ishikawa/go-shell/internal/completion"
)

type FilePlugin struct {
	completionUi completion.Completion
	homeDir      string
}

var _ Plugin = (*FilePlugin)(nil)

func NewFilePlugin(completionUi completion.Completion, homeDir string) Plugin {
	return &FilePlugin{
		completionUi: completionUi,
		homeDir:      homeDir,
	}
}

func (f FilePlugin) Command() string {
	return ""
}

func (f FilePlugin) GetContext(_ string) (map[string]string, error) {
	return nil, nil
}

func (f FilePlugin) Suggest(arg SuggestArg) ([]string, error) {
	suggestedValuesFromHistory, err := arg.GetSuggestedValues()
	if err != nil {
		return []string{}, fmt.Errorf("arg.GetSuggestedValues failed: %w", err)
	}

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

	suggestValues := make([]string, 0, len(suggestedValuesFromHistory)+len(filePaths))
	for _, suggestValue := range suggestedValuesFromHistory {
		suggestValues = append(suggestValues, suggestValue)
	}
	for _, filePath := range filePaths {
		suggestValues = append(suggestValues, filePath)
	}

	return f.completionUi.CompleteMulti(suggestValues, completion.CompleteOptions{
		InitialQuery: query,
	})
}
