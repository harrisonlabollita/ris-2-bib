# shelf

![GitHub](https://img.shields.io/github/license/harrisonlabollita/shelf)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/harrisonlabollita/shelf)

A single-file bibliography on the command line.

One canonical `master.bib` lives on your machine, in a git repo if you like. `shelf`
imports references into it from RIS or BibTeX, keeps it in a stable canonical
format, and refuses to add the same paper twice.

## Install

```shell
git clone https://github.com/harrisonlabollita/shelf
cd shelf
make build      # produces ./shelf
make install    # moves it to /opt/homebrew/bin
```

## The library

`shelf` reads and writes one file, resolved in this order:

1. `$SHELF_FILE`
2. `$XDG_DATA_HOME/shelf/master.bib`
3. `~/.local/share/shelf/master.bib`

It is plain BibTeX. Point LaTeX straight at it, grep it, edit it by hand, and
put it under version control — `shelf fmt` keeps the formatting canonical so the
diffs stay readable.

## Usage

```
shelf                                                search the library (same as: shelf find)
shelf find [-f key|cite|entry]                       search, print the selection
shelf add [-to FILE] [-n] <file.ris|file.bib|->...   add references to the library
shelf fmt [FILE]                                     rewrite a .bib in canonical form
shelf convert <file.ris|->...                        convert to BibTeX on stdout
shelf browse [DIR]                                   page through .ris files, deleting rejects
```

## Searching

Run `shelf` with no arguments and type. The list narrows as you go, and the entry
under the cursor is previewed in full underneath.

```
❯ dmft                                                                    7/26
──────────────────────────────────────────────────────────────────────────────
  Fictitious             2021  Charge self-consistency in DFT+DMFT for laye…
▌ Pavarini               2011  The LDA+DMFT Approach
  Kotliar et al.         2006  Electronic structure calculations with dynam…
  Georges et al.         1996  Dynamical mean-field theory of strongly corr…
──────────────────────────────────────────────────────────────────────────────
@incollection{Pavarini2011LDADMFT,
  author    = Pavarini, Eva
  title     = The LDA+DMFT Approach
  booktitle = The LDA+DMFT approach to strongly correlated materials
  year      = 2011
}
```

`↑`/`↓` or `ctrl-p`/`ctrl-n` to move, `ctrl-w` to delete a word, `ctrl-u` to
clear, enter to select, escape to quit.

Enter prints the cite key on stdout, so the picker composes:

```shell
shelf | pbcopy                 # cite key to the clipboard
shelf find -f cite | pbcopy    # \cite{key} instead
shelf find -f entry            # the whole BibTeX entry
```

The interface is drawn on `/dev/tty`, which is why piping the result works.

### How matching works

Scoring is fzf's own `FuzzyMatchV2`, used as a library, but each field is
matched separately and the best score wins. Matching one concatenated string
ranks badly: fzf rewards compact matches, so `gga` scored higher against
"Kresse, **G**eor**g** **a**nd ..." than against "**G**eneralized **G**radient
**A**pproximation". Two things fix that — author *surnames* are searched rather
than full names, and the initials of the title are a search field of their own,
because physics is written in acronyms:

| query | finds |
|---|---|
| `gga` | Perdew, *Generalized Gradient Approximation Made Simple* |
| `paw` | Blöchl, *Projector augmented-wave method* |
| `mlwf` | Marzari, *Maximally localized Wannier functions* |
| `kresse` | the VASP paper, by surname |
| `10.1038` | the two Nature papers, by DOI |

DOIs, keywords and cite keys are searched even though the list never shows
them. The whole library is rescanned on every keystroke: about 2 ms at 3000
entries, so there is no need for the incremental narrowing fzf does.

## Adding references

Download an `.ris` from a journal page and add it:

```shell
$ shelf add ~/Downloads/revmodphys.ris
+ Georges1996Dynamical
shelf: 1 added, 0 duplicate, 348 total in ~/.local/share/shelf/master.bib
```

Add it again, or add a `.bib` from a collaborator that overlaps with what you
already have, and the duplicates are skipped. Papers are matched on DOI first,
then arXiv id, then author/year/title — so the same reference in two different
formats is still recognised as one paper:

```shell
$ shelf add collaborator.bib
+ Novel2011Something
= Georges 1996 "Dynamical mean-field theory of strongly correlated fer..." (already in library)
shelf: 1 added, 1 duplicate, 349 total in ~/.local/share/shelf/master.bib
```

`-n` shows what would happen without writing. `-` reads standard input, so
`shelf` composes with anything:

```shell
$ curl -sL "$URL" | shelf add -
```

`shelf convert` skips the library entirely and prints BibTeX to stdout.

## Cite keys

Keys are `AuthorYearWord` — `Georges1996Dynamical` — with accents folded to
ASCII and a letter suffix on collision.

**A key is assigned once, at import, and never recomputed.** Once a key is in a
manuscript, changing it silently breaks a paper you may not touch again for a
year. Keys arriving with an imported `.bib` are kept for the same reason.

## What gets converted

RIS records are split on `ER`, so a multi-record export becomes multiple
entries. `TY` picks the BibTeX entry type (`JOUR` → `@article`, `BOOK` →
`@book`, `CHAP` → `@incollection`, `THES` → `@phdthesis`, and so on), and that
type then decides how the ambiguous tags are read: `T2` is a journal on an
article but a book title on a chapter, `SN` is an ISSN on a periodical and an
ISBN on a book, `PB` becomes `school` on a thesis.

Values are escaped once, on import, and only for `&`, `%` and `#`. The math and
markup characters `$ _ ^ \ { }` are deliberately left alone: publisher exports
do carry real LaTeX, and mangling `$T_c$` is worse than leaving a rare stray
character to fix by hand. Braces are escaped only when unbalanced, which would
otherwise corrupt the entry.

Tags with no BibTeX equivalent are reported rather than dropped in silence.

## Roadmap

- `shelf export --from paper.aux` — emit only the references a manuscript cites,
  so the master file never has to be shipped
- `shelf merge` — fold a preprint entry into its published version, leaving the
  old key working

## Contributing

Contributions are welcome. Please open an issue or a pull request.
