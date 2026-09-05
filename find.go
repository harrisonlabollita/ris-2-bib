package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
	"golang.org/x/term"
)

const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiDim     = "\033[2m"
	ansiCyan    = "\033[36m"
	ansiYellow  = "\033[33m"
	ansiGreen   = "\033[32m"
	ansiMagenta = "\033[35m"
)

// candidate is one library entry prepared for matching. Each field is scored
// separately and the best score wins.
//
// Matching one concatenated blob instead ranks badly: fzf rewards compact
// matches, so "gga" scored higher against "Kresse, Georg and ..." than against
// "Generalized Gradient Approximation". Splitting the fields and dropping
// author first names removes that whole class of coincidence.
//
// Fields are lowercased once, at load. fzf's matcher folds the pattern but not
// the haystack, so doing it per keystroke would be both wrong and wasteful.
type candidate struct {
	entry   *Entry
	display string
	fields  []util.Chars
}

// searchFields is what a query is matched against, most specific first. The
// initials of the title are included as a field of their own because physics is
// written in acronyms: "gga", "dmft", "paw", "mlwf".
func searchFields(e *Entry) []string {
	title := plainText(e.Get("title"))
	container := e.Get("journal")
	if container == "" {
		container = e.Get("booktitle")
	}
	return []string{
		authorSurnames(e.Get("author")),
		title,
		wordInitials(title),
		plainText(container),
		wordInitials(plainText(container)),
		plainText(e.Get("keywords")),
		e.Key,
		e.Get("year"),
		normalizeDOI(e.Get("doi")),
	}
}

func newCandidates(entries []*Entry) []candidate {
	out := make([]candidate, len(entries))
	for i, e := range entries {
		fields := searchFields(e)
		chars := make([]util.Chars, 0, len(fields))
		for _, f := range fields {
			if f == "" {
				continue
			}
			chars = append(chars, util.ToChars([]byte(strings.ToLower(f))))
		}
		out[i] = candidate{entry: e, display: displayLine(e), fields: chars}
	}
	return out
}

// authorSurnames reduces an author list to just the surnames. First names are
// pure noise in a search: nobody looks a paper up by "Georg".
func authorSurnames(authors string) string {
	if authors == "" {
		return ""
	}
	var names []string
	for _, a := range strings.Split(authors, " and ") {
		a = strings.TrimSpace(a)
		if i := strings.Index(a, ","); i >= 0 {
			names = append(names, strings.TrimSpace(a[:i]))
			continue
		}
		if fields := strings.Fields(a); len(fields) > 0 {
			names = append(names, fields[len(fields)-1])
		}
	}
	return plainText(strings.Join(names, " "))
}

// wordInitials builds the acronym of a phrase. Words break on any non-letter,
// so "mean-field" contributes both m and f and "dmft" finds "dynamical
// mean-field theory".
func wordInitials(s string) string {
	var sb strings.Builder
	inWord := false
	for _, r := range s {
		letter := r == '\'' || unicode.IsLetter(r)
		if letter && !inWord && r != '\'' {
			sb.WriteRune(r)
		}
		inWord = letter
	}
	return sb.String()
}

// plainText strips the LaTeX that makes a value correct in a bibliography but
// noisy on screen.
func plainText(s string) string {
	r := strings.NewReplacer(
		`\&`, "&", `\%`, "%", `\#`, "#", `\{`, "{", `\}`, "}",
		"{", "", "}", "", `\`, "",
	)
	return r.Replace(s)
}

func displayLine(e *Entry) string {
	author := firstAuthorLast(e.Get("author"))
	if author == "" {
		author = "?"
	}
	if strings.Contains(e.Get("author"), " and ") {
		author += " et al."
	}
	year := e.Get("year")
	if year == "" {
		year = "????"
	}
	title := plainText(e.Get("title"))
	return fmt.Sprintf("%-22s %-4s  %s", truncate(author, 22), year, title)
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 1 {
		return string(runes[:n])
	}
	return string(runes[:n-1]) + "…"
}

type finder struct {
	all     []candidate
	matches []int // indices into all, best first
	query   []rune
	cursor  int // index into matches
	offset  int // first visible row
	slab    *util.Slab
	width   int
	height  int
}

func newFinder(entries []*Entry) *finder {
	f := &finder{
		all:   newCandidates(entries),
		slab:  util.MakeSlab(100*1024, 2048),
		width: 80, height: 24,
	}
	f.filter()
	return f
}

// filter re-ranks the whole library against the current query. A full rescan
// costs about two milliseconds at a few thousand entries, so there is no need
// for the incremental narrowing fzf does.
func (f *finder) filter() {
	pattern := []rune(strings.ToLower(strings.TrimSpace(string(f.query))))
	f.matches = f.matches[:0]

	if len(pattern) == 0 {
		for i := range f.all {
			f.matches = append(f.matches, i)
		}
	} else {
		scores := make(map[int]int, len(f.all))
		for i := range f.all {
			best := 0
			for j := range f.all[i].fields {
				res, _ := algo.FuzzyMatchV2(false, true, true, &f.all[i].fields[j], pattern, false, f.slab)
				if res.Score > best {
					best = res.Score
				}
			}
			if best > 0 {
				scores[i] = best
				f.matches = append(f.matches, i)
			}
		}
		sort.SliceStable(f.matches, func(a, b int) bool {
			ia, ib := f.matches[a], f.matches[b]
			if scores[ia] != scores[ib] {
				return scores[ia] > scores[ib]
			}
			// Ties break towards the more recent paper.
			ya, yb := f.all[ia].entry.Get("year"), f.all[ib].entry.Get("year")
			if ya != yb {
				return ya > yb
			}
			return f.all[ia].entry.Key < f.all[ib].entry.Key
		})
	}

	if f.cursor >= len(f.matches) {
		f.cursor = len(f.matches) - 1
	}
	if f.cursor < 0 {
		f.cursor = 0
	}
}

func (f *finder) selected() *Entry {
	if f.cursor < 0 || f.cursor >= len(f.matches) {
		return nil
	}
	return f.all[f.matches[f.cursor]].entry
}

// listHeight splits the screen between the result list and the preview, giving
// the list the smaller share unless there is little to preview.
func (f *finder) listHeight() int {
	avail := f.height - 3 // prompt plus two rules
	if avail < 4 {
		avail = 4
	}
	h := avail * 45 / 100
	if h < 3 {
		h = 3
	}
	if h > len(f.matches) {
		h = len(f.matches)
	}
	if h < 1 {
		h = 1
	}
	return h
}

func (f *finder) scroll() {
	h := f.listHeight()
	if f.cursor < f.offset {
		f.offset = f.cursor
	}
	if f.cursor >= f.offset+h {
		f.offset = f.cursor - h + 1
	}
	if f.offset < 0 {
		f.offset = 0
	}
}

// line writes one screen row, clearing whatever the previous frame left there.
func line(w io.Writer, s string) {
	fmt.Fprint(w, s, "\033[K\r\n")
}

func (f *finder) render(w io.Writer) {
	f.scroll()
	listH := f.listHeight()
	var sb strings.Builder

	sb.WriteString("\033[H") // home, without the clear that causes flicker

	count := fmt.Sprintf("%d/%d", len(f.matches), len(f.all))
	prompt := ansiCyan + "❯ " + ansiReset + string(f.query) + ansiDim + "▏" + ansiReset
	pad := f.width - 2 - len([]rune(f.query)) - 1 - len(count)
	if pad < 1 {
		pad = 1
	}
	line(&sb, prompt+strings.Repeat(" ", pad)+ansiDim+count+ansiReset)
	line(&sb, ansiDim+strings.Repeat("─", f.width)+ansiReset)

	if len(f.matches) == 0 {
		line(&sb, "  "+ansiDim+"no matches"+ansiReset)
		for i := 1; i < listH; i++ {
			line(&sb, "")
		}
	}
	for row := 0; row < listH; row++ {
		i := f.offset + row
		if i >= len(f.matches) {
			line(&sb, "")
			continue
		}
		text := truncate(f.all[f.matches[i]].display, f.width-2)
		if i == f.cursor {
			line(&sb, ansiCyan+"▌ "+ansiReset+ansiBold+text+ansiReset)
		} else {
			line(&sb, "  "+ansiDim+text+ansiReset)
		}
	}

	line(&sb, ansiDim+strings.Repeat("─", f.width)+ansiReset)
	f.renderPreview(&sb, f.height-3-listH)

	fmt.Fprint(&sb, "\033[J") // drop anything left below the frame
	io.WriteString(w, sb.String())
}

func (f *finder) renderPreview(sb *strings.Builder, rows int) {
	e := f.selected()
	if e == nil || rows <= 0 {
		return
	}
	fields, width := e.OrderedFields()

	line(sb, ansiGreen+"@"+e.Type+"{"+ansiReset+ansiYellow+ansiBold+e.Key+ansiReset+ansiGreen+","+ansiReset)
	rows--

	for _, fl := range fields {
		if rows <= 1 {
			break
		}
		label := fmt.Sprintf("  %-*s", width, fl.Name)
		value := truncate(plainText(fl.Value), f.width-width-8)
		line(sb, ansiMagenta+label+ansiReset+ansiDim+" = "+ansiReset+value)
		rows--
	}
	if rows > 0 {
		line(sb, ansiGreen+"}"+ansiReset)
		rows--
	}
	for ; rows > 0; rows-- {
		line(sb, "")
	}
}

// action is what a keypress asked for.
type action int

const (
	actNone action = iota
	actQuit
	actAccept
	actUp
	actDown
)

func (f *finder) handle(buf []byte) action {
	if len(buf) == 0 {
		return actNone
	}
	switch b := buf[0]; {
	case b == 3, b == 4: // ctrl-c, ctrl-d
		return actQuit
	case b == 13, b == 10: // enter
		return actAccept
	case b == 127, b == 8: // backspace
		if n := len(f.query); n > 0 {
			f.query = f.query[:n-1]
			f.filter()
		}
	case b == 21: // ctrl-u, clear the query
		f.query = f.query[:0]
		f.filter()
	case b == 23: // ctrl-w, delete the last word
		q := strings.TrimRight(string(f.query), " ")
		if i := strings.LastIndex(q, " "); i >= 0 {
			f.query = []rune(q[:i+1])
		} else {
			f.query = f.query[:0]
		}
		f.filter()
	case b == 14: // ctrl-n
		return actDown
	case b == 16: // ctrl-p
		return actUp
	case b == 27:
		if len(buf) == 1 {
			return actQuit // bare escape
		}
		if len(buf) >= 3 && buf[1] == '[' {
			switch buf[2] {
			case 'A':
				return actUp
			case 'B':
				return actDown
			}
		}
	default:
		changed := false
		for _, r := range string(buf) {
			if r >= 32 && r != 127 {
				f.query = append(f.query, r)
				changed = true
			}
		}
		if changed {
			f.filter()
		}
	}
	return actNone
}

// RunFinder drives the interactive picker and returns the chosen entry, or nil
// if the user quit.
//
// The interface is drawn on /dev/tty rather than stdout, so that the selection
// can be piped: `bib | pbcopy` works.
func RunFinder(entries []*Entry) (*Entry, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("no terminal available: %w", err)
	}
	defer tty.Close()

	fd := int(tty.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	defer term.Restore(fd, oldState)

	f := newFinder(entries)
	if w, h, err := term.GetSize(fd); err == nil {
		f.width, f.height = w, h
	}

	fmt.Fprint(tty, "\033[?25l")       // hide the cursor
	defer fmt.Fprint(tty, "\033[?25h") // and put it back
	defer fmt.Fprint(tty, "\033[H\033[J")

	buf := make([]byte, 16)
	for {
		if w, h, err := term.GetSize(fd); err == nil {
			f.width, f.height = w, h // pick up a resize
		}
		f.render(tty)

		n, err := tty.Read(buf)
		if err != nil || n == 0 {
			return nil, err
		}
		switch f.handle(buf[:n]) {
		case actQuit:
			return nil, nil
		case actAccept:
			return f.selected(), nil
		case actUp:
			if f.cursor > 0 {
				f.cursor--
			}
		case actDown:
			if f.cursor < len(f.matches)-1 {
				f.cursor++
			}
		}
	}
}
