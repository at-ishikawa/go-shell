package shell

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/at-ishikawa/go-shell/internal/completion"
	"github.com/at-ishikawa/go-shell/internal/keyboard"
	"github.com/at-ishikawa/go-shell/internal/plugin"
	"github.com/at-ishikawa/go-shell/internal/plugin/kubectl"
)

type Shell struct {
	history            history
	in                 input
	out                output
	isEscapeKeyPressed bool
	completionUi       *completion.Fzf
	plugins            map[string]plugin.Plugin
}

func NewShell(inFile *os.File, outFile *os.File) (Shell, error) {
	in, err := initInput(inFile)
	if err != nil {
		return Shell{}, err
	}
	out := initOutput(outFile)

	conf, err := newConfig()
	if err != nil {
		return Shell{}, err
	}
	if err := conf.makeDir(); err != nil {
		return Shell{}, fmt.Errorf("failed to make a config directory: %w", err)
	}
	hist := newHistory(conf)
	if err := hist.loadFile(); err != nil {
		return Shell{}, fmt.Errorf("failed to load a history file: %w", err)
	}

	completionUi := completion.NewFzf()
	kubeCtlPlugin := kubectl.NewKubeCtlPlugin(completionUi)
	plugins := map[string]plugin.Plugin{
		kubeCtlPlugin.Command(): kubeCtlPlugin,
	}

	return Shell{
		history:      hist,
		in:           in,
		out:          out,
		completionUi: completionUi,
		plugins:      plugins,
	}, nil
}

// https://hackernoon.com/today-i-learned-making-a-simple-interactive-shell-application-in-golang-aa83adcb266a
func (s Shell) Run() error {
	defer s.in.finalize()
	if err := s.in.makeRaw(); err != nil {
		return err
	}

	for {
		kubeCtx, err := kubectl.GetContext()
		if err != nil {
			fmt.Println(err)
		} else {
			kubeNamespace, err := kubectl.GetNamespace(kubeCtx)
			if err != nil {
				fmt.Println(err)
			} else {
				s.out.setPrompt(fmt.Sprintf("[%s|%s] $ ", kubeCtx, kubeNamespace))
			}
		}

		inputCommand, err := s.getInputCommand()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		inputCommand = strings.TrimSpace(inputCommand)
		if inputCommand == "" {
			continue
		}
		if inputCommand == "exit" {
			break
		}

		// For some reason, term.Restore for an input is required before executing a command
		if err := s.in.restore(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		exitCode, err := s.runCommand(inputCommand)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if err := s.in.makeRaw(); err != nil {
			return err
		}
		s.history.add(inputCommand, exitCode)
	}
	if err := s.history.saveFile(); err != nil {
		return fmt.Errorf("failed to write a history to a file: %w", err)
	}

	return nil
}

func getPreviousWord(str string, cursor int) string {
	subStrBeforeCursor := str[:len(str)+cursor]
	previousChar := str[len(str)+cursor-1]

	var subStrLastIndex int
	if previousChar != ' ' {
		subStrLastIndex = strings.LastIndex(subStrBeforeCursor, " ") + 1
	} else {
		for subStrLastIndex = len(subStrBeforeCursor) - 2; subStrLastIndex >= 0; subStrLastIndex-- {
			if subStrBeforeCursor[subStrLastIndex] != ' ' {
				break
			}
		}
		subStrLastIndex++
	}
	return subStrBeforeCursor[subStrLastIndex:]
}

func getNextWord(str string, cursor int) string {
	subStrAfterCursor := str[len(str)+cursor:]
	nextChar := str[len(str)+cursor]

	var subStrFirstIndex int
	if nextChar != ' ' {
		subStrFirstIndex = strings.Index(subStrAfterCursor, " ")
		if subStrFirstIndex < 0 {
			subStrFirstIndex = len(subStrAfterCursor)
		}
	} else {
		var ch rune
		for subStrFirstIndex, ch = range subStrAfterCursor {
			if ch == ' ' {
				break
			}
		}
		subStrFirstIndex++
	}
	return subStrAfterCursor[:subStrFirstIndex]
}

func (s *Shell) handleShortcutKey(inputCommand string, char rune, key keyboard.Key) (string, error) {
	if s.isEscapeKeyPressed {
		switch key {
		case keyboard.B:
			if -s.out.cursor >= len(inputCommand) {
				break
			}

			previousWord := getPreviousWord(inputCommand, s.out.cursor)
			s.out.cursor = -len(previousWord) + s.out.cursor
			break
		case keyboard.F:
			if s.out.cursor == 0 {
				break
			}

			nextWord := getNextWord(inputCommand, s.out.cursor)
			s.out.cursor += len(nextWord)
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
		lines := make([]string, 0, len(s.history.list)+1)
		lines = append(lines, fmt.Sprintf("%-50s %20s", "command", "status"))
		for _, historyItem := range s.history.list {
			lines = append(lines, fmt.Sprintf("%-50s %20d",
				historyItem.Command,
				historyItem.Status))
		}
		// todo: show a preview like
		//     item := s.history.list[index]
		//     return fmt.Sprintf("status: %d\nRunning at: %s", item.Status, item.RunAt.Format(time.RFC3339))
		result, err := s.completionUi.Complete(lines, completion.FzfOption{
			HeaderLines: 1,
		})
		if err != nil {
			return "", err
		} else if result != "" {
			selectedCommand := strings.Fields(result)
			inputCommand = strings.Join(selectedCommand[:len(selectedCommand)-1], " ")
		}
	case keyboard.ControlP:
		previousCommand := s.history.previous()
		if previousCommand != "" {
			inputCommand = previousCommand
		}
		break
	case keyboard.ControlN:
		nextCommand, ok := s.history.next()
		if ok {
			inputCommand = nextCommand
		}
		break

	case keyboard.ControlW:
		if -s.out.cursor >= len(inputCommand) {
			break
		}

		previousWord := getPreviousWord(inputCommand, s.out.cursor)
		a := inputCommand[:len(inputCommand)+s.out.cursor-len(previousWord)]
		b := inputCommand[len(inputCommand)+s.out.cursor:]
		inputCommand = a + b

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
		var suggested []string
		suggestPlugin, ok := s.plugins[args[0]]
		if ok {
			var err error
			suggested, err = suggestPlugin.Suggest(args)
			if err != nil {
				fmt.Println(err)
				break
			}
		} else {
			previousChar := inputCommand[len(inputCommand)+s.out.cursor-1]
			if previousChar == ' ' {
				// TODO: fix not only the new argument
				// todo: fix max depth configuration
				// todo: fix a skip list

				maxDepth := 3
				// var files []fs.DirEntry
				skipList := map[string]struct{}{
					".git":   {},
					"vendor": {},
				}
				var filePaths []string
				if err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if _, ok := skipList[d.Name()]; ok {
						return fs.SkipDir
					}
					if d.IsDir() && strings.Count(path, string(os.PathSeparator)) > maxDepth {
						return fs.SkipDir
					}
					filePaths = append(filePaths, path)
					return nil
				}); err != nil {
					return "", err
				}

				selected, err := s.completionUi.Complete(filePaths, completion.FzfOption{})
				if err != nil {
					return "", err
				} else {
					inputCommand += selected
				}
			}
		}
		if len(suggested) > 0 {
			inputCommand = inputCommand + strings.Join(suggested, " ")
		}
		break
	default:
		if !utf8.ValidRune(char) {
			break
		}
		if keyboard.ControlA <= key && key <= keyboard.ControlZ {
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
	s.out.initNewLine()
	s.out.setCursor(0)

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

func (s Shell) runCommand(commandStr string) (int, error) {
	commandStr = strings.TrimSuffix(commandStr, "\n")
	arrCommandStr := strings.Fields(commandStr)

	cmd := exec.Command(arrCommandStr[0], arrCommandStr[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = s.out.file
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
