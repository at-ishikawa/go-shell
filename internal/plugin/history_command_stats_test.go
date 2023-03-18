package plugin

import (
	"testing"

	"github.com/at-ishikawa/go-shell/internal/config"

	"github.com/stretchr/testify/assert"
)

func Test_getCommandStats(t *testing.T) {
	testCases := []struct {
		name        string
		historyList []config.HistoryItem
		want        HistoryCommandStats
	}{
		{
			name: "analyze command stats",
			historyList: []config.HistoryItem{
				{
					Command: "command --global-option-no-value --global-option value subcommand -o --subcommand-option-with-value option_value",
				},
				{
					Command: "command subcommand option_value --global-option value",
				},
				{
					Command: "ls",
				},
				{
					Command: "failed command",
					Status:  1,
				},
			},
			want: HistoryCommandStats{
				"command": {
					count: 2,
					options: map[string]optionStats{
						"--global-option": {
							values: map[string]int{
								"value": 1,
							},
						},
						"--global-option-no-value": {
							noValue: 1,
							values:  map[string]int{},
						},
					},
					args: map[string]commandStats{
						"subcommand": {
							count: 2,
							options: map[string]optionStats{
								"-o": {
									noValue: 1,
									values:  map[string]int{},
								},
								"--subcommand-option-with-value": {
									values: map[string]int{
										"option_value": 1,
									},
								},
							},
							args: map[string]commandStats{
								"option_value": {
									count: 1,
									options: map[string]optionStats{
										"--global-option": {
											values: map[string]int{
												"value": 1,
											},
										},
									},
									args: map[string]commandStats{},
								},
							},
						},
					},
				},
				"ls": {
					count:   1,
					options: map[string]optionStats{},
					args:    map[string]commandStats{},
				},
			},
		},
		{
			name: "no command history",
			want: HistoryCommandStats{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, getHistoryCommandStats(tc.historyList))
		})
	}
}

func TestHistoryCommandStats_getSuggestedValues(t *testing.T) {
	testCases := []struct {
		name                string
		args                []string
		currentToken        string
		historyCommandStats HistoryCommandStats
		want                []string
	}{
		{
			name:         "new command",
			args:         []string{},
			currentToken: "",
			historyCommandStats: HistoryCommandStats{
				"command": {
					count:   2,
					options: map[string]optionStats{},
					args:    map[string]commandStats{},
				},
				"ls": {
					count:   1,
					options: map[string]optionStats{},
					args:    map[string]commandStats{},
				},
			},
			want: []string{
				"command",
				"ls",
			},
		},
		{
			name:         "new argument",
			args:         []string{"command"},
			currentToken: "",
			historyCommandStats: HistoryCommandStats{
				"command": {
					count: 2,
					options: map[string]optionStats{
						"--global-option": {
							values: map[string]int{
								"value": 1,
							},
						},
						"--global-option-no-value": {
							noValue: 1,
							values:  map[string]int{},
						},
					},
					args: map[string]commandStats{
						"subcommand": {
							count:   2,
							options: map[string]optionStats{},
							args:    map[string]commandStats{},
						},
					},
				},
			},
			want: []string{
				"--global-option",
				"--global-option-no-value",
				"subcommand",
			},
		},
		{
			name:         "new option value for a required option",
			args:         []string{"command", "--global-option2", "--global-option"},
			currentToken: "",
			historyCommandStats: HistoryCommandStats{
				"command": {
					count: 2,
					options: map[string]optionStats{
						"--global-option": {
							noValue: 0,
							values: map[string]int{
								"value": 1,
							},
						},
						"--global-option2": {
							noValue: 2,
							values:  map[string]int{},
						},
						"--global-option-no-value": {
							noValue: 1,
							values:  map[string]int{},
						},
					},
					args: map[string]commandStats{
						"subcommand": {
							count:   2,
							options: map[string]optionStats{},
							args:    map[string]commandStats{},
						},
					},
				},
			},
			want: []string{
				"value",
			},
		},
		{
			name:         "new option value for an optional option",
			args:         []string{"command", "--global-option2", "--global-optional-value"},
			currentToken: "",
			historyCommandStats: HistoryCommandStats{
				"command": {
					count: 2,
					options: map[string]optionStats{
						"--global-option": {
							noValue: 0,
							values: map[string]int{
								"value": 1,
							},
						},
						"--global-option2": {
							noValue: 2,
							values:  map[string]int{},
						},
						"--global-optional-value": {
							noValue: 1,
							values: map[string]int{
								"optional-value": 1,
							},
						},
					},
					args: map[string]commandStats{
						"subcommand": {
							count:   2,
							options: map[string]optionStats{},
							args:    map[string]commandStats{},
						},
					},
				},
			},
			want: []string{
				"optional-value",
				"--global-option",
				"subcommand",
			},
		},
		{
			name:         "new subcommand",
			args:         []string{"newcommand"},
			currentToken: "",
			historyCommandStats: HistoryCommandStats{
				"command": {
					count: 2,
					options: map[string]optionStats{
						"--global-option": {
							values: map[string]int{},
						},
					},
					args: map[string]commandStats{
						"subcommand": {
							count:   2,
							options: map[string]optionStats{},
							args:    map[string]commandStats{},
						},
					},
				},
			},
			want: []string{},
		},
		{
			name:         "new option",
			args:         []string{"command", "--new-option"},
			currentToken: "",
			historyCommandStats: HistoryCommandStats{
				"command": {
					count: 2,
					options: map[string]optionStats{
						"--global-option": {
							values: map[string]int{},
						},
					},
					args: map[string]commandStats{
						"subcommand": {
							count:   2,
							options: map[string]optionStats{},
							args:    map[string]commandStats{},
						},
					},
				},
			},
			want: []string{
				"--global-option",
				"subcommand",
			},
		},
		{
			name:         "during subcommand input",
			args:         []string{"command", "sub"},
			currentToken: "sub",
			historyCommandStats: HistoryCommandStats{
				"command": {
					count: 2,
					options: map[string]optionStats{
						"--global-option": {
							values: map[string]int{},
						},
					},
					args: map[string]commandStats{
						"subcommand": {
							count:   2,
							options: map[string]optionStats{},
							args:    map[string]commandStats{},
						},
					},
				},
			},
			want: []string{
				"--global-option",
				"subcommand",
			},
		},
		{
			name:         "during option input",
			args:         []string{"command", "--global-"},
			currentToken: "--global-",
			historyCommandStats: HistoryCommandStats{
				"command": {
					count: 2,
					options: map[string]optionStats{
						"--global-option": {
							values: map[string]int{},
						},
					},
					args: map[string]commandStats{
						"subcommand": {
							count:   2,
							options: map[string]optionStats{},
							args:    map[string]commandStats{},
						},
					},
				},
			},
			want: []string{
				"--global-option",
				"subcommand",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.historyCommandStats.getSuggestedValues(tc.args, tc.currentToken))
		})
	}
}
