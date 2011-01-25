package charset

import (
	"fmt"
	"utf8"
	"io"
	"os"
)

type codePageCvtToUTF8 struct {
	byte2rune []int
	scratch   []byte
}

func (p *codePageCvtToUTF8) Convert(data []byte, eof bool) (int, []byte, os.Error) {
	p.scratch = p.scratch[:0]
	for _, x := range data {
		p.scratch = appendRune(p.scratch, p.byte2rune[x])
	}
	return len(data), p.scratch, nil
}

type codePageCvtFromUTF8 struct {
	rune2byte map[int]byte
	scratch   []byte
}

func (p *codePageCvtFromUTF8) Convert(data []byte, eof bool) (int, []byte, os.Error) {
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

func codepageToUTF8(arg string) (func() Converter, os.Error) {
	file := filename(arg)
	fd, err := os.Open(file, os.O_RDONLY, 0)
	if fd == nil {
		return nil, err
	}
	buf := make([]byte, 256*utf8.UTFMax)
	n, _ := io.ReadFull(fd, buf)
	if err != nil {
		return nil, err
	}

	runes := []int(string(buf[0:n]))
	if len(runes) != 256 {
		return nil, fmt.Errorf("charset: code page at %q has too few runes", file)
	}
	return func() Converter {
		return &codePageCvtToUTF8{byte2rune: runes}
	},
		nil
}

func codePageFromUTF8(arg string) (func() Converter, os.Error) {
	file := filename(arg)
	fd, err := os.Open(file, os.O_RDONLY, 0)
	if fd == nil {
		return nil, err
	}
	buf := make([]byte, 256*utf8.UTFMax)
	n, _ := io.ReadFull(fd, buf)
	if err != nil {
		return nil, err
	}

	m := make(map[int]byte)
	i := 0
	for _, r := range string(buf[0:n]) {
		m[r] = byte(i)
		i++
	}
	if i != 256 {
		return nil, fmt.Errorf("charset: code page at %q has too few runes", file)
	}
	return func() Converter {
		return &codePageCvtFromUTF8{
			rune2byte: m,
		}
	},
		nil
}
