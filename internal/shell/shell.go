package shell

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"

	"github.com/at-ishikawa/go-shell/internal/keyboard"
	"github.com/at-ishikawa/go-shell/internal/kubectl"
)

type Shell struct {
	historyIndex int
	histories    []string
}

func NewShell() Shell {
	return Shell{
		histories: []string{},
	}
}

// https://hackernoon.com/today-i-learned-making-a-simple-interactive-shell-application-in-golang-aa83adcb266a
func (s Shell) Run(inFile *os.File, outFile *os.File) error {
	in, err := initInput(inFile)
	if err != nil {
		return err
	}
	defer in.finalize()
	if err := in.makeRaw(); err != nil {
		return err
	}

	out := initOutput(outFile)

	for {
		out.initNewLine()
		out.cursor = 0

		line, err := s.getCommand(in, out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}

		s.histories = append(s.histories, line)
		s.historyIndex = len(s.histories)
		// For some reason, term.Restore for an input is required before executing a command
		if err := in.restore(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if err := s.runCommand(line, outFile); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if err := in.makeRaw(); err != nil {
			return err
		}
	}
}

func (s Shell) getCommand(in input, out output) (string, error) {
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
		case keyboard.ControlR:
			idx, err := fuzzyfinder.Find(s.histories,
				func(i int) string {
					return s.histories[i]
				})
			if err != nil {
				// ignore
				// todo: fix this
			} else {
				line = s.histories[idx]
			}
		case keyboard.ControlP:
			if 0 < s.historyIndex {
				s.historyIndex--
				line = s.histories[s.historyIndex]
			}

			break
		case keyboard.ControlN:
			if len(s.histories)-1 > s.historyIndex {
				s.historyIndex++
				line = s.histories[s.historyIndex]
			} else if len(s.histories) > s.historyIndex {
				s.historyIndex++
				line = ""
			}
			break
		case keyboard.ControlA:
			out.moveCursor(-len(line))
			break
		case keyboard.ControlE:
			out.cursor = 0
			break
		case keyboard.ControlF:
			if out.cursor < 0 {
				out.moveCursor(1)
			}
			break
		case keyboard.ControlB:
			if -out.cursor < len(line) {
				out.moveCursor(-1)
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

func (s Shell) runCommand(commandStr string, outFile io.Writer) error {
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
