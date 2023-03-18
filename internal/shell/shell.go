package shell

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/at-ishikawa/go-shell/internal/config"
	"go.uber.org/zap"

	"github.com/at-ishikawa/go-shell/internal/completion"
	"github.com/at-ishikawa/go-shell/internal/plugin/kubectl"
)

type Shell struct {
	logger         *zap.Logger
	history        config.History
	terminal       terminal
	shellSuggester suggester
	commandRunner  commandRunner
}

type Options struct {
	IsDebug bool
}

func NewShell(inFile *os.File, outFile *os.File, errorFile *os.File, options Options) (Shell, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Shell{}, err
	}
	conf, err := config.NewConfig(homeDir)
	if err != nil {
		return Shell{}, err
	}
	logger, err := func(isDebug bool) (*zap.Logger, error) {
		tempDir := os.TempDir()
		loggerPath := fmt.Sprintf("%sgo-shell-%s.log", tempDir, time.Now().Format("2006-01-02"))
		loggerConfig := zap.NewProductionConfig()
		if isDebug {
			loggerConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		}
		loggerConfig.OutputPaths = []string{
			loggerPath,
		}
		loggerConfig.ErrorOutputPaths = []string{
			loggerPath,
		}

		fmt.Printf("You can see a log on %s\n", loggerPath)
		return loggerConfig.Build()
	}(options.IsDebug)
	if err != nil {
		return Shell{}, err
	}

	commandHistory := config.NewHistory(conf)
	if err := commandHistory.LoadFile(); err != nil {
		return Shell{}, fmt.Errorf("failed to load a history file: %w", err)
	}
	terminal, err := newTerminal(
		inFile,
		outFile,
		errorFile,
		&commandHistory,
		logger,
	)
	if err != nil {
		return Shell{}, fmt.Errorf("failed to initialize a terminal: %w", err)
	}

	completionUi := completion.NewFzf()
	suggester := newSuggester(&terminal, completionUi, &commandHistory, homeDir)
	// todo remove a circular dependency
	terminal.suggester = suggester

	return Shell{
		logger:         logger,
		history:        commandHistory,
		terminal:       terminal,
		shellSuggester: suggester,
		commandRunner:  newCommandRunner(homeDir),
	}, nil
}

// https://hackernoon.com/today-i-learned-making-a-simple-interactive-shell-application-in-golang-aa83adcb266a
func (s Shell) Run() error {
	defer func() {
		if err := s.logger.Sync(); err != nil {
			s.logger.Error("Failed to zap.Logger.sync", zap.Error(err))
		}
	}()
	defer func() {
		if err := s.terminal.finalize(); err != nil {
			s.logger.Error("Failed to finalize a terminal input", zap.Error(err))
		}
	}()
	if err := s.terminal.makeRaw(); err != nil {
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
				s.terminal.setPrompt(fmt.Sprintf("[%s|%s] $ ", kubeCtx, kubeNamespace))
			}
		}

		inputCommand, err := s.terminal.getInputCommand()
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
		if err := s.terminal.restore(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		exitCode, err := s.commandRunner.run(inputCommand, s.terminal.commandFactory())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if err := s.terminal.makeRaw(); err != nil {
			return err
		}

		// wait for the previous stored history process will be done
		if historyChannel != nil {
			<-historyChannel
		}
		// In order to avoid storing commands with syntax error, do not store commands failed
		historyChannel = s.history.Sync(inputCommand, exitCode, s.logger)
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
