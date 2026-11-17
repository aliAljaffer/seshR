BIN := $(HOME)/.local/bin
SHARE := $(HOME)/.local/share

VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')-dev
ifeq ($(VERSION),-dev)
	VERSION := dev
endif

TAP := alialjaffer/tap

.PHONY: build test install upgrade brew clean
build:
	go vet ./...
	go build -ldflags "-X main.version=$(VERSION)" -o seshr .

test:
	go test ./...

install: build
	mkdir -p $(BIN)
	ln -sf $(CURDIR)/seshr $(BIN)/seshr
	mkdir -p $(SHARE)/bash-completion/completions $(SHARE)/zsh/site-functions
	ln -sf $(CURDIR)/completions/seshr.bash $(SHARE)/bash-completion/completions/seshr
	ln -sf $(CURDIR)/completions/_seshr $(SHARE)/zsh/site-functions/_seshr

upgrade:
	git pull --ff-only
	$(MAKE) install

brew:
	@if brew tap-info $(TAP) 2>/dev/null | grep -q Untrusted; then \
		printf 'Trust tap $(TAP)? [y/N] '; read a; \
		case "$$a" in y|Y|yes) brew trust --tap $(TAP) ;; *) echo aborted; exit 1 ;; esac; \
	fi
	rm -f $(BIN)/seshr $(SHARE)/bash-completion/completions/seshr $(SHARE)/zsh/site-functions/_seshr
	brew install $(TAP)/seshr

clean:
	rm -f seshr
