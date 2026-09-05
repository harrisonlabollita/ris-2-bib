package main

import (
	"fmt"
	"sort"
	"strings"
)

// Entry is one BibTeX record.
//
// Field values are stored as they appear between the delimiters and are assumed
// to already be LaTeX. Conversion from foreign formats happens at the import
// boundary, never at write time, so that formatting an already formatted file
// is a no-op.
type Entry struct {
	Type   string // lowercase, without the '@'
	Key    string
	Fields []Field
}

// Field is one name/value pair. Names are lowercased on parse.
type Field struct {
	Name  string
	Value string
}

func (e *Entry) Get(name string) string {
	for _, f := range e.Fields {
		if f.Name == name {
			return f.Value
		}
	}
	return ""
}

// Set replaces the value of name, appending the field if it is absent. An empty
// value deletes the field, so callers can Set unconditionally.
func (e *Entry) Set(name, value string) {
	if value == "" {
		e.Del(name)
		return
	}
	for i := range e.Fields {
		if e.Fields[i].Name == name {
			e.Fields[i].Value = value
			return
		}
	}
	e.Fields = append(e.Fields, Field{name, value})
}

func (e *Entry) Del(name string) {
	kept := e.Fields[:0]
	for _, f := range e.Fields {
		if f.Name != name {
			kept = append(kept, f)
		}
	}
	e.Fields = kept
}

// canonicalOrder is the field order used when writing. Anything not listed is
// written afterwards in alphabetical order.
var canonicalOrder = []string{
	"author", "editor", "title", "booktitle", "journal", "series",
	"volume", "number", "pages", "eid", "year", "month",
	"publisher", "address", "school", "institution", "edition", "howpublished",
	"doi", "url", "eprint", "archiveprefix", "primaryclass",
	"issn", "isbn", "language", "note", "keywords", "abstract", "file",
}

var orderIndex = func() map[string]int {
	m := make(map[string]int, len(canonicalOrder))
	for i, name := range canonicalOrder {
		m[name] = i
	}
	return m
}()

func fieldRank(name string) int {
	if i, ok := orderIndex[name]; ok {
		return i
	}
	return len(canonicalOrder)
}

// OrderedFields returns the fields in the order they are written: known
// bibliographic fields first, then anything else alphabetically. The width is
// the longest field name, for aligning '='.
func (e *Entry) OrderedFields() (fields []Field, width int) {
	fields = append([]Field(nil), e.Fields...)
	sort.SliceStable(fields, func(i, j int) bool {
		ri, rj := fieldRank(fields[i].Name), fieldRank(fields[j].Name)
		if ri != rj {
			return ri < rj
		}
		return fields[i].Name < fields[j].Name
	})
	for _, f := range fields {
		if len(f.Name) > width {
			width = len(f.Name)
		}
	}
	return fields, width
}

// String renders the entry in canonical layout: bibliographic field order, '='
// aligned, and a trailing comma so adding a field is a one-line diff.
func (e *Entry) String() string {
	fields, width := e.OrderedFields()

	var sb strings.Builder
	fmt.Fprintf(&sb, "@%s{%s,\n", e.Type, e.Key)
	for _, f := range fields {
		fmt.Fprintf(&sb, "  %-*s = {%s},\n", width, f.Name, f.Value)
	}
	sb.WriteString("}\n")
	return sb.String()
}

// FormatLibrary renders entries sorted by cite key, which keeps the master file
// stable under git.
func FormatLibrary(entries []*Entry) string {
	sorted := append([]*Entry(nil), entries...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Key < sorted[j].Key })

	var sb strings.Builder
	for i, e := range sorted {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(e.String())
	}
	return sb.String()
}

type parser struct {
	src      string
	pos      int
	macros   map[string]string
	warnings []string
}

func (p *parser) warnf(format string, args ...any) {
	p.warnings = append(p.warnings, fmt.Sprintf(format, args...))
}

// ParseBibTeX parses BibTeX source, returning entries in file order plus any
// non-fatal complaints. @string macros are expanded in place; @comment and
// @preamble blocks are skipped with a warning.
func ParseBibTeX(src string) ([]*Entry, []string, error) {
	p := &parser{src: src, macros: map[string]string{}}
	var entries []*Entry

	for p.seekAt() {
		kind := strings.ToLower(p.ident())
		p.skipSpace()
		if p.pos >= len(p.src) {
			break
		}
		switch p.src[p.pos] {
		case '{':
			// fall through to the block below
		case '(':
			p.warnf("skipped parenthesis-delimited @%s block", kind)
			continue
		default:
			continue // a bare '@' in running text, e.g. an email address
		}

		switch kind {
		case "string":
			if err := p.macro(); err != nil {
				return nil, p.warnings, err
			}
		case "comment", "preamble":
			if _, err := p.balanced(); err != nil {
				return nil, p.warnings, err
			}
			p.warnf("skipped @%s block", kind)
		default:
			p.pos++ // consume '{'
			e, err := p.entry(kind)
			if err != nil {
				return nil, p.warnings, err
			}
			entries = append(entries, e)
		}
	}
	return entries, p.warnings, nil
}

// seekAt advances to just past the next '@', reporting whether one was found.
func (p *parser) seekAt() bool {
	i := strings.IndexByte(p.src[p.pos:], '@')
	if i < 0 {
		p.pos = len(p.src)
		return false
	}
	p.pos += i + 1
	return true
}

func (p *parser) skipSpace() {
	for p.pos < len(p.src) {
		switch p.src[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

func isIdentByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
		strings.IndexByte("-_.:+/'", c) >= 0
}

func (p *parser) ident() string {
	p.skipSpace()
	start := p.pos
	for p.pos < len(p.src) && isIdentByte(p.src[p.pos]) {
		p.pos++
	}
	return p.src[start:p.pos]
}

// macro reads an @string definition. p.pos is at the opening brace.
func (p *parser) macro() error {
	p.pos++
	name := strings.ToLower(p.ident())
	p.skipSpace()
	if p.pos >= len(p.src) || p.src[p.pos] != '=' {
		return fmt.Errorf("@string %q: expected '='", name)
	}
	p.pos++
	value, err := p.value()
	if err != nil {
		return fmt.Errorf("@string %q: %w", name, err)
	}
	p.macros[name] = value
	p.skipSpace()
	if p.pos < len(p.src) && p.src[p.pos] == '}' {
		p.pos++
	}
	return nil
}

// entry reads the body of an entry. p.pos is just past the opening brace.
func (p *parser) entry(kind string) (*Entry, error) {
	e := &Entry{Type: kind}

	// The cite key runs to the first comma, or to the closing brace when the
	// entry has no fields at all.
	start := p.pos
	for p.pos < len(p.src) && p.src[p.pos] != ',' && p.src[p.pos] != '}' {
		p.pos++
	}
	e.Key = strings.TrimSpace(p.src[start:p.pos])

	for p.pos < len(p.src) {
		if p.src[p.pos] == '}' {
			p.pos++
			return e, nil
		}
		p.pos++ // consume ','
		p.skipSpace()
		if p.pos < len(p.src) && p.src[p.pos] == '}' {
			p.pos++ // trailing comma before the close
			return e, nil
		}
		name := strings.ToLower(p.ident())
		if name == "" {
			return nil, fmt.Errorf("entry %q: expected a field name", e.Key)
		}
		p.skipSpace()
		if p.pos >= len(p.src) || p.src[p.pos] != '=' {
			return nil, fmt.Errorf("entry %q: expected '=' after field %q", e.Key, name)
		}
		p.pos++
		value, err := p.value()
		if err != nil {
			return nil, fmt.Errorf("entry %q field %q: %w", e.Key, name, err)
		}
		e.Set(name, value)
		p.skipSpace()
	}
	return nil, fmt.Errorf("entry %q: unexpected end of input", e.Key)
}

// value reads a field value: a braced group, a quoted string, a number, or a
// macro name, optionally concatenated with '#'.
func (p *parser) value() (string, error) {
	var sb strings.Builder
	for {
		p.skipSpace()
		if p.pos >= len(p.src) {
			return "", fmt.Errorf("unexpected end of input in value")
		}
		switch p.src[p.pos] {
		case '{':
			part, err := p.balanced()
			if err != nil {
				return "", err
			}
			sb.WriteString(part)
		case '"':
			part, err := p.quoted()
			if err != nil {
				return "", err
			}
			sb.WriteString(part)
		default:
			word := p.ident()
			if word == "" {
				return "", fmt.Errorf("unexpected %q in value", string(p.src[p.pos]))
			}
			if expansion, ok := p.macros[strings.ToLower(word)]; ok {
				sb.WriteString(expansion)
			} else {
				if !isNumber(word) {
					p.warnf("undefined macro %q, kept literally", word)
				}
				sb.WriteString(word)
			}
		}

		p.skipSpace()
		if p.pos < len(p.src) && p.src[p.pos] == '#' {
			p.pos++
			continue
		}
		return sb.String(), nil
	}
}

func isNumber(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

// balanced reads a {...} group and returns its contents. p.pos is at the open
// brace; on return it is just past the matching close.
func (p *parser) balanced() (string, error) {
	depth := 0
	start := p.pos + 1
	for p.pos < len(p.src) {
		switch p.src[p.pos] {
		case '\\':
			p.pos++ // the escaped character cannot close a group
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				inner := p.src[start:p.pos]
				p.pos++
				return inner, nil
			}
		}
		p.pos++
	}
	return "", fmt.Errorf("unbalanced '{'")
}

// quoted reads a "..." value. Braces inside it still nest, so a quote within a
// braced group does not terminate the value.
func (p *parser) quoted() (string, error) {
	p.pos++
	start := p.pos
	depth := 0
	for p.pos < len(p.src) {
		switch p.src[p.pos] {
		case '\\':
			p.pos++
		case '{':
			depth++
		case '}':
			depth--
		case '"':
			if depth == 0 {
				inner := p.src[start:p.pos]
				p.pos++
				return inner, nil
			}
		}
		p.pos++
	}
	return "", fmt.Errorf("unterminated quoted value")
}
