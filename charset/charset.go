package charset

import (
	"fmt"
	"io"
	"json"
	"os"
	"strings"
	"sync"
	"utf8"
)

type newConverter func(arg string) (func() Converter, os.Error)
type class struct {
	fromUTF8 newConverter
	toUTF8   newConverter
}

// CharsetDir is the directory where charset will look for
// its data files.
var CharsetDir = "/usr/local/lib/go-charset/data"

func init() {
	root := os.Getenv("GOROOT")
	if root != "" {
		CharsetDir = root + "/src/pkg/go-charset.googlecode.com/hg/charset/data"
	}
}

var classes = map[string]class{
	"cp": {codePageFromUTF8, codepageToUTF8},
	"big5": {nil, big5ToUTF8},
	//	"cp932": {readCP932, writeCP932},
	//	"euc-jp", {readEUC_JP, nil},
	//	"gb2312", {readGB2312, nil},
	"utf16": {utf16FromUTF8, utf16ToUTF8},
	"utf8": {utf8ToUTF8, utf8ToUTF8},
	//	"8bit": {nil, write8bit},
}

type info struct {
	Alias    string
	Desc     string
	Class    string
	Arg      string
}

// Converter represents a character set converter.
// The Convert method converts the given data,
// and returns the number of bytes of data consumed,
// a slice containing the converted data (which may be
// overwritten on the next call to Convert), and any
// conversion error. If eof is true, the data represents
// the final bytes of the input.
type Converter interface {
	Convert(data []byte, eof bool) (n int, cdata []byte, err os.Error)
}

var (
	readCharsetsOnce sync.Once
	charsets = make(map[string]info)
)

func readCharsets() {
	file := filename("charsets.json")
	csdata, err := os.Open(file, os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "charset: cannot open %q: %v\n", file, err)
		return
	}
	dec := json.NewDecoder(csdata)
	err = dec.Decode(&charsets)
	if err != nil {
		fmt.Fprintf(os.Stderr, "charset: cannot decode %q: %v\n", file, err)
	}
}

// NewWriter returns a new Reader; reading from
// it returns UTF-8 text translated from text
// in the named charset read from r.
func NewReader(name string, r io.Reader) (io.Reader, os.Error) {
	cvt, err := ToUTF8ConverterFunc(name)
	if err != nil {
		return nil, err
	}
	return newCharsetReader(r, cvt()), nil
}

// NewWriter returns a new Writer; writing
// UTF-8 text to it to causes text in the named charset
// to be written on w.
func NewWriter(charset string, w io.Writer) (io.WriteCloser, os.Error) {
	cvt, err := ToCharsetConverterFunc(charset)
	if err != nil {
		return nil, err
	}
	return newCharsetWriter(w, cvt()), nil
}

// ToUTF8ConverterFunc finds the character set named name,
// and returns a function that returns new instances of
// a Converter from the character set to UTF-8.
func ToUTF8ConverterFunc(name string) (func() Converter, os.Error) {
	class, arg, err := findClass(name)
	if err != nil {
		return nil, err
	}
	if class.toUTF8 == nil {
		return nil, fmt.Errorf("charset: %q does not support conversion to UTF8", name)
	}
	return class.toUTF8(arg)
}

// ToCharsetConverterFunc finds the character set named name,
// and returns a function that returns new instances of
// a Converter from UTF-8 to the character set.
func ToCharsetConverterFunc(name string) (func() Converter, os.Error) {
	class, arg, err := findClass(name)
	if err != nil {
		return nil, err
	}
	if class.fromUTF8 == nil {
		return nil, fmt.Errorf("charset: %q does not support conversion from UTF8", name)
	}
	return class.fromUTF8(arg)
}


func findClass(name string) (class class, arg string, err os.Error) {
	readCharsetsOnce.Do(readCharsets)

	lookup := strings.Map(toLower, name)
	var cs info
	found := false

	// allow one level of alias redirection
	for i := 0; i < 2; i++ {
		if cs, found = charsets[lookup]; found {
			if cs.Alias == "" {
				break
			}
			lookup = cs.Alias
		}
	}
	if !found || cs.Alias != "" {
		err = os.ErrorString("charset: no such character set")
		return
	}
	class, ok := classes[cs.Class]
	if !ok {
		err = fmt.Errorf("charset: bad converter found: %q", cs.Class)
		return
	}
	return class, cs.Arg, nil
}

// Names returns set of known canonical character set names.
func Names() []string {
	var names []string
	readCharsetsOnce.Do(readCharsets)
	for name, info := range charsets {
		if info.Class != "" {
			names = append(names, name)
		}
	}
	return names
}

// Aliases returns the known aliases of a character set.
func Aliases(name string) []string {
	return nil // TODO
}

func toLower(c int) int {
	if c >= 'A' && c <= 'Z' {
		c = c - 'A' + 'a'
	}
	return c
}

func filename(f string) string {
	if f != "" && f[0] == '/' {
		return f
	}
	return CharsetDir + "/" + f
}

type charsetWriter struct {
	w   io.Writer
	cvt Converter
	buf []byte // unconsumed data from writer.
}

func newCharsetWriter(w io.Writer, cvt Converter) *charsetWriter {
	return &charsetWriter{w: w, cvt: cvt}
}

func (w *charsetWriter) Write(data []byte) (rn int, rerr os.Error) {

	wdata := data
	if len(w.buf) > 0 {
		w.buf = append(w.buf, data...)
		wdata = w.buf
	}
	n, cdata, err := w.cvt.Convert(wdata, false)
	if err != nil {
		// TODO
	}
	if n > 0 {
		_, err = w.w.Write(cdata)
		if err != nil {
			return 0, err
		}
	}
	w.buf = w.buf[:0]
	if n < len(wdata) {
		w.buf = append(w.buf, wdata[n:]...)
	}
	return len(data), nil
}


func (p *charsetWriter) Close() os.Error {
	if len(p.buf) == 0 {
		return nil
	}
	_, data, err := p.cvt.Convert(p.buf, true)
	p.buf = nil
	if err != nil {
		// TODO
	}
	if len(data) == 0 {
		return nil
	}
	_, err = p.w.Write(p.buf)
	return err
}

type charsetReader struct {
	r     io.Reader
	cvt   Converter
	cdata []byte   // unconsumed data from converter.
	rdata []byte   // unconverted data from reader.
	err   os.Error // final error from reader.
}

func newCharsetReader(r io.Reader, cvt Converter) *charsetReader {
	return &charsetReader{r: r, cvt: cvt}
}

func (r *charsetReader) Read(buf []byte) (int, os.Error) {
	for {
		if len(r.cdata) > 0 {
			n := copy(buf, r.cdata)
			r.cdata = r.cdata[n:]
			return n, nil
		}
		r.rdata = ensure(r.rdata, len(r.rdata)+len(buf))
		if r.err == nil {
			n, err := r.r.Read(r.rdata[len(r.rdata):cap(r.rdata)])
			r.rdata = r.rdata[0 : len(r.rdata)+n]
			r.err = err
		}
		if len(r.rdata) == 0 {
			break
		}
		nc, cdata, cvterr := r.cvt.Convert(r.rdata, r.err != nil)
		if cvterr != nil {
			// TODO
		}
		r.cdata = cdata

		// Ensure that we consume all bytes at eof
		// if the converter refuses them.
		if nc == 0 && r.err != nil {
			nc = len(r.rdata)
		}

		// Copy unconsumed data to the start of the rdata buffer.
		r.rdata = r.rdata[0:copy(r.rdata, r.rdata[nc:])]
	}
	return 0, r.err
}

// ensure makes sure that x has capacity for at least n extra bytes;
// If it does, it returns x; otherwise it returns a new sufficiently large slice
// with a copy of the old contents.
func ensure(x []byte, n int) []byte {
	// logic filched from appendslice1 in runtime
	if len(x) + n <= cap(x) {
		return x
	}
	m := cap(x)
	if m == 0 {
		m = utf8.UTFMax
	}else{
		for{
			if len(x) < 1024 {
				m += m
			}else{
				m += m / 4
			}
			if m >= len(x) + n {
				break
			}
		}
	}
	y := make([]byte, len(x), m)
	copy(y, x)
	return y
}

func appendRune(buf []byte, r int) []byte {
	n := len(buf)
	buf = ensure(buf, n + utf8.UTFMax)
	nu := utf8.EncodeRune(buf[n:n + utf8.UTFMax], r)
	return buf[0:n + nu]
}
