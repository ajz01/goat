// A package for walking directories concurrently. Inspired
// heavily by "The Go Programming Language" book by Donovan
// and Kerninghan. Great book for learning Go programming.
package walk

import (
	"fmt"
	"os"
	"sync"
	"io/ioutil"
	"path/filepath"
	"github.com/ajz01/goat/read"
)

// Helper function to get directory entries
func dirents(dir string) []os.FileInfo {
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goat: %v\n", err)
		return nil
	}
	return entries
}

// Walk directories concurrently
func WalkDir(dir string, n *sync.WaitGroup, dch chan<- read.Decl) {
	defer n.Done()
	for _, entry := range dirents(dir) {
		subdir := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			n.Add(1)
			WalkDir(subdir, n, dch)
		} else {
			if filepath.Ext(entry.Name()) == ".go" {
				fileDecl, err := read.ReadDecl(subdir)
				if err != nil {
					fmt.Fprintf(os.Stderr, "goat ReadDecl: %v\n", err)
				}
				for _, d := range fileDecl {
					dch <- d
				}
			}
		}
	}
}
