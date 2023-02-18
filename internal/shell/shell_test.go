package shell

import (
	"testing"

	"github.com/at-ishikawa/go-shell/internal/keyboard"
	"github.com/stretchr/testify/assert"
)

func Test_HandleShortcutKey(t *testing.T) {
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
		testCases := []struct {
			name             string
			shell            Shell
			command          string
			typedChar        rune
			keyCode          keyboard.Key
			wantCommand      string
			wantHistoryIndex int
			wantCursor       int
			wantErr          error
		}{
			{
				name: "Show the previous command from a history from a command",
				shell: Shell{
					histories: []string{
						"command1",
						"command2",
					},
					historyIndex: 2,
				},
				command:          "ab",
				keyCode:          keyboard.ControlP,
				wantCommand:      "command2",
				wantHistoryIndex: 1,
			},
			{
				name: "Show the previous command from a history from the last command",
				shell: Shell{
					histories: []string{
						"command1",
						"command2",
					},
					historyIndex: 1,
				},
				command:          "command2",
				keyCode:          keyboard.ControlP,
				wantCommand:      "command1",
				wantHistoryIndex: 0,
			},
			{
				name:    "Show the previous command from a history when no history",
				keyCode: keyboard.ControlP,
			},
			{
				name: "Show the previous command from a history when there is no history",
				shell: Shell{
					histories: []string{
						"command1",
					},
					historyIndex: 1,
				},
				command:          "command1",
				keyCode:          keyboard.ControlP,
				wantCommand:      "command1",
				wantHistoryIndex: 0,
			},

			{
				name: "Show the next command from a history",
				shell: Shell{
					histories: []string{
						"command1",
						"command2",
					},
					historyIndex: 0,
				},
				command:          "command1",
				keyCode:          keyboard.ControlN,
				wantCommand:      "command2",
				wantHistoryIndex: 1,
			},
			{
				name: "Show the next command when no more command",
				shell: Shell{
					histories: []string{
						"command1",
						"command2",
					},
					historyIndex: 1,
				},
				command:          "command2",
				keyCode:          keyboard.ControlN,
				wantCommand:      "",
				wantHistoryIndex: 2,
			},
			{
				name:    "Show the next command when no history",
				keyCode: keyboard.ControlN,
			},
			{
				name: "Show the next history when there is no history",
				shell: Shell{
					histories: []string{
						"command1",
					},
					historyIndex: 2,
				},
				command:          "abc",
				keyCode:          keyboard.ControlN,
				wantCommand:      "abc",
				wantHistoryIndex: 2,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				gotLine, gotErr := tc.shell.handleShortcutKey(tc.command, tc.typedChar, tc.keyCode)
				assert.Equal(t, tc.wantCommand, gotLine)
				assert.Equal(t, tc.wantCursor, tc.shell.out.cursor)
				assert.Equal(t, tc.wantHistoryIndex, tc.shell.historyIndex)
				assert.Equal(t, tc.wantErr, gotErr)
			})
		}
	})
}
