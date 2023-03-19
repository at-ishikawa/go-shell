package ansi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParserString(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want []AnsiString
	}{
		{
			name: "no ansi code",
			str:  "no ansi",
			want: []AnsiString{
				{String: "no ansi"},
			},
		},
		{
			name: "only ansi reset code",
			str:  " \u001b[m",
			want: []AnsiString{
				{String: " "},
			},
		},
		{
			name: "single ansi code",
			str:  "\u001b[1m+++ b/internal/completion/tcell.go\u001b[m",
			want: []AnsiString{
				{String: "+++ b/internal/completion/tcell.go", Style: StyleBold},
			},
		},
		{
			name: "multiple ansi codes",
			str:  "\u001b[32m+\u001b[m\t\u001b[32m\"regexp\"\u001b[m",
			want: []AnsiString{
				{String: "+", ForegroundColor: ColorGreen},
				{String: "\t"},
				{String: `"regexp"`, ForegroundColor: ColorGreen},
			},
		},
		{
			name: "style and color ansi code",
			str:  "\u001b[1;36m+\u001b[m\u001b[1;36m}\u001b[m",
			want: []AnsiString{
				{String: "+", ForegroundColor: ColorCyan, Style: StyleBold},
				{String: "}", ForegroundColor: ColorCyan, Style: StyleBold},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseString(tc.str))
		})
	}
}
