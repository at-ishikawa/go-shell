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
	switch args[1] {
	case "add":
		return g.suggestFiles()
	case "branch":
		return g.suggestLocalBranches()
	}
	return []string{}, nil
}

func (g GitPlugin) suggestFiles() ([]string, error) {
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

func (g GitPlugin) suggestLocalBranches() ([]string, error) {
	// output, err := exec.Command(g.command, "-c", "color.status=always", "status", "-s").Output()
	// Show only unstaged and untracked files
	localBranchRefs := "refs/heads/"
	sortByLatestCommitted := "--sort=-committerdate"
	output, err := exec.Command(g.command, "for-each-ref", sortByLatestCommitted, localBranchRefs, "--format=%(refname:short) %(committerdate:relative)").Output()
	if err != nil {
		return []string{}, fmt.Errorf("failed to run a git for-each-ref: %w", err)
	}
	formattedBranches := formatLines(output, 0, 50, 100)
	lines, err := g.completionUi.CompleteMulti(formattedBranches, completion.FzfOption{
		IsAnsiColor:    true,
		PreviewCommand: "git log {1}",
	})
	if err != nil {
		return []string{}, err
	}

	return getResultFromLine(lines, 0), nil
}

func getResultFromLine(lines []string, column int) []string {
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}

		items := strings.Fields(line)
		result = append(result, items[column])
	}
	return result
}

func formatLines(output []byte, formatColumn, minLength, maxLength int) []string {
	// todo: formatColumn only works if it's 0
	// left padding only
	selectList := strings.Split(string(output), "\n")
	allBranches := getResultFromLine(selectList, formatColumn)

	maxBranchNameLength := 0
	for _, branch := range allBranches {
		if maxBranchNameLength < len(branch) {
			maxBranchNameLength = len(branch)
		}
	}
	if maxBranchNameLength < minLength {
		maxBranchNameLength = minLength
	}
	if maxBranchNameLength > maxLength {
		maxBranchNameLength = maxLength
	}

	formattedBranches := make([]string, 0, len(selectList))
	for _, selectItem := range selectList {
		if selectItem == "" {
			continue
		}

		fields := strings.Fields(selectItem)
		format := fmt.Sprintf("%%-%ds %%s", maxBranchNameLength)
		formattedBranches = append(formattedBranches, fmt.Sprintf(format, fields[formatColumn], strings.Join(fields[formatColumn+1:], " ")))
	}
	return formattedBranches
}
