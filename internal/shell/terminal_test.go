package shell

import (
	"bufio"
	"bytes"
	"testing"
	"time"

	"github.com/at-ishikawa/go-shell/internal/config"
	"github.com/at-ishikawa/go-shell/internal/keyboard"
	"github.com/at-ishikawa/go-shell/internal/plugin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestGetPreviousWord(t *testing.T) {
	testCases := []struct {
		name   string
		token  string
		cursor int
		want   string
	}{
		{
			name:  "get a word before a letter",
			token: "file --line-numbers0",
			want:  "numbers0",
		},
		{
			name:   "get a word before non letter nor digit",
			token:  "file --line-numbers0",
			cursor: -8,
			want:   "line-",
		},
		{
			name:   "get a word before non letter nor digit including a space",
			token:  "file --line-numbers0",
			cursor: -13,
			want:   "file --",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := getPreviousWord(tc.token, tc.cursor)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetNextWord(t *testing.T) {
	testCases := []struct {
		name   string
		token  string
		cursor int
		want   string
	}{
		{
			name:   "get a word before a letter",
			token:  "file --line-numbers0",
			cursor: -20,
			want:   "file",
		},
		{
			name:   "get a word before a space and a symbol",
			token:  "file --line-numbers0",
			cursor: -16,
			want:   " --line",
		},
		{
			name:   "get a word before non letter nor digit",
			token:  "file --line-numbers0",
			cursor: -9,
			want:   "-numbers0",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := getNextWord(tc.token, tc.cursor)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTerminal_getInputCommand(t *testing.T) {
	testCases := []struct {
		name        string
		keyCodes    keyboard.Code
		wantCommand string
	}{
		{
			name:     "Enter only",
			keyCodes: keyboard.Enter,
		},
		{
			name:     "Control C",
			keyCodes: keyboard.ControlC,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			term := terminal{
				in: input{
					// Currently, this only reads the first letter
					reader:     bufio.NewReaderSize(bytes.NewReader(tc.keyCodes.Bytes()), 1),
					bufferSize: 8,
				},
				logger: zap.NewNop(),
			}
			got, gotErr := term.getInputCommand()
			assert.NoError(t, gotErr)
			assert.Equal(t, tc.wantCommand, got)
		})
	}
}

func TestTerminal_HandleShortcutKey(t *testing.T) {
	newHistoryFromCommands := func(strs []string, countPrevious int) *config.History {
		hist := config.History{}
		for _, str := range strs {
			hist.Add(str, 0, nil, time.Now())
		}
		for i := 0; i < countPrevious; i++ {
			hist.Previous()
		}

		return &hist
	}

	t.Run("type a character", func(t *testing.T) {
		testCases := []struct {
			name        string
			term        terminal
			command     string
			typedChar   rune
			wantCommand string
			wantCursor  int
			wantErr     error
		}{
			{
				name:        "Type a letter",
				command:     "ab",
				typedChar:   'c',
				wantCommand: "abc",
			},
			{
				name:        "Type a letter when no command",
				typedChar:   'c',
				wantCommand: "c",
			},
			{
				name: "Type a letter when the cursor is the middle of the command",
				term: terminal{
					out: output{
						cursor: -1,
					},
				},
				command:     "ab",
				typedChar:   'c',
				wantCommand: "acb",
				wantCursor:  -1,
			},

			{
				name:        "Type a space",
				command:     "ab",
				typedChar:   ' ',
				wantCommand: "ab ",
			},
			{
				name:        "Type a space when no command",
				typedChar:   ' ',
				wantCommand: " ",
			},
			{
				name: "Type a space when the cursor is the middle of the command",
				term: terminal{
					out: output{
						cursor: -1,
					},
				},
				command:     "ab",
				typedChar:   ' ',
				wantCommand: "a b",
				wantCursor:  -1,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tc.term.history = &config.History{}
				gotLine, gotErr := tc.term.handleShortcutKey(tc.command, keyboard.KeyEvent{
					Rune: tc.typedChar,
				})
				assert.Equal(t, tc.wantCommand, gotLine)
				assert.Equal(t, tc.wantCursor, tc.term.out.cursor)
				assert.Equal(t, tc.wantErr, gotErr)
			})
		}
	})

	t.Run("Move cursor shortcuts", func(t *testing.T) {

		testCases := []struct {
			name                 string
			term                 terminal
			command              string
			keyEvent             keyboard.KeyEvent
			wantCommand          string
			wantCandidateCommand string
			wantCursor           int
			wantErr              error
		}{
			{
				name:    "Move a cursor back",
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.B,
					IsControlPressed: true,
				},
				wantCommand: "ab",
				wantCursor:  -1,
			},
			{
				name:    "Move a cursor back",
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode: keyboard.ArrowLeft,
				},
				wantCommand: "ab",
				wantCursor:  -1,
			},
			{
				name: "Move back when no command",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.B,
					IsControlPressed: true,
				},
			},
			{
				name: "Move a cursor back when the cursor is the beginning of the command",
				term: terminal{
					out: output{
						cursor: -2,
					},
				},
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.B,
					IsControlPressed: true,
				},
				wantCommand: "ab",
				wantCursor:  -2,
			},

			{
				name: "Move a cursor forward",
				term: terminal{
					out: output{
						cursor: -1,
					},
				},
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.F,
					IsControlPressed: true,
				},
				wantCommand: "ab",
			},
			{
				name: "Move a cursor forward by arrow key",
				term: terminal{
					out: output{
						cursor: -1,
					},
				},
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode: keyboard.ArrowRight,
				},
				wantCommand: "ab",
			},
			{
				name: "Move forward when no command",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.F,
					IsControlPressed: true,
				},
			},
			{
				name:    "Move a cursor forward when the cursor is the end of the command",
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.F,
					IsControlPressed: true,
				},
				wantCommand: "ab",
			},

			{
				name:    "Move a cursor on the beginning of a command",
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.A,
					IsControlPressed: true,
				},
				wantCommand: "ab",
				wantCursor:  -2,
			},
			{
				name: "Move a cursor on the beginning on the command when no command",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.A,
					IsControlPressed: true,
				},
			},
			{
				name: "Move a cursor on the beginning on the command when it's already on the beginning on the command",
				term: terminal{
					out: output{
						cursor: -2,
					},
				},
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.A,
					IsControlPressed: true,
				},
				wantCommand: "ab",
				wantCursor:  -2,
			},

			{
				name: "Move a cursor on the end of a command",
				term: terminal{
					out: output{
						cursor: -2,
					},
				},
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.E,
					IsControlPressed: true,
				},
				wantCommand: "ab",
			},
			{
				name: "Move a cursor on the end of a command with a candidate",
				term: terminal{
					out: output{
						cursor: -2,
					},
					candidateCommand: "ab cd ef",
				},
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.E,
					IsControlPressed: true,
				},
				wantCommand: "ab cd ef",
			},
			{
				name: "Move a cursor on the end on the command when no command",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.E,
					IsControlPressed: true,
				},
			},
			{
				name:    "Move a cursor on the end on the command when it's already on the beginning on the command",
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.E,
					IsControlPressed: true,
				},
				wantCommand: "ab",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				gotLine, gotErr := tc.term.handleShortcutKey(tc.command, tc.keyEvent)
				assert.Equal(t, tc.wantCommand, gotLine)
				assert.Equal(t, tc.wantCursor, tc.term.out.cursor)
				assert.Equal(t, tc.wantErr, gotErr)
			})
		}
	})

	t.Run("Delete shortcuts", func(t *testing.T) {
		testCases := []struct {
			name                 string
			terminal             terminal
			command              string
			keyEvent             keyboard.KeyEvent
			wantCommand          string
			wantCandidateCommand string
			wantCursor           int
			wantErr              error
		}{
			{
				name: "Backspace",
				terminal: terminal{
					candidateCommand: "abcde",
				},
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode: keyboard.Backspace,
				},
				wantCommand: "a",
			},
			{
				name: "Backspace when no command",
				keyEvent: keyboard.KeyEvent{
					KeyCode: keyboard.Backspace,
				},
			},
			{
				name: "Backspace when a cursor is in the middle",
				terminal: terminal{
					out: output{
						cursor: -1,
					},
				},
				command: "abc",
				keyEvent: keyboard.KeyEvent{
					KeyCode: keyboard.Backspace,
				},
				wantCommand: "ac",
				wantCursor:  -1,
			},

			{
				name: "Delete one char forward",
				terminal: terminal{
					out: output{
						cursor: -1,
					},
					candidateCommand: "abcde",
				},
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.D,
					IsControlPressed: true,
				},
				wantCommand: "a",
			},
			{
				name: "Delete one char forward when no command",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.D,
					IsControlPressed: true,
				},
			},
			{
				name: "Delete one char forward when a cursor is in the beginning",
				terminal: terminal{
					out: output{
						cursor: -3,
					},
				},
				command: "abc",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.D,
					IsControlPressed: true,
				},
				wantCommand: "bc",
				wantCursor:  -2,
			},

			{
				name: "Delete a word before a cursor",
				terminal: terminal{
					out: output{
						cursor: -2,
					},
					candidateCommand: "abc de",
				},
				command: "abc d",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.W,
					IsControlPressed: true,
				},
				wantCommand: " d",
				wantCursor:  -2,
			},
			{
				name: "Delete a word before a cursor when no command",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.W,
					IsControlPressed: true,
				},
			},
			{
				name: "Delete a word before a cursor on the beginning of the command",
				terminal: terminal{
					out: output{
						cursor: -2,
					},
				},
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.W,
					IsControlPressed: true,
				},
				wantCommand: "ab",
				wantCursor:  -2,
			},

			{
				name: "Delete a line after a cursor",
				terminal: terminal{
					out: output{
						cursor: -2,
					},
					candidateCommand: "abcde",
				},
				command: "abc",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.K,
					IsControlPressed: true,
				},
				wantCommand: "a",
			},
			{
				name: "Delete a line after a cursor when no command",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.K,
					IsControlPressed: true,
				},
			},
			{
				name:    "Delete a line after a cursor on the end of the command",
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.K,
					IsControlPressed: true,
				},
				wantCommand: "ab",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				gotLine, gotErr := tc.terminal.handleShortcutKey(tc.command, tc.keyEvent)
				assert.Equal(t, tc.wantCommand, gotLine)
				assert.Equal(t, tc.wantCandidateCommand, tc.terminal.candidateCommand)
				assert.Equal(t, tc.wantCursor, tc.terminal.out.cursor)
				assert.Equal(t, tc.wantErr, gotErr)
			})
		}
	})

	t.Run("History shortcuts", func(t *testing.T) {
		testCases := []struct {
			name                 string
			terminal             terminal
			command              string
			keyEvent             keyboard.KeyEvent
			wantCommand          string
			wantCandidateCommand string
			wantCursor           int
			wantErr              error
		}{
			{
				name: "Show the previous command from a history from a command",
				terminal: terminal{
					history: newHistoryFromCommands([]string{
						"command1",
						"command2",
					}, 0),
					candidateCommand: "abcde",
				},
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.P,
					IsControlPressed: true,
				},
				wantCommand: "command2",
			},
			{
				name: "Show the previous command from a history from a command by ArrowUp",
				terminal: terminal{
					history: newHistoryFromCommands([]string{
						"command1",
						"command2",
					}, 0),
					candidateCommand: "abcde",
				},
				command: "ab",
				keyEvent: keyboard.KeyEvent{
					KeyCode: keyboard.ArrowUp,
				},
				wantCommand: "command2",
			},
			{
				name: "Show the previous command from a history from the last command",
				terminal: terminal{
					history: newHistoryFromCommands([]string{
						"command1",
						"command2",
					}, 1),
				},
				command: "command2",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.P,
					IsControlPressed: true,
				},
				wantCommand: "command1",
			},
			{
				name: "Show the previous command from a history when no history",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.P,
					IsControlPressed: true,
				},
			},
			{
				name: "Show the previous command from a history when there is no history",
				terminal: terminal{
					history: newHistoryFromCommands([]string{
						"command1",
					}, 0),
				},
				command: "command1",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.P,
					IsControlPressed: true,
				},
				wantCommand: "command1",
			},

			{
				name: "Show the next command from a history",
				terminal: terminal{
					history: newHistoryFromCommands([]string{
						"command1",
						"command2",
					}, 2),
					candidateCommand: "command1 abc",
				},
				command: "command1",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.N,
					IsControlPressed: true,
				},
				wantCommand: "command2",
			},
			{
				name: "Show the next command from a history by ArrowDown",
				terminal: terminal{
					history: newHistoryFromCommands([]string{
						"command1",
						"command2",
					}, 2),
					candidateCommand: "command1 abc",
				},
				command: "command1",
				keyEvent: keyboard.KeyEvent{
					KeyCode: keyboard.ArrowDown,
				},
				wantCommand: "command2",
			},
			{
				name: "Show the next command when no more command",
				terminal: terminal{
					history: newHistoryFromCommands([]string{
						"command1",
						"command2",
					}, 1),
				},
				command: "command2",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.N,
					IsControlPressed: true,
				},
				wantCommand: "",
			},
			{
				name: "Show the next command when no history",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.N,
					IsControlPressed: true,
				},
			},
			{
				name: "Show the next history when there is no history",
				terminal: terminal{
					history: newHistoryFromCommands([]string{
						"command1",
					}, 0),
				},
				command: "abc",
				keyEvent: keyboard.KeyEvent{
					KeyCode:          keyboard.N,
					IsControlPressed: true,
				},
				wantCommand: "abc",
			},

			// todo: Test Control R
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.terminal.history == nil {
					tc.terminal.history = &config.History{}
				}
				gotLine, gotErr := tc.terminal.handleShortcutKey(tc.command, tc.keyEvent)
				assert.Equal(t, tc.wantCommand, gotLine)
				assert.Equal(t, tc.wantCandidateCommand, tc.terminal.candidateCommand)
				assert.Equal(t, tc.wantCursor, tc.terminal.out.cursor)
				assert.Equal(t, tc.wantErr, gotErr)
			})
		}
	})

	t.Run("Escape mode", func(t *testing.T) {
		testCases := []struct {
			name                 string
			terminal             terminal
			command              string
			keyEvent             keyboard.KeyEvent
			wantCommand          string
			wantCursor           int
			wantCandidateCommand string
			wantErr              error
		}{
			{
				name: "Move a cursor a word back if a previous char is a space",
				terminal: terminal{
					candidateCommand: "a b  c d",
				},
				command: "a b  ",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.B,
					IsEscapePressed: true,
				},
				wantCommand:          "a b  ",
				wantCursor:           -3,
				wantCandidateCommand: "a b  c d",
			},
			{
				name:    "Move a cursor a word back if a previous char is a letter",
				command: "a bc",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.B,
					IsEscapePressed: true,
				},
				wantCommand: "a bc",
				wantCursor:  -2,
			},
			{
				name: "Move a cursor a word back if a previous char is a space in the middle of the command",
				terminal: terminal{
					out: output{
						cursor: -5,
					},
				},
				command: "a b  c d e",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.B,
					IsEscapePressed: true,
				},
				wantCommand: "a b  c d e",
				wantCursor:  -8,
			},
			{
				name: "Move a cursor a word back if a previous char is a letter in the middle of a command",
				terminal: terminal{
					out: output{
						cursor: -3,
					},
				},
				command: "a bcd e",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.B,
					IsEscapePressed: true,
				},
				wantCommand: "a bcd e",
				wantCursor:  -5,
			},
			{
				name: "Move a cursor a word back when no command",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.B,
					IsEscapePressed: true,
				},
			},
			{
				name: "Move a cursor a word back if a cursor is on the beginning of the command",
				terminal: terminal{
					out: output{
						cursor: -1,
					},
				},
				command: "a",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.B,
					IsEscapePressed: true,
				},
				wantCommand: "a",
				wantCursor:  -1,
			},
			{
				name:    "Move a cursor a word back if a command is only space",
				command: " ",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.B,
					IsEscapePressed: true,
				},
				wantCommand: " ",
				wantCursor:  -1,
			},

			{
				name:     "Move a cursor a word forward if the next char is a space",
				terminal: terminal{out: output{cursor: -2}},
				command:  "a b",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.F,
					IsEscapePressed: true,
				},
				wantCommand: "a b",
				wantCursor:  0,
			},
			{
				name:     "Move a cursor a word forward if the next char is a letter before the last word",
				terminal: terminal{out: output{cursor: -1}},
				command:  "a bc d",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.F,
					IsEscapePressed: true,
				},
				wantCommand: "a bc d",
			},
			{
				name:     "Move a cursor a word forward if the next char is a letter",
				terminal: terminal{out: output{cursor: -4}},
				command:  "a bc d",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.F,
					IsEscapePressed: true,
				},
				wantCommand: "a bc d",
				wantCursor:  -2,
			},
			{
				name: "Move a cursor a word forward when no command",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.F,
					IsEscapePressed: true,
				},
			},
			{
				name:    "Move a cursor a word forward if a cursor is on the end of the command",
				command: "a",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.F,
					IsEscapePressed: true,
				},
				wantCommand: "a",
			},
			{
				name:     "Move a cursor a word forward if a command is only space",
				terminal: terminal{out: output{cursor: -1}},
				command:  " ",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.F,
					IsEscapePressed: true,
				},
				wantCommand: " ",
			},

			{
				name:     "Delete a word forward if the next char is a space",
				terminal: terminal{out: output{cursor: -2}},
				command:  "a b",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.D,
					IsEscapePressed: true,
				},
				wantCommand: "a",
				wantCursor:  0,
			},
			{
				name:     "Delete a word forward if the next char is a letter before the last word",
				terminal: terminal{out: output{cursor: -1}},
				command:  "a bc d",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.D,
					IsEscapePressed: true,
				},
				wantCommand: "a bc ",
			},
			{
				name:     "Delete a word forward if the next char is a letter",
				terminal: terminal{out: output{cursor: -4}},
				command:  "a bc d",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.D,
					IsEscapePressed: true,
				},
				wantCommand: "a  d",
				wantCursor:  -2,
			},
			{
				name: "Delete a word forward when no command",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.D,
					IsEscapePressed: true,
				},
			},
			{
				name:    "Delete a word forward if a cursor is on the end of the command",
				command: "a",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.D,
					IsEscapePressed: true,
				},
				wantCommand: "a",
			},
			{
				name:     "Delete a word forward if a command is only space",
				terminal: terminal{out: output{cursor: -1}},
				command:  " ",
				keyEvent: keyboard.KeyEvent{
					KeyCode:         keyboard.D,
					IsEscapePressed: true,
				},
				wantCommand: "",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				gotLine, gotErr := tc.terminal.handleShortcutKey(tc.command, tc.keyEvent)
				assert.Equal(t, tc.wantCommand, gotLine)
				assert.Equal(t, tc.wantCursor, tc.terminal.out.cursor)
				assert.Equal(t, tc.wantCandidateCommand, tc.terminal.candidateCommand)
				assert.Equal(t, tc.wantErr, gotErr)
			})
		}
	})

	t.Run("Suggest", func(t *testing.T) {
		testCases := []struct {
			name     string
			terminal terminal

			inputCommand string
			keyEvent     keyboard.KeyEvent

			mockCommandSuggester func(mockController *gomock.Controller) commandSuggester

			wantCommand          string
			wantCursor           int
			wantCandidateCommand string
			wantErr              error
		}{
			{
				name: "No command. default plugin",
				keyEvent: keyboard.KeyEvent{
					KeyCode: keyboard.Tab,
				},
				mockCommandSuggester: func(mockController *gomock.Controller) commandSuggester {
					mockPlugin := plugin.NewMockPlugin(mockController)
					mockPlugin.EXPECT().Suggest(gomock.Any()).Return([]string{"cd"}, nil).Times(1)
					mockCommandSuggester := commandSuggester{
						defaultPlugin: mockPlugin,
					}
					return mockCommandSuggester
				},
				wantCommand: "cd ",
			},
			{
				name: "no supported plugin for a command. fallback to the default plugin",

				inputCommand: "ls ",
				keyEvent: keyboard.KeyEvent{
					KeyCode: keyboard.Tab,
				},

				mockCommandSuggester: func(mockController *gomock.Controller) commandSuggester {
					mockPlugin := plugin.NewMockPlugin(mockController)
					mockPlugin.EXPECT().Suggest(gomock.Any()).Return([]string{"/tmp"}, nil).Times(1)
					mockCommandSuggester := commandSuggester{
						defaultPlugin: mockPlugin,
					}
					return mockCommandSuggester
				},

				wantCommand: "ls /tmp ",
			},
			{
				name: "supported plugin for a command",

				inputCommand: "ls ",
				keyEvent: keyboard.KeyEvent{
					KeyCode: keyboard.Tab,
				},

				mockCommandSuggester: func(mockController *gomock.Controller) commandSuggester {
					mockPlugin := plugin.NewMockPlugin(mockController)
					mockPlugin.EXPECT().Suggest(gomock.Any()).Return([]string{"/tmp"}, nil).Times(1)
					mockCommandSuggester := commandSuggester{
						plugins: map[string]plugin.Plugin{
							"ls": mockPlugin,
						},
					}
					return mockCommandSuggester
				},

				wantCommand: "ls /tmp ",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockController := gomock.NewController(t)
				suggester := tc.mockCommandSuggester(mockController)
				tc.terminal.commandSuggester = suggester

				gotLine, gotErr := tc.terminal.handleShortcutKey(tc.inputCommand, tc.keyEvent)
				assert.Equal(t, tc.wantCommand, gotLine)
				assert.Equal(t, tc.wantCursor, tc.terminal.out.cursor)
				assert.Equal(t, tc.wantCandidateCommand, tc.terminal.candidateCommand)
				assert.Equal(t, tc.wantErr, gotErr)
			})
		}
	})
}
