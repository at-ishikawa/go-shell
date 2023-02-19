package kubectl

import (
	"testing"

	"github.com/at-ishikawa/go-shell/internal/plugin/kubectl/kubectloptions"

	"github.com/stretchr/testify/assert"
)

func Test_filterOptions(t *testing.T) {

	t.Run("test global options", func(t *testing.T) {
		testCases := []struct {
			name        string
			args        []string
			wantArgs    []string
			wantOptions map[string]string
		}{
			{
				name:     "a short option",
				args:     []string{"kubectl", "-n", "kube-system", "describe"},
				wantArgs: []string{"kubectl", "describe"},
				wantOptions: map[string]string{
					"namespace": "kube-system",
				},
			},
			{
				name:     "a long namespace",
				args:     []string{"kubectl", "describe", "--namespace", "kube-system"},
				wantArgs: []string{"kubectl", "describe"},
				wantOptions: map[string]string{
					"namespace": "kube-system",
				},
			},
			{
				name:        "no option",
				args:        []string{"kubectl", "describe"},
				wantArgs:    []string{"kubectl", "describe"},
				wantOptions: map[string]string{},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				gotArgs, gotOptions := filterOptions(tc.args, kubectloptions.KubeCtlGlobalOptions)
				assert.Equal(t, tc.wantArgs, gotArgs)
				assert.Equal(t, tc.wantOptions, gotOptions)
			})
		}

	})
}
