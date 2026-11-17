BIN := $(HOME)/.local/bin

VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')-dev
ifeq ($(VERSION),-dev)
	VERSION := dev
endif

.PHONY: build test install upgrade
build:
	go vet ./...
	go build -ldflags "-X main.version=$(VERSION)" -o seshr .

test:
	go test ./...

install: build
	mkdir -p $(BIN)
	ln -sf $(CURDIR)/seshr $(BIN)/seshr

upgrade:
	git pull --ff-only
	$(MAKE) install
