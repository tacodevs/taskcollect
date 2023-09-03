package main

import (
	"os"

	"git.sr.ht/~kvo/libgo/defs"

	"main/logger"
	"main/plat"
	"main/server"
)

func main() {
	tlsConn := true
	if len(os.Args) > 2 || len(os.Args) == 2 && os.Args[1] != "-w" {
		logger.Fatal(plat.ErrBadCommandUsage)
	}
	if defs.Has(os.Args, "-w") {
		tlsConn = false
	}
	server.Run(version, tlsConn)
}
