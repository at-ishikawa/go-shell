package completion

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

type Completion interface {
	Complete(lines []string, options CompleteOptions) (string, error)
	CompleteMulti(lines []string, options CompleteOptions) ([]string, error)
}

type CompleteOptions struct {
	IsAnsiColor    bool
	PreviewCommand string
	InitialQuery   string
}

type TcellCompletion struct {
}

var _ Completion = (*TcellCompletion)(nil)

func NewTcellCompletion() (*TcellCompletion, error) {
	return &TcellCompletion{}, nil
}

func (complete *TcellCompletion) CompleteMulti(rows []string, options CompleteOptions) ([]string, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return []string{}, err
	}
	if err := screen.Init(); err != nil {
		return []string{}, err
	}
	defer screen.Fini()

	screen.SetCursorStyle(tcell.CursorStyleDefault)

	selectedRows := make(map[int]string)
	cursorRow := 0
	query := options.InitialQuery

	emitStr := func(s tcell.Screen, x, y int, style tcell.Style, str string) int {
		for _, char := range str {
			var combinings []rune
			runeWidth := runewidth.RuneWidth(char)
			s.SetContent(x, y, char, combinings, style)
			x += runeWidth
		}
		return x
	}

	show := func() {
		screen.Clear()
		prompt := fmt.Sprintf("> %s", query)
		currentX := emitStr(screen, 0, 0, tcell.StyleDefault, prompt)
		screen.ShowCursor(currentX, 0)

		showY := 2
		for rowIndex := 0; rowIndex < len(rows); rowIndex++ {
			row := rows[rowIndex]
			if query != "" && !strings.Contains(row, query) {
				continue
			}

			style := tcell.StyleDefault
			if cursorRow == rowIndex {
				style = tcell.StyleDefault.Foreground(tcell.ColorCadetBlue.TrueColor()).Background(tcell.ColorWhite)
			}
			if _, ok := selectedRows[rowIndex]; ok {
				screen.SetCell(0, showY, style, '>', ' ')
			}
			emitStr(screen, 2, showY, style, fmt.Sprintf("%s", row))
			showY++
		}
		screen.Show()
	}

	show()
loop:
	for {
		switch event := screen.PollEvent().(type) {
		case *tcell.EventKey:
			switch event.Key() {
			case tcell.KeyTAB:
				if _, ok := selectedRows[cursorRow]; ok {
					delete(selectedRows, cursorRow)
				} else {
					selectedRows[cursorRow] = rows[cursorRow]
				}
				if cursorRow < len(rows)-1 {
					cursorRow++
				}
			case tcell.KeyCtrlC:
				return []string{""}, nil

			case tcell.KeyEnter:
				// todo: Do we want to unselect if it's selected?
				selectedRows[cursorRow] = rows[cursorRow]
				break loop
			case tcell.KeyCtrlP:
				if cursorRow > 1 {
					cursorRow--
				}
			case tcell.KeyCtrlN:
				if cursorRow < len(rows)-1 {
					cursorRow++
				}
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if len(query) > 0 {
					query = query[:len(query)-1]
				}
			default:
				ch := event.Rune()
				query = query + string(ch)
			}
			screen.Sync()
			show()
		}
	}

	result := make([]string, 0, len(selectedRows))
	for _, val := range selectedRows {
		result = append(result, val)
	}
	return result, nil
}

func (complete TcellCompletion) Complete(lines []string, options CompleteOptions) (string, error) {
	panic("Not implemented")
}
