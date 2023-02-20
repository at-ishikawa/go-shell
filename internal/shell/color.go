// https://gist.github.com/ik5/d8ecde700972d4378d87

package shell

import "fmt"

var (
	Dim = Color("\033[1;2m%s\033[0m")
)

func Color(colorString string) func(...interface{}) string {
	sprint := func(args ...interface{}) string {
		return fmt.Sprintf(colorString,
			fmt.Sprint(args...))
	}
	return sprint
}
