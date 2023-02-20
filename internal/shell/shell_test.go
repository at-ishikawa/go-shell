package shell

import (
	"testing"

	"github.com/at-ishikawa/go-shell/internal/config"

	"github.com/at-ishikawa/go-shell/internal/keyboard"
	"github.com/stretchr/testify/assert"
)

func Test_HandleShortcutKey(t *testing.T) {
	t.Run("type a character", func(t *testing.T) {
		testCases := []struct {
			name        string
			shell       Shell
			command     string
			typedChar   rune
			wantCommand string
			wantCursor  int
			wantErr     error
		}{
			{
				name:        "Type a letter",
				command:     "ab",
				typedChar:   rune(byte('c')),
				wantCommand: "abc",
			},
			{
				name:        "Type a letter when no command",
				typedChar:   rune(byte('c')),
				wantCommand: "c",
			},
			{
				name: "Type a letter when the cursor is the middle of the command",
				shell: Shell{
					out: output{
						cursor: -1,
					},
				},
				command:     "ab",
				typedChar:   rune(byte('c')),
				wantCommand: "acb",
				wantCursor:  -1,
			},

			{
				name:        "Type a space",
				command:     "ab",
				typedChar:   rune(byte(' ')),
				wantCommand: "ab ",
			},
			{
				name:        "Type a space when no command",
				typedChar:   rune(byte(' ')),
				wantCommand: " ",
			},
			{
				name: "Type a space when the cursor is the middle of the command",
				shell: Shell{
					out: output{
						cursor: -1,
					},
				},
				command:     "ab",
				typedChar:   rune(byte(' ')),
				wantCommand: "a b",
				wantCursor:  -1,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				gotLine, gotErr := tc.shell.handleShortcutKey(tc.command, tc.typedChar, keyboard.Key_Unknown)
				assert.Equal(t, tc.wantCommand, gotLine)
				assert.Equal(t, tc.wantCursor, tc.shell.out.cursor)
				assert.Equal(t, tc.wantErr, gotErr)
			})
		}

	})

	t.Run("Move cursor shortcuts", func(t *testing.T) {

		testCases := []struct {
			name        string
			shell       Shell
			command     string
			typedChar   rune
			keyCode     keyboard.Key
			wantCommand string
			wantCursor  int
			wantErr     error
		}{
			{
				name:        "Move a cursor back",
				command:     "ab",
				keyCode:     keyboard.ControlB,
				wantCommand: "ab",
				wantCursor:  -1,
			},
			{
				name:    "Move back when no command",
				keyCode: keyboard.ControlB,
			},
			{
				name: "Move a cursor back when the cursor is the beginning of the command",
				shell: Shell{
					out: output{
						cursor: -2,
					},
				},
				command:     "ab",
				keyCode:     keyboard.ControlB,
				wantCommand: "ab",
				wantCursor:  -2,
			},

			{
				name: "Move a cursor forward",
				shell: Shell{
					out: output{
						cursor: -1,
					},
				},
				command:     "ab",
				keyCode:     keyboard.ControlF,
				wantCommand: "ab",
			},
			{
				name:    "Move forward when no command",
				keyCode: keyboard.ControlF,
			},
			{
				name:        "Move a cursor forward when the cursor is the end of the command",
				command:     "ab",
				keyCode:     keyboard.ControlF,
				wantCommand: "ab",
			},

			{
				name:        "Move a cursor on the beginning of a command",
				command:     "ab",
				keyCode:     keyboard.ControlA,
				wantCommand: "ab",
				wantCursor:  -2,
			},
			{
				name:    "Move a cursor on the beginning on the command when no command",
				keyCode: keyboard.ControlA,
			},
			{
				name: "Move a cursor on the beginning on the command when it's already on the beginning on the command",
				shell: Shell{
					out: output{
						cursor: -2,
					},
				},
				command:     "ab",
				keyCode:     keyboard.ControlA,
				wantCommand: "ab",
				wantCursor:  -2,
			},

			{
				name: "Move a cursor on the end of a command",
				shell: Shell{
					out: output{
						cursor: -2,
					},
				},
				command:     "ab",
				keyCode:     keyboard.ControlE,
				wantCommand: "ab",
			},
			{
				name:    "Move a cursor on the end on the command when no command",
				keyCode: keyboard.ControlE,
			},
			{
				name:        "Move a cursor on the end on the command when it's already on the beginning on the command",
				command:     "ab",
				keyCode:     keyboard.ControlE,
				wantCommand: "ab",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				gotLine, gotErr := tc.shell.handleShortcutKey(tc.command, tc.typedChar, tc.keyCode)
				assert.Equal(t, tc.wantCommand, gotLine)
				assert.Equal(t, tc.wantCursor, tc.shell.out.cursor)
				assert.Equal(t, tc.wantErr, gotErr)
			})
		}
	})

	t.Run("Delete shortcuts", func(t *testing.T) {
		testCases := []struct {
			name        string
			shell       Shell
			command     string
			typedChar   rune
			keyCode     keyboard.Key
			wantCommand string
			wantCursor  int
			wantErr     error
		}{
			{
				name:        "Backspace",
				command:     "ab",
				keyCode:     keyboard.Backspace,
				wantCommand: "a",
			},
			{
				name:    "Backspace when no command",
				keyCode: keyboard.Backspace,
			},
			{
				name: "Backspace when a cursor is in the middle",
				shell: Shell{
					out: output{
						cursor: -1,
					},
				},
				command:     "abc",
				keyCode:     keyboard.Backspace,
				wantCommand: "ac",
				wantCursor:  -1,
			},

			{
				name: "Delete one char forward",
				shell: Shell{
					out: output{
						cursor: -1,
					},
				},
				command:     "ab",
				keyCode:     keyboard.ControlD,
				wantCommand: "a",
			},
			{
				name:    "Delete one char forward when no command",
				keyCode: keyboard.ControlD,
			},
			{
				name: "Delete one char forward when a cursor is in the beginning",
				shell: Shell{
					out: output{
						cursor: -3,
					},
				},
				command:     "abc",
				keyCode:     keyboard.ControlD,
				wantCommand: "bc",
				wantCursor:  -2,
			},

			{
				name: "Delete a word before a cursor",
				shell: Shell{
					out: output{
						cursor: -2,
					},
				},
				command:     "abc d",
				keyCode:     keyboard.ControlW,
				wantCommand: " d",
				wantCursor:  -2,
			},
			{
				name:    "Delete a word before a cursor when no command",
				keyCode: keyboard.ControlW,
			},
			{
				name: "Delete a word before a cursor on the beginning of the command",
				shell: Shell{
					out: output{
						cursor: -2,
					},
				},
				command:     "ab",
				keyCode:     keyboard.ControlW,
				wantCommand: "ab",
				wantCursor:  -2,
			},

			{
				name: "Delete a line after a cursor",
				shell: Shell{
					out: output{
						cursor: -2,
					},
				},
				command:     "abc",
				keyCode:     keyboard.ControlK,
				wantCommand: "a",
			},
			{
				name:    "Delete a line after a cursor when no command",
				keyCode: keyboard.ControlK,
			},
			{
				name:        "Delete a line after a cursor on the end of the command",
				command:     "ab",
				keyCode:     keyboard.ControlK,
				wantCommand: "ab",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				gotLine, gotErr := tc.shell.handleShortcutKey(tc.command, tc.typedChar, tc.keyCode)
				assert.Equal(t, tc.wantCommand, gotLine)
				assert.Equal(t, tc.wantCursor, tc.shell.out.cursor)
				assert.Equal(t, tc.wantErr, gotErr)
			})
		}
	})

	t.Run("History shortcuts", func(t *testing.T) {
		newHistoryFromCommands := func(strs []string, countPrevious int) config.History {
			hist := config.History{}
			for _, str := range strs {
				hist.Add(str, 0)
			}
			for i := 0; i < countPrevious; i++ {
				hist.Previous()
			}

			return hist
		}
		testCases := []struct {
			name        string
			shell       Shell
			command     string
			typedChar   rune
			keyCode     keyboard.Key
			wantCommand string
			wantCursor  int
			wantErr     error
		}{
			{
				name: "Show the previous command from a history from a command",
				shell: Shell{
					history: newHistoryFromCommands([]string{
						"command1",
						"command2",
					}, 0),
				},
				command:     "ab",
				keyCode:     keyboard.ControlP,
				wantCommand: "command2",
			},
			{
				name: "Show the previous command from a history from the last command",
				shell: Shell{
					history: newHistoryFromCommands([]string{
						"command1",
						"command2",
					}, 1),
				},
				command:     "command2",
				keyCode:     keyboard.ControlP,
				wantCommand: "command1",
			},
			{
				name:    "Show the previous command from a history when no history",
				keyCode: keyboard.ControlP,
			},
			{
				name: "Show the previous command from a history when there is no history",
				shell: Shell{
					history: newHistoryFromCommands([]string{
						"command1",
					}, 0),
				},
				command:     "command1",
				keyCode:     keyboard.ControlP,
				wantCommand: "command1",
			},

			{
				name: "Show the next command from a history",
				shell: Shell{
					history: newHistoryFromCommands([]string{
						"command1",
						"command2",
					}, 2),
				},
				command:     "command1",
				keyCode:     keyboard.ControlN,
				wantCommand: "command2",
			},
			{
				name: "Show the next command when no more command",
				shell: Shell{
					history: newHistoryFromCommands([]string{
						"command1",
						"command2",
					}, 1),
				},
				command:     "command2",
				keyCode:     keyboard.ControlN,
				wantCommand: "",
			},
			{
				name:    "Show the next command when no history",
				keyCode: keyboard.ControlN,
			},
			{
				name: "Show the next history when there is no history",
				shell: Shell{
					history: newHistoryFromCommands([]string{
						"command1",
					}, 0),
				},
				command:     "abc",
				keyCode:     keyboard.ControlN,
				wantCommand: "abc",
			},

			// todo: Test Control R
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				gotLine, gotErr := tc.shell.handleShortcutKey(tc.command, tc.typedChar, tc.keyCode)
				assert.Equal(t, tc.wantCommand, gotLine)
				assert.Equal(t, tc.wantCursor, tc.shell.out.cursor)
				assert.Equal(t, tc.wantErr, gotErr)
			})
		}
	})

	t.Run("Escape mode", func(t *testing.T) {
		testCases := []struct {
			name        string
			shell       Shell
			command     string
			typedChar   rune
			keyCode     keyboard.Key
			wantCommand string
			wantCursor  int
			wantErr     error
		}{
			{
				name:        "Move a cursor a word back if a previous char is a space",
				command:     "a b  ",
				keyCode:     keyboard.B,
				wantCommand: "a b  ",
				wantCursor:  -3,
			},
			{
				name:        "Move a cursor a word back if a previous char is a letter",
				command:     "a bc",
				keyCode:     keyboard.B,
				wantCommand: "a bc",
				wantCursor:  -2,
			},
			{
				name: "Move a cursor a word back if a previous char is a space in the middle of the command",
				shell: Shell{
					out: output{
						cursor: -5,
					},
				},
				command:     "a b  c d e",
				keyCode:     keyboard.B,
				wantCommand: "a b  c d e",
				wantCursor:  -8,
			},
			{
				name: "Move a cursor a word back if a previous char is a letter in the middle of a command",
				shell: Shell{
					out: output{
						cursor: -3,
					},
				},
				command:     "a bcd e",
				keyCode:     keyboard.B,
				wantCommand: "a bcd e",
				wantCursor:  -5,
			},
			{
				name:    "Move a cursor a word back when no command",
				keyCode: keyboard.B,
			},
			{
				name: "Move a cursor a word back if a cursor is on the beginning of the command",
				shell: Shell{
					out: output{
						cursor: -1,
					},
				},
				command:     "a",
				keyCode:     keyboard.B,
				wantCommand: "a",
				wantCursor:  -1,
			},
			{
				name:        "Move a cursor a word back if a command is only space",
				command:     " ",
				keyCode:     keyboard.B,
				wantCommand: " ",
				wantCursor:  -1,
			},

			{
				name:        "Move a cursor a word forward if the next char is a space",
				shell:       Shell{out: output{cursor: -2}},
				command:     "a b",
				keyCode:     keyboard.F,
				wantCommand: "a b",
				wantCursor:  0,
			},
			{
				name:        "Move a cursor a word forward if the next char is a letter before the last word",
				shell:       Shell{out: output{cursor: -1}},
				command:     "a bc d",
				keyCode:     keyboard.F,
				wantCommand: "a bc d",
			},
			{
				name:        "Move a cursor a word forward if the next char is a letter",
				shell:       Shell{out: output{cursor: -4}},
				command:     "a bc d",
				keyCode:     keyboard.F,
				wantCommand: "a bc d",
				wantCursor:  -2,
			},
			{
				name:    "Move a cursor a word forward when no command",
				keyCode: keyboard.F,
			},
			{
				name:        "Move a cursor a word forward if a cursor is on the end of the command",
				command:     "a",
				keyCode:     keyboard.F,
				wantCommand: "a",
			},
			{
				name:        "Move a cursor a word forward if a command is only space",
				shell:       Shell{out: output{cursor: -1}},
				command:     " ",
				keyCode:     keyboard.F,
				wantCommand: " ",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				gotLine, gotErr := tc.shell.handleShortcutKey(tc.command, tc.typedChar, keyboard.Escape)
				assert.True(t, tc.shell.isEscapeKeyPressed)

				gotLine, gotErr = tc.shell.handleShortcutKey(gotLine, tc.typedChar, tc.keyCode)
				assert.False(t, tc.shell.isEscapeKeyPressed)
				assert.Equal(t, tc.wantCommand, gotLine)
				assert.Equal(t, tc.wantCursor, tc.shell.out.cursor)
				assert.Equal(t, tc.wantErr, gotErr)
			})
		}
	})

	// todo: Test Tab
}
