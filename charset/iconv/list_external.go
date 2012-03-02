// +build linux

// This "portable" way of getting the names assumes that an
// iconv binary is available and uses it to extract the names.
// TODO make it so we do this on demand rather than
// at init time.

package iconv
import (
	"log"
	"os"
	"exec"
)

TODO

func Names() [][]string {
}
//	cmd := exec.Command("iconv", "--list")
//	stdout, err := cmd.StdoutPipe()
//	if err != nil {
//		log.Printf("iconv: %v", err)
//		return nil
//	}
	