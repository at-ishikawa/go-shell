package shell

import (
	"fmt"
	"os"
	"time"

	"github.com/at-ishikawa/go-shell/internal/config"
	"go.uber.org/zap"
)

type Shell struct {
	logger        *zap.Logger
	terminal      terminal
	commandRunner commandRunner
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

		if isDebug {
			fmt.Printf("You can see a log on %s\n", loggerPath)
		}
		return loggerConfig.Build()
	}(options.IsDebug)
	if err != nil {
		return Shell{}, err
	}

	commandHistory := config.NewHistory(conf)
	if err := commandHistory.LoadFile(); err != nil {
		return Shell{}, fmt.Errorf("failed to load a history file: %w", err)
	}
	suggester, err := newCommandSuggester(&commandHistory, homeDir)
	if err != nil {
		return Shell{}, err
	}

	terminal, err := newTerminal(
		inFile,
		outFile,
		errorFile,
		suggester,
		&commandHistory,
		logger,
	)
	if err != nil {
		return Shell{}, fmt.Errorf("failed to initialize a terminal: %w", err)
	}

	return Shell{
		logger:        logger,
		terminal:      terminal,
		commandRunner: newCommandRunner(homeDir),
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

	return s.terminal.start(func(inputCommand string) (int, error) {
		return s.commandRunner.run(inputCommand, s.terminal.commandFactory())
	})
}
