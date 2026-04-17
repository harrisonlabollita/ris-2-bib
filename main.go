package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

func IsDir(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), nil
}

func ProcessFile(file string, id string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(id) != "" {
		return Convert(string(data), id)
	}
	return ConvertWithoutId(file, string(data))
}

func RunPager(files []string) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		log.Fatal(err)
	}
	defer term.Restore(fd, oldState)

	idx := 0
	total := len(files)
	buf := make([]byte, 1)

	for {
		fmt.Print("\033[H\033[2J")
		formatted, err := ProcessFile(files[idx], "")
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", files[idx], err)
		} else {
			fmt.Printf("[%d/%d] %s\n\n", idx+1, total, filepath.Base(files[idx]))
			fmt.Println(formatted)
		}
		fmt.Print("\n-- (n/j) next  (p/k) prev  (q) quit --")

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
		case 'q', 3: // 3 = Ctrl-C
			fmt.Println()
			return
		}
	}
}

func main() {
	FilePtr := flag.String("file", ".", "filename of ris file or directory path to ris file(s).")
	IdPtr   := flag.String("id", "", "BibTeX article id (single file only)")

	flag.Parse()

	file := *FilePtr
	id   := *IdPtr

	var err error
	var files []string

	isDir, err := IsDir(file)
	if err != nil {
		log.Fatal(err)
	}

	if isDir || file == "." {
		files, err = filepath.Glob(filepath.Join(file, "*.ris"))
		if err != nil {
			log.Fatal(err)
		}
	} else {
		files = append(files, file)
	}

	if len(files) == 0 {
		log.Println("No .ris files found")
		return
	}

	if len(files) == 1 {
		formatted, err := ProcessFile(files[0], id)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(formatted)
	} else {
		RunPager(files)
	}
}
