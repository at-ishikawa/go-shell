package keyboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEvent(t *testing.T) {
	testCases := []struct {
		name  string
		input []byte
		want  KeyEvent
	}{
		{
			name:  "alphabet upper case",
			input: []byte{65},
			want: KeyEvent{
				Key:  'A',
				Rune: 'A',
			},
		},
		{
			name:  "alphabet lower case",
			input: []byte{A},
			want: KeyEvent{
				Key:  'a',
				Rune: 'a',
			},
		},
		{
			name:  "Control key",
			input: []byte{controlA},
			want: KeyEvent{
				Key:              A,
				IsControlPressed: true,
			},
		},
		{
			name:  "Tab key",
			input: []byte{Tab},
			want: KeyEvent{
				Key: Tab,
			},
		},
		{
			name:  "Backspace key",
			input: []byte{Backspace},
			want: KeyEvent{
				Key: Backspace,
			},
		},
		{
			name:  "Enter key",
			input: []byte{Enter},
			want: KeyEvent{
				Key: Enter,
			},
		},
		{
			name:  "Escape key",
			input: []byte{Escape, A},
			want: KeyEvent{
				Key:             A,
				Rune:            'a',
				IsEscapePressed: true,
			},
		},
		{
			name:  "Arrow key",
			input: []byte{Escape, LeftSquareBracket, 0x41},
			want: KeyEvent{
				Key: ArrowUp,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, GetKeyEvent(tc.input))
		})
	}
}
