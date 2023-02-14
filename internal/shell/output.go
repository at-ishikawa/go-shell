package shell

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

type output struct {
	fd        int
	file      *os.File
	termState *term.State
}

func initOutput(out *os.File) output {
	return output{
		fd:   int(out.Fd()),
		file: out,
	}
}

func (o *output) makeRaw() error {
	var err error
	o.termState, err = term.MakeRaw(int(o.fd))
	return err
}

func (o *output) restore() error {
	return term.Restore(o.fd, o.termState)
}

func (o *output) finalize() error {
	return o.restore()
}

func (o *output) newLine() error {
	o.file.WriteString("\n")
	// For some reasons, it's required to reset the cursor position
	o.file.Write([]byte{'\r'})
	return nil
}

func (o *output) moveLeft(count int) error {
	_, err := fmt.Fprintf(o.file, "\033[%dD", count)
	return err
}

func (o *output) clearLine() error {
	// https://gist.github.com/fnky/458719343aabd01cfb17a3a4f7296797
	_, err := fmt.Fprint(o.file, "\033[2K\r")
	return err
}

func (o *output) writeLine(str string) error {
	o.clearLine()
	o.file.WriteString("$ " + str)
	return nil
}
