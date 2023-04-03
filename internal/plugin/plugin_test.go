package plugin

import (
	"testing"

	"github.com/at-ishikawa/go-shell/internal/completion"
	"github.com/stretchr/testify/assert"
)

func TestSuggestArg_GetDefaultCompletionOption(t *testing.T) {
	testCases := []struct {
		name string
		args SuggestArg
		want completion.CompleteOptions
	}{
		{
			name: "no argument",
			args: SuggestArg{
				CurrentArgToken: "",
			},
			want: completion.CompleteOptions{
				InitialQuery: "",
			},
		},
		{
			name: "one argument",
			args: SuggestArg{
				CurrentArgToken: "test",
			},
			want: completion.CompleteOptions{
				InitialQuery: "test",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.args.GetDefaultCompletionOption()
			assert.Equal(t, tc.want, got)
		})
	}
}
