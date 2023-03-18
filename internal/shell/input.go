package shell

import (
	"bufio"
	"io"
	"os"

	"github.com/at-ishikawa/go-shell/internal/keyboard"

	"golang.org/x/term"
)

type input struct {
	fd        int
	file      *os.File
	termState *term.State
	reader    io.Reader
}

func initInput(in *os.File) (input, error) {
	reader := bufio.NewReader(in)
	return input{
		fd:     int(in.Fd()),
		file:   in,
		reader: reader,
	}, nil
}

func (i *input) makeRaw() error {
	var err error
	i.termState, err = term.MakeRaw(i.fd)

	return err
}

func (i *input) restore() error {
	return term.Restore(i.fd, i.termState)
}

func (i *input) finalize() error {
	return i.restore()
}

func (i *input) Read() (keyboard.KeyEvent, error) {
	buffer := make([]byte, 8)
	bufferSize, err := i.reader.Read(buffer)
	// fmt.Printf("%v\n", buffer)
	return keyboard.GetKeyEvent(buffer[:bufferSize]), err
}
