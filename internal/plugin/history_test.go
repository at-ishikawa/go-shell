package plugin

import (
	"testing"
	"time"

	"github.com/at-ishikawa/go-shell/internal/config"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHistoryPlugin_filterHistoryList(t *testing.T) {
	lastSucceededAt := time.Now()
	type historyPluginFactory func(mockController *gomock.Controller) HistoryPlugin

	testCases := []struct {
		name          string
		pluginFactory historyPluginFactory
		historyList   []config.HistoryItem
		query         string
		want          []config.HistoryItem
	}{
		{
			name: "multiple commands",
			pluginFactory: func(mockController *gomock.Controller) HistoryPlugin {
				mockPlugin := NewMockPlugin(mockController)
				mockPlugin.EXPECT().Command().Return("mock").Times(1)
				mockPlugin.EXPECT().GetContext(gomock.Any()).Return(map[string]string{
					"key":  "value",
					"key2": "value2",
				}, nil).Times(1)
				return HistoryPlugin{
					plugins: map[string]Plugin{
						"mock": mockPlugin,
					},
				}
			},
			historyList: []config.HistoryItem{
				{Command: "no success command"},
				{Command: "mock with the same part of the context", LastSucceededAt: lastSucceededAt, Context: map[string]string{
					"key": "value",
				}},
				{Command: "mock with the same context", LastSucceededAt: lastSucceededAt, Context: map[string]string{
					"key":  "value",
					"key2": "value2",
				}},
				{Command: "mock with the different context", LastSucceededAt: lastSucceededAt, Context: map[string]string{
					"key":  "value",
					"key2": "different values",
				}},
			},
			want: []config.HistoryItem{
				{Command: "mock with the same context", LastSucceededAt: lastSucceededAt, Context: map[string]string{
					"key":  "value",
					"key2": "value2",
				}},
			},
		},

		{
			name: "no command",
			pluginFactory: func(mockController *gomock.Controller) HistoryPlugin {
				mockPlugin := NewMockPlugin(mockController)
				mockPlugin.EXPECT().Command().Return("mock").Times(1)
				mockPlugin.EXPECT().GetContext(gomock.Any()).Return(map[string]string{
					"key": "value",
				}, nil).Times(1)
				return HistoryPlugin{
					plugins: map[string]Plugin{
						"mock": mockPlugin,
					},
				}
			},
		},

		{
			name: "no plugin",
			pluginFactory: func(mockController *gomock.Controller) HistoryPlugin {
				return HistoryPlugin{}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockController := gomock.NewController(t)
			hp := tc.pluginFactory(mockController)
			assert.Equal(t, tc.want, hp.filterHistoryList(tc.historyList, "query"))
		})
	}
}
