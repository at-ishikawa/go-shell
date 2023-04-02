package shell

import (
	"bufio"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/at-ishikawa/go-shell/internal/config"
	"github.com/at-ishikawa/go-shell/internal/plugin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func measureTime(f func()) time.Duration {
	before := time.Now()
	f()
	after := time.Now()
	return after.Sub(before)
}

func TestShell_Run(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "input")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Open the temporary file to get its file descriptor
	in, err := os.OpenFile(tmpFile.Name(), os.O_RDWR, 0)
	require.NoError(t, err)
	defer in.Close()

	out, err := ioutil.TempFile("", "output")
	require.NoError(t, err)
	defer os.Remove(out.Name())

	newShell := func(t *testing.T) Shell {
		logger := zap.NewNop()

		c, err := config.NewConfig(tmpFile.Name())
		require.NoError(t, err)
		history := config.NewHistory(c)

		// terminal := newTerminal(f)
		mockController := gomock.NewController(t)
		mockPlugin := plugin.NewMockPlugin(mockController)
		mockPlugin.EXPECT().GetContext(gomock.Any()).Return(nil, nil).AnyTimes()

		reader := bufio.NewReaderSize(in, 1)
		terminal := terminal{
			in: input{
				reader:     reader,
				bufferSize: 1,
			},
			out: output{
				file: out,
			},
			stdErr: output{
				file: os.Stderr,
			},
			logger:  logger,
			history: &history,
			commandSuggester: commandSuggester{
				defaultPlugin: mockPlugin,
			},
		}
		return Shell{
			logger:        logger,
			terminal:      terminal,
			commandRunner: commandRunner{},
		}
	}

	testCases := []struct {
		name string

		inputCommands [][]byte
	}{
		{
			name: "Confirm exit ends a shell",
			inputCommands: [][]byte{
				[]byte("exit"),
			},
		},
		{
			name: "Confirm run a command",
			inputCommands: [][]byte{
				[]byte("echo something"),
				[]byte("exit"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newShell(t)
			s.commandRunner.execCommandContext = exec.CommandContext

			done := make(chan bool)
			read := make(chan bool)
			written := make(chan bool)
			go func() {
				gotErr := s.run(func(f func() (int, error)) (int, error) {
					<-written
					code, err := f()
					read <- true
					return code, err
				})
				assert.NoError(t, gotErr)
				done <- true
			}()

			for index, command := range tc.inputCommands {
				got, err := tmpFile.Write(command)
				require.Equal(t, len(command), got)
				require.NoError(t, err)

				if index < len(tc.inputCommands)-1 {
					written <- true
					<-read
				}
			}
			<-done
		})
	}

	t.Run("signal tests", func(t *testing.T) {
		testCases := []struct {
			name          string
			inputCommands [][]byte
			signal        syscall.Signal
			assertFunc    func(t *testing.T, f func())
		}{
			{
				name: "Ctrl-C",
				inputCommands: [][]byte{
					[]byte("sleep 2"),
					[]byte("exit"),
				},
				signal: syscall.SIGINT,
				assertFunc: func(t *testing.T, f func()) {
					time := measureTime(f)
					if time.Seconds() >= 2 {
						assert.Fail(t, "sleep 2 should be interrupted")
					}
				},
			},
			{
				name: "Ctrl-Z",
				inputCommands: [][]byte{
					[]byte("sleep 2"),
					[]byte("exit"),
				},
				signal: syscall.SIGTSTP,
				assertFunc: func(t *testing.T, f func()) {
					assert.PanicsWithValue(t, "Pausing a process has not been implemented yet", f)
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				s := newShell(t)
				var cmd *exec.Cmd
				s.commandRunner.execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
					cmd = exec.CommandContext(ctx, name, args...)
					return cmd
				}
                
				tc.assertFunc(t, func() {
					done := make(chan bool)
					read := make(chan bool)
					written := make(chan bool)
					go func() {
						for index, command := range tc.inputCommands {
							got, err := tmpFile.Write(command)
							require.Equal(t, len(command), got)
							require.NoError(t, err)

							if index < len(tc.inputCommands)-1 {
								written <- true

								for {
									time.Sleep(200 * time.Microsecond)
									if cmd.Process == nil {
										continue
									}
									if cmd.Process.Pid != 0 {
										break
									}
								}

								// Confirm the process hasn't been exited yet
								require.True(t, cmd.ProcessState == nil || !cmd.ProcessState.Exited())
								// Send the signal to the test process
								syscall.Kill(syscall.Getpid(), tc.signal)
								<-read
							}
						}
						done <- true
					}()

					// run a command in the main go routine
					// this is to catch a panic
					gotErr := s.run(func(f func() (int, error)) (int, error) {
						<-written
						code, err := f()
						read <- true
						return code, err
					})
					assert.NoError(t, gotErr)
					<-done
				})
			})
		}
	})
}
