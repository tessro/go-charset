package charset
import (
	"testing"
	"testing/iotest"
	"bytes"
	"io"
)

var testReaders = []func(io.Reader) io.Reader {
	func(r io.Reader) io.Reader { return r },
	iotest.OneByteReader,
	iotest.HalfReader,
	iotest.DataErrReader,
}

var codepageCharsets = []string{"latin1"}

func TestCodepages(t *testing.T) {
	for _, name := range codepageCharsets {
		for _, inr := range testReaders {
			for _, outr := range testReaders {
				testCodepage(t, name, inr, outr)
			}
		}
	}
}

func testCodepage(t *testing.T, name string, inReader, outReader func(io.Reader) io.Reader) {
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}
	inr := inReader(bytes.NewBuffer(data))
	r, err := NewReader(name, inr)
	if err != nil {
		t.Fatalf("cannot make reader for charset %q: %v", name, err)
	}
	outr := outReader(r)
	r = outr

	var outbuf bytes.Buffer
	w, err := NewWriter(name, &outbuf)
	if err != nil {
		t.Fatalf("cannot make writer  for charset %q: %v", name, err)
	}
	_, err = io.Copy(w, r)
	if err != nil {
		t.Fatalf("copy failed: %v", err)
	}
	err = w.Close()
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if len(outbuf.Bytes()) != len(data) {
		t.Fatalf("short result of roundtrip, charset %q, readers %T, %T; expected 256, got %d", name, inr, outr, len(outbuf.Bytes()))
	}
	for i, x := range outbuf.Bytes() {
		if data[i] != x {
			t.Fatalf("charset %q, round trip expected %d, got %d", name, i, data[i])
		}
	}
}