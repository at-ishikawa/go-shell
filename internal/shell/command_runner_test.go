package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandRunner_ParseInput(t *testing.T) {
	homeDir := "/home/user"
	testCases := []struct {
		name         string
		inputCommand string
		wantCommand  string
		wantArgs     []string
	}{
		{
			name:         "no argument",
			inputCommand: "ls",
			wantCommand:  "ls",
		},
		{
			name:         "multiple arguments",
			inputCommand: "ls /home",
			wantCommand:  "ls",
			wantArgs:     []string{"/home"},
		},
		{
			name:         "replace a tilde with a home directory",
			inputCommand: "cd ~",
			wantCommand:  "cd",
			wantArgs:     []string{homeDir},
		},
		{
			name: "no input",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cr := commandRunner{homeDir: homeDir}
			gotCommand, gotArgs := cr.parseInput(tc.inputCommand)
			assert.Equal(t, tc.wantCommand, gotCommand)
			assert.Equal(t, tc.wantArgs, gotArgs)
		})
	}
}
