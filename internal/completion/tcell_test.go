package completion

import (
	"errors"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func TestFinder_getVisibleRows(t *testing.T) {
	testCases := []struct {
		name string
		rows []finderRow
		want []finderRow
	}{
		{
			name: "All Rows Visible",
			rows: []finderRow{
				{visible: true, index: 0, value: "apple"},
				{visible: true, index: 1, value: "banana"},
			},
			want: []finderRow{
				{visible: true, index: 0, value: "apple"},
				{visible: true, index: 1, value: "banana"},
			},
		},
		{
			name: "Some Rows Hidden",
			rows: []finderRow{
				{visible: true, index: 0, value: "apple"},
				{visible: false, index: 1, value: "banana"},
				{visible: true, index: 2, value: "orange"},
				{visible: false, index: 3, value: "peach"},
			},
			want: []finderRow{
				{visible: true, index: 0, value: "apple"},
				{visible: true, index: 2, value: "orange"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := finder{
				allRows: tc.rows,
			}
			assert.Equal(t, tc.want, f.getVisibleRows())
		})
	}
}

func TestFinder_updateQuery(t *testing.T) {
	testCases := []struct {
		name  string
		rows  []finderRow
		query string
		want  finder
	}{
		{
			name: "empty query",
			rows: []finderRow{
				{visible: true, index: 0, value: "apple"},
				{visible: false, index: 1, value: "banana"},
			},
			want: finder{
				allRows: []finderRow{
					{visible: true, index: 0, value: "apple"},
					{visible: true, index: 1, value: "banana"},
				},
			},
		},
		{
			name: "query matches some rows",
			rows: []finderRow{
				{visible: true, index: 0, value: "apple"},
				{visible: true, index: 1, value: "banana"},
			},
			query: "ap",
			want: finder{
				allRows: []finderRow{
					{visible: true, index: 0, value: "apple"},
					{visible: false, index: 1, value: "banana"},
				},
				query: "ap",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := finder{
				allRows: tc.rows,
			}
			f.updateQuery(tc.query)
			assert.Equal(t, tc.want, f)
		})
	}
}

func TestTcellCompletion_complete(t *testing.T) {
	var emptyRune rune
	enter := tcell.KeyEnter

	type args struct {
		rows              []string
		options           CompleteOptions
		isMultiSelectMode bool
	}
	testCases := []struct {
		name      string
		args      args
		keyEvents func(screen tcell.SimulationScreen)

		want    []string
		wantErr error
	}{
		{
			name: "single selct: return the first item without no input",
			args: args{
				rows: []string{"apple", "banana"},
				options: CompleteOptions{
					PreviewCommand: func(row int) (string, error) {
						return "preview", nil
					},
				},
			},
			keyEvents: func(screen tcell.SimulationScreen) {
				screen.InjectKey(enter, emptyRune, tcell.ModNone)
			},
			want: []string{"apple"},
		},
		{
			name: "single selct: filter by query with an initial query",
			args: args{
				rows:    []string{"apple", "banana"},
				options: CompleteOptions{InitialQuery: "a"},
			},
			keyEvents: func(screen tcell.SimulationScreen) {
				screen.InjectKeyBytes([]byte("n"))
				screen.InjectKey(enter, emptyRune, tcell.ModNone)
			},
			want: []string{"banana"},
		},
		{
			name: "multiple selct: return mutliple items",
			args: args{
				rows:              []string{"apple", "banana"},
				isMultiSelectMode: true,
			},
			keyEvents: func(screen tcell.SimulationScreen) {
				screen.InjectKey(tcell.KeyTAB, emptyRune, tcell.ModNone)
				screen.InjectKey(tcell.KeyTAB, emptyRune, tcell.ModNone)
				screen.InjectKey(enter, emptyRune, tcell.ModNone)
			},
			want: []string{"apple", "banana"},
		},
		{
			name: "live reloading causes an error",
			args: args{
				rows: []string{"apple"},
				options: CompleteOptions{
					LiveReloading: func(row int, query string) ([]string, error) {
						return nil, errors.New("error")
					},
				},
			},
			keyEvents: func(screen tcell.SimulationScreen) {
				screen.InjectKey(tcell.KeyTab, emptyRune, tcell.ModNone)
			},
			wantErr: errors.Join(errors.New("error")),
		},
		{
			name: "return nil when no matches",
			args: args{
				rows: []string{"apple"},
				options: CompleteOptions{
					InitialQuery: "dog",
				},
			},
			keyEvents: func(screen tcell.SimulationScreen) {
				screen.InjectKey(enter, emptyRune, tcell.ModNone)
			},
		},
		{
			name: "no rows",
			args: args{
				rows: []string{},
			},
		},
		{
			name: "a row with empty string",
			args: args{
				rows: []string{""},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			screen := tcell.NewSimulationScreen("")
			screen.Init()
			defer screen.Fini()

			complete := &TcellCompletion{
				logger: zap.NewNop(),
			}

			eg := errgroup.Group{}
			eg.Go(func() error {
				got, err := complete.complete(screen, tc.args.rows, tc.args.options, tc.args.isMultiSelectMode)
				assert.Equal(t, tc.want, got)
				return err
			})

			if tc.keyEvents != nil {
				tc.keyEvents(screen)
			}

			gotErr := eg.Wait()
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}

func TestComplete_handleKeyEvent(t *testing.T) {
	var emptyRune rune
	enterKeyEvent := tcell.NewEventKey(tcell.KeyEnter, emptyRune, tcell.ModNone)
	type args struct {
		keyEvent tcell.Event
		finder   finder
	}

	t.Run("common behaviors in select modes", func(t *testing.T) {
		testCases := []struct {
			name     string
			args     args
			want     finder
			wantErr  error
			wantDone bool
		}{
			{
				name: "add new query, filter one row",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: true, index: 0, value: "apple"},
							{visible: true, index: 1, value: "banana"},
						},
						cursorRow: 1,
					},
					keyEvent: tcell.NewEventKey(tcell.KeyRune, 'b', tcell.ModNone),
				},
				want: finder{
					allRows: []finderRow{
						{visible: false, index: 0, value: "apple"},
						{visible: true, index: 1, value: "banana"},
					},
					query: "b",
				},
			},
			{
				name: "add one letter to a query, filter all rows",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: true, index: 0, value: "apple"},
						},
						query: "p",
					},
					keyEvent: tcell.NewEventKey(tcell.KeyRune, 'b', tcell.ModNone),
				},
				want: finder{
					allRows: []finderRow{
						{visible: false, index: 0, value: "apple"},
					},
					query: "pb",
				},
			},
			{
				name: "delete one letter from a query, show an row",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{index: 0, value: "apple"},
							{selected: true, index: 0, value: "banana"},
						},
						query: "pb",
					},
					keyEvent: tcell.NewEventKey(tcell.KeyBackspace, emptyRune, tcell.ModNone),
				},
				want: finder{
					allRows: []finderRow{
						{visible: true, index: 0, value: "apple"},
						{selected: true, index: 0, value: "banana"},
					},
					query: "p",
				},
			},
			{
				name: "delete letters from a query, show all rows",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{index: 0, value: "apple"},
						},
						query: "z",
					},
					keyEvent: tcell.NewEventKey(tcell.KeyBackspace, emptyRune, tcell.ModNone),
				},
				want: finder{
					allRows: []finderRow{
						{visible: true, index: 0, value: "apple"},
					},
				},
			},

			{
				name: "enter when an item",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: true, index: 0, value: "apple"},
						},
					},
					keyEvent: enterKeyEvent,
				},
				want: finder{
					allRows: []finderRow{
						{visible: true, selected: true, index: 0, value: "apple"},
					},
				},
				wantDone: true,
			},
			{
				name: "enter when no item",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: false, index: 0, value: "apple"},
						},
						query: "z",
					},
					keyEvent: enterKeyEvent,
				},
				want: finder{
					allRows: []finderRow{
						{visible: false, index: 0, value: "apple"},
					},
					query: "z",
				},
				wantDone: true,
			},

			{
				name: "pressing tab key doesn't change anything",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: true, index: 0, value: "apple"},
						},
					},
					keyEvent: tcell.NewEventKey(tcell.KeyTab, emptyRune, tcell.ModNone),
				},
				want: finder{
					allRows: []finderRow{
						{visible: true, index: 0, value: "apple"},
					},
				},
			},
			{
				name: "move cursor up",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: true, index: 0, value: "apple"},
							{visible: true, index: 1, value: "banana"},
						},
						cursorRow: 1,
					},
					keyEvent: tcell.NewEventKey(tcell.KeyCtrlP, emptyRune, tcell.ModCtrl),
				},
				want: finder{
					allRows: []finderRow{
						{visible: true, index: 0, value: "apple"},
						{visible: true, index: 1, value: "banana"},
					},
				},
			},
			{
				name: "move cursor up if it's already on top",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: false, index: 0, value: "apple"},
							{visible: true, index: 1, value: "banana"},
						},
					},
					keyEvent: tcell.NewEventKey(tcell.KeyCtrlP, emptyRune, tcell.ModCtrl),
				},
				want: finder{
					allRows: []finderRow{
						{visible: false, index: 0, value: "apple"},
						{visible: true, index: 1, value: "banana"},
					},
				},
			},

			{
				name: "move cursor down",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: true, index: 0, value: "apple"},
							{visible: true, index: 1, value: "banana"},
						},
					},
					keyEvent: tcell.NewEventKey(tcell.KeyCtrlN, emptyRune, tcell.ModCtrl),
				},
				want: finder{
					allRows: []finderRow{
						{visible: true, index: 0, value: "apple"},
						{visible: true, index: 1, value: "banana"},
					},
					cursorRow: 1,
				},
			},
			{
				name: "move cursor down if it's already on the bottom",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: false, index: 0, value: "apple"},
							{visible: true, index: 1, value: "banana"},
						},
					},
					keyEvent: tcell.NewEventKey(tcell.KeyCtrlN, emptyRune, tcell.ModCtrl),
				},
				want: finder{
					allRows: []finderRow{
						{visible: false, index: 0, value: "apple"},
						{visible: true, index: 1, value: "banana"},
					},
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				c := &TcellCompletion{
					logger: zap.NewNop(),
				}
				got, gotErr, gotDone := c.handleKeyEvent(tc.args.finder, tc.args.keyEvent.(*tcell.EventKey))
				assert.Equal(t, tc.want, got)
				assert.Equal(t, tc.wantErr, gotErr)
				assert.Equal(t, tc.wantDone, gotDone)
			})
		}
	})

	t.Run("single select mode", func(t *testing.T) {
		testCases := []struct {
			name     string
			args     args
			want     finder
			wantErr  error
			wantDone bool
		}{
			{
				name: "pressing tab key doesn't change anything if no live reloading",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: true, index: 0, value: "apple"},
						},
					},
					keyEvent: tcell.NewEventKey(tcell.KeyTab, emptyRune, tcell.ModNone),
				},
				want: finder{
					allRows: []finderRow{
						{visible: true, index: 0, value: "apple"},
					},
				},
			},
			{
				name: "live reloading updates all rows",
				args: args{
					finder: finder{
						query: "ap",
						allRows: []finderRow{
							{visible: true, index: 0, value: "apple"},
						},
						liveReloading: func(row int, query string) ([]string, error) {
							return []string{"dog", "cat"}, nil
						},
					},
					keyEvent: tcell.NewEventKey(tcell.KeyTab, emptyRune, tcell.ModNone),
				},
				want: finder{
					// query: "ap",
					allRows: []finderRow{
						{visible: true, index: 0, value: "dog"},
						{visible: true, index: 1, value: "cat"},
					},
				},
			},
			{
				name: "live reloading doesn't show any rows anymore",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: true, index: 0, value: "apple"},
						},
						liveReloading: func(row int, query string) ([]string, error) {
							return nil, nil
						},
					},
					keyEvent: tcell.NewEventKey(tcell.KeyTab, emptyRune, tcell.ModNone),
				},
				want: finder{
					allRows: []finderRow{
						{visible: true, selected: true, index: 0, value: "apple"},
					},
				},
			},
			{
				name: "live reloading causes an error",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: true, index: 0, value: "apple"},
						},
						liveReloading: func(row int, query string) ([]string, error) {
							return nil, errors.New("error")
						},
					},
					keyEvent: tcell.NewEventKey(tcell.KeyTab, emptyRune, tcell.ModNone),
				},
				want: finder{
					allRows: []finderRow{
						{visible: true, index: 0, value: "apple"},
					},
				},
				wantErr:  errors.New("error"),
				wantDone: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				c := &TcellCompletion{
					logger: zap.NewNop(),
				}
				got, gotErr, gotDone := c.handleKeyEvent(tc.args.finder, tc.args.keyEvent.(*tcell.EventKey))

				// Set nil because functions cannot be compared: https://github.com/stretchr/testify/issues/182
				got.liveReloading = nil
				tc.want.liveReloading = nil

				assert.Equal(t, tc.want, got)
				assert.Equal(t, tc.wantErr, gotErr)
				assert.Equal(t, tc.wantDone, gotDone)
			})
		}
	})

	t.Run("multi select mode", func(t *testing.T) {
		testCases := []struct {
			name     string
			args     args
			want     finder
			wantErr  error
			wantDone bool
		}{
			{
				name: "pressing tab key on mutliple items",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: true, index: 0, value: "apple"},
							{visible: true, index: 1, value: "banana"},
						},
						liveReloading: func(row int, query string) ([]string, error) {
							panic("shouldn't be called")
						},
					},
					keyEvent: tcell.NewEventKey(tcell.KeyTab, emptyRune, tcell.ModNone),
				},
				want: finder{
					allRows: []finderRow{
						{visible: true, selected: true, index: 0, value: "apple"},
						{visible: true, index: 1, value: "banana"},
					},
					cursorRow: 1,
				},
			},
			{
				name: "pressing tab key on a single item",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: true, index: 0, value: "apple"},
						},
					},
					keyEvent: tcell.NewEventKey(tcell.KeyTab, emptyRune, tcell.ModNone),
				},
				want: finder{
					allRows: []finderRow{
						{visible: true, selected: true, index: 0, value: "apple"},
					},
					cursorRow: 0,
				},
			},
			{
				name: "pressing tab key if there are a visible and invisible item",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: false, index: 0, value: "apple"},
							{visible: true, index: 1, value: "banana"},
						},
					},
					keyEvent: tcell.NewEventKey(tcell.KeyTab, emptyRune, tcell.ModNone),
				},
				want: finder{
					allRows: []finderRow{
						{visible: false, index: 0, value: "apple"},
						{visible: true, selected: true, index: 1, value: "banana"},
					},
					cursorRow: 0,
				},
			},
			{
				name: "pressing tab key if no available item",
				args: args{
					finder: finder{
						allRows: []finderRow{
							{visible: false, index: 0, value: "apple"},
						},
						query: "b",
					},
					keyEvent: tcell.NewEventKey(tcell.KeyTab, emptyRune, tcell.ModNone),
				},
				want: finder{
					allRows: []finderRow{
						{visible: false, index: 0, value: "apple"},
					},
					query: "b",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				c := &TcellCompletion{
					logger: zap.NewNop(),
				}

				tc.args.finder.isMultiSelectMode = true
				tc.want.isMultiSelectMode = true
				got, gotErr, gotDone := c.handleKeyEvent(tc.args.finder, tc.args.keyEvent.(*tcell.EventKey))

				// Set nil because functions cannot be compared: https://github.com/stretchr/testify/issues/182
				got.liveReloading = nil
				tc.want.liveReloading = nil
				assert.Equal(t, tc.want, got)
				assert.Equal(t, tc.wantErr, gotErr)
				assert.Equal(t, tc.wantDone, gotDone)
			})
		}
	})
}
