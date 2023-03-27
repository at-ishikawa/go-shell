package shell

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"

	"github.com/at-ishikawa/go-shell/internal/config"
	"github.com/at-ishikawa/go-shell/internal/keyboard"
	"github.com/at-ishikawa/go-shell/internal/plugin"
	"github.com/at-ishikawa/go-shell/internal/plugin/kubectl"
	"go.uber.org/zap"
)

type terminal struct {
	in     input
	out    output
	stdErr output

	prompt           string
	candidateCommand string
	commandSuggester commandSuggester
	history          *config.History
	logger           *zap.Logger
}

func newTerminal(
	inFile *os.File,
	outFile *os.File,
	errorFile *os.File,
	suggester commandSuggester,
	history *config.History,
	logger *zap.Logger,
) (terminal, error) {
	stdinStream, err := initInput(inFile)
	if err != nil {
		return terminal{}, err
	}
	stdoutStream := initOutput(outFile)
	stderrorStream := initOutput(errorFile)

	return terminal{
		in:               stdinStream,
		out:              stdoutStream,
		stdErr:           stderrorStream,
		commandSuggester: suggester,
		history:          history,
		logger:           logger,
	}, nil
}

func (term *terminal) finalize() error {
	return term.in.finalize()
}

func (term *terminal) makeRaw() error {
	return term.in.makeRaw()
}

func (term *terminal) restore() error {
	return term.in.restore()
}

func (term *terminal) setPrompt(prompt string) {
	term.out.setPrompt(prompt)
}

func (term *terminal) readPrompt() (string, error) {
	kubeCtx, err := kubectl.GetContext()
	if err != nil {
		return "", err
	}
	if kubeCtx == "" {
		return "", nil
	}

	kubeNamespace, err := kubectl.GetNamespace(kubeCtx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("[%s|%s] $ ", kubeCtx, kubeNamespace), nil
}

func (term *terminal) start(f func(inputCommand string) (int, error)) error {
	var historyChannel chan struct{}

	for {
		prompt, err := term.readPrompt()
		if err != nil {
			fmt.Println(err)
		}
		if prompt != "" {
			term.setPrompt(prompt)
		}

		inputCommand, err := term.getInputCommand()
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
		if err := term.restore(); err != nil {
			panic(err)
		}
		exitCode, err := f(inputCommand)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if err := term.makeRaw(); err != nil {
			panic(err)
		}

		context, err := term.commandSuggester.getContext(inputCommand)
		if err != nil {
			term.logger.Error("failed term.commandSuggester.getContext: %w", zap.Error(err))
		}

		// wait for the previous stored history process will be done
		if historyChannel != nil {
			<-historyChannel
		}
		// In order to avoid storing commands with syntax error, do not store commands failed
		historyChannel = term.history.Sync(inputCommand, exitCode, context, term.logger)
	}
	if historyChannel != nil {
		<-historyChannel
	}

	return nil
}

func (term terminal) commandFactory() func(name string, args ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		command := exec.Command(name, args...)
		// command.Stdin = term.in.file
		command.Stdout = term.out.file
		command.Stderr = term.stdErr.file
		return command
	}
}

func (term *terminal) updateInputCommand(str string) string {
	term.candidateCommand = ""
	return str
}

func (term *terminal) moveCursorForward() {
	if term.out.cursor < 0 {
		term.out.moveCursor(1)
	}
}

func (term *terminal) moveCursorBackward(inputCommand string) {
	if -term.out.cursor < len(inputCommand) {
		term.out.moveCursor(-1)
	}
}

func (term *terminal) showPreviousCommandFromHistory(inputCommand string) string {
	previousCommand := term.history.Previous()
	if previousCommand != "" {
		inputCommand = term.updateInputCommand(previousCommand)
	}
	return inputCommand
}

func (term *terminal) showNextCommandFromHistory(inputCommand string) string {
	nextCommand, ok := term.history.Next()
	if ok {
		inputCommand = term.updateInputCommand(nextCommand)
	}
	return inputCommand
}

func (term *terminal) handleShortcutKey(inputCommand string, keyEvent keyboard.KeyEvent) (string, error) {
	if keyEvent.IsEscapePressed {
		switch keyEvent.KeyCode {
		case keyboard.B:
			if -term.out.cursor >= len(inputCommand) {
				break
			}

			previousWord := getPreviousWord(inputCommand, term.out.cursor)
			term.out.cursor = -len(previousWord) + term.out.cursor
			break
		case keyboard.F:
			if term.out.cursor == 0 {
				break
			}

			nextWord := getNextWord(inputCommand, term.out.cursor)
			term.out.cursor += len(nextWord)
			break
		case keyboard.D:
			if len(inputCommand) == 0 {
				break
			}
			if term.out.cursor == 0 {
				break
			}
			nextWord := getNextWord(inputCommand, term.out.cursor)
			inputCommandIndex := len(inputCommand) + term.out.cursor
			inputCommand = inputCommand[:inputCommandIndex] + inputCommand[inputCommandIndex+len(nextWord):]
			inputCommand = term.updateInputCommand(inputCommand)
			term.out.cursor += len(nextWord)
			break
		}
		return inputCommand, nil
	}
	if keyEvent.IsControlPressed {
		switch keyEvent.KeyCode {
		case keyboard.D:
			if len(inputCommand) == 0 {
				break
			}
			if term.out.cursor == 0 {
				break
			}

			inputCommandIndex := len(inputCommand) + term.out.cursor
			inputCommand = inputCommand[:inputCommandIndex] + inputCommand[inputCommandIndex+1:]
			inputCommand = term.updateInputCommand(inputCommand)
			term.out.cursor++
			break
		case keyboard.R:
			var err error
			inputCommand, err = term.suggest(inputCommand, func(arg plugin.SuggestArg) ([]string, error) {
				return term.commandSuggester.suggestHistory(arg)
			})
			if err != nil {
				fmt.Println(err)
				break
			}
			inputCommand = term.updateInputCommand(inputCommand)
			break
		case keyboard.P:
			inputCommand = term.showPreviousCommandFromHistory(inputCommand)
			break
		case keyboard.N:
			inputCommand = term.showNextCommandFromHistory(inputCommand)
			break
		case keyboard.W:
			if -term.out.cursor >= len(inputCommand) {
				break
			}

			previousWord := getPreviousWord(inputCommand, term.out.cursor)
			a := inputCommand[:len(inputCommand)+term.out.cursor-len(previousWord)]
			b := inputCommand[len(inputCommand)+term.out.cursor:]
			inputCommand = a + b
			inputCommand = term.updateInputCommand(inputCommand)

			break
		case keyboard.K:
			inputCommandIndex := len(inputCommand) + term.out.cursor
			if inputCommandIndex < len(inputCommand) {
				inputCommand = inputCommand[:inputCommandIndex]
				inputCommand = term.updateInputCommand(inputCommand)
				term.out.cursor = 0
			}
			break
		case keyboard.A:
			term.out.setCursor(-len(inputCommand))
			break
		case keyboard.E:
			if term.candidateCommand != "" {
				inputCommand = term.updateInputCommand(term.candidateCommand)
			}
			term.out.setCursor(0)
			break
		case keyboard.F:
			term.moveCursorForward()
			break
		case keyboard.B:
			term.moveCursorBackward(inputCommand)
			break
		}
		return inputCommand, nil
	}

	switch keyEvent.KeyCode {
	case keyboard.Backspace:
		if len(inputCommand) == 0 {
			break
		}

		if term.out.cursor < 0 {
			inputCommandIndex := len(inputCommand) + term.out.cursor
			inputCommand = inputCommand[:inputCommandIndex-1] + inputCommand[inputCommandIndex:]
		} else {
			inputCommand = inputCommand[:len(inputCommand)-1]
		}
		inputCommand = term.updateInputCommand(inputCommand)
		break
	case keyboard.ArrowUp:
		inputCommand = term.showPreviousCommandFromHistory(inputCommand)
		break
	case keyboard.ArrowDown:
		inputCommand = term.showNextCommandFromHistory(inputCommand)
		break
	case keyboard.ArrowRight:
		term.moveCursorForward()
		break
	case keyboard.ArrowLeft:
		term.moveCursorBackward(inputCommand)
		break
	case keyboard.Tab:
		suggested, err := term.suggest(inputCommand, func(arg plugin.SuggestArg) ([]string, error) {
			return term.commandSuggester.suggestCommand(inputCommand, arg)
		})
		if err != nil {
			term.logger.Error("Failed to suggest", zap.Error(err))
			return suggested, err
		}
		inputCommand = term.updateInputCommand(suggested)
		break
	default:
		if !utf8.ValidRune(keyEvent.Rune) {
			break
		}

		if term.out.cursor < 0 {
			inputCommandIndex := len(inputCommand) + term.out.cursor
			inputCommand = inputCommand[:inputCommandIndex] + string(keyEvent.Rune) + inputCommand[inputCommandIndex:]
		} else {
			inputCommand = inputCommand + string(keyEvent.Rune)
		}
		term.candidateCommand = term.history.StartWith(inputCommand, 0)
		if inputCommand == term.candidateCommand {
			term.candidateCommand = ""
		}
	}

	return inputCommand, nil
}

func (term *terminal) getInputCommand() (string, error) {
	term.out.initNewLine()
	term.out.setCursor(0)

	term.candidateCommand = ""

	interuptSignals := make(chan os.Signal, 1)
	defer signal.Stop(interuptSignals)
	signal.Notify(interuptSignals, syscall.SIGINT)

	inputCommand := ""
	for {
		keyEvent, err := term.in.Read()
		term.logger.Debug("type", zap.ByteString("bytes", keyEvent.Bytes))

		if err == io.EOF {
			term.out.writeLine(inputCommand, "")
			term.out.newLine()
			break
		} else if err != nil {
			term.out.writeLine(inputCommand, "")
			return "", err
		}
		if keyEvent.KeyCode == keyboard.Enter {
			term.out.writeLine(inputCommand, "")
			term.out.newLine()
			break
		}
		if keyEvent.IsControlPressed && keyEvent.KeyCode == keyboard.C {
			term.out.writeLine(inputCommand, "")
			term.out.newLine()
			inputCommand = ""
			break
		}

		go func() {
			// Don't cancel a shell when the child command is canceled
			<-interuptSignals
		}()
		inputCommand, err = term.handleShortcutKey(inputCommand, keyEvent)
		if err != nil {
			term.out.writeLine("", "")
			return "", err
		}

		if len(inputCommand) <= 0 {
			term.out.writeLine("", "")
			continue
		}

		term.out.writeLine(inputCommand, term.candidateCommand)
	}
	term.candidateCommand = ""
	return inputCommand, nil
}

func (term *terminal) suggest(inputCommand string, suggestFunc func(plugin.SuggestArg) ([]string, error)) (string, error) {
	if strings.TrimSpace(inputCommand) == "" {
		return inputCommand, nil
	}

	// move these logics to terminal
	var currentArgToken string
	var previousArgs string
	if len(inputCommand) > 1 {
		previousChar := inputCommand[len(inputCommand)+term.out.cursor-1]
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
		Command:         inputCommand,
		Args:            strings.Fields(inputCommand),
		History:         term.history,
		CurrentArgToken: strings.TrimSpace(currentArgToken),
	}
	suggested, err := suggestFunc(arg)
	if err != nil {
		return inputCommand, err
	}
	if len(suggested) > 0 {
		if previousArgs != "" {
			inputCommand = previousArgs + " " + strings.Join(suggested, " ")
		} else {
			inputCommand = inputCommand + strings.Join(suggested, " ")
		}
		inputCommand = inputCommand + " "
	}
	return inputCommand, nil
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
