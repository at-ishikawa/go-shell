package shell

import (
	"os"
	"os/exec"
	"strings"
)

type commandRunner struct {
	homeDir string
}

func newCommandRunner(homeDir string) commandRunner {
	return commandRunner{
		homeDir: homeDir,
	}
}

// todo: may replace with a oss tokenizer and parser like goyacc
func (cr commandRunner) parseInput(inputCommand string) []string {
	// inputFields := strings.Fields(inputCommand)
	var inputFields []string
	var isInStr bool
	lastIndex := 0
	lastQuoteIndex := 0
	for i, char := range inputCommand {
		if char == ' ' && !isInStr {
			if i > lastIndex+1 {
				inputFields = append(inputFields, inputCommand[lastIndex:i])
			}
			lastIndex = i + 1
		} else if char == '"' {
			if i > 0 {
				lastChar := inputCommand[i-1]
				if lastChar == '\\' {
					continue
				}
			}
			if isInStr {
				if lastQuoteIndex == 0 || inputCommand[lastQuoteIndex-1] == ' ' {
					inputFields = append(inputFields, inputCommand[lastIndex+1:i])
				} else {
					inputFields = append(inputFields, inputCommand[lastIndex:i])
				}
				lastIndex = i + 1
				isInStr = false
			} else {
				lastQuoteIndex = i
				isInStr = true
			}
		}
	}
	if lastIndex < len(inputCommand) {
		inputFields = append(inputFields, inputCommand[lastIndex:])
	}
	return inputFields
}

func (cr commandRunner) compileInput(inputCommand string) (string, []string) {
	inputFields := cr.parseInput(inputCommand)
	command := inputFields[0]
	var args []string
	if len(inputFields) > 1 {
		args = inputFields[1:]
		for i := 0; i < len(args); i++ {
			arg := args[i]
			// todo: file path only
			args[i] = strings.ReplaceAll(arg, "~", cr.homeDir)
		}
	}
	return command, args
}

func (cr commandRunner) run(inputCommand string, commandFactory func(name string, args ...string) *exec.Cmd) (int, error) {
	command, args := cr.compileInput(inputCommand)
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

	cmd := commandFactory(command, args...)
	if err := cmd.Run(); err != nil {
		// var exitError *exec.ExitError
		// if errors.As(err, &exitError) {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			// Don't show a message when a command was canceled by an interruption
			if exitError.String() == "signal: interrupt" {
				return exitCode, nil
			}
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
