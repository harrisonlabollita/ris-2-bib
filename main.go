package main

import (
	"flag"
	"fmt"
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
	outfileName := flag.String("out", ".", "new filename of bib file")

	flag.Parse()

	File := *FilePtr
	OutFile := *outfileName

	if File == "." { // this is our current directory
		files, glob_err := filepath.Glob("*.ris")
		if glob_err != nil {
			log.Fatal(glob_err)
		}

		wd, _ := os.Getwd()
		fmt.Println("Found", len(files), "RIS files in", wd)
		for f := 0; f < len(files); f++ {
			fmt.Println("Processing file: ", files[f])
			data, err := os.ReadFile(files[f])
			if err != nil {
				log.Fatal(err)
			}
			ConvertRisFile(files[f], string(data))
		}
	} else if IsDir(File) {
		files, glob_err := filepath.Glob(File + "/*.ris")
		if glob_err != nil {
			log.Fatal(glob_err)
		}
		fmt.Println("Found", len(files), "ris files in", File)
		for f := 0; f < len(files); f++ {
			fmt.Println("Processing file: ", files[f])
			data, err := os.ReadFile(files[f])
			if err != nil {
				log.Fatal(err)
			}
			ConvertRisFile(files[f], string(data))
		}
	} else {
		fmt.Println("Processing file:", File)
		data, err := os.ReadFile(File)
		if err != nil {
			log.Fatal(err)
		}
		if OutFile == "." {
			ConvertRisFile(File, string(data))
		} else {
			ConvertRisFile(OutFile, string(data))
		}
	}
}
