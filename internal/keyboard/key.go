package keyboard

type Key int

const (
	Key_Unknown Key = 0
	ControlB        = 0x02
	Escape          = 0x1b
	Enter           = 0x0D
	Tab             = 0x9
	Backspace       = 0x7f
)

var keys = map[byte]Key{
	ControlB:  ControlB,
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
