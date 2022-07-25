all: i386-linux amd64-linux arm-linux arm64-linux i386-windows amd64-windows amd64-osx arm64-osx

i386-linux:
	CGO_ENABLED=0 GOARCH=386 GOOS=linux go build -ldflags="-s -w" -o ../prg/i386/linux/taskcollect .

amd64-linux:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o ../prg/amd64/linux/taskcollect .

arm-linux:
	CGO_ENABLED=0 GOARCH=arm GOOS=linux go build -ldflags="-s -w" -o ../prg/arm/linux/taskcollect .

arm64-linux:
	CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -ldflags="-s -w" -o ../prg/arm64/linux/taskcollect .

i386-windows:
	CGO_ENABLED=0 GOARCH=386 GOOS=windows go build -ldflags="-s -w" -o ../prg/i386/windows/taskcollect.exe .

amd64-windows:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -ldflags="-s -w" -o ../prg/amd64/windows/taskcollect.exe .

amd64-osx:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -ldflags="-s -w" -o ../prg/amd64/osx/taskcollect .

arm64-osx:
	CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build -ldflags="-s -w" -o ../prg/arm64/osx/taskcollect .
