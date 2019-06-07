.PHONY: build run-dev build-docker clean

default: clean install test crossbinary

clean:
	rm -rf dist/

binary: install test
	GOOS=linux CGO_ENABLED=0 GOGC=off GOARCH=amd64 go build -a -tags netgo -ldflags '-w' -o "$(CURDIR)/dist/log-shipper"

crossbinary: binary
	GOOS=linux GOARCH=amd64 go build -o "$(CURDIR)/dist/log-shipper-linux-amd64"
	GOOS=linux GOARCH=386 go build -o "$(CURDIR)/dist/log-shipper-linux-386"
	GOOS=darwin GOARCH=amd64 go build -o "$(CURDIR)/dist/log-shipper-darwin-amd64"
	GOOS=darwin GOARCH=386 go build -o "$(CURDIR)/dist/log-shipper-darwin-386"
	GOOS=windows GOARCH=amd64 go build -o "$(CURDIR)/dist/log-shipper-windows-amd64.exe"
	GOOS=windows GOARCH=386 go build -o "$(CURDIR)/dist/log-shipper-windows-386.exe"

install: clean
	go mod download
	go mod verify
	go mod vendor
	go generate ./...

test:
	go test ./...

dist:
	mkdir dist

build-docker:
	docker build -t "$(DEV_DOCKER_IMAGE)" .
