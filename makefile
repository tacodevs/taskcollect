.PHONY: build all run deploy

ARCH := $(shell go env GOARCH)
OS := $(shell go env GOOS)

ifeq ($(OS), windows)
	EXT := .exe
endif

# TODO: update src/version.go on all builds

build: res/styles.css
	mkdir -p prg/$(OS)/$(ARCH)/
	cd src/; CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -o ../prg/$(OS)/$(ARCH)/taskcollect$(EXT) .

all: res/styles.css
	cd src/; for os in linux android; do \
		for arch in amd64 386 arm64 arm; do \
			echo make: building for $$os/$$arch...; \
			mkdir -p ../prg/$$os/$$arch/; \
			CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -o ../prg/$$os/$$arch/taskcollect .; \
		done; \
	done
	cd src/; for os in darwin ios; do \
		for arch in arm64 amd64; do \
			echo make: building for $$os/$$arch...; \
			mkdir -p ../prg/$$os/$$arch/; \
			CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -o ../prg/$$os/$$arch/taskcollect .; \
		done; \
	done
	cd src/; for arch in amd64 386; do \
		echo make: building for windows/$$arch...; \
		mkdir -p ../prg/windows/$$arch/; \
		CGO_ENABLED=0 GOOS=windows GOARCH=$$arch go build -o ../prg/windows/$$arch/taskcollect.exe .; \
	done

res/styles.css:
	sass src/styles/styles.scss res/styles.css --no-source-map

run:	
	./prg/$(OS)/$(ARCH)/taskcollect$(EXT) -w

deploy:
	./prg/$(OS)/$(ARCH)/taskcollect$(EXT)
