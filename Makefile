build:
	go build -o ris2bib
	mv ris2bib /opt/homebrew/bin/
test:
	go test -v ./...
