package keyboard

type Key int

const (
	Key_Unknown Key = 0
	ControlB        = 0x02
	Enter           = 0x0D
)

var keys = map[byte]Key{
	ControlB: ControlB,
	Enter:    Enter,
}

func GetKey(b byte) Key {
	if k, ok := keys[b]; ok {
		return k
	}
	return Key_Unknown
}
