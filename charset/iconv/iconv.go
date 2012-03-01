// The iconv package provides an interface to the GNU iconv character set
// conversion library (see http://www.gnu.org/software/libiconv/).
// It automatically registers all the character sets with the charset package,
// so it is usually used simply for the side effects of importing it.
// Example:
//   import (
//		"go-charset.googlecode.com/hg/charset"
//		_ "go-charset.googlecode.com/hg/charset/iconv"
//   )
package iconv

//#cgo LDFLAGS: -liconv -L/opt/local/lib
//#include <stdlib.h>
//#include <string.h>
//#include <iconv.h>
//iconv_t iconv_open_error = (iconv_t)-1;
//size_t iconv_error = (size_t)-1;
//
//void go_iconvlist(void*x);
//int nameCallback(unsigned int, char **, void*);
//
//typedef struct nameList nameList;
//struct nameList {
//	int n;
//	char **names;
//	nameList *next;
//};
//
//int
//addNames(unsigned int n, const char *const *names, void *data) {
//	// we can't call back to Go because of the stack size issue,
//	// so copy all the names.
//	nameList *hd, *e;
//	int i;
//
//	hd = data;
//	e = malloc(sizeof(nameList));
//	e->n = n;
//	e->names = malloc(sizeof(char*) * n);
//	for(i = 0; i < n; i++){
//		e->names[i] = strdup(names[i]);
//	}
//	e->next = hd->next;
//	hd->next = e;
//	return 0;
//}
//
//nameList *
//listNames(void) {
//	nameList hd;
//	hd.next = 0;
//	iconvlist(addNames, &hd);
//	return hd.next;
//}
import "C"
import (
	"code.google.com/p/go-charset/charset"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"unicode/utf8"
	"unsafe"
)

type iconvTranslator struct {
	cd      C.iconv_t
	scratch []byte
}

// Names returns all the character set names known to iconv
// as a slice of slices. Each entry in the returned slice holds
// the name and the aliases of a given character set.
func Names() [][]string {
	var next *C.nameList
	var names [][]string
	for p := C.listNames(); p != nil; p = next {
		next = p.next
		aliases := make([]string, p.n)
		pnames := (*[1e9]*C.char)(unsafe.Pointer(p.names))
		for i := range aliases {
			aliases[i] = C.GoString(pnames[i])
			C.free(unsafe.Pointer(pnames[i]))
		}
		C.free(unsafe.Pointer(p.names))
		C.free(unsafe.Pointer(p))
		names = append(names, aliases)
	}
	return names
}

func canonicalChar(c int) int {
	if c >= 'a' && c <= 'z' {
		return c - 'a' + 'A'
	}
	return c
}

func canonicalName(s string) string {
	return strings.Map(canonicalChar, s)
}

func init() {
	for _, aliases := range Names() {
		aliases := aliases
		cs := &charset.Charset{
			Name:    aliases[0],
			Aliases: aliases[1:],
			TranslatorFrom: func() (charset.Translator, error) {
				return Translator("UTF-8", aliases[0])
			},
			TranslatorTo: func() (charset.Translator, error) {
				return Translator(aliases[0], "UTF-8")
			},
		}
		cs.Register(true)
	}
}

// Translator returns a Translator that translates between
// the named character sets.
func Translator(toCharset, fromCharset string) (charset.Translator, error) {
	cto, cfrom := C.CString(toCharset), C.CString(fromCharset)
	cd, err := C.iconv_open(cto, cfrom)

	C.free(unsafe.Pointer(cfrom))
	C.free(unsafe.Pointer(cto))

	if cd == C.iconv_open_error {
		if err == os.EINVAL {
			return nil, errors.New("iconv: conversion not supported")
		}
		return nil, err
	}
	t := &iconvTranslator{cd: cd}
	runtime.SetFinalizer(t, func(*iconvTranslator) {
		C.iconv_close(cd)
	})
	return t, nil
}

func (p *iconvTranslator) Translate(data []byte, eof bool) (rn int, rd []byte, rerr error) {
	n := 0
	p.scratch = p.scratch[:0]
	for len(data) > 0 {
		p.scratch = ensureCap(p.scratch, len(p.scratch)+len(data)*utf8.UTFMax)
		cData := (*C.char)(unsafe.Pointer(&data[:1][0]))
		nData := C.size_t(len(data))

		ns := len(p.scratch)
		cScratch := (*C.char)(unsafe.Pointer(&p.scratch[ns : ns+1][0]))
		nScratch := C.size_t(cap(p.scratch) - ns)
		r, err := C.iconv(p.cd, &cData, &nData, &cScratch, &nScratch)

		p.scratch = p.scratch[0 : cap(p.scratch)-int(nScratch)]
		n += len(data) - int(nData)
		data = data[len(data)-int(nData):]

		if r != C.iconv_error || err == nil {
			return n, p.scratch, nil
		}
		switch err := err.(os.Errno); err {
		case C.EILSEQ:
			// invalid multibyte sequence - skip one byte and continue
			p.scratch = appendRune(p.scratch, utf8.RuneError)
			n++
			data = data[1:]
		case C.EINVAL:
			// incomplete multibyte sequence
			return n, p.scratch, nil
		case C.E2BIG:
			// output buffer not large enough; try again with larger buffer.
			p.scratch = ensureCap(p.scratch, cap(p.scratch)+utf8.UTFMax)
		default:
			panic(fmt.Sprintf("unexpected error code: %v", err))
		}
	}
	return n, p.scratch, nil
}

// ensureCap returns s with a capacity of at least n bytes.
// If cap(s) < n, then it returns a new copy of s with the
// required capacity.
func ensureCap(s []byte, n int) []byte {
	if n <= cap(s) {
		return s
	}
	// logic adapted from appendslice1 in runtime
	m := cap(s)
	if m == 0 {
		m = n
	} else {
		for {
			if m < 1024 {
				m += m
			} else {
				m += m / 4
			}
			if m >= n {
				break
			}
		}
	}
	t := make([]byte, len(s), m)
	copy(t, s)
	return t
}

func appendRune(buf []byte, r int) []byte {
	n := len(buf)
	buf = ensureCap(buf, n+utf8.UTFMax)
	nu := utf8.EncodeRune(buf[n:n+utf8.UTFMax], r)
	return buf[0 : n+nu]
}
