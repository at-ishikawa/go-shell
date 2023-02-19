package completion

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Fzf struct {
	command string
}

func NewFzf() *Fzf {
	return &Fzf{
		command: "fzf",
	}
}

func (f Fzf) CompleteBytes(bytes []byte, fzfOptions []string) ([]string, error) {
	command := fmt.Sprintf("echo '%s' | %s %s", string(bytes), f.command, strings.Join(fzfOptions, " "))
	execCmd := exec.Command("sh", "-c", command)
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin
	out, err := execCmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Script canceled by Ctrl-c
			// Only for bash?: http://tldp.org/LDP/abs/html/exitcodes.html
			if exitErr.ExitCode() == 130 {
				return []string{}, nil
			}
		}
		return []string{}, fmt.Errorf("failed to run the command %s: %w", command, err)
	}

	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}
