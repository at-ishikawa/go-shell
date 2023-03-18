package keyboard

type Key int

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
	// Escape + Left Square Brancket + Capital A
	ArrowUp    = 0x1b5b41
	ArrowDown  = 0x1b5b42
	ArrowRight = 0x1b5b43
	ArrowLeft  = 0x1b5b44
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

var specialKeys = []Key{
	Enter,
	Tab,
	Backspace,

	ArrowUp,
	ArrowDown,
	ArrowRight,
	ArrowLeft,
}

func GetKeyEvent(bytes []byte) KeyEvent {
	keyEvent := KeyEvent{}
	if len(bytes) == 0 {
		return keyEvent
	}

	keyBytes := Key(0)
	for _, b := range bytes {
		keyBytes = (keyBytes << 8) | Key(b)
	}

	for _, specialKey := range specialKeys {
		if keyBytes == specialKey {
			keyEvent.Key = specialKey
			return keyEvent
		}
	}

	if bytes[0] == Escape {
		keyEvent.IsEscapePressed = true
		bytes = bytes[1:]
	}
	if bytes[0] >= controlA && bytes[0] <= controlZ {
		keyEvent.Key = Key(bytes[0] - controlA + A)
		keyEvent.IsControlPressed = true
	} else {
		keyEvent.Key = Key(bytes[0])
		keyEvent.Rune = rune(bytes[0])
	}

	return keyEvent
}
