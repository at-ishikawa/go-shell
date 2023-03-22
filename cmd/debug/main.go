package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

func main() {
	if len(os.Args) < 1 {
		consoleKey()
		return
	}
	arg := os.Args[1]
	fmt.Println(arg)
	switch arg {
	case "keycode":
		consoleKey()
	case "interrupt":
		interrupt()
	default:
		runTcell()
	}
}

func interrupt() {
	log.Println("sleep")
	errCh := make(chan error)
	defer close(errCh)
	cmd := exec.Command("sleep", "60")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	go func() {
		errCh <- cmd.Run()
		log.Println("end command")
	}()

	interuptSignals := make(chan os.Signal, 1)
	defer signal.Stop(interuptSignals)
	signal.Notify(interuptSignals, os.Interrupt)
	go func() {
		sig := <-interuptSignals
		log.Println("canceled")
		if err := cmd.Process.Signal(sig); err != nil {
			log.Printf("failed to send a signal %d: ", sig)
			log.Println(err)
		}
	}()
	if err := <-errCh; err != nil {
		log.Printf("failed to run the command: ")
		log.Println(err)
	}
}

func consoleKey() {
	fmt.Println("Press Ctrl-C to exit this program")
	fmt.Println("Press any key to see their ASCII code follow by Enter")

	for {
		// only read single characters, the rest will be ignored!!
		consoleReader := bufio.NewReaderSize(os.Stdin, 1)
		fmt.Print(">")
		input, _ := consoleReader.ReadByte()

		ascii := input

		fmt.Println("ASCII : ", ascii)
		// Ctrl-C = 3
		if ascii == 3 {
			fmt.Println("Exiting...")
			os.Exit(0)
		}
	}
	return
}

var query = ""
var cursorY int = 1

var rows = []string{
	"Apple",
	"Orange",
	"Banana",
	"Pear",
	"Strawberry",
	"Blueberry",
}

func emitStr(s tcell.Screen, x, y int, style tcell.Style, str string) {
	for _, c := range str {
		var comb []rune
		w := runewidth.RuneWidth(c)
		s.SetContent(x, y, c, comb, style)
		x += w
	}
}

func displayHelloWorld(s tcell.Screen) {
	s.Clear()
	prompt := fmt.Sprintf("Input: %s", query)
	emitStr(s, 2, 0, tcell.StyleDefault, prompt)
	s.SetCursorStyle(tcell.CursorStyleDefault)
	s.ShowCursor(2+len(prompt), 0)

	y := 1
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if query != "" && !strings.Contains(row, query) {
			continue
		}

		style := tcell.StyleDefault
		if cursorY == y {
			style = tcell.StyleDefault.Foreground(tcell.ColorCadetBlue.TrueColor()).Background(tcell.ColorWhite)
			s.SetCell(0, y, style, '>', ' ')
		}
		emitStr(s, 2, y, style, fmt.Sprintf("%s", row))
		y++
	}
	s.Show()
}

func runTcell() {
	fmt.Println("before screen")
	s, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	if e := s.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}

	displayHelloWorld(s)

loop:
	for {
		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
			displayHelloWorld(s)
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape {
				break loop
			}

			switch ev.Key() {
			case tcell.KeyEnter:
				query = ""
				break loop
			case tcell.KeyCtrlP:
				if cursorY > 1 {
					cursorY--
					s.Sync()
					displayHelloWorld(s)
				}
			case tcell.KeyCtrlN:
				if cursorY < len(rows) {
					cursorY++
					s.Sync()
					displayHelloWorld(s)
				}
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				query = query[:len(query)-1]
				s.Sync()
				displayHelloWorld(s)
			default:
				rune := ev.Rune()
				query = query + string(rune)
				s.Sync()
				displayHelloWorld(s)
			}
		}
	}

	s.Fini()
	fmt.Println("end fini")
}
