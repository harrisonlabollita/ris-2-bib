# ris-2-bib

Convert RIS formatted citation files into BIB format from the command line.

## Build

```bash
git clone https://github.com/harrisonlabollita/ris-2-bib.git
go build .
```
The executable can then be copied to your ``bin`` folder.

## Usage
Currently, there are two modes to use:

1. Provid a filename with the ``-file`` tag
```bash
ris2bib -file=name-of-file
```
2. Call the executable in the directory where there is at least 1 ``.ris`` file. ``ris2bib`` will convert all RIS files that it finds.
```bash
ris2bib
```
