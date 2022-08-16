package main

import (
    "github.com/harrisonlabollita/ris-2-bib"
    "fmt"
    "os"
    "log"
    "path/filepath"
    "flag"
)

func main() {
    //define flag
    file_ptr := flag.String("file", ".", "filename of ris file")
    // TODO(#1): provide a directory path
    //           ris2bib should be able to receive a directory path
    //           instead of just filenames
    // TODO(#2): additional flag for controlling the name of bib file
    //           produced?
    flag.Parse()

    FILE := *file_ptr
    if FILE == "." {
        files, glob_err := filepath.Glob("*.ris")
        if glob_err != nil {
            log.Fatal(glob_err)
        }
        wd, _ := os.Getwd()
        fmt.Println("Found", len(files), "RIS files in", wd)
        for f := 0; f<len(files); f++ {
            fmt.Println("Processing file: ", files[f])
            data, err := os.ReadFile(files[f])
            if err != nil {
                log.Fatal(err)
            }
            ris2bib.ConvertRIS(files[f], string(data))
        }
    } else {
        fmt.Println("Processing file:", FILE)
        data, err := os.ReadFile(FILE)
        if err != nil {
            log.Fatal(err)
        }
        ris2bib.ConvertRIS(FILE, string(data))
    }
}
