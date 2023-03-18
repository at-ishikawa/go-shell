package shell

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"

	"github.com/at-ishikawa/go-shell/internal/config"

	"github.com/at-ishikawa/go-shell/internal/plugin/git"

	"github.com/at-ishikawa/go-shell/internal/completion"
	"github.com/at-ishikawa/go-shell/internal/keyboard"
	"github.com/at-ishikawa/go-shell/internal/plugin"
	"github.com/at-ishikawa/go-shell/internal/plugin/kubectl"
)

type Shell struct {
	history          config.History
	in               input
	out              output
	completionUi     *completion.Fzf
	plugins          map[string]plugin.Plugin
	defaultPlugin    plugin.Plugin
	historyPlugin    plugin.Plugin
	commandRunner    commandRunner
	candidateCommand string
}

func NewShell(inFile *os.File, outFile *os.File) (Shell, error) {
	in, err := initInput(inFile)
	if err != nil {
		return Shell{}, err
	}
	out := initOutput(outFile)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Shell{}, err
	}
	conf, err := config.NewConfig(homeDir)
	if err != nil {
		return Shell{}, err
	}
	hist := config.NewHistory(conf)
	if err := hist.LoadFile(); err != nil {
		return Shell{}, fmt.Errorf("failed to load a history file: %w", err)
	}

	completionUi := completion.NewFzf()
	pluginList := []plugin.Plugin{
		kubectl.NewKubeCtlPlugin(completionUi),
		git.NewGitPlugin(completionUi),
	}
	plugins := make(map[string]plugin.Plugin, len(pluginList))
	for _, p := range pluginList {
		plugins[p.Command()] = p
	}

	return Shell{
		history:       hist,
		in:            in,
		out:           out,
		completionUi:  completionUi,
		plugins:       plugins,
		defaultPlugin: plugin.NewFilePlugin(completionUi, homeDir),
		historyPlugin: plugin.NewHistoryPlugin(completionUi),
		commandRunner: newCommandRunner(out, homeDir),
	}, nil
}

// https://hackernoon.com/today-i-learned-making-a-simple-interactive-shell-application-in-golang-aa83adcb266a
func (s Shell) Run() error {
	defer s.in.finalize()
	if err := s.in.makeRaw(); err != nil {
		return err
	}

	var historyChannel chan struct{}
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
		s.candidateCommand = ""
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
		exitCode, err := s.commandRunner.run(inputCommand)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if err := s.in.makeRaw(); err != nil {
			return err
		}

		// wait for the previous stored history process will be done
		if historyChannel != nil {
			<-historyChannel
		}
		// In order to avoid storing commands with syntax error, do not store commands failed
		historyChannel = s.history.Sync(inputCommand, exitCode)
	}
	if historyChannel != nil {
		<-historyChannel
	}

	return nil
}

func getPreviousWord(str string, cursor int) string {
	subStrBeforeCursor := str[:len(str)+cursor]
	previousChar := rune(str[len(str)+cursor-1])

	var subStrLastIndex int
	if !(unicode.IsLetter(previousChar) || unicode.IsDigit(previousChar)) {
		for subStrLastIndex = len(subStrBeforeCursor) - 2; subStrLastIndex >= 0; subStrLastIndex-- {
			ch := rune(subStrBeforeCursor[subStrLastIndex])
			if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
				break
			}
		}
		for ; subStrLastIndex >= 0; subStrLastIndex-- {
			ch := rune(subStrBeforeCursor[subStrLastIndex])
			if !(unicode.IsLetter(ch) || unicode.IsDigit(ch)) {
				break
			}
		}
		subStrLastIndex++
	} else {
		subStrLastIndex = strings.LastIndexFunc(subStrBeforeCursor, func(r rune) bool {
			return !(unicode.IsLetter(r) || unicode.IsDigit(r))
		}) + 1
	}
	return subStrBeforeCursor[subStrLastIndex:]
}

func getNextWord(str string, cursor int) string {
	subStrAfterCursor := str[len(str)+cursor:]
	nextChar := rune(str[len(str)+cursor])

	var subStrFirstIndex int
	if unicode.IsLetter(nextChar) || unicode.IsDigit(nextChar) {
		subStrFirstIndex = strings.IndexFunc(subStrAfterCursor, func(r rune) bool {
			return !(unicode.IsLetter(r) || unicode.IsDigit(r))
		})
		if subStrFirstIndex < 0 {
			subStrFirstIndex = len(subStrAfterCursor)
		}
	} else {
		subStrFirstIndex = 0
		for ; subStrFirstIndex < len(subStrAfterCursor); subStrFirstIndex++ {
			ch := rune(subStrAfterCursor[subStrFirstIndex])
			if !(unicode.IsLetter(ch) || unicode.IsDigit(ch)) {
				break
			}
		}
		for ; subStrFirstIndex < len(subStrAfterCursor); subStrFirstIndex++ {
			ch := rune(subStrAfterCursor[subStrFirstIndex])
			if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
				break
			}
		}
		for ; subStrFirstIndex < len(subStrAfterCursor); subStrFirstIndex++ {
			ch := rune(subStrAfterCursor[subStrFirstIndex])
			if !(unicode.IsLetter(ch) || unicode.IsDigit(ch)) {
				break
			}
		}
	}
	return subStrAfterCursor[:subStrFirstIndex]
}

func (s *Shell) updateInputCommand(str string) string {
	s.candidateCommand = ""
	return str
}

func (s *Shell) moveCursorForward() {
	if s.out.cursor < 0 {
		s.out.moveCursor(1)
	}
}

func (s *Shell) moveCursorBackward(inputCommand string) {
	if -s.out.cursor < len(inputCommand) {
		s.out.moveCursor(-1)
	}
}

func (s *Shell) showPreviousCommandFromHistory(inputCommand string) string {
	previousCommand := s.history.Previous()
	if previousCommand != "" {
		inputCommand = s.updateInputCommand(previousCommand)
	}
	return inputCommand
}

func (s *Shell) showNextCommandFromHistory(inputCommand string) string {
	nextCommand, ok := s.history.Next()
	if ok {
		inputCommand = s.updateInputCommand(nextCommand)
	}
	return inputCommand
}

func (s *Shell) handleShortcutKey(inputCommand string, keyEvent keyboard.KeyEvent) (string, error) {
	if keyEvent.IsEscapePressed {
		switch keyEvent.Key {
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
		case keyboard.D:
			if len(inputCommand) == 0 {
				break
			}
			if s.out.cursor == 0 {
				break
			}
			nextWord := getNextWord(inputCommand, s.out.cursor)
			inputCommandIndex := len(inputCommand) + s.out.cursor
			inputCommand = inputCommand[:inputCommandIndex] + inputCommand[inputCommandIndex+len(nextWord):]
			inputCommand = s.updateInputCommand(inputCommand)
			s.out.cursor += len(nextWord)
			break
		}
		return inputCommand, nil
	}
	if keyEvent.IsControlPressed {
		switch keyEvent.Key {
		case keyboard.D:
			if len(inputCommand) == 0 {
				break
			}
			if s.out.cursor == 0 {
				break
			}

			inputCommandIndex := len(inputCommand) + s.out.cursor
			inputCommand = inputCommand[:inputCommandIndex] + inputCommand[inputCommandIndex+1:]
			inputCommand = s.updateInputCommand(inputCommand)
			s.out.cursor++
			break
		case keyboard.R:
			var err error
			inputCommand, err = s.suggest(s.historyPlugin, strings.Fields(inputCommand), inputCommand)
			if err != nil {
				fmt.Println(err)
				break
			}
			inputCommand = s.updateInputCommand(inputCommand)
			break
		case keyboard.P:
			inputCommand = s.showPreviousCommandFromHistory(inputCommand)
			break
		case keyboard.N:
			inputCommand = s.showNextCommandFromHistory(inputCommand)
			break
		case keyboard.W:
			if -s.out.cursor >= len(inputCommand) {
				break
			}

			previousWord := getPreviousWord(inputCommand, s.out.cursor)
			a := inputCommand[:len(inputCommand)+s.out.cursor-len(previousWord)]
			b := inputCommand[len(inputCommand)+s.out.cursor:]
			inputCommand = a + b
			inputCommand = s.updateInputCommand(inputCommand)

			break
		case keyboard.K:
			inputCommandIndex := len(inputCommand) + s.out.cursor
			if inputCommandIndex < len(inputCommand) {
				inputCommand = inputCommand[:inputCommandIndex]
				inputCommand = s.updateInputCommand(inputCommand)
				s.out.cursor = 0
			}
			break
		case keyboard.A:
			s.out.setCursor(-len(inputCommand))
			break
		case keyboard.E:
			if s.candidateCommand != "" {
				inputCommand = s.updateInputCommand(s.candidateCommand)
			}
			s.out.setCursor(0)
			break
		case keyboard.F:
			s.moveCursorForward()
			break
		case keyboard.B:
			s.moveCursorBackward(inputCommand)
			break
		}
		return inputCommand, nil
	}

	switch keyEvent.Key {
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
		inputCommand = s.updateInputCommand(inputCommand)
		break
	case keyboard.ArrowUp:
		inputCommand = s.showPreviousCommandFromHistory(inputCommand)
		break
	case keyboard.ArrowDown:
		inputCommand = s.showNextCommandFromHistory(inputCommand)
		break
	case keyboard.ArrowRight:
		s.moveCursorForward()
		break
	case keyboard.ArrowLeft:
		s.moveCursorBackward(inputCommand)
		break
	case keyboard.Tab:
		args := strings.Fields(inputCommand)
		if len(args) == 0 {
			break
		}

		suggestPlugin, ok := s.plugins[args[0]]
		if !ok {
			suggestPlugin = s.defaultPlugin
		}
		var err error
		inputCommand, err = s.suggest(suggestPlugin, args, inputCommand)
		if err != nil {
			fmt.Println(err)
			break
		}
		inputCommand = s.updateInputCommand(inputCommand)
		break
	default:
		if !utf8.ValidRune(keyEvent.Rune) {
			break
		}

		if s.out.cursor < 0 {
			inputCommandIndex := len(inputCommand) + s.out.cursor
			inputCommand = inputCommand[:inputCommandIndex] + string(keyEvent.Rune) + inputCommand[inputCommandIndex:]
		} else {
			inputCommand = inputCommand + string(keyEvent.Rune)
		}
		s.candidateCommand = s.history.StartWith(inputCommand, 0)
		if inputCommand == s.candidateCommand {
			s.candidateCommand = ""
		}
	}

	return inputCommand, nil
}

func (s Shell) getInputCommand() (string, error) {
	s.out.initNewLine()
	s.out.setCursor(0)
	s.candidateCommand = ""

	interuptSignals := make(chan os.Signal, 1)
	defer close(interuptSignals)
	signal.Notify(interuptSignals, syscall.SIGINT)

	inputCommand := ""
	for {
		keyEvent, err := s.in.Read()
		fmt.Printf("%v\n", keyEvent)
		if err == io.EOF {
			s.out.writeLine(inputCommand, "")
			s.out.newLine()
			break
		} else if err != nil {
			s.out.writeLine(inputCommand, "")
			return "", err
		}
		if keyEvent.Key == keyboard.Enter {
			s.out.writeLine(inputCommand, "")
			s.out.newLine()
			break
		}
		if keyEvent.IsControlPressed && keyEvent.Key == keyboard.C {
			s.out.writeLine(inputCommand, "")
			s.out.newLine()
			inputCommand = ""
			break
		}

		go func() {
			// Don't cancel a shell when the child command is canceled
			<-interuptSignals
		}()
		inputCommand, err = s.handleShortcutKey(inputCommand, keyEvent)
		if err != nil {
			s.out.writeLine("", "")
			return "", err
		}

		if len(inputCommand) <= 0 {
			s.out.writeLine("", "")
			continue
		}

		s.out.writeLine(inputCommand, s.candidateCommand)
	}
	return inputCommand, nil
}

func (s Shell) suggest(p plugin.Plugin, args []string, inputCommand string) (string, error) {
	var currentArgToken string
	var previousArgs string
	if len(inputCommand) > 1 {
		previousChar := inputCommand[len(inputCommand)+s.out.cursor-1]
		if previousChar != ' ' {
			lastSpaceIndex := strings.LastIndex(inputCommand, " ")
			if lastSpaceIndex != -1 {
				currentArgToken = inputCommand[lastSpaceIndex:]
				previousArgs = inputCommand[:lastSpaceIndex]
			} else {
				currentArgToken = inputCommand
			}
		}
	}

	arg := plugin.SuggestArg{
		Args:            args,
		History:         &s.history,
		CurrentArgToken: strings.TrimSpace(currentArgToken),
	}
	var suggested []string
	var err error
	suggested, err = p.Suggest(arg)
	if err != nil {
		return inputCommand, err
	}
	if len(suggested) > 0 {
		if previousArgs != "" {
			inputCommand = previousArgs + " " + strings.Join(suggested, " ")
		} else {
			inputCommand = inputCommand + strings.Join(suggested, " ")
		}
	}
	return inputCommand, nil
}
