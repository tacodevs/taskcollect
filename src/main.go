package main

import (
	"fmt"
	"os"

	"codeberg.org/kvo/std"

	"main/errors"
	"main/logger"
	"main/server"
)

func main() {
	tlsConn := true

	if len(os.Args) > 2 || len(os.Args) == 2 && os.Args[1] != "-w" {
		logger.Fatal(errors.ErrBadCommandUsage)
	}

	if std.Contains(os.Args, "-w") {
		tlsConn = false
	}

	fmt.Printf("%v\n\n", tcVersion)
	server.Run(tcVersion, tlsConn)
}
