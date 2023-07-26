package main

import (
	"strings"
	"testing"
)

var fileData string = `TY  - JOUR
            AU  - LastName1, FirstName1
            AU  - LastName2, FirstName2
            AU  - LastName3, FirstName3
            AU  - LastName4, FirstName4
            PY  - 2023
            DA  - 2023/01/01
            TI  - An interesting title would be here
            JO  - Journal Name
            SP  - 123
            EP  - 123
            VL  - 123
            IS  - 1234
            AB  - The quick brown fox jumped over the lazy sheep dog.
            SN  - 1234-2468
            UR  - https://doi.org/10.0000/journal0000
            DO  - 10.0000/journal0000`

var contents []string = strings.Split(fileData, "\n")

var bib *bibEntry = createBibEntry(contents)

func TestAuthors(t *testing.T) {
	if len(bib.authors) != 4 { // 4 Authors;
		t.Errorf("Number of authors parsed %d, Number of authors expected %d", len(bib.authors), 4)
	}
	authors := []string{"LastName1, FirstName1",
		"LastName2, FirstName2",
		"LastName3, FirstName3",
		"LastName4, FirstName4"}
	for i, a := range authors {
		if a != bib.authors[i] {
			t.Errorf("Author error: parsed %s, expected %s", bib.authors[i], a)
		}

	}
}

func TestCheck(t *testing.T) {
	if bib.checkBibEntry() != nil {
		t.Errorf("Error: bib did not pass check!")
	}
}

func TestTitle(t *testing.T) {
	title := "An interesting title would be here"
	if bib.title != title {
		t.Errorf("Error: parsed %s, expected %s", bib.title, title)
	}
}

func TestJounral(t *testing.T) {
	journal := "Journal Name"
	if bib.journal != journal {
		t.Errorf("Error: parsed %s, expected %s", bib.journal, journal)
	}
}

func TestVolume(t *testing.T) {
	volume := "123"
	if bib.volume != volume {
		t.Errorf("Error: parsed %s, expected %s", bib.volume, volume)
	}
}
