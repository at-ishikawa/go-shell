package shell

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestCommandRunner_Run(t *testing.T) {
	t.Run("change directory", func(t *testing.T) {
		backupDirectory, err := os.Getwd()
		require.NoError(t, err)

		testCases := []struct {
			name          string
			inputCommand  string
			wantDirectory string
			wantExitCode  int
			wantError     error
		}{
			{
				name:          "directory exists",
				inputCommand:  "cd /",
				wantDirectory: "/",
			},
			{
				name:          "no directory exists",
				inputCommand:  "cd /unknown",
				wantDirectory: backupDirectory,
				wantExitCode:  1,
				wantError: &fs.PathError{
					Op:   "chdir",
					Path: "/unknown",
					Err:  syscall.ENOENT,
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				defer func() {
					err := os.Chdir(backupDirectory)
					require.NoError(t, err)
				}()

				cr := commandRunner{}
				gotExitCode, gotError := cr.run(tc.inputCommand, nil)
				assert.Equal(t, tc.wantExitCode, gotExitCode)
				assert.Equal(t, tc.wantError, gotError)

				currentDirectory, err := os.Getwd()
				require.NoError(t, err)
				assert.Equal(t, tc.wantDirectory, currentDirectory)
			})
		}
	})

	t.Run("run a command", func(t *testing.T) {
		testCases := []struct {
			name string

			inputCommand string
			sleepSecond  string
			mockOutput   string
			signal       syscall.Signal

			wantPanic       string
			wantExitCode    int
			wantError       error
			wantErrorString string
		}{
			{
				name:         "no error",
				inputCommand: "unknown",
				mockOutput:   "file1",
			},
			{
				name:         "an error",
				inputCommand: "unknown",
				mockOutput:   "PermissionDenied",

				wantExitCode:    1,
				wantError:       &exec.ExitError{},
				wantErrorString: "exit status 1",
			},
			{
				name:         "Cancel signal",
				inputCommand: "unknown",
				sleepSecond:  "2",
				mockOutput:   "signal is supposed to be sent",
				signal:       syscall.SIGINT,

				wantExitCode: -1,
			},
			{
				name:         "Pause a command",
				inputCommand: "unknown",
				sleepSecond:  "2",
				mockOutput:   "signal is supposed to be sent",
				signal:       syscall.SIGTSTP,

				wantPanic: "Pausing a process has not been implemented yet",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cr := commandRunner{
					homeDir: "/home/user",
					execCommandContext: func(ctx context.Context, command string, args ...string) *exec.Cmd {
						// https://npf.io/2015/06/testing-exec-command/
						cs := []string{"-test.run=TestCommandRunner_RunHelperProcess", "--", command}
						cs = append(cs, args...)
						cmd := exec.Command(os.Args[0], cs...)
						cmd.Env = []string{
							"GO_WANT_HELPER_PROCESS_EXIT_CODE=" + strconv.Itoa(tc.wantExitCode),
							"GO_WANT_HELPER_PROCESS_OUTPUT=" + tc.mockOutput,
						}
						if tc.inputCommand != "" {
							cmd.Env = append(cmd.Env, "GO_WANT_HELPER_PROCESS_SLEEP_SECOND="+tc.sleepSecond)
						}
						return cmd
					},
				}

				term := terminal{
					in: input{
						file: os.Stdin,
					},
					out: output{
						file: os.Stdout,
					},
					stdErr: output{
						file: os.Stderr,
					},
				}
				if tc.signal != 0 {
					go func() {
						s := make(chan os.Signal, 1)
						signal.Notify(s, tc.signal)
						time.Sleep(100 * time.Millisecond)
						syscall.Kill(syscall.Getpid(), tc.signal)
					}()
				}
				if tc.wantPanic != "" {
					assert.PanicsWithValue(t, tc.wantPanic, func() {
						cr.run(tc.inputCommand, &term)
					})
					return
				}

				got, gotErr := cr.run(tc.inputCommand, &term)
				assert.Equal(t, tc.wantExitCode, got)
				if tc.wantError != nil {
					assert.ErrorAs(t, gotErr, &tc.wantError)
					assert.Equal(t, tc.wantErrorString, gotErr.Error())
				} else {
					assert.NoError(t, gotErr)
				}
			})
		}
	})
}

func TestCommandRunner_RunHelperProcess(t *testing.T) {
	wantExitCode := os.Getenv("GO_WANT_HELPER_PROCESS_EXIT_CODE")
	if wantExitCode == "" {
		// It runs with normal test process
		return
	}

	i, err := strconv.Atoi(wantExitCode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get GO_WANT_HELPER_PROCESS_EXIT_CODE")
		os.Exit(1)
	}

	sleepTime := os.Getenv("GO_WANT_HELPER_PROCESS_SLEEP_SECOND")
	if sleepTime != "" {
		second, _ := strconv.Atoi(sleepTime)
		time.Sleep(time.Duration(second) * time.Second)
		os.Exit(1)
	}

	wantOutput := os.Getenv("GO_WANT_HELPER_PROCESS_OUTPUT")
	if i > 0 {
		fmt.Fprintf(os.Stderr, wantOutput)
		os.Exit(i)
	}
	fmt.Fprintf(os.Stdout, wantOutput)
	os.Exit(i)
}
