package shell

import (
	"bufio"
	"os"

	"github.com/at-ishikawa/go-shell/internal/keyboard"

	"golang.org/x/term"
)

type input struct {
	fd        int
	file      *os.File
	termState *term.State
	reader    *bufio.Reader
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

func (i *input) Read() (rune, keyboard.Key, error) {
	b, err := i.reader.ReadByte()
	return rune(b), keyboard.GetKey(b), err
}
