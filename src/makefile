PLATFORMS=\
	386/linux \
	386/windows \
	amd64/darwin \
	amd64/windows \
	amd64/linux \
	arm/linux \
	arm64/darwin \
	arm64/linux

all: $(PLATFORMS)

8LG=CGO_ENABLED=0  GOARCH=386    GOOS=linux    go build
8WG=CGO_ENABLED=0  GOARCH=386    GOOS=windows  go build
6DG=CGO_ENABLED=0  GOARCH=amd64  GOOS=darwin   go build
6LG=CGO_ENABLED=0  GOARCH=amd64  GOOS=linux    go build
6WG=CGO_ENABLED=0  GOARCH=amd64  GOOS=windows  go build
5LG=CGO_ENABLED=0  GOARCH=arm    GOOS=linux    go build
ADG=CGO_ENABLED=0  GOARCH=arm64  GOOS=darwin   go build
ALG=CGO_ENABLED=0  GOARCH=arm64  GOOS=linux    go build

GOFLAGS=-ldflags="-s -w"

386/linux:
	$(8LG) $(GOFLAGS) -o ../prg/386/linux/taskcollect .

386/windows:
	$(8WG) $(GOFLAGS) -o ../prg/386/windows/taskcollect.exe .

amd64/darwin:
	$(6DG) $(GOFLAGS) -o ../prg/amd64/darwin/taskcollect .

amd64/linux:
	$(6LG) $(GOFLAGS) -o ../prg/amd64/linux/taskcollect .

amd64/windows:
	$(6WG) $(GOFLAGS) -o ../prg/amd64/windows/taskcollect.exe .

arm/linux:
	$(5LG) $(GOFLAGS) -o ../prg/arm/linux/taskcollect .

arm64/darwin:
	$(ADG) $(GOFLAGS) -o ../prg/arm64/darwin/taskcollect .

arm64/linux:
	$(ALG) $(GOFLAGS) -o ../prg/arm64/linux/taskcollect .

.PHONY: $(PLATFORMS)
