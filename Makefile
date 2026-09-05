build:
	go build -o shelf

install: build
	mv shelf /opt/homebrew/bin/

test:
	go test -v ./...

golden:
	go test -run TestGolden -update ./...
