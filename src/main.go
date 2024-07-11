package main

import (
	"os"

	"git.sr.ht/~kvo/go-std/defs"

	"main/logger"
	"main/server"
)

func main() {
	tlsConn := true
	if len(os.Args) > 2 || len(os.Args) == 2 && os.Args[1] != "-w" {
		logger.Fatal("invalid invocation")
	}
	if defs.Has(os.Args, "-w") {
		tlsConn = false
	}
	server.Run(version, tlsConn)
}
