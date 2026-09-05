package main

import (
	"strings"
	"testing"
)

func mustParseBib(t *testing.T, src string) []*Entry {
	t.Helper()
	entries, _, err := ParseBibTeX(src)
	if err != nil {
		t.Fatalf("ParseBibTeX: %v", err)
	}
	return entries
}

func TestParseValueDelimiters(t *testing.T) {
	src := `@article{key1,
  title   = {The {DFT}+{DMFT} Method},
  journal = "Phys. Rev. B",
  year    = 1996,
  note    = {a \} escaped brace},
}`
	e := mustParseBib(t, src)
	if len(e) != 1 {
		t.Fatalf("got %d entries, want 1", len(e))
	}
	for _, tc := range []struct{ field, want string }{
		{"title", "The {DFT}+{DMFT} Method"}, // nested braces preserved verbatim
		{"journal", "Phys. Rev. B"},          // quoted
		{"year", "1996"},                     // bare number
		{"note", `a \} escaped brace`},       // an escaped brace does not close the group
	} {
		if got := e[0].Get(tc.field); got != tc.want {
			t.Errorf("%s = %q, want %q", tc.field, got, tc.want)
		}
	}
}

func TestParseStringMacroAndConcat(t *testing.T) {
	src := `@string{prb = "Phys. Rev. B"}
@string{lett = " Letters"}
@article{k,
  title   = {T},
  journal = prb # lett,
}`
	e := mustParseBib(t, src)
	if got, want := e[0].Get("journal"), "Phys. Rev. B Letters"; got != want {
		t.Errorf("journal = %q, want %q", got, want)
	}
}

func TestParseSkipsNonEntryAt(t *testing.T) {
	// A bare '@' in running text between entries must not derail the parser.
	src := `Contact me@example.com about this.
@article{k, title = {T}}`
	if e := mustParseBib(t, src); len(e) != 1 || e[0].Key != "k" {
		t.Fatalf("got %+v, want the single entry k", e)
	}
}

func TestParseUnbalancedIsAnError(t *testing.T) {
	// Both of these are malformed. The parser must say so rather than silently
	// truncating the value, which is how bibliographies quietly lose data.
	for _, src := range []string{
		`@article{k, title = {unterminated}`,
		`@article{k, title = {a } stray close brace}`,
	} {
		if _, _, err := ParseBibTeX(src); err == nil {
			t.Errorf("want an error for %q, got nil", src)
		}
	}
}

func TestFormatFieldOrder(t *testing.T) {
	e := &Entry{Type: "article", Key: "k"}
	// Deliberately inserted out of order, with an unknown field last.
	for _, f := range []Field{
		{"zzz", "unknown"}, {"doi", "10.1/x"}, {"title", "T"},
		{"author", "A"}, {"aaa", "unknown"},
	} {
		e.Set(f.Name, f.Value)
	}
	var names []string
	for _, line := range strings.Split(e.String(), "\n") {
		if i := strings.Index(line, " ="); i > 0 {
			names = append(names, strings.TrimSpace(line[:i]))
		}
	}
	want := []string{"author", "title", "doi", "aaa", "zzz"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("field order = %v, want %v", names, want)
	}
}

// TestFormatIdempotent is the property that makes `bib fmt` safe to run in a
// pre-commit hook: formatting an already formatted library changes nothing.
func TestFormatIdempotent(t *testing.T) {
	src := `@article{b1, title={T1}, author={Doe, J.}, year=2020, doi={10.1/a}}
@book{a1, title={T2}, author={Roe, R.}, publisher={Pub}, year=1999}
@misc{c1, title={Escaped \& kept}, note={brace \{ left}}`

	once := FormatLibrary(mustParseBib(t, src))
	twice := FormatLibrary(mustParseBib(t, once))
	if once != twice {
		t.Errorf("format is not idempotent:\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
	// Sorted by cite key, so the library file stays stable under git.
	if !strings.HasPrefix(once, "@book{a1,") {
		t.Errorf("entries are not sorted by key:\n%s", once)
	}
}

func TestRoundTripPreservesValues(t *testing.T) {
	src := `@article{k,
  title    = {A {NiO} study of $T_c$ \& friends},
  author   = {M{\"u}ller, Karl},
  pages    = {13--125},
  abstract = {Contains a \} brace and 100\% coverage},
}`
	before := mustParseBib(t, src)[0]
	after := mustParseBib(t, FormatLibrary([]*Entry{before}))[0]

	if after.Type != before.Type || after.Key != before.Key {
		t.Errorf("header changed: %s{%s} -> %s{%s}", before.Type, before.Key, after.Type, after.Key)
	}
	for _, f := range before.Fields {
		if got := after.Get(f.Name); got != f.Value {
			t.Errorf("%s: %q -> %q", f.Name, f.Value, got)
		}
	}
}

func TestSetEmptyDeletes(t *testing.T) {
	e := &Entry{Type: "misc", Key: "k"}
	e.Set("note", "x")
	e.Set("note", "")
	if got := e.Get("note"); got != "" || len(e.Fields) != 0 {
		t.Errorf("Set to empty left %d fields (%q)", len(e.Fields), got)
	}
}
