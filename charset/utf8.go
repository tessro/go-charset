package charset

import "unicode/utf8"

func init() {
	registerClass("utf8", toUTF8, toUTF8)
}

type translateToUTF8 struct {
	scratch []byte
}

const errorRuneLen = len(string(utf8.RuneError))

func (p *translateToUTF8) Translate(data []byte, eof bool) (int, []byte, error) {
	p.scratch = ensureCap(p.scratch, (len(data))*errorRuneLen)[:0]
	n := 0
	for len(data) > 0 {
		if !utf8.FullRune(data) && !eof {
			break
		}
		r, size := utf8.DecodeRune(data)
		p.scratch = appendRune(p.scratch, r)
		data = data[size:]
		n += size
	}
	return n, p.scratch, nil
}

func toUTF8(arg string) (Translator, error) {
	return new(translateToUTF8), nil
}
