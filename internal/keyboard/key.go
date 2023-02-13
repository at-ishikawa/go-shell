package keyboard

type Key int

const (
	Key_Unknown Key = 0
	ControlB        = 0x02
	Enter           = 0x0D
	Backspace       = 0x7f
	Escape          = 0x1b
)

var keys = map[byte]Key{
	Escape:    Escape,
	ControlB:  ControlB,
	Enter:     Enter,
	Backspace: Backspace,
}

func GetKey(b byte) Key {
	if k, ok := keys[b]; ok {
		return k
	}
	return Key_Unknown
}
