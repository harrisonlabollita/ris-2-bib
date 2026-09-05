package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// LibraryPath resolves the master bibliography: $SHELF_FILE, else the XDG data
// directory.
func LibraryPath() string {
	if p := os.Getenv("SHELF_FILE"); p != "" {
		return p
	}
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "master.bib"
		}
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "shelf", "master.bib")
}

// LoadLibrary reads the master file. A missing file is an empty library, not an
// error, so the first `shelf add` works with no setup.
func LoadLibrary(path string) ([]*Entry, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	entries, warnings, err := ParseBibTeX(string(data))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "shelf: %s: %s\n", path, w)
	}
	return entries, nil
}

// SaveLibrary writes entries atomically, so an interrupted write cannot leave a
// truncated bibliography behind.
func SaveLibrary(path string, entries []*Entry) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".shelf-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := io.WriteString(tmp, FormatLibrary(entries)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// ReadSource loads entries from a .bib or .ris file, or from stdin when path is
// "-". The format is sniffed from the content rather than the extension.
func ReadSource(path string) ([]*Entry, []string, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, nil, err
	}
	if strings.HasPrefix(strings.TrimLeft(string(data), " \t\r\n"), "@") {
		return ParseBibTeX(string(data))
	}
	return ParseRIS(string(data))
}

// Identity is the value duplicate detection compares on: a DOI when there is
// one, then an arXiv id, then a slug built from author, year and title.
func Identity(e *Entry) string {
	if doi := normalizeDOI(e.Get("doi")); doi != "" {
		return "doi:" + doi
	}
	if id := arxivID(e); id != "" {
		return "arxiv:" + id
	}
	return "slug:" + slug(firstAuthorLast(e.Get("author"))) + ":" + e.Get("year") + ":" + slug(e.Get("title"))
}

func normalizeDOI(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, prefix := range []string{"https://doi.org/", "http://doi.org/", "https://dx.doi.org/", "http://dx.doi.org/", "doi:"} {
		s = strings.TrimPrefix(s, prefix)
	}
	return s
}

func arxivID(e *Entry) string {
	if v := e.Get("eprint"); v != "" {
		return strings.ToLower(strings.TrimPrefix(v, "arXiv:"))
	}
	const marker = "arxiv.org/abs/"
	if u := e.Get("url"); strings.Contains(u, marker) {
		id := u[strings.Index(u, marker)+len(marker):]
		return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(id), "/"))
	}
	return ""
}

// firstAuthorLast extracts the surname of the first author from a BibTeX author
// list, handling both "Last, First" and "First Last".
func firstAuthorLast(authors string) string {
	first := authors
	if i := strings.Index(authors, " and "); i >= 0 {
		first = authors[:i]
	}
	first = strings.TrimSpace(first)
	if i := strings.Index(first, ","); i >= 0 {
		return strings.TrimSpace(first[:i])
	}
	if fields := strings.Fields(first); len(fields) > 0 {
		return fields[len(fields)-1]
	}
	return ""
}

func slug(s string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(asciiFold(s)) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func keepAlnum(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

var foldMap = map[rune]string{
	'á': "a", 'à': "a", 'â': "a", 'ä': "a", 'ã': "a", 'å': "a", 'ā': "a", 'ą': "a",
	'é': "e", 'è': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ę': "e", 'ě': "e",
	'í': "i", 'ì': "i", 'î': "i", 'ï': "i", 'ī': "i",
	'ó': "o", 'ò': "o", 'ô': "o", 'ö': "o", 'õ': "o", 'ø': "o", 'ő': "o",
	'ú': "u", 'ù': "u", 'û': "u", 'ü': "u", 'ū': "u", 'ů': "u",
	'ñ': "n", 'ń': "n", 'ç': "c", 'ć': "c", 'č': "c", 'ß': "ss",
	'ł': "l", 'š': "s", 'ś': "s", 'ž': "z", 'ź': "z", 'ż': "z",
	'ý': "y", 'ÿ': "y", 'ď': "d", 'ť': "t", 'ř': "r",
}

// asciiFold reduces accented Latin letters to ASCII so that cite keys stay
// typeable. Characters with no mapping are dropped: they cannot appear in a key.
func asciiFold(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r < 128 {
			sb.WriteRune(r)
			continue
		}
		lower := unicode.ToLower(r)
		replacement, ok := foldMap[lower]
		if !ok {
			continue
		}
		if lower != r {
			replacement = strings.ToUpper(replacement[:1]) + replacement[1:]
		}
		sb.WriteString(replacement)
	}
	return sb.String()
}

// keyStopwords are skipped when picking the title word for a cite key; they
// carry no information about which paper it is.
var keyStopwords = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "from": true,
	"that": true, "this": true, "are": true, "was": true, "were": true,
	"but": true, "not": true, "its": true, "via": true, "using": true,
}

// MakeKey builds a NameYearWord cite key that is not already in taken.
//
// Keys are assigned once, at import, and never recomputed. A key that has
// reached a manuscript has to keep resolving, so no later change of scheme may
// be applied retroactively.
func MakeKey(e *Entry, taken map[string]bool) string {
	name := keepAlnum(asciiFold(firstAuthorLast(e.Get("author"))))
	if name == "" {
		name = "Anon"
	}

	word := ""
	for _, w := range strings.Fields(e.Get("title")) {
		w = keepAlnum(asciiFold(w))
		if len(w) >= 3 && !keyStopwords[strings.ToLower(w)] {
			word = w
			break
		}
	}

	base := name + keepAlnum(e.Get("year")) + word
	key := base
	for i := 0; taken[key]; i++ {
		if i < 26 {
			key = base + string(rune('a'+i))
		} else {
			key = fmt.Sprintf("%s%d", base, i)
		}
	}
	return key
}

// AddEntries merges incoming into lib, assigning cite keys and skipping
// anything already present. It returns the merged library alongside what was
// added and what was skipped, so the caller can report honestly.
func AddEntries(lib, incoming []*Entry) (merged, added, skipped []*Entry) {
	seen := make(map[string]bool, len(lib))
	taken := make(map[string]bool, len(lib))
	for _, e := range lib {
		seen[Identity(e)] = true
		taken[e.Key] = true
	}

	merged = lib
	for _, e := range incoming {
		id := Identity(e)
		if seen[id] {
			skipped = append(skipped, e)
			continue
		}
		// A .bib import arrives with a key already; keep it unless it clashes.
		if e.Key == "" || taken[e.Key] {
			e.Key = MakeKey(e, taken)
		}
		seen[id] = true
		taken[e.Key] = true
		merged = append(merged, e)
		added = append(added, e)
	}
	return merged, added, skipped
}
