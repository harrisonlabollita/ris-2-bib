package main

import (
    "github.com/harrisonlabollita/ris-2-bib"
    "fmt"
    "os"
    "log"
    "path/filepath"
    "flag"
)

func IsDir(path string) bool {
    fileInfo, info_err := os.Stat(path)
    if info_err != nil{
        log.Fatal(info_err)
    }
    return fileInfo.IsDir()
}

func main() {
    //define flag
    file_ptr := flag.String("file", ".", "filename of ris file or directory path to ris file(s).")
    outfile_name := flag.String("out", ".", "new filename of bib file")

    flag.Parse()

    FILE := *file_ptr
    OUT_FILE := *outfile_name
    if FILE == "." { // this is our current directory
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
    } else if IsDir(FILE) {
        files, glob_err := filepath.Glob(FILE+"/*.ris")
        if glob_err != nil {
            log.Fatal(glob_err)
        }
        fmt.Println("Found", len(files), "RIS files in", FILE)
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
        if OUT_FILE == "." {
            ris2bib.ConvertRIS(FILE, string(data))
        } else {
            ris2bib.ConvertRIS(OUT_FILE, string(data))
        }
    }
}
