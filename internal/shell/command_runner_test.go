package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandRunner_CompileInput(t *testing.T) {
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
			name:         "an argument with double quotes ",
			inputCommand: `git commit -m "commit message \"test\" 1" -s`,
			wantCommand:  "git",
			wantArgs: []string{
				"commit",
				"-m",
				`commit message \"test\" 1`,
				"-s",
			},
		},
		{
			name:         "an argument includes double quotes ",
			inputCommand: `git commit -m"commit message \"test\" 1" -s`,
			wantCommand:  "git",
			wantArgs: []string{
				"commit",
				"-m\"commit message \\\"test\\\" 1",
				"-s",
			},
		},
		{
			name:         "a command ends double quotes ",
			inputCommand: `git commit -m "commit message \"test\""`,
			wantCommand:  "git",
			wantArgs: []string{
				"commit",
				"-m",
				"commit message \\\"test\\\"",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cr := commandRunner{homeDir: homeDir}
			gotCommand, gotArgs := cr.compileInput(tc.inputCommand)
			assert.Equal(t, tc.wantCommand, gotCommand)
			assert.Equal(t, tc.wantArgs, gotArgs)
		})
	}
}
