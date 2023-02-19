package git

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/at-ishikawa/go-shell/internal/completion"
	"github.com/at-ishikawa/go-shell/internal/plugin"
)

type GitPlugin struct {
	command      string
	completionUi *completion.Fzf
}

var _ plugin.Plugin = (*GitPlugin)(nil)

func NewGitPlugin(completionUi *completion.Fzf) plugin.Plugin {
	return &GitPlugin{
		command:      "git",
		completionUi: completionUi,
	}
}

func (g GitPlugin) Command() string {
	return g.command
}

func (g GitPlugin) Suggest(arg plugin.SuggestArg) ([]string, error) {
	args := arg.Args
	if args[1] == "add" {
		// output, err := exec.Command(g.command, "-c", "color.status=always", "status", "-s").Output()
		// Show only unstaged and untracked files
		output, err := exec.Command(g.command, "ls-files", "--modified", "--others", "--exclude-standard").Output()
		if err != nil {
			return []string{}, fmt.Errorf("failed to run a git status: %w", err)
		}

		lines, err := g.completionUi.CompleteMulti(strings.Split(string(output), "\n"), completion.FzfOption{
			IsAnsiColor:    true,
			PreviewCommand: "git diff --color HEAD {1}",
		})
		if err != nil {
			return []string{}, err
		}
		return lines, nil
	}
	return []string{}, nil
}
