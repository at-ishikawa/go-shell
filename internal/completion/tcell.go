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

type Completion interface {
	Complete(rows []string, options CompleteOptions) (string, error)
	CompleteMulti(rows []string, options CompleteOptions) ([]string, error)
}

type PreviewCommandType func(row int) (string, error)

type CompleteOptions struct {
	PreviewCommand PreviewCommandType
	Header         string
	InitialQuery   string
	IsAnsiColor    bool
}

type finderRow struct {
	visible  bool
	selected bool
	index    int
	value    string
}

type finderRowsType []finderRow

func (vr *finderRowsType) filterByQuery(query string) finderRowsType {
	result := finderRowsType{}
	for i, row := range *vr {
		if query != "" && !strings.Contains(row.value, query) {
			(*vr)[i].visible = false
			continue
		}
		(*vr)[i].visible = true
		result = append(result, row)
	}
	return result
}

func (vr finderRowsType) Len() int {
	count := 0
	for _, r := range vr {
		if r.visible {
			count++
		}
	}
	return count
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
	return complete.complete(rows, options, true)
}

func (complete TcellCompletion) Complete(rows []string, options CompleteOptions) (string, error) {
	result, err := complete.complete(rows, options, false)
	if err != nil {
		return "", err
	}
	if len(result) == 0 {
		return "", nil
	}
	return result[0], nil
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
	visibleRows finderRowsType,
	cursorRow int,
) {
	if previewCommand == nil {
		return
	}
	if visibleRows.Len() == 0 {
		return
	}
	if cursorRow >= visibleRows.Len() {
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

func (complete TcellCompletion) showVisibleRows(
	screen tcell.Screen,
	visibleRows finderRowsType,
	cursorRow int,
	query string,
	header string,
) {
	width, _ := screen.Size()
	prompt := fmt.Sprintf("> %s", query)
	currentX := emitStr(screen, 0, 0, tcell.StyleDefault, prompt)
	screen.ShowCursor(currentX, 0)

	if len(header) > 0 {
		emitStr(screen, 2, 1, tcell.StyleDefault, header)
	} else {
		emitStr(screen, 0, 1, tcell.StyleDefault, strings.Repeat("-", width))
	}
	showY := 2

	for rowIndex := 0; rowIndex < visibleRows.Len(); rowIndex++ {
		row := visibleRows[rowIndex]

		style := tcell.StyleDefault
		if cursorRow == rowIndex {
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

func (complete TcellCompletion) complete(rows []string, options CompleteOptions, isMultiSelectMode bool) ([]string, error) {
	header := options.Header
	screen, err := tcell.NewScreen()
	if err != nil {
		return []string{}, err
	}
	if err := screen.Init(); err != nil {
		return []string{}, err
	}
	defer screen.Fini()

	screen.SetCursorStyle(tcell.CursorStyleDefault)

	cursorRow := 0
	query := options.InitialQuery

	allRows := make(finderRowsType, 0, len(rows))
	for index, r := range rows {
		allRows = append(allRows, finderRow{
			visible: true,
			index:   index,
			value:   r,
		})
	}
	visibleRows := allRows
	if query != "" {
		visibleRows = allRows.filterByQuery(query)
	}

	eg := errgroup.Group{}
	complete.showVisibleRows(screen, visibleRows, cursorRow, query, header)
	complete.showPreview(screen, options.PreviewCommand, visibleRows, cursorRow)
loop:
	for {
		switch event := screen.PollEvent().(type) {
		case *tcell.EventKey:
			switch event.Key() {
			case tcell.KeyTAB:
				if !isMultiSelectMode {
					// disable a tab key for a single selection mode
					continue
				}

				index := visibleRows[cursorRow].index
				allRows[index].selected = !allRows[index].selected
				visibleRows[cursorRow].selected = allRows[index].selected
				if cursorRow < visibleRows.Len()-1 {
					cursorRow++
				}
			case tcell.KeyCtrlC:
				return []string{}, nil

			case tcell.KeyEnter:
				if !isMultiSelectMode {
					index := visibleRows[cursorRow].index
					allRows[index].selected = !allRows[index].selected
				}
				break loop
			case tcell.KeyCtrlP:
				if cursorRow > 0 {
					cursorRow--
				}
			case tcell.KeyCtrlN:
				if cursorRow < visibleRows.Len()-1 {
					cursorRow++
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
				visibleRows = allRows.filterByQuery(query)
				complete.logger.Debug("query was changed",
					zap.String("path", "internal/completion/tcell"),
					zap.String("query", query),
					zap.Any("allRows", allRows),
					zap.Any("visibleRows", visibleRows))

				if cursorRow > visibleRows.Len() {
					cursorRow = visibleRows.Len() - 1
				}
			}

			cursorRow := cursorRow
			visibleRows := visibleRows
			query := query
			eg.Go(func() error {
				screen.Sync()
				screen.Clear()
				complete.showVisibleRows(screen, visibleRows, cursorRow, query, header)
				complete.showPreview(screen, options.PreviewCommand, visibleRows, cursorRow)
				return nil
			})
		}
	}
	if err := eg.Wait(); err != nil {
		return []string{}, err
	}

	result := make([]string, 0, len(allRows))
	for _, row := range allRows {
		if !row.selected {
			continue
		}
		result = append(result, row.value)
	}
	return result, nil
}
