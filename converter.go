package ris2bib

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type unused_bib_key map[string]string

type bib_entry struct {
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
	unused  []unused_bib_key
}

func (bib *bib_entry) bib_map(key string, val string) {

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
		bib.unused = append(bib.unused, unused_bib_key{"key": key, "val": val})
	}
}

func (bib *bib_entry) check_bib_entry() error {
	if len(bib.title) < 4 {
		return fmt.Errorf("Error: title is incorrect! Parsed title : " + bib.title)
	}
	return nil
}

func (bib *bib_entry) output_debug() {
	fmt.Println("unused information for debugging: ")
	for i := 0; i < len(bib.unused); i++ {
		fmt.Println(bib.unused[i]["key"] + ":  " + bib.unused[i]["val"])
	}
}

func create_bib_entry(content []string) *bib_entry {

	var bib *bib_entry = &bib_entry{}

	for i := 0; i < len(content); i++ {
		var sep string = " - "
		l := strings.Split(content[i], sep)
		if len(l) > 1 {
			key, val := strings.TrimSpace(l[0]), strings.TrimSpace(l[1])
			bib.bib_map(key, val)
		}
	}

	return bib
}

func ConvertRIS(filename string, filedata string) {
	contents := strings.Split(filedata, "\n")

	bib := create_bib_entry(contents)

	Ok := bib.check_bib_entry()
	if Ok != nil {
		bib.output_debug()
		log.Fatal(Ok)
	}

	id := strings.Split(bib.authors[0], ",")[0] + bib.year + bib.title[:5]

	var bib_file string

	if strings.Contains(filename, ".ris") {
		bib_file = strings.Split(filename, ".ris")[0] + ".bib"
	} else {
		bib_file = filename
	}

	fmt.Println("Creating file: ", bib_file)

	out, err := os.Create(bib_file)
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
