package shell

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/at-ishikawa/go-shell/internal/kubectl"
	"github.com/at-ishikawa/go-shell/internal/keyboard"
)

// https://hackernoon.com/today-i-learned-making-a-simple-interactive-shell-application-in-golang-aa83adcb266a
func Run(inFile *os.File, outFile *os.File) error {
	in, err := initInput(inFile)
	if err != nil {
		return err
	}
	defer in.finalize()
	out := initOutput(outFile)

	for {
		out.initNewLine()

		if err := in.makeRaw(); err != nil {
			return err
		}
		line, err := getCommand(in, out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		// For some reason, term.Restore for an input is required before executing a command
		if err := in.restore(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		if err := runCommand(line, outFile); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func getCommand(in input, out output) (string, error) {
	line := ""

	for {
		char, key, err := in.Read()
		if err != nil {
			return "", err
		}

		if key == keyboard.Enter {
			out.newLine()
			break
		}

		switch key {
		case keyboard.Backspace:
			if len(line) > 0 {
				line = line[:len(line)-1]
			}
			break
		case keyboard.ControlB:
			if -out.cursor < len(line) {
				out.moveLeft(-1)
			}
			break
		case keyboard.Tab:
			args := strings.Split(line, " ")
			if args[0] == "kubectl" {
				suggested, err := kubectl.Suggest(args)
				if err != nil {
					fmt.Println(err)
					break
				}
				line = line + strings.Join(suggested, " ")
			}
			break
		default:
			line = line + char
		}

		if len(line) <= 0 {
			out.writeLine("")
			continue
		}
		out.writeLine(line)
	}
	return line, nil
}

func runCommand(commandStr string, outFile io.Writer) error {
	commandStr = strings.TrimSuffix(commandStr, "\n")
	arrCommandStr := strings.Fields(commandStr)
	switch arrCommandStr[0] {
	case "exit":
		os.Exit(0)
		// add another case here for custom commands.
	}
	cmd := exec.Command(arrCommandStr[0], arrCommandStr[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = outFile
	return cmd.Run()
}
