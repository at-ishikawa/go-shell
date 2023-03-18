package keyboard

type Code int

func (c Code) Bytes() []byte {
	value := c
	var bytes []byte
	for value > 0 {
		bytes = append(bytes, byte(value)&0xff)
		value = value >> 8
	}

	// sort reverse
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}
	return bytes
}

type KeyEvent struct {
	Bytes   []byte
	KeyCode Code
	Rune    rune

	IsEscapePressed  bool
	IsControlPressed bool
}

// https://pkg.go.dev/gobot.io/x/gobot/platforms/keyboard
const (
	Tilde Code = iota + 96
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

// https://github.com/c-bata/go-prompt/blob/82a912274504477990ecf7c852eebb7c85291772/input.go#L34
const (
	controlA Code = 0x1
	ControlC Code = 0x3
	controlZ Code = 0x1a
	Escape   Code = 0x1b
)

const (
	// https://pkg.go.dev/gobot.io/x/gobot/platforms/keyboard#pkg-constants
	// Escape + Left Square Brancket + Capital A
	ArrowUp    Code = 0x1b5b41
	ArrowDown       = 0x1b5b42
	ArrowRight      = 0x1b5b43
	ArrowLeft       = 0x1b5b44
	// Enter key is the same as Control B
	Enter     Code = 0xD
	Tab       Code = 0x9
	Backspace Code = 0x7f
)

var specialKeys = []Code{
	Enter,
	Tab,
	Backspace,

	ArrowUp,
	ArrowDown,
	ArrowRight,
	ArrowLeft,
}

func GetKeyEvent(bytes []byte) KeyEvent {
	keyEvent := KeyEvent{
		Bytes: bytes,
	}
	if len(bytes) == 0 {
		return keyEvent
	}

	keyBytes := Code(0)
	for _, b := range bytes {
		keyBytes = (keyBytes << 8) | Code(b)
	}

	for _, specialKey := range specialKeys {
		if keyBytes == specialKey {
			keyEvent.KeyCode = specialKey
			return keyEvent
		}
	}

	if Code(bytes[0]) == Escape {
		keyEvent.IsEscapePressed = true
		bytes = bytes[1:]
	}
	keyCode := Code(bytes[0])
	if keyCode >= controlA && keyCode <= controlZ {
		keyEvent.KeyCode = keyCode - controlA + A
		keyEvent.IsControlPressed = true
	} else {
		keyEvent.KeyCode = keyCode
		keyEvent.Rune = rune(bytes[0])
	}

	return keyEvent
}
