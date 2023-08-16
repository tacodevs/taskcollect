package tests

import (
	"fmt"
	"os"
	"os/user"
	fp "path/filepath"
)

// Get the resource path (resPath). You can specify directories that are located within
// resPath as well.
func GetResPath(f ...string) string {
	curUser, err := user.Current()
	if err != nil {
		fmt.Println("tests: Cannot determine current user's home folder")
		os.Exit(1)
	}

	joined := fp.Join(f...)
	return fp.Join(curUser.HomeDir, "res/taskcollect", joined)
}
