package shell

import (
	"os"
	"os/exec"
	"strings"
)

type commandRunner struct {
	out     output
	homeDir string
}

func newCommandRunner(out output, homeDir string) commandRunner {
	return commandRunner{
		out:     out,
		homeDir: homeDir,
	}
}

func (cr commandRunner) parseInput(inputCommand string) (string, []string) {
	inputFields := strings.Fields(inputCommand)
	if len(inputFields) == 0 {
		return "", nil
	}
	command := inputFields[0]

	var args []string
	if len(inputFields) > 1 {
		args = inputFields[1:]
		for i, arg := range args {
			// todo: file path only
			args[i] = strings.ReplaceAll(arg, "~", cr.homeDir)
		}
	}
	return command, args
}

func (cr commandRunner) run(inputCommand string) (int, error) {
	command, args := cr.parseInput(inputCommand)
	if command == "" {
		return 0, nil
	}

	switch command {
	case "cd":
		if err := cr.changeDir(args); err != nil {
			return 1, err
		}
		return 0, nil
	}

	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = cr.out.file
	if err := cmd.Run(); err != nil {
		// var exitError *exec.ExitError
		// if errors.As(err, &exitError) {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			return exitCode, err
		}
		return 1, err
	}
	return 0, nil
}

func (cr commandRunner) changeDir(args []string) error {
	dir := cr.homeDir
	if len(args) >= 1 {
		dir = args[0]
	}
	if err := os.Chdir(dir); err != nil {
		return err
	}
	return nil
}
