package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
)

func IsDir(path string) bool {
	fileInfo, info_err := os.Stat(path)
	if info_err != nil {
		log.Fatal(info_err)
	}
	return fileInfo.IsDir()
}

func main() {
	FilePtr := flag.String("file", ".", "filename of ris file or directory path to ris file(s).")
	OutFilePtr := flag.String("out", ".", "new filename of bib file")
	idPtr := flag.String("id", " ", "BibTeX article id")

	flag.Parse()

	File := *FilePtr
	OutFile := *OutFilePtr
	id := *idPtr

	if File == "." { // this is our current directory
		files, glob_err := filepath.Glob("*.ris")
		if glob_err != nil {
			log.Fatal(glob_err)
		}

		wd, _ := os.Getwd()
		log.Println("Found", len(files), "RIS files in", wd)
		for f := 0; f < len(files); f++ {
			log.Println("Processing file: ", files[f])
			data, err := os.ReadFile(files[f])
			if err != nil {
				log.Fatal(err)
			}
			ConvertWithoutId(files[f], string(data))
		}

	} else if IsDir(File) {
		files, glob_err := filepath.Glob(File + "/*.ris")
		if glob_err != nil {
			log.Fatal(glob_err)
		}
		log.Println("Found", len(files), ".ris files in", File)
		for f := 0; f < len(files); f++ {
			log.Println("Processing file: ", files[f])
			data, err := os.ReadFile(files[f])
			if err != nil {
				log.Fatal(err)
			}
			ConvertWithoutId(files[f], string(data))
		}
	} else {

		log.Println("Processing file:", File)
		data, err := os.ReadFile(File)
		if err != nil {
			log.Fatal(err)
		}

		if OutFile == "." {
			if id != " " {
				Convert(File, string(data), string(id))
			} else {
				ConvertWithoutId(File, string(data))
			}
		} else {
			if id != " " {
				Convert(OutFile, string(data), string(id))
			} else {
				ConvertWithoutId(OutFile, string(data))
			}
		}
	}
}
