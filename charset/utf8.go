package charset
import (
	"os"
	"utf8"
)

type utf8CvtToUTF8 struct {
	scratch []byte
}

const errorRuneLen = len(string(utf8.RuneError))

func (p *utf8CvtToUTF8) Convert(data []byte, eof bool) (int, []byte, os.Error) {
	p.scratch = ensure(p.scratch, (len(data)) * errorRuneLen)[:0]
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
	
func utf8ToUTF8(arg string) (func() Converter, os.Error) {
	return func() Converter {
		return new(utf8CvtToUTF8)
	}, nil
}
