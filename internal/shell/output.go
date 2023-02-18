package shell

import (
	"fmt"
	"os"
)

type output struct {
	fd     int
	file   *os.File
	cursor int
}

func initOutput(out *os.File) output {
	return output{
		fd:     int(out.Fd()),
		file:   out,
		cursor: 0,
	}
}

func (o *output) initNewLine() error {
	o.cursor = 0
	return o.writeLine("")
}

func (o *output) newLine() error {
	o.file.WriteString("\n")
	// For some reasons, it's required to reset the cursor position
	o.file.Write([]byte{'\r'})
	return nil
}

func (o *output) setCursor(position int) {
	o.cursor = position
}

func (o *output) moveCursor(count int) {
	o.cursor = o.cursor + count
}

func (o *output) clearLine() error {
	// https://gist.github.com/fnky/458719343aabd01cfb17a3a4f7296797
	_, err := fmt.Fprint(o.file, "\033[2K\r")
	return err
}

func (o *output) writeLine(str string) error {
	o.clearLine()
	o.file.WriteString("$ " + str)

	if o.cursor < 0 {
		fmt.Fprintf(o.file, "\033[%dD", -o.cursor)
	}

	/*
		log.Println("%v", map[string]interface{}{
			"str":    str,
			"cursor": o.cursor,
		})
	*/
	return nil
}
