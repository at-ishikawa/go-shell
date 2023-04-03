package kubectl

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/at-ishikawa/go-shell/internal/completion"
	"github.com/at-ishikawa/go-shell/internal/config"
	"github.com/at-ishikawa/go-shell/internal/plugin"
	"github.com/at-ishikawa/go-shell/internal/plugin/kubectl/kubectloptions"
	"github.com/golang/mock/gomock"

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

func TestKubeCtlPlugin_Suggest(t *testing.T) {
	backupExecCommand := execCommand
	defer func() { execCommand = backupExecCommand }()

	history := config.History{}
	now := time.Now()
	history.Add("kubectl get pods", 0, nil, now)
	history.Add("kubectl describe service -n kube-system", 0, nil, now.Add(time.Second))

	testCases := []struct {
		name string
		args plugin.SuggestArg

		mockExecCommand func(command string, args ...string) ([]byte, error)

		mockCompletion          func(t *testing.T, mockCompletion *completion.MockCompletion)
		mockWantGetResult       []string
		mockWantCompleteOptions completion.CompleteOptions
		mockCompleteTimes       int

		want    []string
		wantErr error
	}{
		{
			name: "no sub command. suggest from a history",
			args: plugin.SuggestArg{
				Args: []string{
					Cli,
					"d",
				},
				CurrentArgToken: "d",
				History:         &history,
			},
			mockExecCommand: func(command string, args ...string) ([]byte, error) {
				return nil, errors.New("shouldn't happen")
			},
			mockCompletion: func(_ *testing.T, mockCompletion *completion.MockCompletion) {
				mockCompletion.EXPECT().
					Complete([]string{
						"get",
						"describe",
					}, completion.CompleteOptions{
						InitialQuery: "d",
					}).
					Return("describe", nil).
					Times(1)
			},
			want: []string{"describe"},
		},
		{
			name: "suggest a single resource in different namespace",
			args: plugin.SuggestArg{
				Args: []string{
					Cli,
					"describe",
					"-n",
					"kube-system",
					"pods",
					"p",
				},
				CurrentArgToken: "p",
			},
			mockExecCommand: func(command string, args ...string) ([]byte, error) {
				want := []string{
					"get",
					"pods",
					"--namespace",
					"kube-system",
				}
				if !reflect.DeepEqual(want, args) {
					return nil, fmt.Errorf("want: %v, got: %v", want, args)
				}

				return []byte("NAME\npod1\npod2\n"), nil
			},
			mockCompletion: func(t *testing.T, mockCompletion *completion.MockCompletion) {
				mockCompletion.EXPECT().
					CompleteMulti(gomock.Any(), gomock.Any()).
					DoAndReturn(func(args []string, options completion.CompleteOptions) ([]string, error) {
						want := []string{
							"pod1",
							"pod2",
						}
						assert.Equal(t, want, args)
						assert.Equal(t, "NAME", options.Header)
						assert.Equal(t, "p", options.InitialQuery)

						backupExecCommand := execCommand
						defer func() {
							execCommand = backupExecCommand
						}()

						runPreview := func(row int, want string) {
							execCommand = func(command string, args ...string) ([]byte, error) {
								want := []string{
									"describe",
									"--namespace",
									"kube-system",
									"pods",
									want,
								}
								assert.Equal(t, want, args)
								return []byte("preview"), nil
							}
							got, gotErr := options.PreviewCommand(row)
							assert.Equal(t, "preview", got)
							assert.NoError(t, gotErr)
						}
						runPreview(0, "pod1")
						runPreview(1, "pod2")

						return []string{"pod2"}, nil
					}).
					Times(1)
			},
			want: []string{"pod2"},
		},
		{
			name: "suggest multiple resources",
			args: plugin.SuggestArg{
				Args: []string{
					Cli,
					"port-forward",
					"--address",
					"localhost",
					"p",
				},
				CurrentArgToken: "p",
			},
			mockExecCommand: func(command string, args ...string) ([]byte, error) {
				want := []string{
					"get",
					"pods,services",
				}
				if !reflect.DeepEqual(want, args) {
					return nil, fmt.Errorf("want: %v, got: %v", want, args)
				}

				return []byte("pod/pod1\nservice/service1\n"), nil
			},
			mockCompletion: func(t *testing.T, mockCompletion *completion.MockCompletion) {
				mockCompletion.EXPECT().
					CompleteMulti(gomock.Any(), gomock.Any()).
					DoAndReturn(func(args []string, options completion.CompleteOptions) ([]string, error) {
						want := []string{
							"pod/pod1",
							"service/service1",
						}
						assert.Equal(t, want, args)
						assert.Equal(t, "p", options.InitialQuery)

						backupExecCommand := execCommand
						defer func() {
							execCommand = backupExecCommand
						}()

						runPreview := func(row int, want string) {
							execCommand = func(command string, args ...string) ([]byte, error) {
								want := []string{
									"describe",
									want,
								}
								assert.Equal(t, want, args)
								return []byte("preview"), nil
							}
							got, gotErr := options.PreviewCommand(row)
							assert.Equal(t, "preview", got)
							assert.NoError(t, gotErr)
						}
						runPreview(0, "pod/pod1")
						runPreview(1, "service/service1")

						return []string{"service/service1"}, nil
					}).
					Times(1)
			},
			want: []string{"service/service1"},
		},
		{
			name: "no kubectl get result",
			args: plugin.SuggestArg{
				Args: []string{
					Cli,
					"rollout",
					"restart",
					"deployment",
					"d",
				},
				CurrentArgToken: "d",
			},
			mockExecCommand: func(command string, args ...string) ([]byte, error) {
				want := []string{
					"get",
					"deployment",
				}
				if !reflect.DeepEqual(want, args) {
					return nil, fmt.Errorf("want: %v, got: %v", want, args)
				}

				return []byte("NAME\n"), nil
			},
			mockCompletion: func(t *testing.T, mockCompletion *completion.MockCompletion) {
				mockCompletion.EXPECT().
					CompleteMulti(gomock.Any(), gomock.Any()).
					Times(0)
			},
		},
		{
			name: "suggest no row",
			args: plugin.SuggestArg{
				Args: []string{
					Cli,
					"logs",
					"-f",
					"p",
				},
				CurrentArgToken: "p",
			},
			mockExecCommand: func(command string, args ...string) ([]byte, error) {
				want := []string{
					"get",
					"pods",
				}
				if !reflect.DeepEqual(want, args) {
					return nil, fmt.Errorf("want: %v, got: %v", want, args)
				}

				return []byte("NAME\npod1\npod2\n"), nil
			},
			mockCompletion: func(t *testing.T, mockCompletion *completion.MockCompletion) {
				mockCompletion.EXPECT().
					CompleteMulti(gomock.Any(), gomock.Any()).
					DoAndReturn(func(args []string, options completion.CompleteOptions) ([]string, error) {
						want := []string{
							"pod1",
							"pod2",
						}
						assert.Equal(t, want, args)
						assert.Equal(t, "p", options.InitialQuery)

						backupExecCommand := execCommand
						defer func() {
							execCommand = backupExecCommand
						}()

						runPreview := func(row int, want string) {
							execCommand = func(command string, args ...string) ([]byte, error) {
								want := []string{
									"describe",
									"pods",
									want,
								}
								assert.Equal(t, want, args)
								return []byte("preview"), nil
							}
							got, gotErr := options.PreviewCommand(row)
							assert.Equal(t, "preview", got)
							assert.NoError(t, gotErr)
						}
						runPreview(0, "pod1")
						runPreview(1, "pod2")

						return nil, nil
					}).
					Times(1)
			},
		},

		{
			name: "error when a kubectl get",
			args: plugin.SuggestArg{
				Args: []string{
					Cli,
					"describe",
					"pods",
					"p",
				},
				CurrentArgToken: "p",
			},
			mockExecCommand: func(command string, args ...string) ([]byte, error) {
				want := []string{
					"get",
					"pods",
				}
				if !reflect.DeepEqual(want, args) {
					return nil, fmt.Errorf("want: %v, got: %v", want, args)
				}

				return nil, errors.New("error")
			},
			mockCompletion: func(t *testing.T, mockCompletion *completion.MockCompletion) {
				mockCompletion.EXPECT().CompleteMulti(gomock.Any(), gomock.Any()).Times(0)
			},
			wantErr: errors.New("error"),
		},
		{
			name: "error when completion and preview",
			args: plugin.SuggestArg{
				Args: []string{
					Cli,
					"logs",
					"-f",
					"p",
				},
				CurrentArgToken: "p",
			},
			mockExecCommand: func(command string, args ...string) ([]byte, error) {
				want := []string{
					"get",
					"pods",
				}
				if !reflect.DeepEqual(want, args) {
					return nil, fmt.Errorf("want: %v, got: %v", want, args)
				}

				return []byte("NAME\npod1\npod2\n"), nil
			},
			mockCompletion: func(t *testing.T, mockCompletion *completion.MockCompletion) {
				mockCompletion.EXPECT().
					CompleteMulti(gomock.Any(), gomock.Any()).
					DoAndReturn(func(args []string, options completion.CompleteOptions) ([]string, error) {
						want := []string{
							"pod1",
							"pod2",
						}
						assert.Equal(t, want, args)
						assert.Equal(t, "p", options.InitialQuery)

						backupExecCommand := execCommand
						defer func() {
							execCommand = backupExecCommand
						}()

						runPreview := func(row int, want string) {
							execCommand = func(command string, args ...string) ([]byte, error) {
								want := []string{
									"describe",
									"pods",
									want,
								}
								assert.Equal(t, want, args)
								return nil, errors.New("preview error")
							}
							got, gotErr := options.PreviewCommand(row)
							assert.Empty(t, got)
							assert.Equal(t, gotErr, errors.New("preview error"))
						}
						runPreview(0, "pod1")
						runPreview(1, "pod2")

						return nil, errors.New("completion error")
					}).
					Times(1)
			},
			wantErr: errors.New("completion error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockController := gomock.NewController(t)
			defer mockController.Finish()

			mockCompletion := completion.NewMockCompletion(mockController)
			tc.mockCompletion(t, mockCompletion)

			execCommand = tc.mockExecCommand

			kubeCtlPlugin := KubeCtlPlugin{
				completionUi: mockCompletion,
			}
			got, gotErr := kubeCtlPlugin.Suggest(tc.args)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}
