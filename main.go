package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
)

func IsDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}
	return fileInfo.IsDir()
}

func ProcessFile(file string , OutFile string, id string) {
  log.Println("Processing file: ", file)
  data, err := os.ReadFile(file)
  if err != nil {
    log.Fatal(err)
  }

  if id != " " {
    if OutFile == "." {
      Convert(file, string(data), id)
    } else {
      Convert(OutFile, string(data), id)
    }
  } else {
    if OutFile == "." {
      ConvertWithoutId(file, string(data))
    } else {
      ConvertWithoutId(OutFile, string(data))
    }
  }
}


func main() {
	FilePtr    := flag.String("file", ".", "filename of ris file or directory path to ris file(s).")
	OutFilePtr := flag.String("out", ".", "new filename of bib file")
	IdPtr      := flag.String("id", " ", "BibTeX article id")

	flag.Parse()

	File := *FilePtr
	OutFile := *OutFilePtr
	id := *IdPtr

  var files []string
  var err error

  if File == "." {
    files, err = filepath.Glob("*.ris")
  } else if IsDir(File) {
    files, err = filepath.Glob(filepath.Join(File, "*.ris"))
  } else {
    ProcessFile(File, OutFile, id)
    return 
  }

  if err != nil {
    log.Fatal(err)
  }

  log.Println("Found", len(files), ".ris files in ", File)
  for _, file := range files {
    ProcessFile(file, OutFile, id)
  }
}

