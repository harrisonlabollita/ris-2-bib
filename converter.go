package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type unusedBibKey map[string]string

type bibEntry struct {
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
	unused  []unusedBibKey
}

func (b *bibEntry) bibMap(key, val string) {

	if key == "AU" {
		b.authors = append(b.authors, val)
	} else if key == "TI" {
		b.title = val
	} else if key == "JO" {
		b.journal = val
	} else if key == "UR" {
		b.url = val
	} else if key == "DO" {
		b.doi = val
	} else if key == "VL" {
		b.volume = val
	} else if key == "ID" {
		b.id = val
	} else if key == "TY" {
		b.bibtype = val
	} else if key == "PY" {
		b.year = val
	} else if key == "SP" {
		b.startpg = val
	} else if key == "EP" {
		b.endpg = val
	} else if key == "IS" {
		b.issue = val
	} else {
		b.unused = append(b.unused, unusedBibKey{"key": key, "val": val})
	}
}

func (b *bibEntry) checkBibEntry() error {
	if len(b.title) < 4 {
		return fmt.Errorf("Error: title is incorrect! Parsed title : " + b.title)
	}
	return nil
}

func (b *bibEntry) outputDebug() {
	fmt.Println("unused information for debugging: ")
	for i := 0; i < len(b.unused); i++ {
		fmt.Println(b.unused[i]["key"] + ":  " + b.unused[i]["val"])
	}
}

func createBibEntry(content []string) *bibEntry {

	var b *bibEntry = &bibEntry{}

	for i := 0; i < len(content); i++ {
		var sep string = " - "
		l := strings.Split(content[i], sep)
		if len(l) > 1 {
			key, val := strings.TrimSpace(l[0]), strings.TrimSpace(l[1])
			b.bibMap(key, val)
		}
	}

	return b
}

func ConvertRisFile(filename string, filedata string) {

	contents := strings.Split(filedata, "\n")

	bib := createBibEntry(contents)

	Ok := bib.checkBibEntry()
	if Ok != nil {
		bib.outputDebug()
		log.Fatal(Ok)
	}

	id := strings.Split(bib.authors[0], ",")[0] + bib.year + bib.title[:5]

	var bibFile string

	if strings.Contains(filename, ".ris") {
		bibFile = strings.Split(filename, ".ris")[0] + ".bib"
	} else {
		bibFile = filename
	}

	fmt.Println("Creating file: ", bibFile)

	out, err := os.Create(bibFile)
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
