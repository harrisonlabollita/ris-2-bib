package main

import (
	"strings"
	"testing"
)

func mustParseRIS(t *testing.T, src string) []*Entry {
	t.Helper()
	entries, _, err := ParseRIS(src)
	if err != nil {
		t.Fatalf("ParseRIS: %v", err)
	}
	return entries
}

// TestMultiRecord covers the bug that motivated the rewrite: a file holding
// several references used to be merged into one entry with the authors of all
// of them and the volume of whichever came first.
func TestMultiRecord(t *testing.T) {
	src := `TY  - JOUR
AU  - Doe, Jane
PY  - 2020
TI  - First paper on things
T2  - Physical Review B
VL  - 101
DO  - 10.1/a
ER  -
TY  - JOUR
AU  - Roe, Rick
AU  - Poe, Pat
PY  - 2021
TI  - Second paper on stuff
JF  - Nature
DO  - 10.2/b
ER  -
`
	entries := mustParseRIS(t, src)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if got, want := entries[0].Get("author"), "Doe, Jane"; got != want {
		t.Errorf("entry 0 author = %q, want %q", got, want)
	}
	if got, want := entries[1].Get("author"), "Roe, Rick and Poe, Pat"; got != want {
		t.Errorf("entry 1 author = %q, want %q", got, want)
	}
	if got := entries[1].Get("volume"); got != "" {
		t.Errorf("entry 1 leaked volume %q from entry 0", got)
	}
	// T2 and JF both name the containing journal; neither used to be mapped.
	if got, want := entries[0].Get("journal"), "Physical Review B"; got != want {
		t.Errorf("T2 -> journal = %q, want %q", got, want)
	}
	if got, want := entries[1].Get("journal"), "Nature"; got != want {
		t.Errorf("JF -> journal = %q, want %q", got, want)
	}
}

// A record with no terminating ER still has to come out whole.
func TestMissingEndOfRecord(t *testing.T) {
	src := "TY  - JOUR\nAU  - A, B\nTI  - One\nTY  - JOUR\nAU  - C, D\nTI  - Two\n"
	if entries := mustParseRIS(t, src); len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
}

func TestContinuationLinesAndHyphenInValue(t *testing.T) {
	src := `TY  - JOUR
TI  - A title that the exporter
      wrapped onto a second line
AB  - Well - actually - this abstract contains hyphens
ER  -
`
	e := mustParseRIS(t, src)[0]
	if got, want := e.Get("title"), "A title that the exporter wrapped onto a second line"; got != want {
		t.Errorf("title = %q, want %q", got, want)
	}
	if got, want := e.Get("abstract"), "Well - actually - this abstract contains hyphens"; got != want {
		t.Errorf("abstract = %q, want %q", got, want)
	}
}

func TestCRLF(t *testing.T) {
	src := "TY  - JOUR\r\nAU  - Doe, Jane\r\nTI  - Windows line endings\r\nVL  - 7\r\nER  - \r\n"
	e := mustParseRIS(t, src)[0]
	if got := e.Get("volume"); got != "7" {
		t.Errorf("volume = %q, want %q (stray carriage return?)", got, "7")
	}
}

func TestTypeMapping(t *testing.T) {
	for _, tc := range []struct{ ty, want string }{
		{"JOUR", "article"},
		{"BOOK", "book"},
		{"CHAP", "incollection"},
		{"CONF", "inproceedings"},
		{"THES", "phdthesis"},
		{"RPRT", "techreport"},
		{"WEIRD", "misc"},
	} {
		e := mustParseRIS(t, "TY  - "+tc.ty+"\nTI  - Title\nER  - \n")[0]
		if e.Type != tc.want {
			t.Errorf("TY %s -> %s, want %s", tc.ty, e.Type, tc.want)
		}
	}
}

// T2 is the journal for an article but the book title for a chapter, and SN is
// an ISSN or an ISBN depending on the same distinction.
func TestContainerFieldsDependOnType(t *testing.T) {
	chapter := mustParseRIS(t, "TY  - CHAP\nTI  - A chapter\nT2  - The Big Book\nSN  - 978-0-13-110362-7\nER  - \n")[0]
	if got := chapter.Get("booktitle"); got != "The Big Book" {
		t.Errorf("chapter booktitle = %q", got)
	}
	if got := chapter.Get("journal"); got != "" {
		t.Errorf("chapter should have no journal, got %q", got)
	}
	if got := chapter.Get("isbn"); got != "978-0-13-110362-7" {
		t.Errorf("chapter isbn = %q", got)
	}

	article := mustParseRIS(t, "TY  - JOUR\nTI  - A paper\nT2  - Nature\nSN  - 0028-0836\nER  - \n")[0]
	if got := article.Get("journal"); got != "Nature" {
		t.Errorf("article journal = %q", got)
	}
	if got := article.Get("issn"); got != "0028-0836" {
		t.Errorf("article issn = %q", got)
	}
}

func TestPages(t *testing.T) {
	for _, tc := range []struct{ sp, ep, want string }{
		{"123", "145", "123--145"}, // en-dash range, not a single hyphen
		{"123", "123", "123"},      // one page, not "123-123"
		{"123", "", "123"},
		{"", "", ""},
	} {
		src := "TY  - JOUR\nTI  - T\nSP  - " + tc.sp + "\nEP  - " + tc.ep + "\nER  - \n"
		if got := mustParseRIS(t, src)[0].Get("pages"); got != tc.want {
			t.Errorf("SP=%q EP=%q -> pages %q, want %q", tc.sp, tc.ep, got, tc.want)
		}
	}
}

// IS is the issue number, and standard BibTeX styles read `number`; an `issue`
// field is silently ignored by plain/unsrt/revtex.
func TestIssueBecomesNumber(t *testing.T) {
	e := mustParseRIS(t, "TY  - JOUR\nTI  - T\nIS  - 4\nER  - \n")[0]
	if got := e.Get("number"); got != "4" {
		t.Errorf("number = %q, want 4", got)
	}
	if got := e.Get("issue"); got != "" {
		t.Errorf("should not emit an `issue` field, got %q", got)
	}
}

func TestEscaping(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Ashcroft & Mermin", `Ashcroft \& Mermin`},
		{"50% of cases", `50\% of cases`},
		{`already \& escaped`, `already \& escaped`}, // no double escaping
		{"math $T_c$ survives", "math $T_c$ survives"},
		{"balanced {NiO} kept", "balanced {NiO} kept"},
		{"stray { brace", `stray \{ brace`}, // would corrupt the entry otherwise
	} {
		if got := escapeRIS(tc.in); got != tc.want {
			t.Errorf("escapeRIS(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// Escaping happens once, at import. If it ran at write time instead, `bib fmt`
// would turn \& into \\& on every run.
func TestEscapeThenFormatIsStable(t *testing.T) {
	e := mustParseRIS(t, "TY  - JOUR\nTI  - Ashcroft & Mermin\nER  - \n")[0]
	once := FormatLibrary([]*Entry{e})
	reparsed, _, err := ParseBibTeX(once)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	if twice := FormatLibrary(reparsed); once != twice {
		t.Errorf("escaping is not stable across a format cycle:\n%s\nvs\n%s", once, twice)
	}
}

func TestYearFromDate(t *testing.T) {
	e := mustParseRIS(t, "TY  - JOUR\nTI  - T\nDA  - 2023/01/01\nER  - \n")[0]
	if got := e.Get("year"); got != "2023" {
		t.Errorf("year from DA = %q, want 2023", got)
	}
}

func TestThesisUsesSchool(t *testing.T) {
	e := mustParseRIS(t, "TY  - THES\nTI  - A thesis\nPB  - Some University\nER  - \n")[0]
	if got := e.Get("school"); got != "Some University" {
		t.Errorf("school = %q", got)
	}
	if got := e.Get("publisher"); got != "" {
		t.Errorf("publisher should have moved to school, got %q", got)
	}
}

func TestUnmappedTagsAreReported(t *testing.T) {
	_, warnings, err := ParseRIS("TY  - JOUR\nTI  - T\nXY  - mystery\nER  - \n")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "XY") {
		t.Errorf("dropped tag XY was not reported, warnings: %q", joined)
	}
}

func TestRecordWithoutTitleIsSkipped(t *testing.T) {
	entries, warnings, err := ParseRIS("TY  - JOUR\nAU  - Doe, J\nER  - \n")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
	if len(warnings) == 0 {
		t.Error("skipping a titleless record should warn")
	}
}
