package completion

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Fzf struct {
	command          string
	defaultFzfOption FzfOption
}

type FzfOptionLayout string

const (
	FzfOptionLayoutReverse FzfOptionLayout = "reverse"
)

type FzfOption struct {
	Info           string
	Bind           string
	HeaderLines    int
	Layout         FzfOptionLayout
	PreviewWindow  string
	PreviewCommand string
	IsAnsiColor    bool
	isMulti        bool
}

func (o FzfOption) String() string {
	result := []string{}

	if o.Info != "" {
		result = append(result, fmt.Sprintf("--info=%s", o.Info))
	}
	if o.Bind != "" {
		result = append(result, fmt.Sprintf("--bind=%s", o.Bind))
	}
	if o.HeaderLines != 0 {
		result = append(result, fmt.Sprintf("--header-lines=%d", o.HeaderLines))
	}
	if o.Layout != "" {
		result = append(result, fmt.Sprintf("--layout=%s", o.Layout))
	}
	if o.PreviewWindow != "" {
		result = append(result, fmt.Sprintf("--preview-window=%s", o.PreviewWindow))
	}
	if o.PreviewCommand != "" {
		result = append(result, fmt.Sprintf("--preview=\"%s\"", o.PreviewCommand))
	}
	if o.IsAnsiColor {
		result = append(result, fmt.Sprintf("--ansi"))
	}
	if o.isMulti {
		result = append(result, fmt.Sprintf("--multi"))
	}

	return strings.Join(result, " ")
}

func NewFzf() *Fzf {
	defaultFzfOption := FzfOption{
		Info: "inline",
		Bind: "ctrl-k:kill-line,ctrl-alt-t:toggle-preview,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down",
	}

	return &Fzf{
		command:          "fzf",
		defaultFzfOption: defaultFzfOption,
	}
}

func (f Fzf) Complete(lines []string, fzfOptions FzfOption) (string, error) {
	if fzfOptions.Bind != "" {
		fzfOptions.Bind = f.defaultFzfOption.Bind
	}

	str := strings.Join(lines, "\n")
	command := fmt.Sprintf("echo '%s' | %s %s", str, f.command, fzfOptions.String())
	execCmd := exec.Command("sh", "-c", command)
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin
	out, err := execCmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Script canceled by Ctrl-c
			// Only for bash?: http://tldp.org/LDP/abs/html/exitcodes.html
			if exitErr.ExitCode() == 130 {
				return "", nil
			}
		}
		return strings.TrimSpace(string(out)), fmt.Errorf("failed to run the command %s: %w", command, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (f Fzf) CompleteMulti(lines []string, fzfOptions FzfOption) ([]string, error) {
	fzfOptions.isMulti = true
	rawResult, err := f.Complete(lines, fzfOptions)
	if err != nil {
		return []string{}, err
	}
	if rawResult == "" {
		return []string{}, nil
	}
	return strings.Split(rawResult, "\n"), nil
}
