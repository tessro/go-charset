package charset

import (
	"fmt"
	"unicode/utf8"
)

func init() {
	registerClass("cp", fromCodePage, toCodePage)
}

type translateFromCodePage struct {
	byte2rune []rune
	scratch   []byte
}

type cpKeyFrom string
type cpKeyTo string

func (p *translateFromCodePage) Translate(data []byte, eof bool) (int, []byte, error) {
	p.scratch = p.scratch[:0]
	for _, x := range data {
		p.scratch = appendRune(p.scratch, p.byte2rune[x])
	}
	return len(data), p.scratch, nil
}

type translateToCodePage struct {
	rune2byte map[rune]byte
	scratch   []byte
}

func (p *translateToCodePage) Translate(data []byte, eof bool) (int, []byte, error) {
	n := 0
	p.scratch = p.scratch[:0]
	for len(data) > 0 {
		if !utf8.FullRune(data) && !eof {
			break
		}
		r, size := utf8.DecodeRune(data)
		b, ok := p.rune2byte[r]
		if !ok {
			b = '?'
		}
		p.scratch = append(p.scratch, b)
		n += size
		data = data[size:]
	}
	return n, p.scratch, nil
}

func fromCodePage(arg string) (Translator, error) {
	runes, err := cache(cpKeyFrom(arg), func() (interface{}, error) {
		data, err := readFile(arg)
		if err != nil {
			return nil, err
		}
		runes := []rune(string(data))
		if len(runes) != 256 {
			return nil, fmt.Errorf("charset: %q has wrong rune count", arg)
		}
		return runes, nil
	})
	if err != nil {
		return nil, err
	}
	return &translateFromCodePage{byte2rune: runes.([]rune)}, nil
}

func toCodePage(arg string) (Translator, error) {
	m, err := cache(cpKeyTo(arg), func() (interface{}, error) {
		data, err := readFile(arg)
		if err != nil {
			return nil, err
		}

		m := make(map[rune]byte)
		i := 0
		for _, r := range string(data) {
			m[r] = byte(i)
			i++
		}
		if i != 256 {
			return nil, fmt.Errorf("charset: %q has wrong rune count", arg)
		}
		return m, nil
	})
	if err != nil {
		return nil, err
	}
	return &translateToCodePage{rune2byte: m.(map[rune]byte)}, nil
}
