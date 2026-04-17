package main

import (
	"fmt"
	"strings"
)

var fieldMap = map[string]string{
	"TI": "title",
	"T1": "title",
	"JO": "journal",
	"UR": "url",
	"DO": "doi",
	"VL": "volume",
	"TY": "bibtype",
	"PY": "year",
	"SP": "startpg",
	"EP": "endpg",
	"IS": "issue",
}

type BibEntry struct {
	bibtype string
	authors []string
	title   string
	journal string
	year    string
	volume  string
	issue   string
	startpg string
	endpg   string
	url     string
	doi     string
	unused  map[string]string
}

func (b *BibEntry) BibMap(key, val string) {
	if key == "AU" {
		b.authors = append(b.authors, val)
		return
	}
	switch key {
	case "TI", "T1":
		b.title = val
	case "JO":
		b.journal = val
	case "UR":
		b.url = val
	case "DO":
		b.doi = val
	case "VL":
		b.volume = val
	case "TY":
		b.bibtype = val
	case "PY":
		b.year = val
	case "SP":
		b.startpg = val
	case "EP":
		b.endpg = val
	case "IS":
		b.issue = val
	default:
		if b.unused == nil {
			b.unused = make(map[string]string)
		}
		b.unused[key] = val
	}
}

func (b *BibEntry) CheckBibEntry() error {
	if len(b.title) < 4 {
		return fmt.Errorf("title is too short or missing: %q", b.title)
	}
	return nil
}

func (b *BibEntry) OutputDebug() {
	fmt.Println("unused fields:")
	for k, v := range b.unused {
		fmt.Printf("  %s: %s\n", k, v)
	}
}

func CreateBibEntry(content []string) *BibEntry {
	b := &BibEntry{}
	for _, line := range content {
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) == 2 {
			key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			b.BibMap(key, val)
		}
	}
	return b
}

func FormatBibEntry(bib *BibEntry, id string) string {
	var sb strings.Builder
	sb.WriteString("@article{" + id + ",\n")
	sb.WriteString("author = {" + strings.Join(bib.authors, "  and  ") + "},\n")
	sb.WriteString("title = {" + bib.title + "},\n")
	sb.WriteString("journal = {" + bib.journal + "},\n")
	sb.WriteString("year  = {" + bib.year + "},\n")
	sb.WriteString("volume  = {" + bib.volume + "},\n")
	sb.WriteString("issue = {" + bib.issue + "},\n")
	sb.WriteString("pages = {" + bib.startpg + "-" + bib.endpg + "},\n")
	sb.WriteString("doi  = {" + bib.doi + "},\n")
	sb.WriteString("url  = {" + bib.url + "}\n")
	sb.WriteString("}")
	return sb.String()
}

func Convert(filedata string, id string) (string, error) {
	contents := strings.Split(filedata, "\n")
	bib := CreateBibEntry(contents)
	if err := bib.CheckBibEntry(); err != nil {
		bib.OutputDebug()
		return "", err
	}
	return FormatBibEntry(bib, id), nil
}

func ConvertWithoutId(filename string, filedata string) (string, error) {
	contents := strings.Split(filedata, "\n")
	bib := CreateBibEntry(contents)
	if err := bib.CheckBibEntry(); err != nil {
		bib.OutputDebug()
		return "", err
	}

	titleWords := strings.Split(bib.title, " ")
	idx := 0
	for idx < len(titleWords) && len(titleWords[idx]) < 3 {
		idx++
	}
	if idx >= len(titleWords) {
		idx = 0
	}
	name := strings.Replace(strings.Split(bib.authors[0], ",")[0], " ", "", -1)
	id := name + bib.year + titleWords[idx]
	return FormatBibEntry(bib, id), nil
}
