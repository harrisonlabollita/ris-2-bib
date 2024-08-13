package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
  "strings"
)

func IsDir(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), nil
}

func ProcessFile(file string , OutFile string, id string) error {
  log.Println("Processing file: ", file)
  data, err := os.ReadFile(file)
  if err != nil {
    return err
  }

  if strings.TrimSpace(id) != "" {
    if OutFile == "." {
      return Convert(file, string(data), id)
    }
      return  Convert(OutFile, string(data), id)
  } 
  if OutFile == "." {
    return  ConvertWithoutId(file, string(data))
  }
  return  ConvertWithoutId(OutFile, string(data))
}


func main() {
	FilePtr    := flag.String("file", ".", "filename of ris file or directory path to ris file(s).")
	OutFilePtr := flag.String("out", ".", "new filename of bib file")
	IdPtr      := flag.String("id", " ", "BibTeX article id")

	flag.Parse()

  files := []string{}

	file    := *FilePtr
	OutFile := *OutFilePtr
	id      := *IdPtr

  var err error
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

  log.Printf("Found %d .ris files in %s\n", len(files), file)

  for _, f := range files {
    if err := ProcessFile(f, OutFile, id); err != nil {
      log.Printf("Failed to process file %s: %v", f, err)
    }
  }
}

