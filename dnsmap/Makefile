GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

FLAGS ?= -ldflags '-w -s'

all: bin/dnsmap-$(GOOS)-$(GOARCH)
.PHONY: all

bin:
	mkdir bin

bin/dnsmap-$(GOOS)-$(GOARCH): bin
	go build $(FLAGS) -o $@

clean:
	rm -rf bin
.PHONY: clean
