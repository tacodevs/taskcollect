package server

import (
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	path "path/filepath"
	_ "time/tzdata"

	"git.sr.ht/~kvo/go-std/errors"
	"git.sr.ht/~kvo/go-std/slices"

	"main/logger"
	"main/site"
)

var (
	creds     Creds
	respath   string
	schools   = make(map[string]*site.Mux)
	templates *template.Template
)

func Announce(version string) {
	logger.Info("Running %s", version)
}

// TODO: refactor
type config struct {
	Logging loggingConfig `json:"logging"`
}

// TODO: refactor
type loggingConfig struct {
	UseLogFile bool `json:"useLogFile"`
	//LogFileOptions logFileOptions `json:"logFileOptions"`
}

// TODO: refactor
func getConfig(cfgPath string) (config, error) {
	// gets stuff from config.json

	// Default config
	cfg := config{
		loggingConfig{
			UseLogFile: false,
		},
	}

	jsonFile, err := os.OpenFile(cfgPath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return cfg, errors.New(err, "failed to open config.json")
	}

	b, err := io.ReadAll(jsonFile)
	if err != nil {
		return cfg, errors.New(err, "failed to read config.json")
	}

	err = jsonFile.Close()
	if err != nil {
		return cfg, errors.New(err, "failed to close config.json")
	}

	jsonFile, err = os.OpenFile(cfgPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0622)
	if err != nil {
		return cfg, errors.New(err, "failed to open config.json")
	}
	defer jsonFile.Close()

	if len(b) > 0 {
		err = json.Unmarshal(b, &cfg)
		if err != nil {
			return cfg, errors.New(err, "failed to unmarshal config.json")
		}
	} else {
		logger.Info("Using default configuration settings. These can be edited in the config.json file")
	}

	rawJson, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return config{}, errors.New(err, "failed to marshal config.json")
	}

	_, err = jsonFile.Write(rawJson)
	if err != nil {
		return cfg, errors.New(err, "failed to write to config.json")
	}

	return cfg, nil
}

func loadTmpl(respath string) error {
	tmplPath := path.Join(respath, "templates")
	err := os.MkdirAll(tmplPath, os.ModePerm)
	if err != nil {
		return errors.Wrap(err)
	}
	required := []string{
		"body/error",
		"body/grades",
		"body/login",
		"body/main",
		"body/resource",
		"body/resources",
		"body/task",
		"body/tasks",
		"body/timetable",
		"components/footer",
		"components/header",
		"components/nav",
		"head",
		"page",
	}
	for i := range required {
		required[i] = path.Join(tmplPath, required[i]+".tmpl")
	}
	var files []string
	err = path.WalkDir(tmplPath, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == ".DS_Store" {
			return fs.SkipDir
		} else if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	files = slices.Remove(files, tmplPath)
	missing := make([]string, 0, len(required))
	for _, file := range required {
		if !slices.Has(files, file) {
			missing = append(missing, file)
		}
	}
	if len(missing) != 0 {
		return errors.New(nil, "Missing templates:\n%+v", missing)
	}
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
	}
	templates = template.Must(template.New("").Funcs(funcMap).ParseFiles(files...))
	return nil
}

func Configure() error {
	creds.Tokens = make(map[string]site.Uid)
	creds.Users = make(map[site.Uid]site.User)

	execpath, err := os.Executable()
	if err != nil {
		logger.Fatal(errors.New(err, "cannot get path to executable"))
	}
	respath = path.Join(path.Dir(execpath), "../../../res/")
	config := path.Join(respath, "config.json")

	// TODO: refactor
	cfg, err := getConfig(config)
	if err != nil {
		logger.Error(errors.New(err, "Cannot read config file:"))
		logger.Warn("Resorting to default configuration settings...")
	}
	if cfg.Logging.UseLogFile {
		logPath := path.Join(respath, "logs")
		err = logger.UseConfigFile(logPath)
		if err != nil {
			return errors.New(err, "Log file was not set up successfully")
		}
		logger.Info("Log file set up successfully")
	}

	err = loadTmpl(respath)
	if err != nil {
		return errors.New(err, "cannot load HTML templates")
	}
	logger.Info("Successfully loaded HTML templates")
	return nil
}

func Run(tls bool) error {
	cert := path.Join(respath, "cert.pem")
	key := path.Join(respath, "key.pem")

	mux := http.NewServeMux()

	mux.HandleFunc("/assets/", assetHandler)
	mux.HandleFunc("/res", resHandler)
	mux.HandleFunc("/res/", resourceHandler)
	mux.HandleFunc("/tasks", tasksHandler)
	mux.HandleFunc("/tasks/", taskHandler)
	mux.HandleFunc("/timetable", timetableHandler)
	mux.HandleFunc("/grades", gradesHandler)

	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/logout", logoutHandler)
	mux.HandleFunc("/auth", authHandler)
	mux.HandleFunc("/", rootHandler)

	if tls {
		logger.Info("Running on port 443")
		return http.ListenAndServeTLS(":443", cert, key, mux)
	} else {
		logger.Warn("Running on port 8080 (without TLS). DO NOT USE THIS IN PRODUCTION!")
		return http.ListenAndServe("localhost:8080", mux)
	}
}
