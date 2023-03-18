package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
