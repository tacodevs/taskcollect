.PHONY: run deploy

ARCH := $(shell go env GOARCH)
OS := $(shell go env GOOS)

ifeq ($(OS), windows)
	EXT := .exe
endif

run:	
	./prg/$(OS)/$(ARCH)/taskcollect$(EXT) -w

deploy:
	./prg/$(OS)/$(ARCH)/taskcollect$(EXT)
