package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const usage = `bib - a single-file bibliography on the command line

usage:
  bib add [-to FILE] [-n] <file.ris|file.bib|->...   add references to the library
  bib fmt [FILE]                                     rewrite a .bib in canonical form
  bib convert <file.ris|->...                        convert to BibTeX on stdout
  bib browse [DIR]                                   page through .ris files, deleting rejects

The library defaults to $BIB_FILE, else $XDG_DATA_HOME/bib/master.bib.
`

func main() {
	log.SetFlags(0)
	log.SetPrefix("bib: ")

	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "add":
		err = cmdAdd(os.Args[2:])
	case "fmt":
		err = cmdFmt(os.Args[2:])
	case "convert":
		err = cmdConvert(os.Args[2:])
	case "browse":
		err = cmdBrowse(os.Args[2:])
	case "help", "-h", "--help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "bib: unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		log.Fatal(err)
	}
}

// readAll loads every named source, reporting parse warnings to stderr but
// continuing, so one malformed file does not sink a batch.
func readAll(paths []string) ([]*Entry, error) {
	var all []*Entry
	for _, path := range paths {
		entries, warnings, err := ReadSource(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "bib: %s: %s\n", path, w)
		}
		all = append(all, entries...)
	}
	return all, nil
}

func cmdAdd(argv []string) error {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	to := fs.String("to", "", "library file to add to (default: the master library)")
	dry := fs.Bool("n", false, "print what would be added without writing")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("add: no input files")
	}

	path := *to
	if path == "" {
		path = LibraryPath()
	}
	lib, err := LoadLibrary(path)
	if err != nil {
		return err
	}
	incoming, err := readAll(fs.Args())
	if err != nil {
		return err
	}

	merged, added, skipped := AddEntries(lib, incoming)
	for _, e := range added {
		fmt.Printf("+ %s\n", e.Key)
	}
	for _, e := range skipped {
		fmt.Printf("= %s (already in library)\n", firstLine(e))
	}

	if *dry {
		fmt.Fprintf(os.Stderr, "bib: dry run, %s not written\n", path)
		return nil
	}
	if len(added) == 0 {
		return nil
	}
	if err := SaveLibrary(path, merged); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "bib: %d added, %d duplicate, %d total in %s\n",
		len(added), len(skipped), len(merged), path)
	return nil
}

func cmdFmt(argv []string) error {
	fs := flag.NewFlagSet("fmt", flag.ExitOnError)
	if err := fs.Parse(argv); err != nil {
		return err
	}
	path := LibraryPath()
	if fs.NArg() > 0 {
		path = fs.Arg(0)
	}
	entries, err := LoadLibrary(path)
	if err != nil {
		return err
	}
	if entries == nil {
		return fmt.Errorf("%s: no entries", path)
	}
	return SaveLibrary(path, entries)
}

func cmdConvert(argv []string) error {
	fs := flag.NewFlagSet("convert", flag.ExitOnError)
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("convert: no input files")
	}
	entries, err := readAll(fs.Args())
	if err != nil {
		return err
	}
	taken := map[string]bool{}
	for _, e := range entries {
		if e.Key == "" || taken[e.Key] {
			e.Key = MakeKey(e, taken)
		}
		taken[e.Key] = true
	}
	fmt.Print(FormatLibrary(entries))
	return nil
}

func cmdBrowse(argv []string) error {
	fs := flag.NewFlagSet("browse", flag.ExitOnError)
	if err := fs.Parse(argv); err != nil {
		return err
	}
	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.ris"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no .ris files in %s", dir)
	}
	RunPager(files)
	return nil
}

// firstLine describes an entry compactly for duplicate reports, where the
// incoming entry has no cite key of its own yet.
func firstLine(e *Entry) string {
	title := e.Get("title")
	if len(title) > 60 {
		title = title[:57] + "..."
	}
	author := firstAuthorLast(e.Get("author"))
	if author == "" {
		author = "?"
	}
	// Not %q: the value is LaTeX, and quoting it would re-escape the backslashes.
	return fmt.Sprintf("%s %s \"%s\"", author, e.Get("year"), title)
}
