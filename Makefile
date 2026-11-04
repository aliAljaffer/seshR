BIN := $(HOME)/.local/bin

.PHONY: build install
build:
	go build -o seshr .

install: build
	mkdir -p $(BIN)
	ln -sf $(CURDIR)/seshr $(BIN)/seshr
