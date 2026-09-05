package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func playgroundEntries(t *testing.T) []*Entry {
	t.Helper()
	src, err := os.ReadFile("playground/seed.ris")
	if err != nil {
		t.Skip("no playground data")
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
	return entries
}

func (f *finder) type_(s string) {
	f.handle([]byte(s))
}

func (f *finder) topKeys(n int) []string {
	var out []string
	for i := 0; i < n && i < len(f.matches); i++ {
		out = append(out, f.all[f.matches[i]].entry.Key)
	}
	return out
}

func TestFilterFindsAcrossFields(t *testing.T) {
	f := newFinder(playgroundEntries(t))
	if len(f.matches) != len(f.all) {
		t.Errorf("empty query shows %d of %d", len(f.matches), len(f.all))
	}

	for _, tc := range []struct {
		query, want string
		why         string
	}{
		{"wannier", "Marzari2012Maximally", "title word"},
		{"gga", "Perdew1996Generalized", "acronym over title initials"},
		{"paw", "Blochl1994Projector", "acronym spanning a hyphenated word"},
		{"mlwf", "Marzari2012Maximally", "four-letter acronym"},
		{"kresse", "Kresse1996Efficient", "author surname"},
		{"blochl", "Blochl1994Projector", "accented author, folded in the key"},
		{"10.1038/s41586-019", "Li2019Superconductivity", "DOI, which the list never shows"},
		{"wannier90", "Mostofi2008wannier90", "exact token with digits"},
		{"revmodphys", "", "no match is fine, it just must not crash"},
	} {
		f.query = f.query[:0]
		f.type_(tc.query)
		got := f.topKeys(1)
		if tc.want == "" {
			continue
		}
		if len(got) == 0 || got[0] != tc.want {
			t.Errorf("query %q (%s): top hit %v, want %s", tc.query, tc.why, got, tc.want)
		}
	}
}

// A concatenated haystack ranked "Kresse, Georg and ..." above "Generalized
// Gradient Approximation" for the query "gga", because fzf rewards compact
// matches over spread-out ones. Scoring fields separately, and searching author
// surnames rather than full names, is what fixes it.
func TestRankingIgnoresAuthorFirstNames(t *testing.T) {
	f := newFinder(playgroundEntries(t))
	f.type_("gga")
	for i, key := range f.topKeys(len(f.matches)) {
		if key == "Perdew1996Generalized" {
			break
		}
		if key == "Kresse1996Efficient" || key == "Bednorz1986Possible" {
			t.Fatalf("first-name coincidence %q outranks the GGA paper (position %d)", key, i)
		}
	}
}

func TestAuthorSurnames(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Kresse, Georg and Furthmüller, Jürgen", "Kresse Furthmüller"},
		{"Antoine Georges and Gabriel Kotliar", "Georges Kotliar"},
		{"", ""},
	} {
		if got := authorSurnames(tc.in); got != tc.want {
			t.Errorf("authorSurnames(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestWordInitials(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Generalized Gradient Approximation Made Simple", "GGAMS"},
		// The hyphen has to break the word, or "dmft" never finds this.
		{"dynamical mean-field theory", "dmft"},
		{"Projector augmented-wave method", "Pawm"},
		{"", ""},
	} {
		if got := wordInitials(tc.in); got != tc.want {
			t.Errorf("wordInitials(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFilterCaseInsensitive(t *testing.T) {
	// The haystack is folded at load; if that ever regresses, capitalised
	// titles silently stop matching lowercase queries.
	f := newFinder(playgroundEntries(t))
	f.type_("WANNIER")
	upper := len(f.matches)
	f.query = f.query[:0]
	f.type_("wannier")
	if lower := len(f.matches); lower != upper || lower == 0 {
		t.Errorf("case sensitivity leaked: %d upper vs %d lower", upper, lower)
	}
}

func TestCursorStaysInRange(t *testing.T) {
	f := newFinder(playgroundEntries(t))
	f.cursor = len(f.matches) - 1
	f.type_("zzzzzzzz") // nothing matches
	if f.selected() != nil {
		t.Error("selected() should be nil with no matches")
	}
	f.handle([]byte{21}) // ctrl-u, back to the full list
	if f.cursor < 0 || f.cursor >= len(f.matches) {
		t.Errorf("cursor %d out of range for %d matches", f.cursor, len(f.matches))
	}
}

func TestHandleKeys(t *testing.T) {
	f := newFinder(playgroundEntries(t))
	f.type_("dmft theory")
	if a := f.handle([]byte{23}); a != actNone || string(f.query) != "dmft " {
		t.Errorf("ctrl-w gave %q", string(f.query))
	}
	if a := f.handle([]byte{21}); a != actNone || string(f.query) != "" {
		t.Errorf("ctrl-u gave %q", string(f.query))
	}
	for _, tc := range []struct {
		in   []byte
		want action
		name string
	}{
		{[]byte{3}, actQuit, "ctrl-c"},
		{[]byte{27}, actQuit, "bare escape"},
		{[]byte{13}, actAccept, "enter"},
		{[]byte{27, '[', 'A'}, actUp, "up arrow"},
		{[]byte{27, '[', 'B'}, actDown, "down arrow"},
		{[]byte{16}, actUp, "ctrl-p"},
		{[]byte{14}, actDown, "ctrl-n"},
	} {
		if got := f.handle(tc.in); got != tc.want {
			t.Errorf("%s: action %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestRenderFrame is a smoke test that also prints the interface, so a frame
// can be eyeballed with `go test -run RenderFrame -v`.
func TestRenderFrame(t *testing.T) {
	f := newFinder(playgroundEntries(t))
	f.width, f.height = 96, 30
	f.type_("dmft")
	f.cursor = 1

	var sb strings.Builder
	f.render(&sb)
	out := sb.String()

	// The prompt carries an ANSI reset between the marker and the query text.
	if !strings.Contains(out, "❯") || !strings.Contains(out, "dmft") {
		t.Error("prompt missing from frame")
	}
	if sel := f.selected(); sel == nil || !strings.Contains(out, sel.Key) {
		t.Error("preview does not show the selected entry")
	}
	t.Log("\n" + strings.ReplaceAll(out, "\r\n", "\n"))
}

// BenchmarkFilter measures one keystroke against a library far larger than any
// real one. The picker redraws on every keypress, so this is the latency budget.
func BenchmarkFilter(b *testing.B) {
	words := []string{"dynamical", "mean-field", "theory", "electronic", "structure",
		"correlated", "superconductivity", "nickelate", "Wannier", "functions",
		"density", "functional", "Hubbard", "lattice", "spectral", "impurity"}
	entries := make([]*Entry, 3000)
	for i := range entries {
		e := &Entry{Type: "article", Key: fmt.Sprintf("Author%d%dWord", i%400, 1970+i%55)}
		e.Set("author", fmt.Sprintf("Author%d, A. B. and Other%d, C.", i%400, i%37))
		e.Set("year", fmt.Sprint(1970+i%55))
		title := ""
		for j := 0; j < 9; j++ {
			title += words[(i*7+j*3)%len(words)] + " "
		}
		e.Set("title", title)
		e.Set("journal", "Physical Review B")
		e.Set("doi", fmt.Sprintf("10.1103/PhysRevB.%d.%d", 80+i%40, 100000+i))
		entries[i] = e
	}
	f := newFinder(entries)
	for _, pat := range []string{"d", "dmft", "nickelsup"} {
		b.Run("pat="+pat, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				f.query = []rune(pat)
				f.filter()
			}
		})
	}
}
