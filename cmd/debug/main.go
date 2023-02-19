package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
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
