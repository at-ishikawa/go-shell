package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/at-ishikawa/go-shell/internal/completion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilePlugin_readDirectory(t *testing.T) {
	osTempDir := os.TempDir()
	tempDir, err := os.MkdirTemp(osTempDir, "go-shell-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	type fields struct {
		completionUi completion.Completion
		homeDir      string
	}
	type args struct {
		query                      string
		suggestedValuesFromHistory []string
	}
	testCases := []struct {
		name   string
		fields fields
		args   args
		files  []string

		want    []string
		wantErr error
	}{
		{
			name: "empty dir",
			fields: fields{
				homeDir: "/home/user",
			},
			args: args{
				query:                      tempDir + string(os.PathSeparator) + "part_of_file_name",
				suggestedValuesFromHistory: []string{},
			},
			want: []string{
				osTempDir + string(os.PathSeparator),
			},
		},
		{
			name: "directory with files. Directory first",
			fields: fields{
				homeDir: "/home/user",
			},
			args: args{
				query:                      tempDir + string(os.PathSeparator),
				suggestedValuesFromHistory: []string{},
			},
			files: []string{
				"test.txt",
				"subdir1/",
				"subdir1/test2.txt",
			},
			want: []string{
				tempDir + "/subdir1/",
				tempDir + "/test.txt",
				osTempDir + string(os.PathSeparator),
			},
		},
		{
			name: "directory with files. Set the order by history",
			fields: fields{
				homeDir: "/home/user",
			},
			args: args{
				query: tempDir + string(os.PathSeparator),
				suggestedValuesFromHistory: []string{
					tempDir + "/test.txt",
					tempDir + "/subdir2/test2.txt",
				},
			},
			files: []string{
				"test.txt",
				"subdir1/",
				"subdir2/",
				"subdir2/test2.txt",
			},
			want: []string{
				tempDir + "/test.txt",
				tempDir + "/subdir2/",
				tempDir + "/subdir1/",
				osTempDir + string(os.PathSeparator),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, file := range tc.files {
				filePath := filepath.Join(tempDir, file)
				if strings.HasSuffix(file, "/") {
					err = os.Mkdir(filePath, os.ModePerm)
				} else {
					err = os.WriteFile(filePath, []byte{}, os.ModePerm)
				}
				require.NoError(t, err)
				t.Cleanup(func() {
					os.Remove(filePath)
				})
			}

			f := &FilePlugin{
				homeDir: tc.fields.homeDir,
			}
			got, gotErr := f.readDirectory(tc.args.query, tc.args.suggestedValuesFromHistory)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}
