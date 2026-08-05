# Compilers and tools
GO ?= go

# Compilation flags
GO_LDFLAGS=-w

build:
	mkdir -p build
	cd src; $(GO) build -ldflags "$(GO_LDFLAGS)" -o ../build/GameDiscPlayer

clean:
	rm -r build/

.PHONY: build clean
