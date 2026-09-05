package interactive

import "testing"

func TestKeyDecoder(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want keyKind
	}{
		{name: "character", data: []byte{'a'}, want: keyCharacter},
		{name: "enter", data: []byte{'\r'}, want: keyEnter},
		{name: "tab", data: []byte{'\t'}, want: keyTab},
		{name: "backspace", data: []byte{127}, want: keyBackspace},
		{name: "up", data: []byte("\x1b[A"), want: keyArrowUp},
		{name: "down", data: []byte("\x1b[B"), want: keyArrowDown},
		{name: "interrupt", data: []byte{3}, want: keyInterrupt},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decoder := &keyDecoder{}
			var got keyEvent
			var complete bool
			for _, value := range test.data {
				got, complete = decoder.decode(value)
			}
			if !complete {
				t.Fatal("decode() did not produce a complete key event")
			}
			if got.kind != test.want {
				t.Fatalf("key kind = %d, want %d", got.kind, test.want)
			}
		})
	}
}
