# ris-2-bib

![GitHub](https://img.shields.io/github/license/harrisonlabollita/ris-2-bib)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/harrisonlabollita/ris-2-bib)
![GitHub Workflow Status](https://img.shields.io/github/workflow/status/harrisonlabollita/ris-2-bib/Build%20and%20Test)

A command-line tool to convert RIS formatted files to BibTeX format.

## Introduction

The RIS to BibTeX Converter is a handy command-line utility that allows you to convert RIS formatted files to BibTeX format, making it easier to manage your bibliographic references. Whether you have a single file or a directory of RIS files, this tool simplifies the process.

## Features
- Convert RIS files to BibTeX format
- Supports both single files and directories
- Customizable output file names and BibTeX article IDs

## Installation
To use the RIS to BibTeX Converter, you'll need to have Go installed. If you haven't already, you can download and install Go from the [official website](https://golang.org/).

Once you have Go installed, you can install the tool using the following command:

```shell
go get github.com/harrisonlabollita/ris-2-bib.git
```
From there build the exectubale. You can use the ``Makefile``, but make sure the build is appropriate for your architecture.

## Example
Given an RIS formatted file ``example.ris``
```
AU  - LastName1, FirstName1
AU  - LastName2, FirstName2
AU  - LastName3, FirstName3
AU  - LastName4, FirstName4
PY  - 2023
DA  - 2023/01/01
TI  - An interesting title would be here
JO  - Journal Name
SP  - 123
EP  - 123
VL  - 123
IS  - 1234
AB  - The quick brown fox jumped over the lazy sheep dog.
SN  - 1234-2468
UR  - https://doi.org/10.0000/journal0000
DO  - 10.0000/journal0000`
```

Calling ``ris2bib`` generates the BibTeX formatted file ``example.bib``

```
@article{LastName12023interesting,
author = {LastName1, FirstName1  and  LastName2, FirstName2  and  LastName3, FirstName3  and  LastName4, FirstName4},
title = {An interesting title would be here},
journal = {Journal Name},
year  = {2023},
volume  = {123},
issue = {1234},
pages = {123-123},
doi  = {10.0000/journal0000},
url  = {https://doi.org/10.0000/journal0000}
}
```


## Usage

```bash
ris2bib -h

Usage of ris2bib:
  -file string
        filename of ris file or directory path to ris file(s). (default ".")
  -id string
        BibTeX article id (default " ")
  -out string
        new filename of bib file (default ".")
```

The executbale has two working modes. You can explicitly provide a file name to be converted or a directory path.
```bash
ris2bib -file=name-of-file/directory-path
```
The cli will keep the original file name, but change the file extenstion to ``*.bib``.

If you have many ``*.ris`` files in a directory you can convert all of them, by simply calling ``ris2bib`` from within the directory
```bash
ris2bib
```
The name of the output file can be controlled with ``-out`` flag. Note that this only works on single file conversions.

## Contributing
Contributions are welcome! If you have any suggestions, bug reports, or feature requests, please open an issue or a pull request in this repository.
