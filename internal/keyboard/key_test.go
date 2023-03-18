package keyboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCode_Bytes(t *testing.T) {
	testCases := []struct {
		name string
		code Code
		want []byte
	}{
		{
			name: "single byte",
			code: Enter,
			want: []byte{0xD},
		},
		{
			name: "multiple bytes",
			code: ArrowUp,
			want: []byte{0x1b, 0x5b, 0x41},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.code.Bytes())
		})
	}
}

func TestGetEvent(t *testing.T) {
	keyBytes := func(codes ...Code) []byte {
		result := make([]byte, 0, len(codes))
		for _, code := range codes {
			result = append(result, byte(code))
		}
		return result
	}

	testCases := []struct {
		name  string
		input []byte
		want  KeyEvent
	}{
		{
			name:  "alphabet upper case",
			input: []byte{65},
			want: KeyEvent{
				Bytes:   []byte{65},
				KeyCode: 'A',
				Rune:    'A',
			},
		},
		{
			name:  "alphabet lower case",
			input: keyBytes(A),
			want: KeyEvent{
				Bytes:   keyBytes(A),
				KeyCode: 'a',
				Rune:    'a',
			},
		},
		{
			name:  "Control key",
			input: keyBytes(controlA),
			want: KeyEvent{
				Bytes:            keyBytes(controlA),
				KeyCode:          A,
				IsControlPressed: true,
			},
		},
		{
			name:  "Tab key",
			input: keyBytes(Tab),
			want: KeyEvent{
				Bytes:   keyBytes(Tab),
				KeyCode: Tab,
			},
		},
		{
			name:  "Backspace key",
			input: keyBytes(Backspace),
			want: KeyEvent{
				Bytes:   keyBytes(Backspace),
				KeyCode: Backspace,
			},
		},
		{
			name:  "Enter key",
			input: keyBytes(Enter),
			want: KeyEvent{
				Bytes:   keyBytes(Enter),
				KeyCode: Enter,
			},
		},
		{
			name:  "Escape key",
			input: keyBytes(Escape, A),
			want: KeyEvent{
				Bytes:           keyBytes(Escape, A),
				KeyCode:         A,
				Rune:            'a',
				IsEscapePressed: true,
			},
		},
		{
			name:  "Arrow key",
			input: keyBytes(Escape, 0x5b, 0x41),
			want: KeyEvent{
				Bytes:   keyBytes(Escape, 0x5b, 0x41),
				KeyCode: ArrowUp,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, GetKeyEvent(tc.input))
		})
	}
}
