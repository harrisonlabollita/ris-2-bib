package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata")

func TestIdentityPrefersDOI(t *testing.T) {
	a := &Entry{Type: "article", Key: "a"}
	a.Set("doi", "10.1103/PhysRevB.101.054101")
	b := &Entry{Type: "article", Key: "b"}
	b.Set("doi", "https://doi.org/10.1103/PhysRevB.101.054101") // same paper, URL form
	if Identity(a) != Identity(b) {
		t.Errorf("DOI forms differ:\n %s\n %s", Identity(a), Identity(b))
	}
}

func TestIdentityFallsBackToArxivThenSlug(t *testing.T) {
	pre := &Entry{Type: "misc", Key: "p"}
	pre.Set("url", "https://arxiv.org/abs/2401.01234")
	if got, want := Identity(pre), "arxiv:2401.01234"; got != want {
		t.Errorf("Identity = %q, want %q", got, want)
	}

	bare := &Entry{Type: "misc", Key: "b"}
	bare.Set("author", "Doe, Jane and Roe, R.")
	bare.Set("year", "2020")
	bare.Set("title", "A Study of Things!")
	if got, want := Identity(bare), "slug:doe:2020:astudyofthings"; got != want {
		t.Errorf("Identity = %q, want %q", got, want)
	}
}

func TestAddEntriesSkipsDuplicates(t *testing.T) {
	mk := func(doi, title string) *Entry {
		e := &Entry{Type: "article"}
		e.Set("author", "Doe, Jane")
		e.Set("year", "2020")
		e.Set("title", title)
		e.Set("doi", doi)
		return e
	}
	lib, added, _ := AddEntries(nil, []*Entry{mk("10.1/a", "First")})
	if len(added) != 1 {
		t.Fatalf("first add: got %d added, want 1", len(added))
	}

	// The same DOI arriving again, with different surrounding metadata.
	lib, added, skipped := AddEntries(lib, []*Entry{mk("10.1/a", "First, reprinted")})
	if len(added) != 0 || len(skipped) != 1 {
		t.Errorf("re-add: %d added, %d skipped; want 0 and 1", len(added), len(skipped))
	}
	if len(lib) != 1 {
		t.Errorf("library grew to %d entries", len(lib))
	}
}

func TestMakeKey(t *testing.T) {
	mk := func(author, year, title string) *Entry {
		e := &Entry{Type: "article"}
		e.Set("author", author)
		e.Set("year", year)
		e.Set("title", title)
		return e
	}
	for _, tc := range []struct {
		name string
		e    *Entry
		want string
	}{
		{"plain", mk("Georges, Antoine and Kotliar, G.", "1996", "Dynamical mean-field theory"), "Georges1996Dynamical"},
		{"stopword skipped", mk("Doe, J.", "2020", "The quick brown fox"), "Doe2020quick"},
		{"first-last form", mk("Antoine Georges", "1996", "Hubbard model"), "Georges1996Hubbard"},
		{"accents folded", mk("Müller, K.", "1986", "Über Supraleitung"), "Muller1986Uber"},
		{"no author", mk("", "2001", "Anonymous report"), "Anon2001Anonymous"},
	} {
		if got := MakeKey(tc.e, map[string]bool{}); got != tc.want {
			t.Errorf("%s: MakeKey = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestMakeKeyCollisions(t *testing.T) {
	e := &Entry{Type: "article"}
	e.Set("author", "Doe, J.")
	e.Set("year", "2020")
	e.Set("title", "Same title")

	taken := map[string]bool{}
	for _, want := range []string{"Doe2020Same", "Doe2020Samea", "Doe2020Sameb"} {
		got := MakeKey(e, taken)
		if got != want {
			t.Fatalf("MakeKey = %q, want %q", got, want)
		}
		taken[got] = true
	}
}

// A cite key that arrives with a .bib import is kept, because it may already be
// referenced by a manuscript.
func TestAddKeepsImportedKey(t *testing.T) {
	e := &Entry{Type: "article", Key: "TheirKey2020"}
	e.Set("title", "T")
	e.Set("doi", "10.9/z")
	_, added, _ := AddEntries(nil, []*Entry{e})
	if added[0].Key != "TheirKey2020" {
		t.Errorf("key = %q, want it preserved", added[0].Key)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "master.bib")
	e := &Entry{Type: "article", Key: "k"}
	e.Set("title", "T")
	e.Set("author", "Doe, J.")

	if err := SaveLibrary(path, []*Entry{e}); err != nil {
		t.Fatalf("SaveLibrary: %v", err)
	}
	got, err := LoadLibrary(path)
	if err != nil {
		t.Fatalf("LoadLibrary: %v", err)
	}
	if len(got) != 1 || got[0].Get("title") != "T" {
		t.Errorf("round trip lost data: %+v", got)
	}
}

func TestLoadMissingLibraryIsEmpty(t *testing.T) {
	got, err := LoadLibrary(filepath.Join(t.TempDir(), "absent.bib"))
	if err != nil || got != nil {
		t.Errorf("missing library: got %v, %v; want nil, nil", got, err)
	}
}

// TestGolden converts a realistic export end to end. Run with -update to
// regenerate after an intentional change, then read the diff.
func TestGolden(t *testing.T) {
	src, err := os.ReadFile("testdata/sample.ris")
	if err != nil {
		t.Fatal(err)
	}
	entries, _, err := ParseRIS(string(src))
	if err != nil {
		t.Fatal(err)
	}
	taken := map[string]bool{}
	for _, e := range entries {
		e.Key = MakeKey(e, taken)
		taken[e.Key] = true
	}
	got := FormatLibrary(entries)

	golden := "testdata/sample.golden.bib"
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log("wrote " + golden)
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("output differs from %s:\n--- got ---\n%s", golden, got)
	}
}
