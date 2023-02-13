package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/at-ishikawa/go-shell/internal/keyboard"
)

// https://hackernoon.com/today-i-learned-making-a-simple-interactive-shell-application-in-golang-aa83adcb266a
func Run(inFile *os.File, outFile *os.File) error {
	in, err := initInput(inFile)
	if err != nil {
		return err
	}
	defer in.finalize()

	for {
		outFile.WriteString("$ ")

		if err := in.makeRaw(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		line, err := getCommand(in, outFile)
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

func getCommand(in input, outFile *os.File) (string, error) {
	line := ""
	for {
		char, key, err := in.Read()
		if err != nil {
			return "", err
		}

		if key == keyboard.Enter {
			outFile.WriteString("\n")
			outFile.Write([]byte{'\r'})
			break
		}

		switch key {
		case keyboard.Backspace:
			line = line[:len(line)-1]
			break
		case keyboard.ControlB:
			// TODO: Move back to a cursor. For some reasons, this doesn't work
			count := strconv.Itoa(1)
			outFile.Write([]byte{keyboard.Escape, '['})
			outFile.Write([]byte(count))
			outFile.Write([]byte{'D'})
			break
		default:
			line = line + char
		}
		// https://gist.github.com/fnky/458719343aabd01cfb17a3a4f7296797
		// fmt.Printf("\\033[2K")
		// fmt.Printf("\\r")
		// fmt.Print("$ " + line)
		outFile.Write([]byte{keyboard.Escape, '[', '2', 'K'})
		outFile.Write([]byte{'\r'})
		outFile.WriteString("$ ")
		outFile.WriteString(line)
	}
	return line, nil
}

func runCommand(commandStr string, outFile *os.File) error {
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
