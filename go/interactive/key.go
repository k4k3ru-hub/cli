package interactive

type keyKind uint8

const (
	keyCharacter keyKind = iota
	keyEnter
	keyTab
	keyBackspace
	keyArrowUp
	keyArrowDown
	keyInterrupt
)

type keyEvent struct {
	kind      keyKind
	character rune
}

type keyDecoder struct {
	escapeSequence []byte
}

func (d *keyDecoder) decode(value byte) (keyEvent, bool) {
	if len(d.escapeSequence) > 0 || value == 27 {
		d.escapeSequence = append(d.escapeSequence, value)
		switch string(d.escapeSequence) {
		case "\x1b", "\x1b[":
			return keyEvent{}, false
		case "\x1b[A":
			d.escapeSequence = d.escapeSequence[:0]
			return keyEvent{kind: keyArrowUp}, true
		case "\x1b[B":
			d.escapeSequence = d.escapeSequence[:0]
			return keyEvent{kind: keyArrowDown}, true
		default:
			d.escapeSequence = d.escapeSequence[:0]
			return keyEvent{}, false
		}
	}

	switch value {
	case 3, 4:
		return keyEvent{kind: keyInterrupt}, true
	case '\r', '\n':
		return keyEvent{kind: keyEnter}, true
	case '\t':
		return keyEvent{kind: keyTab}, true
	case 8, 127:
		return keyEvent{kind: keyBackspace}, true
	default:
		return keyEvent{kind: keyCharacter, character: rune(value)}, true
	}
}
