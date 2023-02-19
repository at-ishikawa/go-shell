package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"github.com/at-ishikawa/go-shell/internal/keyboard"
	"github.com/at-ishikawa/go-shell/internal/kubectl"
	"github.com/ktr0731/go-fuzzyfinder"
)

type Shell struct {
	historyIndex       int
	histories          []string
	in                 input
	out                output
	isEscapeKeyPressed bool
}

func NewShell(inFile *os.File, outFile *os.File) (Shell, error) {
	in, err := initInput(inFile)
	if err != nil {
		return Shell{}, err
	}
	out := initOutput(outFile)

	return Shell{
		histories: []string{},
		in:        in,
		out:       out,
	}, nil
}

// https://hackernoon.com/today-i-learned-making-a-simple-interactive-shell-application-in-golang-aa83adcb266a
func (s Shell) Run() error {
	defer s.in.finalize()
	if err := s.in.makeRaw(); err != nil {
		return err
	}

	for {
		s.out.initNewLine()
		s.out.setCursor(0)

		inputCommand, err := s.getInputCommand()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		if strings.TrimSpace(inputCommand) == "" {
			continue
		}

		s.histories = append(s.histories, inputCommand)
		s.historyIndex = len(s.histories)
		// For some reason, term.Restore for an input is required before executing a command
		if err := s.in.restore(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if err := s.runCommand(inputCommand); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if err := s.in.makeRaw(); err != nil {
			return err
		}
	}
}

func (s *Shell) handleShortcutKey(inputCommand string, char rune, key keyboard.Key) (string, error) {
	if s.isEscapeKeyPressed {
		switch key {
		case keyboard.B:
			if -s.out.cursor >= len(inputCommand) {
				break
			}

			subStrBeforeCursor := inputCommand[:len(inputCommand)+s.out.cursor]
			previousChar := inputCommand[len(inputCommand)+s.out.cursor-1]

			var subStrLastIndex int
			if previousChar != ' ' {
				subStrLastIndex = strings.LastIndex(subStrBeforeCursor, " ") + 1
				s.out.cursor = -(len(subStrBeforeCursor) - subStrLastIndex) + s.out.cursor
			} else {
				for subStrLastIndex = len(subStrBeforeCursor) - 2; subStrLastIndex >= 0; subStrLastIndex-- {
					if subStrBeforeCursor[subStrLastIndex] != ' ' {
						break
					}
				}
				s.out.cursor = -(len(subStrBeforeCursor) - subStrLastIndex) + s.out.cursor + 1
			}

			break
		case keyboard.F:
			if s.out.cursor == 0 {
				break
			}

			subStrAfterCursor := inputCommand[len(inputCommand)+s.out.cursor:]
			nextChar := inputCommand[len(inputCommand)+s.out.cursor]

			var subStrFirstIndex int
			if nextChar != ' ' {
				subStrFirstIndex = strings.Index(subStrAfterCursor, " ")
				if subStrFirstIndex > 0 {
					s.out.cursor += subStrFirstIndex
				} else {
					s.out.cursor = 0
				}
			} else {
				var ch rune
				for subStrFirstIndex, ch = range subStrAfterCursor {
					if ch == ' ' {
						break
					}
				}
				s.out.cursor += subStrFirstIndex + 1
			}

			break
		}
		s.isEscapeKeyPressed = false
		return inputCommand, nil
	}

	switch key {
	case keyboard.Backspace:
		if len(inputCommand) == 0 {
			break
		}

		if s.out.cursor < 0 {
			inputCommandIndex := len(inputCommand) + s.out.cursor
			inputCommand = inputCommand[:inputCommandIndex-1] + inputCommand[inputCommandIndex:]
		} else {
			inputCommand = inputCommand[:len(inputCommand)-1]
		}
		break
	case keyboard.ControlR:
		idx, err := fuzzyfinder.Find(s.histories,
			func(i int) string {
				return s.histories[i]
			})
		if err != nil {
			return "", err
		} else {
			inputCommand = s.histories[idx]
		}
	case keyboard.ControlP:
		if 0 < s.historyIndex {
			s.historyIndex--
			inputCommand = s.histories[s.historyIndex]
		}

		break
	case keyboard.ControlN:
		if len(s.histories)-1 > s.historyIndex {
			s.historyIndex++
			inputCommand = s.histories[s.historyIndex]
		} else if len(s.histories) > s.historyIndex {
			s.historyIndex++
			inputCommand = ""
		}
		break
	case keyboard.ControlK:
		inputCommandIndex := len(inputCommand) + s.out.cursor
		if inputCommandIndex < len(inputCommand) {
			inputCommand = inputCommand[:inputCommandIndex]
			s.out.cursor = 0
		}
		break
	case keyboard.ControlA:
		s.out.setCursor(-len(inputCommand))
		break
	case keyboard.ControlE:
		s.out.setCursor(0)
		break
	case keyboard.ControlF:
		if s.out.cursor < 0 {
			s.out.moveCursor(1)
		}
		break
	case keyboard.ControlB:
		if -s.out.cursor < len(inputCommand) {
			s.out.moveCursor(-1)
		}
		break
	case keyboard.Escape:
		s.isEscapeKeyPressed = true
		break
	case keyboard.Tab:
		args := strings.Split(inputCommand, " ")
		if args[0] == "kubectl" {
			suggested, err := kubectl.Suggest(args)
			if err != nil {
				fmt.Println(err)
				break
			}
			inputCommand = inputCommand + strings.Join(suggested, " ")
		}
		break
	default:
		if !utf8.ValidRune(char) {
			break
		}

		if s.out.cursor < 0 {
			inputCommandIndex := len(inputCommand) + s.out.cursor
			inputCommand = inputCommand[:inputCommandIndex] + string(char) + inputCommand[inputCommandIndex:]
		} else {
			inputCommand = inputCommand + string(char)
		}
	}

	return inputCommand, nil
}

func (s Shell) getInputCommand() (string, error) {
	inputCommand := ""

	for {
		char, key, err := s.in.Read()
		if err != nil {
			s.out.newLine()
			return "", err
		}

		if key == keyboard.Enter {
			s.out.newLine()
			break
		}
		if key == keyboard.ControlC {
			inputCommand = ""
			s.out.newLine()
			break
		}

		inputCommand, err = s.handleShortcutKey(inputCommand, char, key)
		if err != nil {
			s.out.writeLine("")
			return "", err
		}

		if len(inputCommand) <= 0 {
			s.out.writeLine("")
			continue
		}
		s.out.writeLine(inputCommand)
	}
	return inputCommand, nil
}

func (s Shell) runCommand(commandStr string) error {
	commandStr = strings.TrimSuffix(commandStr, "\n")
	arrCommandStr := strings.Fields(commandStr)
	switch arrCommandStr[0] {
	case "exit":
		os.Exit(0)
		// add another case here for custom commands.
	}
	cmd := exec.Command(arrCommandStr[0], arrCommandStr[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = s.out.file
	return cmd.Run()
}
