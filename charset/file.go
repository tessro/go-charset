package charset
import (
	"path/filepath"
	"io"
)

var files map[string] func() io.ReadCloser

// RegisterDataFile registers the existence of a given data
// file that may be used by a character-set converter.
// It is intended to be used by packages that wish to embed
// data in the executable binary, and should not be
// used normally.
func RegisterDataFile(name string, open func(name string) io.ReadCloser) {
	files[name] = open
}

// CharsetDir gives the location of the default data file directory.
// This directory will be used for files with names that have not
// been registered with RegisterDataFile.
var CharsetDir = "/usr/local/lib/go-charset/data"

func readFile(name string) (data []byte, err error) {
	var r io.ReadCloser
	if open := files[name]; open != nil {
		r, err = open(name)
		if err != nil {
			return
		}
	} else {
		r, err = os.Open(filepath.Join(CharsetDir, name))
		if err != nil {
			return
		}
	}
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading %q: %v", file, err)
	}
	return data, nil
}
