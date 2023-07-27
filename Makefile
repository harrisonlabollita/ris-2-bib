build:
	go build -o ris2bib
	mv ris2bib /usr/local/bin/
test:
	go test -v ./...
