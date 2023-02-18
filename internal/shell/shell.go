package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"

	"github.com/at-ishikawa/go-shell/internal/keyboard"
	"github.com/at-ishikawa/go-shell/internal/kubectl"
	"github.com/ktr0731/go-fuzzyfinder"
)

type Shell struct {
	historyIndex int
	histories    []string
	in           input
	out          output
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

		line, err := s.getCommand()
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
		if err := s.in.restore(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if err := s.runCommand(line); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if err := s.in.makeRaw(); err != nil {
			return err
		}
	}
}

func (s *Shell) handleShortcutKey(line string, char rune, key keyboard.Key) (string, error) {
	switch key {
	case keyboard.Backspace:
		if len(line) == 0 {
			break
		}

		if s.out.cursor < 0 {
			lineIndex := len(line) + s.out.cursor
			line = line[:lineIndex-1] + line[lineIndex:]
		} else {
			line = line[:len(line)-1]
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
	case keyboard.ControlK:
		lineIndex := len(line) + s.out.cursor
		if lineIndex < len(line) {
			line = line[:lineIndex]
			s.out.cursor = 0
		}
		break
	case keyboard.ControlA:
		s.out.setCursor(-len(line))
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
		if -s.out.cursor < len(line) {
			s.out.moveCursor(-1)
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
		if !unicode.IsLetter(char) {
			break
		}

		if s.out.cursor < 0 {
			lineIndex := len(line) + s.out.cursor
			line = line[:lineIndex] + string(char) + line[lineIndex:]
		} else {
			line = line + string(char)
		}
	}

	return line, nil
}

func (s Shell) getCommand() (string, error) {
	line := ""

	for {
		char, key, err := s.in.Read()
		if err != nil {
			s.out.writeLine("")
			return "", err
		}

		if key == keyboard.Enter {
			s.out.newLine()
			break
		}
		if key == keyboard.ControlC {
			line = ""
			s.out.newLine()
			break
		}

		line, err = s.handleShortcutKey(line, char, key)
		if err != nil {
			s.out.writeLine("")
			return "", err
		}

		if len(line) <= 0 {
			s.out.writeLine("")
			continue
		}
		s.out.writeLine(line)
	}
	return line, nil
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
