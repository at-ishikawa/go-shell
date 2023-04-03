package completion

import (
	"fmt"
	"strings"

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

func (complete TcellCompletion) showPreview(
	screen tcell.Screen,
	previewCommand PreviewCommandType,
	visibleRows []finderRow,
	cursorRow int,
) {
	if previewCommand == nil {
		return
	}
	if len(visibleRows) == 0 {
		return
	}
	if cursorRow >= len(visibleRows) {
		return
	}
	if cursorRow < 0 {
		return
	}

	_, height := screen.Size()
	previewResult, err := previewCommand(visibleRows[cursorRow].index)
	if err != nil {
		emitStr(screen, 2, height/2, tcell.StyleDefault, err.Error())
		return
	}

	lines := strings.Split(previewResult, "\n")
	for i, line := range lines {
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

func (complete TcellCompletion) showVisibleRows(screen tcell.Screen, currentFinder finder) {
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

	screen.SetCursorStyle(tcell.CursorStyleDefault)
	currentFinder := newFinder(rows, options, isMultiSelectMode)
	visibleRows := currentFinder.getVisibleRows()

	complete.showVisibleRows(screen, currentFinder)
	complete.showPreview(screen, options.PreviewCommand, visibleRows, currentFinder.cursorRow)

	var err error
	eg := errgroup.Group{}
loop:
	for {
		switch event := screen.PollEvent().(type) {
		case *tcell.EventKey:
			var done bool
			currentFinder, err, done = complete.handleKeyEvent(currentFinder, event)
			if err != nil {
				break loop
			}
			if done {
				break loop
			}

			cursorRow := currentFinder.cursorRow
			visibleRows := currentFinder.getVisibleRows()
			eg.Go(func() error {
				screen.Sync()
				screen.Clear()
				complete.showVisibleRows(screen, currentFinder)
				complete.showPreview(screen, options.PreviewCommand, visibleRows, cursorRow)
				return nil
			})
		}
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
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
