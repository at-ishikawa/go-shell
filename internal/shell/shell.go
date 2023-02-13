package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/at-ishikawa/go-shell/internal/keyboard"
)

// https://hackernoon.com/today-i-learned-making-a-simple-interactive-shell-application-in-golang-aa83adcb266a
func Run(inFile *os.File) error {
	in, err := initInput(inFile)
	if err != nil {
		return err
	}
	defer in.finalize()

	for {
		fmt.Print("$ ")

		if err := in.makeRaw(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		line := ""
		isInputComplete := false
		for !isInputComplete {
			char, key, err := in.Read()
			if err != nil {
				fmt.Println(err)
				return nil
			}

			switch key {
			case keyboard.ControlB:
				break
			case keyboard.Enter:
				isInputComplete = true
				fmt.Println()
				break
			}

			fmt.Print(char)
			line = line + char
		}

		if err := in.restore(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if err := runCommand(line); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func runCommand(commandStr string) error {
	commandStr = strings.TrimSuffix(commandStr, "\n")
	arrCommandStr := strings.Fields(commandStr)
	switch arrCommandStr[0] {
	case "exit":
		os.Exit(0)
		// add another case here for custom commands.
	}
	cmd := exec.Command(arrCommandStr[0], arrCommandStr[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
