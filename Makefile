# communikey — build, test, install
# Binaire Go zéro-dépendance. `make install` produit un binaire universel
# (arm64 + x86_64) et le symlinke sur le PATH via install.sh.

BINARY := communikey
GO     ?= go

.PHONY: all build vet test cover install clean

all: vet test build

build:
	$(GO) build -o $(BINARY) .

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

cover:
	$(GO) test -cover ./...

install:
	./install.sh

clean:
	rm -f $(BINARY)
	rm -rf .build
