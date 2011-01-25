package charset
import (
	"fmt"
	"os"
	"utf8"
	"io/ioutil"
)

// Big5 consists of 89 fonts of 157 chars each
const (
	big5Max = 13973
	big5Font = 157
	big5Data = "big5.dat"
)

type big5CvtToUTF8 struct {
	font int
	scratch []byte
	big5map []int
}

func (p *big5CvtToUTF8) Convert(data []byte, eof bool) (int, []byte, os.Error) {
	p.scratch = p.scratch[:0]
	n := 0
	for len(data) > 0 {
		c := int(data[0])
		data = data[1:]
		n++
		if p.font == -1 {
			// idle state
			if c >= 0xa1 {
				p.font = c
				continue
			}
			if c == 26 {
				c = '\n'
			}
			continue
		}
		f := p.font
		p.font = -1
		r := utf8.RuneError
		switch {
		case c >= 64 && c <= 126:
			c -= 64
		case c >= 161 && c <= 254:
			c = c - 161 + 63
		default:
			// bad big5 char
			f = 255
		}
		if f <= 254 {
			f -= 161
			ix := f * big5Font + c
			if ix < len(p.big5map) {
				r = p.big5map[ix]
			}
			if r == -1 {
				r = utf8.RuneError
			}
		}
		p.scratch = appendRune(p.scratch, r)
	}
	return n, p.scratch, nil
}

func big5ToUTF8(arg string) (func() Converter, os.Error) {
	file := filename(big5Data)
	fd, err := os.Open(file, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("charset: cannot open %q: %v", file, err)
	}
	defer fd.Close()
	buf, err := ioutil.ReadAll(fd)
	if err != nil {
		return nil, fmt.Errorf("charset: error reading %q: %v", file, err)
	}
	big5map := []int(string(buf))
	if len(big5map) != big5Max {
		return nil, fmt.Errorf("charset: corrupt data in %q", file)
	}
	
	return func() Converter {
		return &big5CvtToUTF8{big5map: big5map, font: -1}
	}, nil
}
