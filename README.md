# ris-2-bib

Convert RIS formatted citation files into BIB format from the command line.

## Build

```bash
go get github.com:harrisonlabollita/ris-2-bib.git
```
From there you can build the exectuable. For an Intel Macs,
```bash
GOOS=darwin GOARCH=amd64 go build -o ris2bib cmd/ris2bib/main.go
```
For amd/intel Linux, 
```bash
GOOS=linux GOARCH=amd64 go build -o ris2bib cmd/ris2bib/main.go
```
The exectuable ``ris2bib`` can then be moved to your bin folder. On Mac, this would be
```bash
mv ris2bib /usr/local/bin
```


## Usage
The executbale has two working modes. To provide the file that you would like to convert simply execute the command 
```bash
ris2bib -file=name-of-file
```
The CLI will keep the original file name, but change the file extenstion to ``*.bib``.

If you have many ``*.ris`` files in a directory you can convert all of them, by simply calling ``ris2bib`` from within the directory
```bash
ris2bib
```
