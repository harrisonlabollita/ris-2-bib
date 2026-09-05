package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const usage = `shelf - a single-file bibliography on the command line

usage:
  shelf                                                search the library (same as: shelf find)
  shelf find [-f key|cite|entry]                       search, print the selection
  shelf add [-to FILE] [-n] <file.ris|file.bib|->...   add references to the library
  shelf fmt [FILE]                                     rewrite a .bib in canonical form
  shelf convert <file.ris|->...                        convert to BibTeX on stdout
  shelf browse [DIR]                                   page through .ris files, deleting rejects

The library defaults to $SHELF_FILE, else $XDG_DATA_HOME/shelf/master.bib.
`

func main() {
	log.SetFlags(0)
	log.SetPrefix("shelf: ")

	// Bare `bib` opens the picker: searching is what you do all day.
	if len(os.Args) < 2 {
		if err := cmdFind(nil); err != nil {
			log.Fatal(err)
		}
		return
	}

	var err error
	switch os.Args[1] {
	case "find":
		err = cmdFind(os.Args[2:])
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
		fmt.Fprintf(os.Stderr, "shelf: unknown command %q\n\n%s", os.Args[1], usage)
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
			fmt.Fprintf(os.Stderr, "shelf: %s: %s\n", path, w)
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
		fmt.Fprintf(os.Stderr, "shelf: dry run, %s not written\n", path)
		return nil
	}
	if len(added) == 0 {
		return nil
	}
	if err := SaveLibrary(path, merged); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "shelf: %d added, %d duplicate, %d total in %s\n",
		len(added), len(skipped), len(merged), path)
	return nil
}

func cmdFind(argv []string) error {
	fs := flag.NewFlagSet("find", flag.ExitOnError)
	format := fs.String("f", "key", "what to print on selection: key, cite, or entry")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	path := LibraryPath()
	entries, err := LoadLibrary(path)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("%s is empty; add something with `shelf add`", path)
	}

	chosen, err := RunFinder(entries)
	if err != nil {
		return err
	}
	if chosen == nil {
		os.Exit(130) // quit without choosing
	}

	switch *format {
	case "key":
		fmt.Println(chosen.Key)
	case "cite":
		fmt.Printf("\\cite{%s}\n", chosen.Key)
	case "entry":
		fmt.Print(chosen.String())
	default:
		return fmt.Errorf("unknown -f value %q: want key, cite, or entry", *format)
	}
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
