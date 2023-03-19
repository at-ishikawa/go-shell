package ansi

import (
	"regexp"
	"strconv"
	"strings"
)

type Style int
type Color int

type AnsiString struct {
	String          string
	Style           Style
	ForegroundColor Color

	// TODO Unsupported yet
	BackgroundColor Color
}

const (
	StyleRegular   = 0
	StyleBold      = 1
	StyleUnderline = 4

	ColorUnknown = 0
	ColorBlack   = 30
	ColorRed     = 31
	ColorGreen   = 32
	ColorYellow  = 33
	ColorBlue    = 34
	ColorPurple  = 35
	ColorCyan    = 36
	ColorWhite   = 37
)

var (
	// Ansi colors: https://www.lihaoyi.com/post/BuildyourownCommandLinewithANSIescapecodes.html
	// Ansi Bold with colors: https://gist.github.com/JBlond/2fea43a3049b38287e5e9cefc87b2124
	compiledRegexp = regexp.MustCompile(`(?P<escapeCode>\\u001b\[((?P<styleCode>\d+);)?(?P<colorCode>\d+)m)`)

	ansiCodes = map[int]int{
		StyleRegular:   StyleRegular,
		StyleBold:      StyleBold,
		StyleUnderline: StyleUnderline,

		ColorBlack:  ColorBlack,
		ColorRed:    ColorRed,
		ColorGreen:  ColorGreen,
		ColorYellow: ColorYellow,
		ColorBlue:   ColorBlue,
		ColorPurple: ColorPurple,
		ColorCyan:   ColorCyan,
		ColorWhite:  ColorWhite,
	}
)

func ParseString(str string) []AnsiString {
	result := []AnsiString{}
	if len(str) == 0 {
		return result
	}

	str = strings.ReplaceAll(str, "\u001b", "\\u001b")

	var ansiColor Color
	var ansiStyle Style
	currentIndex := 0
	allAnsiEscapeCodeLocations := compiledRegexp.FindAllStringIndex(str, -1)

	for _, locations := range allAnsiEscapeCodeLocations {
		if currentIndex < locations[0] {
			result = append(result, AnsiString{
				String: str[currentIndex:locations[0]],
			})
			currentIndex = locations[0]
		}

		matches := compiledRegexp.FindStringSubmatch(str[currentIndex:])
		if len(matches) > 0 {
			styleCodeIndex := compiledRegexp.SubexpIndex("styleCode")
			if styleCodeIndex >= 0 && matches[styleCodeIndex] != "" {
				styleCode, err := strconv.Atoi(matches[styleCodeIndex])
				if err != nil {
					// unexpected error
					panic(err)
				}
				ansiCode := ansiCodes[styleCode]
				if ansiCode == StyleBold {
					ansiStyle = Style(ansiCode)
				} else {
					ansiColor = Color(ansiCode)
				}
			}

			colorCodeIndex := compiledRegexp.SubexpIndex("colorCode")
			colorCodeStr := matches[colorCodeIndex]
			colorCode, err := strconv.Atoi(colorCodeStr)
			if err != nil {
				// unexpected error
				panic(err)
			}

			ansiCode := ansiCodes[colorCode]
			if ansiCode == StyleBold {
				ansiStyle = Style(ansiCode)
			} else {
				ansiColor = Color(ansiCode)
			}
			escapeCodeIndex := compiledRegexp.SubexpIndex("escapeCode")
			escapeCode := matches[escapeCodeIndex]
			currentIndex = currentIndex + len(escapeCode)
		}

		resetEscapeCode := `\u001b[m`
		lastIndex := strings.Index(str[currentIndex:], resetEscapeCode)
		if lastIndex == -1 {
			panic("unexpected ansi string")
		}

		result = append(result, AnsiString{
			String:          str[currentIndex : currentIndex+lastIndex],
			ForegroundColor: ansiColor,
			Style:           ansiStyle,
		})
		ansiStyle = StyleRegular
		ansiColor = ColorUnknown
		currentIndex += lastIndex + len(resetEscapeCode)
	}

	lastIndex := strings.Index(str, `\u001b[m`)
	if currentIndex < lastIndex {
		result = append(result, AnsiString{
			String: str[currentIndex:lastIndex],
		})
	} else if lastIndex < 0 {
		result = append(result, AnsiString{
			String: str[currentIndex:],
		})
	}
	return result
}
