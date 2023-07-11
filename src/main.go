package main

import (
	"os"

	"codeberg.org/kvo/std"

	"main/logger"
	"main/plat"
	"main/server"
)

func main() {
	tlsConn := true
	if len(os.Args) > 2 || len(os.Args) == 2 && os.Args[1] != "-w" {
		logger.Fatal(plat.ErrBadCommandUsage)
	}
	if std.Contains(os.Args, "-w") {
		tlsConn = false
	}
	server.Run(version, tlsConn)
}
