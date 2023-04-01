package shell

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

type commandRunner struct {
	homeDir            string
	execCommandContext func(context.Context, string, ...string) *exec.Cmd
}

func newCommandRunner(homeDir string) commandRunner {
	return commandRunner{
		homeDir:            homeDir,
		execCommandContext: exec.CommandContext,
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

func (cr commandRunner) run(inputCommand string, term *terminal) (int, error) {
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := cr.execCommandContext(ctx, command, args...)
	cmd.Stdin = term.in.file
	cmd.Stdout = term.out.file
	cmd.Stderr = term.stdErr.file
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	errCh := make(chan error)
	defer close(errCh)
	if err := cmd.Start(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			return exitCode, err
		}
		return 1, err
	}

	stopSignals := make(chan os.Signal, 1)
	defer signal.Stop(stopSignals)
	// SIGINT: Control-C
	signal.Notify(stopSignals, syscall.SIGINT)
	go func() {
		select {
		case <-ctx.Done():
		case sig := <-stopSignals:
			if err := cmd.Process.Signal(sig); err != nil {
				// todo: replace os.Stderr with tty
				fmt.Fprintf(term.stdErr.file, "failed cmd.Process.Signal: %v\n", err)
			}
		}
	}()

	var paused bool
	pauseSignals := make(chan os.Signal, 1)
	defer signal.Stop(pauseSignals)
	// SIGTSTP: Control-Z
	signal.Notify(pauseSignals, syscall.SIGTSTP)
	go func() {
		select {
		case <-ctx.Done():
		case <-pauseSignals:
			// don't make a panic in a goroutine. It's harder to check a result on a unit test
			paused = true
			if err := cmd.Process.Kill(); err != nil {
				fmt.Fprintf(term.stdErr.file, "failed cmd.Process.Kill: %v\n", err)
			} else {
				fmt.Fprintf(term.out.file, "killed process: %s\n", command)
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		if paused {
			panic("Pausing a process has not been implemented yet")
		}

		// var exitError *exec.ExitError
		// if errors.As(err, &exitError) {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			// Don't show a message when a command was canceled by an interruption
			if strings.HasPrefix(exitError.String(), "signal:") {
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
