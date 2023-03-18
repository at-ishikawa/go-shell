package shell

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/at-ishikawa/go-shell/internal/config"
	"github.com/at-ishikawa/go-shell/internal/keyboard"
	"go.uber.org/zap"
)

type terminal struct {
	in     input
	out    output
	stdErr output

	prompt           string
	candidateCommand string
	suggester        suggester
	history          *config.History
	logger           *zap.Logger
}

func newTerminal(
	inFile *os.File,
	outFile *os.File,
	errorFile *os.File,
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
		in:      stdinStream,
		out:     stdoutStream,
		stdErr:  stderrorStream,
		history: history,
		logger:  logger,
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

func (term terminal) commandFactory() func(name string, args ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		command := exec.Command(name, args...)
		command.Stderr = term.stdErr.file
		command.Stdout = term.out.file
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
			inputCommand, err = term.suggester.suggestHistory(strings.Fields(inputCommand), inputCommand)
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
		suggested, err := term.suggester.suggestCommand(inputCommand)
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
	defer close(interuptSignals)
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
