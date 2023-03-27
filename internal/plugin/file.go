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

	suggestedValues, err := f.readDirectory(arg.CurrentArgToken, suggestedValuesFromHistory)
	if err != nil {
		return nil, fmt.Errorf("f.readDirectory failed: %w", err)
	}

	file, err := f.completionUi.Complete(suggestedValues, completion.CompleteOptions{
		InitialQuery: query,
		LiveReloading: func(row int, query string) ([]string, error) {
			files, err := f.readDirectory(query, suggestedValuesFromHistory)
			if err != nil {
				return nil, fmt.Errorf("f.readDirectory failed: %w", err)
			}
			return files, nil
		},
	})
	if err != nil {
		return nil, err
	}
	return []string{file}, nil
}

func (f FilePlugin) readDirectory(directory string, suggestedValuesFromHistory []string) ([]string, error) {
	currentDirectory := filepath.Dir(directory)
	entries, err := os.ReadDir(strings.ReplaceAll(currentDirectory, "~", f.homeDir))
	if err != nil {
		return []string{}, fmt.Errorf("os.ReadDir failed: %w", err)
	}

	isFileSearchedBefore := func(filePath string) bool {
		for _, suggestedValueFromHistory := range suggestedValuesFromHistory {
			if strings.Contains(suggestedValueFromHistory, filePath) {
				return true
			}
		}
		return false
	}

	pathSeparator := string(os.PathSeparator)
	var filePaths []string
	for _, e := range entries {
		filePath := currentDirectory + pathSeparator + e.Name()
		if e.IsDir() {
			filePath = filePath + "/"
		}
		if isFileSearchedBefore(filePath) {
			filePaths = append([]string{filePath}, filePaths...)
			continue
		}
		filePaths = append(filePaths, filePath)
	}

	parentDirectory := filepath.Dir(currentDirectory + pathSeparator + ".." + pathSeparator)
	entry, _ := os.Stat(strings.ReplaceAll(parentDirectory, "~", f.homeDir))
	if entry != nil {
		filePaths = append(filePaths, parentDirectory+"/")
	}

	suggestValues := make([]string, 0, len(filePaths))
	for _, filePath := range filePaths {
		suggestValues = append(suggestValues, filePath)
	}
	return suggestValues, nil
}
