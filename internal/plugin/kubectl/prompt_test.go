package kubectl

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetContext(t *testing.T) {
	backupExecCommand := execCommand
	defer func() {
		execCommand = backupExecCommand
	}()

	wantErr := errors.New("PermissionDenied")
	testCases := []struct {
		name        string
		execCommand func(name string, args ...string) ([]byte, error)
		want        string
		wantErr     error
	}{
		{
			name: "return a context",
			execCommand: func(name string, args ...string) ([]byte, error) {
				cmd := exec.Command("echo", " context ")
				return cmd.CombinedOutput()
			},
			want: "context",
		},
		{
			name: "if no kubectl command",
			execCommand: func(name string, args ...string) ([]byte, error) {
				return nil, exec.ErrNotFound
			},
		},
		{
			name: "if an unexpected error",
			execCommand: func(name string, args ...string) ([]byte, error) {
				return nil, wantErr
			},
			wantErr: wantErr,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			execCommand = tc.execCommand
			got, gotErr := GetContext()
			assert.Equal(t, tc.want, got)
			assert.True(t, errors.Is(gotErr, tc.wantErr))
		})
	}
}

func TestGetNamespace(t *testing.T) {
	backupExecCommand := execCommand
	defer func() {
		execCommand = backupExecCommand
	}()

	wantErr := errors.New("PermissionDenied")
	testCases := []struct {
		name        string
		execCommand func(name string, args ...string) ([]byte, error)
		want        string
		wantErr     error
	}{
		{
			name: "return a context",
			execCommand: func(name string, args ...string) ([]byte, error) {
				return exec.Command("echo", "-n", "'namespace'").CombinedOutput()
			},
			want: "namespace",
		},
		{
			name: "if no kubectl command",
			execCommand: func(name string, args ...string) ([]byte, error) {
				return nil, exec.ErrNotFound
			},
		},
		{
			name: "if an unexpected error",
			execCommand: func(name string, args ...string) ([]byte, error) {
				return nil, wantErr
			},
			wantErr: wantErr,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			execCommand = tc.execCommand
			got, gotErr := GetNamespace("context")
			assert.Equal(t, tc.want, got)
			assert.True(t, errors.Is(gotErr, tc.wantErr))
		})
	}
}
