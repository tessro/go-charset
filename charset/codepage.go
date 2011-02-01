package charset

import (
	"fmt"
	"utf8"
	"os"
)

func init() {
	registerClass("cp", fromCodePage, toCodePage)
}

type translateFromCodePage struct {
	byte2rune []int
	scratch   []byte
}

type cpKeyFrom string
type cpKeyTo string

func (p *translateFromCodePage) Translate(data []byte, eof bool) (int, []byte, os.Error) {
	p.scratch = p.scratch[:0]
	for _, x := range data {
		p.scratch = appendRune(p.scratch, p.byte2rune[x])
	}
	return len(data), p.scratch, nil
}

type translateToCodePage struct {
	rune2byte map[int]byte
	scratch   []byte
}

func (p *translateToCodePage) Translate(data []byte, eof bool) (int, []byte, os.Error) {
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

func fromCodePage(arg string) (Translator, os.Error) {
	runes, err := cache(cpKeyFrom(arg), func() (interface{}, os.Error) {
		data, err := readFile(arg)
		if err != nil {
			return nil, err
		}
		runes := []int(string(data))
		if len(runes) != 256 {
			return nil, fmt.Errorf("charset: %q has wrong rune count", arg)
		}
		return runes, nil
	})
	if err != nil {
		return nil, err
	}
	return &translateFromCodePage{byte2rune: runes.([]int)}, nil
}

func toCodePage(arg string) (Translator, os.Error) {
	m, err := cache(cpKeyTo(arg), func() (interface{}, os.Error) {
		data, err := readFile(arg)
		if err != nil {
			return nil, err
		}

		m := make(map[int]byte)
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
	return &translateToCodePage{rune2byte: m.(map[int]byte)}, nil
}
