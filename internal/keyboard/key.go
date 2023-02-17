package keyboard

type Key int

// https://github.com/c-bata/go-prompt/blob/82a912274504477990ecf7c852eebb7c85291772/input.go#L34
const (
	Key_Unknown Key = 0
	ControlB        = 0x02
	ControlN        = 0xe
	ControlP        = 0x10
	ControlR        = 0x12
	Escape          = 0x1b
	Enter           = 0x0D
	Tab             = 0x9
	Backspace       = 0x7f
)

var keys = map[byte]Key{
	ControlB:  ControlB,
	ControlN:  ControlN,
	ControlP:  ControlP,
	ControlR:  ControlR,
	Escape:    Escape,
	Enter:     Enter,
	Tab:       Tab,
	Backspace: Backspace,
}

func GetKey(b byte) Key {
	if k, ok := keys[b]; ok {
		return k
	}
	return Key_Unknown
}
