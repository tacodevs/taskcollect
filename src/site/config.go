package site

import (
	"net/url"
	"os"
	path "path/filepath"

	"git.sr.ht/~kvo/go-std/errors"
	"github.com/BurntSushi/toml"
)

func readcfg(path string) (map[string]UserConfig, error) {
	config := make(map[string]UserConfig)
	file, err := os.Open(path)
	if err != nil {
		// user has empty config
		return config, nil
	}
	defer file.Close()
	_, err = toml.NewDecoder(file).Decode(&config)
	if err != nil {
		return config, errors.New(err, "cannot parse user config: %s", path)
	}
	return config, nil
}

func LoadConfig(user *User) error {
	username := url.PathEscape(user.Username)
	filename := username + ".cfg"
	execpath, err := os.Executable()
	if err != nil {
		return errors.New(err, "cannot get path to executable")
	}
	cfgpath := path.Join(path.Dir(execpath), "../../../cfg/user/", user.School, filename)
	config, err := readcfg(cfgpath)
	if err != nil {
		return errors.Wrap(err)
	}
	user.Config = config
	return nil
}
