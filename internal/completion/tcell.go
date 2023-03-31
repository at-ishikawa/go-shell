package completion

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/at-ishikawa/go-shell/internal/ansi"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type finder struct {
	header  string
	allRows []finderRow

	// the index of all rows is getVisibleRows()[cursorRow].index
	cursorRow int

	query             string
	previewCommand    PreviewCommandType
	liveReloading     LiveReloading
	isMultiSelectMode bool
}

func (f *finder) setRows(rows []string) {
	allRows := make([]finderRow, 0, len(rows))
	for index, r := range rows {
		allRows = append(allRows, finderRow{
			visible: true,
			index:   index,
			value:   r,
		})
	}
	f.allRows = allRows

	if f.cursorRow >= len(f.getVisibleRows()) {
		f.cursorRow = len(f.getVisibleRows()) - 1
	}
}

func newFinder(rows []string, options CompleteOptions, isMultiSelectMode bool) finder {
	f := finder{
		header: options.Header,

		query:             options.InitialQuery,
		previewCommand:    options.PreviewCommand,
		liveReloading:     options.LiveReloading,
		isMultiSelectMode: isMultiSelectMode,
	}
	f.setRows(rows)
	f.updateQuery(options.InitialQuery)
	return f
}

type finderRow struct {
	visible  bool
	selected bool
	index    int
	value    string
}

func (f finder) getVisibleRows() []finderRow {
	result := []finderRow{}
	for _, row := range f.allRows {
		if !row.visible {
			continue
		}
		result = append(result, row)
	}
	return result
}

func (f *finder) updateQuery(query string) {
	f.query = query
	for i, row := range f.allRows {
		if query != "" && !strings.Contains(row.value, query) {
			f.allRows[i].visible = false
			continue
		}
		f.allRows[i].visible = true
	}
}

type ansiString ansi.AnsiString

func (as ansiString) ToTCellStyle() tcell.Style {
	style := tcell.StyleDefault
	if as.Style == ansi.StyleBold {
		style = style.Bold(true)
	}
	if as.Style == ansi.StyleUnderline {
		style = style.Underline(true)
	}

	colorMaps := map[ansi.Color]tcell.Color{
		ansi.ColorBlack:  tcell.ColorBlack,
		ansi.ColorRed:    tcell.ColorRed,
		ansi.ColorGreen:  tcell.ColorGreen,
		ansi.ColorYellow: tcell.ColorYellow,
		ansi.ColorBlue:   tcell.ColorBlue,
		ansi.ColorPurple: tcell.ColorPurple,
		// too dark or too light for a cyan
		ansi.ColorCyan:  tcell.ColorLightSkyBlue,
		ansi.ColorWhite: tcell.ColorWhite,
	}
	if color, ok := colorMaps[as.ForegroundColor]; ok {
		style = style.Foreground(color)
	}
	if color, ok := colorMaps[as.BackgroundColor]; ok {
		style = style.Background(color)
	}

	return style
}

type TcellCompletion struct {
	logger *zap.Logger
}

var _ Completion = (*TcellCompletion)(nil)

func NewTcellCompletion() (*TcellCompletion, error) {
	return &TcellCompletion{
		logger: zap.L(),
	}, nil
}

func (complete *TcellCompletion) CompleteMulti(rows []string, options CompleteOptions) ([]string, error) {
	if options.LiveReloading != nil {
		panic("not implemented")
	}

	return complete.start(rows, options, true)
}

func (complete TcellCompletion) Complete(rows []string, options CompleteOptions) (string, error) {
	result, err := complete.start(rows, options, false)
	if err != nil {
		return "", fmt.Errorf("failed TcellCompletion.complete(): %w", err)
	}
	if len(result) == 0 {
		return "", nil
	}
	return result[0], nil
}

func (complete TcellCompletion) start(rows []string, options CompleteOptions, isMultiSelectMode bool) ([]string, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("failed tcell.NewScreen: %w", err)
	}
	if err := screen.Init(); err != nil {
		return nil, fmt.Errorf("failed screen.Init(): %w", err)
	}
	defer screen.Fini()

	return complete.complete(screen, rows, options, isMultiSelectMode)
}

func emitStr(s tcell.Screen, x, y int, style tcell.Style, str string) int {
	for _, char := range str {
		var combinings []rune
		runeWidth := runewidth.RuneWidth(char)
		if runeWidth == 0 {
			// complete.logger.Debug("runeWidth = 0",
			// 	zap.String("char", string(char)))

			// \t
			if char == '\t' {
				runeWidth = 4
			} else {
				// combinings = []rune{char}
				// char = ' '
				runeWidth = 1
			}
		}
		s.SetContent(x, y, char, combinings, style)
		x += runeWidth
	}
	return x
}

func (f finder) runPreview() ([]string, error) {
	if f.previewCommand == nil {
		return nil, nil
	}
	visibleRows := f.getVisibleRows()
	if len(visibleRows) == 0 {
		return nil, nil
	}
	if f.cursorRow >= len(visibleRows) {
		return nil, nil
	}
	if f.cursorRow < 0 {
		return nil, nil
	}
	previewResult, err := f.previewCommand(visibleRows[f.cursorRow].index)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(previewResult, "\n")
	return lines, nil
}

func (complete *tcellRenderer) renderPreviewResult(ctx context.Context, screen tcell.Screen, lines []string, err error) {
	_, height := screen.Size()
	if err != nil {
		emitStr(screen, 2, height/2, tcell.StyleDefault, err.Error())
		return
	}

	for i, line := range lines {
		select {
		case <-ctx.Done():
			return
		default:
			break
		}

		y := height/2 + i
		if y > height {
			break
		}

		ansiStrs := ansi.ParseString(line)
		x := 2
		for _, ansiStr := range ansiStrs {
			style := ansiString(ansiStr).ToTCellStyle()
			x = emitStr(screen, x, y, style, ansiStr.String)
		}
	}
	screen.Show()
}

func (complete *tcellRenderer) showVisibleRows(screen tcell.Screen, currentFinder finder) {
	width, _ := screen.Size()
	prompt := fmt.Sprintf("> %s", currentFinder.query)
	currentX := emitStr(screen, 0, 0, tcell.StyleDefault, prompt)
	screen.ShowCursor(currentX, 0)

	header := currentFinder.header
	if len(header) > 0 {
		emitStr(screen, 2, 1, tcell.StyleDefault, header)
	} else {
		emitStr(screen, 0, 1, tcell.StyleDefault, strings.Repeat("-", width))
	}
	showY := 2

	visibleRows := currentFinder.getVisibleRows()
	for rowIndex := 0; rowIndex < len(visibleRows); rowIndex++ {
		row := visibleRows[rowIndex]

		style := tcell.StyleDefault
		if currentFinder.cursorRow == rowIndex {
			style = tcell.StyleDefault.Foreground(tcell.ColorCadetBlue.TrueColor()).Background(tcell.ColorWhite)
		}
		if row.selected {
			screen.SetCell(0, showY, style, '>', ' ')
		}

		emitStr(screen, 2, showY, style, fmt.Sprintf("%s", row.value))
		showY++
	}

	screen.Show()
}

type tcellRenderer struct {
	screen    tcell.Screen
	isDone    atomic.Bool
	isUpdated atomic.Bool
	mutex     sync.Mutex
	latestJob finder
}

func newRenderer(screen tcell.Screen) *tcellRenderer {
	return &tcellRenderer{
		screen: screen,
	}
}

func (renderer *tcellRenderer) finish() {
	renderer.isDone.Store(true)
}

func (renderer *tcellRenderer) requestNewRenderer(latestJob finder) {
	renderer.mutex.Lock()
	renderer.latestJob = latestJob
	renderer.mutex.Unlock()
	renderer.isUpdated.Store(true)
}

func (renderer *tcellRenderer) start(ctx context.Context) func() error {
	screen := renderer.screen
	screen.SetCursorStyle(tcell.CursorStyleDefault)

	return func() error {
		childEg, childEgCtx := errgroup.WithContext(ctx)
		childCtx, cancel := context.WithCancel(childEgCtx)

		for {
			if renderer.isDone.Load() {
				break
			}
			if renderer.isUpdated.Load() {
				renderer.isUpdated.Store(false)
				renderer.mutex.Lock()
				finder := renderer.latestJob
				renderer.mutex.Unlock()

				// cancel previous rendering
				cancel()
				childCtx, cancel = context.WithCancel(childEgCtx)

				screen.Sync()
				screen.Clear()
				renderer.showVisibleRows(screen, finder)
				childEg.Go(func() error {
					lines, err := finder.runPreview()
					renderer.renderPreviewResult(childCtx, screen, lines, err)
					return nil
				})
			}
			time.Sleep(50 * time.Millisecond)
		}
		cancel()
		return childEg.Wait()
	}
}

func (complete TcellCompletion) complete(screen tcell.Screen, rows []string, options CompleteOptions, isMultiSelectMode bool) ([]string, error) {
	rows = func(rows []string) []string {
		result := make([]string, 0, len(rows))
		for _, row := range rows {
			if strings.TrimSpace(row) == "" {
				continue
			}
			result = append(result, row)
		}
		return result
	}(rows)
	if len(rows) == 0 {
		screen.Beep()
		return nil, nil
	}

	currentFinder := newFinder(rows, options, isMultiSelectMode)
	renderer := newRenderer(screen)
	renderer.requestNewRenderer(currentFinder)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(renderer.start(egCtx))

	var keyEventErr error
loop:
	for {
		switch event := screen.PollEvent().(type) {
		case *tcell.EventKey:
			f, err, done := complete.handleKeyEvent(currentFinder, event)
			if err != nil {
				keyEventErr = err
				break loop
			}
			if done {
				break loop
			}

			currentFinder = f
			renderer.requestNewRenderer(f)
		default:
		}
	}
	renderer.finish()
	if egErr := eg.Wait(); keyEventErr != nil || egErr != nil {
		return nil, errors.Join(keyEventErr, egErr)
	}

	result := make([]string, 0, len(currentFinder.allRows))
	for _, row := range currentFinder.allRows {
		if !row.selected {
			continue
		}
		result = append(result, row.value)
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func (complete TcellCompletion) handleKeyEvent(currentFinder finder, event *tcell.EventKey) (finder, error, bool) {
	var done bool

	visibleRows := currentFinder.getVisibleRows()
	cursorRow := currentFinder.cursorRow
	allRows := currentFinder.allRows
	query := currentFinder.query

	switch event.Key() {
	case tcell.KeyTAB:
		if !currentFinder.isMultiSelectMode && currentFinder.liveReloading == nil {
			// disable a tab key for a single selection mode
			break
		}
		if cursorRow >= len(visibleRows) {
			break
		}

		index := visibleRows[cursorRow].index
		if currentFinder.isMultiSelectMode {
			allRows[index].selected = !allRows[index].selected
			currentFinder.allRows = allRows
			if cursorRow < len(visibleRows)-1 {
				currentFinder.cursorRow++
			}
			break
		}

		if currentFinder.liveReloading != nil {
			rows, err := currentFinder.liveReloading(index, visibleRows[cursorRow].value)
			if err != nil {
				return currentFinder, err, true
			}
			if len(rows) > 0 {
				currentFinder.setRows(rows)
				currentFinder.updateQuery("")
			} else {
				index := visibleRows[cursorRow].index
				allRows[index].selected = !allRows[index].selected
			}
			break
		}

	case tcell.KeyCtrlC:
		done = true

	case tcell.KeyEnter:
		if cursorRow < len(visibleRows) {
			index := visibleRows[cursorRow].index
			allRows[index].selected = true
			currentFinder.allRows = allRows
		}
		done = true

	case tcell.KeyCtrlP:
		if cursorRow > 0 {
			currentFinder.cursorRow--
		}
	case tcell.KeyCtrlN:
		if cursorRow < len(visibleRows)-1 {
			currentFinder.cursorRow++
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyRune:
		if event.Key() == tcell.KeyBackspace ||
			event.Key() == tcell.KeyBackspace2 {
			if len(query) > 0 {
				query = query[:len(query)-1]
			}
		} else {
			ch := event.Rune()
			query = query + string(ch)
		}

		currentFinder.updateQuery(query)
		visibleRows = currentFinder.getVisibleRows()
		complete.logger.Debug("query was changed",
			zap.String("path", "internal/completion/tcell"),
			zap.String("query", query),
			zap.Any("allRows", allRows),
			zap.Any("visibleRows", visibleRows))

		if cursorRow >= len(visibleRows) {
			if len(visibleRows) > 0 {
				currentFinder.cursorRow = len(visibleRows) - 1
			} else {
				currentFinder.cursorRow = 0
			}
		}
	}

	return currentFinder, nil, done
}
