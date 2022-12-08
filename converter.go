package ris2bib

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type UnusedBibKey map[string]string

type BibEntry struct {
	bibtype string
	id      string
	authors []string
	title   string
	journal string
	year    string
	volume  string
	issue   string
	startpg string
	endpg   string
	month   string
	url     string
	doi     string
    unused []UnusedBibKey
}

func (bib *BibEntry) Bibmap(key string, val string) {

	if key == "AU" {
		bib.authors = append(bib.authors, val)
	} else if key == "TI" {
		bib.title = val
	} else if key == "JO" {
		bib.journal = val
	} else if key == "UR" {
		bib.url = val
	} else if key == "DO" {
		bib.doi = val
	} else if key == "VL" {
		bib.volume = val
	} else if key == "ID" {
		bib.id = val
	} else if key == "TY" {
		bib.bibtype = val
	} else if key == "PY" {
		bib.year = val
	} else if key == "SP" {
		bib.startpg = val
	} else if key == "EP" {
		bib.endpg = val
	} else if key == "IS" {
		bib.issue = val
	} else {
        bib.unused = append(bib.unused, UnusedBibKey{ "key" : key, "val" : val } )
    }
}


func (bib *BibEntry) CheckBibEntry() error {
    if len(bib.title) < 4 {
        return fmt.Errorf("Error: title is incorrect! Parsed title : "+bib.title)
    }
    return nil
}


func (bib *BibEntry) OutputDebug() {
    fmt.Println("unused information for debugging: ")
    for i:=0; i < len(bib.unused); i++ {
        fmt.Println(bib.unused[i]["key"]+ ":  "+bib.unused[i]["val"])
    }
}

func CreateBibEntry(content []string) *BibEntry {
	var bib *BibEntry = &BibEntry{}

	for i := 0; i < len(content); i++ {
        var sep string = " - "
		l := strings.Split(content[i], sep)
		if len(l) > 1 {
			key, val := strings.TrimSpace(l[0]), strings.TrimSpace(l[1])
			bib.Bibmap(key, val)
		}
	}

	return bib
}


func ConvertRIS(filename string, filedata string) {
	contents := strings.Split(filedata, "\n")

	bib := CreateBibEntry(contents)

    Ok := bib.CheckBibEntry()
    if Ok != nil {
        bib.OutputDebug()
        log.Fatal(Ok)
    }

	id := strings.Split(bib.authors[0], ",")[0] + bib.year + bib.title[:5]

	var BIB_FILE string

	if strings.Contains(filename, ".ris") {
		BIB_FILE = strings.Split(filename, ".ris")[0] + ".bib"
	} else {
		BIB_FILE = filename
	}

	fmt.Println("Creating file: ", BIB_FILE)

	out, err := os.Create(BIB_FILE)
	if err != nil {
		log.Fatal(err)
	}

	defer out.Close()

	out.WriteString("@article{" + id + ",\n")
	out.WriteString("author = " + "\"" + strings.Join(bib.authors, "  and  ") + "\"" + ",\n")
	out.WriteString("title = " + "\"" + bib.title + "\"" + ",\n")
	out.WriteString("journal = " + "\"" + bib.journal + "\"" + ",\n")
	out.WriteString("year  = " + "\"" + bib.year + "\"" + ",\n")
	out.WriteString("volume  = " + "\"" + bib.volume + "\"" + ",\n")
	out.WriteString("issue = " + "\"" + bib.issue + "\"" + ",\n")
	out.WriteString("pages = " + "\"" + bib.startpg + "-" + bib.endpg + "\"" + ",\n")
	out.WriteString("doi  = " + "\"" + bib.doi + "\"" + ",\n")
	out.WriteString("url  = " + "\"" + bib.url + "\"\n")
	out.WriteString("}")
}
