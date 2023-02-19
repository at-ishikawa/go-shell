package keyboard

type Key int

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

// https://github.com/c-bata/go-prompt/blob/82a912274504477990ecf7c852eebb7c85291772/input.go#L34
const (
	Key_Unknown Key = 0
	ControlA        = 0x01
	ControlB        = 0x02
	ControlC        = 0x03
	ControlD        = 0x04
	ControlE        = 0x05
	ControlF        = 0x06
	ControlG        = 0x07
	ControlH        = 0x08
	ControlI        = 0x09
	ControlJ        = 0x0a
	ControlK        = 0x0b
	ControlL        = 0x0c
	ControlM        = 0x0d
	ControlN        = 0x0e
	ControlO        = 0x0f
	ControlP        = 0x10
	ControlQ        = 0x11
	ControlR        = 0x12
	ControlS        = 0x13
	ControlT        = 0x14
	ControlU        = 0x15
	ControlV        = 0x16
	ControlW        = 0x17
	ControlX        = 0x18
	ControlY        = 0x19
	ControlZ        = 0x1a
	Escape          = 0x1b
	Enter           = 0x0D
	Tab             = 0x9
	Backspace       = 0x7f
)

var keys = []Key{
	B,
	F,

	ControlA,
	ControlB,
	ControlC,
	ControlD,
	ControlE,
	ControlF,
	ControlG,
	ControlH,
	ControlI,
	ControlJ,
	ControlK,
	ControlL,
	ControlM,
	ControlN,
	ControlO,
	ControlP,
	ControlQ,
	ControlR,
	ControlS,
	ControlT,
	ControlU,
	ControlV,
	ControlW,
	ControlX,
	ControlY,
	ControlZ,
	Escape,
	Enter,
	Tab,
	Backspace,
}

var keyMap map[byte]Key

func init() {
	keyMap = make(map[byte]Key, len(keys))
	for _, key := range keys {
		keyMap[byte(key)] = key
	}
}

func GetKey(b byte) Key {
	if k, ok := keyMap[b]; ok {
		return k
	}
	return Key_Unknown
}
