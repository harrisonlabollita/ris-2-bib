# ris-2-bib

Convert RIS formatted citation files into BIB format from the command line.

## Build

```bash
go get github.com:harrisonlabollita/ris-2-bib.git
```
From there you can build the exectuable. For an Intel Macs,
```bash
GOOS=darwin GOARCH=amd64 go build -o ris2bib
```
For amd/intel Linux, 
```bash
GOOS=linux GOARCH=amd64 go build -o ris2bib
```
The exectuable ``ris2bib`` can then be moved to your bin folder. On Mac, this would be
```bash
mv ris2bib /usr/local/bin
```


## Usage

```bash
ris2bib -h

Usage of ris2bib:
  -file string
        filename of ris file or directory path to ris file(s). (default ".")
  -out string
        new filename of bib file (default ".")
```

The executbale has two working modes. You can explicitly provide a file name to be converted or a directory path.
```bash
ris2bib -file=name-of-file/directory-path
```
The CLI will keep the original file name, but change the file extenstion to ``*.bib``.

If you have many ``*.ris`` files in a directory you can convert all of them, by simply calling ``ris2bib`` from within the directory
```bash
ris2bib
```
The name of the output file can be controlled with ``-out`` flag. Note that this only works on single file conversions.

## Contributing
PRs and Issues are welcome! If you found this tool useful, please leave a star.
