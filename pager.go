package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

// preview renders a .ris file as BibTeX for display in the pager.
func preview(file string) (string, error) {
	entries, _, err := ReadSource(file)
	if err != nil {
		return "", err
	}
	taken := map[string]bool{}
	for _, e := range entries {
		e.Key = MakeKey(e, taken)
		taken[e.Key] = true
	}
	return FormatLibrary(entries), nil
}

// RunPager steps through .ris files one at a time so rejects can be deleted
// before anything reaches the library.
func RunPager(files []string) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		log.Fatal(err)
	}
	defer term.Restore(fd, oldState)

	idx := 0
	buf := make([]byte, 1)

	for len(files) > 0 {
		total := len(files)
		fmt.Print("\033[H\033[2J")
		formatted, err := preview(files[idx])
		if err != nil {
			fmt.Printf("Error processing %s: %v\r\n", files[idx], err)
		} else {
			fmt.Printf("[%d/%d] %s\r\n\r\n", idx+1, total, filepath.Base(files[idx]))
			fmt.Println(strings.ReplaceAll(formatted, "\n", "\r\n"))
		}
		fmt.Print("\r\n-- (n/j) next  (p/k) prev  (d) delete  (q) quit --")

		os.Stdin.Read(buf)
		switch buf[0] {
		case 'n', 'j':
			if idx < total-1 {
				idx++
			}
		case 'p', 'k':
			if idx > 0 {
				idx--
			}
		case 'd':
			if err := os.Remove(files[idx]); err != nil {
				fmt.Printf("\r\nFailed to delete %s: %v\r\n", files[idx], err)
			}
			files = append(files[:idx], files[idx+1:]...)
			if idx >= len(files) {
				idx = len(files) - 1
			}
			if len(files) == 0 {
				fmt.Print("\r\n")
				return
			}
		case 'q', 3: // 3 = Ctrl-C
			fmt.Print("\r\n")
			return
		}
	}
}
