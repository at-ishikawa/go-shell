package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newHistoryFromCommands(strs []string) []HistoryItem {
	list := []HistoryItem{}
	for _, str := range strs {
		list = append(list, HistoryItem{
			Command: str,
		})
	}
	return list
}

func TestHistory_Previous(t *testing.T) {
	testCases := []struct {
		name      string
		history   History
		command   string
		want      string
		wantIndex int
	}{
		{
			name: "Show the previous command from a history from a command",
			history: History{
				list: newHistoryFromCommands([]string{
					"command1",
					"command2",
				}),
				index: 2,
			},
			want:      "command2",
			wantIndex: 1,
		},
		{
			name: "Show the previous command from a history from the last command",
			history: History{
				list: newHistoryFromCommands([]string{
					"command1",
					"command2",
				}),
				index: 1,
			},
			want:      "command1",
			wantIndex: 0,
		},
		{
			name: "Show the previous command from a history when no history",
		},
		{
			name: "Show the previous command from a history when there is no history",
			history: History{
				list: newHistoryFromCommands([]string{
					"command1",
				}),
				index: 1,
			},
			want:      "command1",
			wantIndex: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.history.Previous()
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantIndex, tc.history.index)
		})
	}
}

func TestHistory_SaveFile(t *testing.T) {
	dir := os.TempDir()
	fmt.Println(dir)
	tmpConfig, err := NewConfig(dir)
	assert.NoError(t, err)
	now := time.Date(2023, 1, 1, 1, 1, 1, 1, time.UTC)

	testCases := []struct {
		name string
		want History
	}{
		{
			name: "file doesn't exist",
			want: History{
				fileName: "tmp-want.json",
				maxSize:  10,
				config:   tmpConfig,
				list: []HistoryItem{
					{Command: "command1", LastSucceededAt: now.Add(1), LastFailedAt: now.Add(2), Count: 1, Directories: []string{"/path/to/dir1", "/path/to/dir2"}},
					{Command: "command2", LastSucceededAt: now.Add(10), LastFailedAt: now.Add(20), Count: 2, Directories: []string{"/path/to/dir11", "/path/to/dir12"}},
				},
				currentTime: now,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NoError(t, tc.want.saveFile())

			got := History{
				fileName:    tc.want.fileName,
				maxSize:     tc.want.maxSize,
				config:      tc.want.config,
				currentTime: tc.want.currentTime,
			}
			assert.NoError(t, got.LoadFile())
			assert.Equal(t, tc.want.list, got.list)
			assert.Equal(t, len(tc.want.list), got.index)
		})
	}
}

func TestHistory_Next(t *testing.T) {
	testCases := []struct {
		name      string
		history   History
		command   string
		want      string
		wantIndex int
		wantOk    bool
	}{
		{
			name: "Show the next command from a history",
			history: History{
				list: newHistoryFromCommands([]string{
					"command1",
					"command2",
				}),
				index: 0,
			},
			want:      "command2",
			wantIndex: 1,
			wantOk:    true,
		},
		{
			name: "Show the next command when no more command",
			history: History{
				list: newHistoryFromCommands([]string{
					"command1",
					"command2",
				}),
				index: 1,
			},
			want:      "",
			wantIndex: 2,
			wantOk:    true,
		},
		{
			name: "Show the next command when no history",
		},
		{
			name: "Show the next history when there is no history",
			history: History{
				list: newHistoryFromCommands([]string{
					"command1",
				}),
				index: 2,
			},
			want:      "",
			wantIndex: 2,
			wantOk:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := tc.history.Next()
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantIndex, tc.history.index)
			assert.Equal(t, tc.wantOk, ok)
		})
	}
}

func TestHistory_Add(t *testing.T) {
	commandRunAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name    string
		history History

		command   string
		status    int
		directory string

		wantList  []HistoryItem
		wantIndex int
	}{
		{
			name: "The first command",
			history: History{
				list:  []HistoryItem{},
				index: 0,
			},
			command:   "command1",
			status:    0,
			directory: "/path/to/dir1",
			wantList: []HistoryItem{
				{
					Command:         "command1",
					Count:           1,
					LastSucceededAt: commandRunAt,
					Directories:     []string{"/path/to/dir1"},
				},
			},
			wantIndex: 1,
		},
		{
			name: "Run a different command",
			history: History{
				list: []HistoryItem{
					{
						Command: "command1",
						Count:   1,
					},
				},
				index: 1,
			},
			command:   "command2",
			status:    1,
			directory: "/path/to/dir1",
			wantList: []HistoryItem{
				{
					Command: "command1",
					Count:   1,
				},
				{
					Command:      "command2",
					Count:        1,
					LastFailedAt: commandRunAt,
					Directories:  []string{"/path/to/dir1"},
				},
			},
			wantIndex: 2,
		},

		{
			name: "Add the same command",
			history: History{
				list: []HistoryItem{
					{
						Command:         "command1",
						Count:           2,
						LastSucceededAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
						Directories:     []string{"/path/to/dir1"},
					},
				},
				index: 1,
			},
			command:   "command1",
			status:    1,
			directory: "/path/to/dir2",
			wantList: []HistoryItem{
				{
					Command:         "command1",
					Count:           3,
					LastSucceededAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
					LastFailedAt:    commandRunAt,
					Directories:     []string{"/path/to/dir1", "/path/to/dir2"},
				},
			},
			wantIndex: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.history.Add(tc.command, tc.status, tc.directory, commandRunAt)
			assert.Equal(t, tc.wantList, tc.history.list)
			assert.Equal(t, tc.wantIndex, tc.history.index)
		})
	}
}
