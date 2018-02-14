.PHONY: all

all: blu-up

blu-up: *.go
	goimports -w *.go
	go fmt
	go build