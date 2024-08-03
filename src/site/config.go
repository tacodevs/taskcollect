package site

import (
	"fmt"
	"net/url"
	"os"
	path "path/filepath"

	"git.sr.ht/~kvo/go-std/errors"
)

func readcfg(path string) (map[string]UserConfig, error) {
	config := make(map[string]UserConfig)
	file, err := os.Open(path)
	if err != nil {
		errstr := fmt.Sprintf("cannot open %s", path)
		return config, errors.New(errstr, err)
	}
	defer file.Close()
	// TODO: parse config
	return config, nil
}

func LoadConfig(user User) (User, error) {
	username := url.PathEscape(user.Username)
	filename := username + ".cfg"
	execpath, err := os.Executable()
	if err != nil {
		return User{}, errors.New("cannot get path to executable", err)
	}
	cfgpath := path.Join(path.Dir(execpath), "../../../cfg/user/", user.School, filename)
	config, err := readcfg(cfgpath)
	if err != nil {
		return User{}, errors.New("", err)
	}
	user.Config = config
	return user, nil
}
