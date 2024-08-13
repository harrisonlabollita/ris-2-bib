package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)


type UnusedBibKey struct {
  key string
  val string
}

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
	unused  []UnusedBibKey
}

func (b *BibEntry) BibMap(key, val string) {
  
  FieldMap := map[string]*string {
    "TI" : &b.title,
    "T1" : &b.title,
    "JO" : &b.journal,
    "UR" : &b.url,
    "DO" : &b.doi,
    "VL" : &b.volume,
    "ID" : &b.id,
    "TY" : &b.bibtype,
    "PY" : &b.year,
    "SP" : &b.startpg,
    "EP" : &b.endpg,
    "IS" : &b.issue,
  }

  if key == "AU" {
    b.authors = append(b.authors, val)
  } else if field, found := FieldMap[key]; found { 
    *field = val
  } else {
		b.unused = append(b.unused, UnusedBibKey{key, val})
  }
}

func (b *BibEntry) CheckBibEntry() error {
	if len(b.title) < 4 {
		return fmt.Errorf("Error: title is incorrect! Parsed title : " + b.title)
	}
	return nil
}

func (b *BibEntry) OutputDebug() {
	fmt.Println("unused information for debugging: ")
  for _, entry := range b.unused {
    fmt.Printf("%s: %s\n", entry.key, entry.val)
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

func Convert(filename string, filedata string, id string) error {
	contents := strings.Split(filedata, "\n")
	bib := CreateBibEntry(contents)
  if err := bib.CheckBibEntry(); err != nil {
		bib.OutputDebug()
    return err
	}
	return WriteToFile(bib, filename, id)
}

func ConvertWithoutId(filename string, filedata string) error {
	contents := strings.Split(filedata, "\n")
	bib := CreateBibEntry(contents)
	if err := bib.CheckBibEntry(); err != nil {
		bib.OutputDebug()
    return err
  }

	TitleWords := strings.Split(bib.title, " ")
	idx := 0
	for len(TitleWords[idx]) < 3 {
		idx++
	}
  name := strings.Replace(strings.Split(bib.authors[0], ",")[0], " ", "", -1)
	id := name + bib.year + TitleWords[idx]
	return WriteToFile(bib, filename, id)
}

func WriteToFile(bib *BibEntry, filename string, id string) error {
	var BibFile string

	if strings.Contains(filename, ".ris") {
		BibFile = id + ".bib"
	} else {
		BibFile = filename
	}

	log.Println("Creating file: ", BibFile)

	out, err := os.Create(BibFile)
	if err != nil {
    return err
	}
	defer out.Close()

	out.WriteString("@article{" + id + ",\n")
	out.WriteString("author = " + "{" + strings.Join(bib.authors, "  and  ") + "}" + ",\n")
	out.WriteString("title = " + "{" + bib.title + "}" + ",\n")
	out.WriteString("journal = " + "{" + bib.journal + "}" + ",\n")
	out.WriteString("year  = " + "{" + bib.year + "}" + ",\n")
	out.WriteString("volume  = " + "{" + bib.volume + "}" + ",\n")
	out.WriteString("issue = " + "{" + bib.issue + "}" + ",\n")
	out.WriteString("pages = " + "{" + bib.startpg + "-" + bib.endpg + "}" + ",\n")
	out.WriteString("doi  = " + "{" + bib.doi + "}" + ",\n")
	out.WriteString("url  = " + "{" + bib.url + "}\n")
	out.WriteString("}")

  return err
}
