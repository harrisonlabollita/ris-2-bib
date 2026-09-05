package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// risLine matches an RIS tag line. The spec says exactly "XX  - value", but
// exports in the wild pad differently, so the separator is matched loosely.
// A line that does not match is a continuation of the previous tag.
var risLine = regexp.MustCompile(`^\s*([A-Z][A-Z0-9])\s{0,4}-\s?(.*)$`)

var risTypes = map[string]string{
	"JOUR": "article", "JFULL": "article", "MGZN": "article", "NEWS": "article",
	"BOOK": "book", "EBOOK": "book", "SER": "book", "EDBOOK": "book",
	"CHAP": "incollection", "ECHAP": "incollection",
	"CONF": "inproceedings", "CPAPER": "inproceedings",
	"THES":    "phdthesis",
	"RPRT":    "techreport",
	"UNPB":    "unpublished",
	"MANSCPT": "unpublished",
	"COMP":    "software",
}

// risFields maps the tags that translate to a single BibTeX field regardless of
// entry type. Tags needing context (T2, SN) or accumulation (AU, KW) are
// handled in risToEntry.
var risFields = map[string]string{
	"TI": "title", "T1": "title", "CT": "title",
	"JO": "journal", "JF": "journal", "JA": "journal",
	"VL": "volume",
	"IS": "number", "M1": "number",
	"PY": "year", "Y1": "year",
	"PB": "publisher",
	"CY": "address", "PP": "address",
	"ET": "edition",
	"DO": "doi",
	"UR": "url",
	"AB": "abstract", "N2": "abstract",
	"LA": "language",
	"C7": "eid",
	"T3": "series",
	"N1": "note",
}

// risIgnored are bookkeeping tags from the exporting database with no
// bibliographic meaning. They are dropped without comment.
var risIgnored = map[string]bool{
	"ID": true, "DB": true, "DP": true, "AN": true, "M3": true,
	"Y2": true, "TY": true, "ER": true,
}

type risTag struct{ key, val string }

// splitRIS divides a source file into records. Records end at an ER tag; a
// missing ER is tolerated by also breaking at each TY, and a trailing record is
// returned even if unterminated.
func splitRIS(src string) [][]risTag {
	var records [][]risTag
	var cur []risTag

	flush := func() {
		if len(cur) > 0 {
			records = append(records, cur)
			cur = nil
		}
	}

	for _, raw := range strings.Split(src, "\n") {
		line := strings.TrimRight(raw, "\r")
		m := risLine.FindStringSubmatch(line)
		if m == nil {
			// A wrapped value: publishers break long abstracts and titles
			// across lines without repeating the tag.
			if n := len(cur); n > 0 {
				if text := strings.TrimSpace(line); text != "" {
					cur[n-1].val += " " + text
				}
			}
			continue
		}
		key, val := m[1], strings.TrimSpace(m[2])
		switch key {
		case "ER":
			flush()
		case "TY":
			flush()
			cur = append(cur, risTag{key, val})
		default:
			cur = append(cur, risTag{key, val})
		}
	}
	flush()
	return records
}

// ParseRIS converts every record in an RIS file. Records without a title are
// skipped rather than emitted as junk; tags with no BibTeX equivalent are
// reported so nothing disappears silently.
func ParseRIS(src string) ([]*Entry, []string, error) {
	records := splitRIS(src)
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("no RIS records found")
	}

	var entries []*Entry
	var warnings []string
	dropped := map[string]int{}

	for i, rec := range records {
		e, drops := risToEntry(rec)
		if e.Get("title") == "" {
			warnings = append(warnings, fmt.Sprintf("record %d has no title, skipped", i+1))
			continue
		}
		for _, tag := range drops {
			dropped[tag]++
		}
		entries = append(entries, e)
	}

	if len(dropped) > 0 {
		var parts []string
		for tag, n := range dropped {
			parts = append(parts, fmt.Sprintf("%s (%d)", tag, n))
		}
		sort.Strings(parts)
		warnings = append(warnings, "dropped unmapped RIS tags: "+strings.Join(parts, ", "))
	}
	return entries, warnings, nil
}

func risToEntry(tags []risTag) (*Entry, []string) {
	e := &Entry{Type: "misc"}
	var authors, editors, keywords, dropped []string
	var startPage, endPage, secondary, serial, date string

	for _, t := range tags {
		switch t.key {
		case "TY":
			if bt, ok := risTypes[strings.ToUpper(t.val)]; ok {
				e.Type = bt
			}
		case "AU", "A1":
			authors = append(authors, escapeRIS(t.val))
		case "A2", "A3", "A4", "ED":
			editors = append(editors, escapeRIS(t.val))
		case "KW":
			keywords = append(keywords, escapeRIS(t.val))
		case "SP":
			startPage = t.val
		case "EP":
			endPage = t.val
		case "T2":
			secondary = t.val
		case "SN":
			serial = t.val
		case "DA":
			date = t.val
		default:
			field, ok := risFields[t.key]
			if !ok {
				if !risIgnored[t.key] {
					dropped = append(dropped, t.key)
				}
				continue
			}
			// First tag wins: some exporters repeat a tag with a worse value.
			if e.Get(field) == "" {
				e.Set(field, escapeRIS(t.val))
			}
		}
	}

	e.Set("author", strings.Join(authors, " and "))
	e.Set("editor", strings.Join(editors, " and "))
	e.Set("keywords", strings.Join(keywords, ", "))

	// T2 is the containing work: a journal for an article, a book for a chapter.
	if secondary != "" {
		switch e.Type {
		case "inproceedings", "incollection", "inbook":
			if e.Get("booktitle") == "" {
				e.Set("booktitle", escapeRIS(secondary))
			}
		default:
			if e.Get("journal") == "" {
				e.Set("journal", escapeRIS(secondary))
			}
		}
	}

	// SN is an ISBN on a book and an ISSN on a periodical.
	if serial != "" {
		switch e.Type {
		case "book", "incollection", "inbook", "inproceedings":
			e.Set("isbn", serial)
		default:
			e.Set("issn", serial)
		}
	}

	switch {
	case startPage != "" && endPage != "" && startPage != endPage:
		e.Set("pages", startPage+"--"+endPage)
	case startPage != "":
		e.Set("pages", startPage)
	}

	if e.Get("year") == "" {
		e.Set("year", yearFrom(date))
	}

	// Theses and reports name their institution rather than a publisher.
	if pub := e.Get("publisher"); pub != "" {
		switch e.Type {
		case "phdthesis", "mastersthesis":
			e.Set("school", pub)
			e.Del("publisher")
		case "techreport":
			e.Set("institution", pub)
			e.Del("publisher")
		}
	}

	return e, dropped
}

// yearFrom pulls a four digit year out of an RIS date such as "2023/01/01".
func yearFrom(date string) string {
	for i := 0; i+4 <= len(date); i++ {
		if isNumber(date[i : i+4]) {
			return date[i : i+4]
		}
	}
	return ""
}

// escapeRIS makes an RIS value safe to drop into a LaTeX bibliography.
//
// Only & % # are escaped. The math and markup characters $ _ ^ \ { } are left
// alone on purpose: publisher exports do carry real LaTeX (T_c, chemical
// formulae), and mangling that is worse than leaving a rare stray character.
// Braces are escaped only when they are unbalanced, which would otherwise
// corrupt the surrounding entry.
func escapeRIS(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c == '&' || c == '%' || c == '#') && (i == 0 || s[i-1] != '\\') {
			sb.WriteByte('\\')
		}
		sb.WriteByte(c)
	}
	out := sb.String()
	if !bracesBalanced(out) {
		out = strings.NewReplacer("{", `\{`, "}", `\}`).Replace(out)
	}
	return out
}

func bracesBalanced(s string) bool {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}
