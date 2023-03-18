package keyboard

type Key byte

type KeyEvent struct {
	Key  Key
	Rune rune

	IsEscapePressed  bool
	IsControlPressed bool
}

// https://pkg.go.dev/gobot.io/x/gobot/platforms/keyboard
const (
	Tilde = iota + 96
	A
	B
	C
	D
	E
	F
	G
	H
	I
	J
	K
	L
	M
	N
	O
	P
	Q
	R
	S
	T
	U
	V
	W
	X
	Y
	Z
)

// https://pkg.go.dev/gobot.io/x/gobot/platforms/keyboard#pkg-constants
const (
	// ArrowUp is the same ANSI code as upper A
	ArrowUp = iota + 65
	ArrowDown
	ArrowRight
	ArrowLeft
)

// https://github.com/c-bata/go-prompt/blob/82a912274504477990ecf7c852eebb7c85291772/input.go#L34
const (
	controlA          = 0x01
	ControlC          = 0x03
	controlZ          = 0x1a
	Escape            = 0x1b
	LeftSquareBracket = 0x5b

	// Enter key is the same as Control B
	Enter     = 0x0D
	Tab       = 0x9
	Backspace = 0x7f
)

var specialKeys = []byte{
	Enter,
	Tab,
	Backspace,
}

func GetKeyEvent(bytes []byte) KeyEvent {
	keyEvent := KeyEvent{}
	if len(bytes) == 0 {
		return KeyEvent{}
	}

	if bytes[0] == Escape {
		if bytes[1] == LeftSquareBracket {
			// This should be arrow keys
			keyEvent.Key = Key(bytes[2])
			return keyEvent
		}

		keyEvent.IsEscapePressed = true
		bytes = bytes[1:]
	}

	for _, specialKey := range specialKeys {
		if bytes[0] == specialKey {
			keyEvent.Key = Key(specialKey)
			return keyEvent
		}
	}

	if bytes[0] >= controlA && bytes[0] <= controlZ {
		keyEvent.Key = Key(bytes[0] - controlA + A)
		keyEvent.IsControlPressed = true
	} else {
		if keyEvent.IsEscapePressed {
			keyEvent.Key = Key(bytes[0])
		}
		keyEvent.Rune = rune(bytes[0])
	}

	return keyEvent
}
