package site

import (
	"fmt"
	"net/url"
	"os"
	path "path/filepath"

	"git.sr.ht/~kvo/go-std/errors"
	"github.com/BurntSushi/toml"
)

func readcfg(path string) (map[string]UserConfig, error) {
	config := make(map[string]UserConfig)
	_, err := toml.DecodeFile(path, &config)
	if err != nil {
		errstr := fmt.Sprintf("cannot parse user config: %s", path)
		return config, errors.New(errstr, err)
	}
	return config, nil
}

func LoadConfig(user *User) error {
	username := url.PathEscape(user.Username)
	filename := username + ".cfg"
	execpath, err := os.Executable()
	if err != nil {
		return errors.New("cannot get path to executable", err)
	}
	cfgpath := path.Join(path.Dir(execpath), "../../../cfg/user/", user.School, filename)
	config, err := readcfg(cfgpath)
	if err != nil {
		return errors.New("", err)
	}
	user.Config = config
	return nil
}
