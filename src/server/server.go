package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	path "path/filepath"
	_ "time/tzdata"

	"git.sr.ht/~kvo/go-std/defs"
	"git.sr.ht/~kvo/go-std/errors"

	"main/logger"
	"main/site"
)

var (
	creds     Creds
	respath   string
	schools   = make(map[string]*site.Mux)
	templates *template.Template
)

func Run(version string, tlsConn bool) {
	logger.Info("Running %s", version)
	enrol("example", "gihs", "uofa")

	creds.Tokens = make(map[string]site.Uid)
	creds.Users = make(map[site.Uid]site.User)

	execpath, err := os.Executable()
	if err != nil {
		logger.Fatal(errors.New("cannot get path to executable", err))
	}
	respath = path.Join(path.Dir(execpath), "../../../res/")
	config := path.Join(respath, "config.json")
	cert := path.Join(respath, "cert.pem")
	key := path.Join(respath, "key.pem")

	cfg, err := getConfig(config)
	if err != nil {
		logger.Error(errors.New("unable to read config file", err))
		logger.Warn("Resorting to default configuration settings")
	}

	// TODO: Implement logging to file with further options
	if cfg.Logging.UseLogFile {
		logPath := path.Join(respath, "logs")
		err = logger.UseConfigFile(logPath)
		if err != nil {
			logger.Error(errors.New("Log file was not set up successfully", err))
		} else {
			logger.Info("Log file set up successfully")
		}
	}

	err = initTemplates(respath)
	if err != nil {
		logger.Fatal(errors.New(site.ErrInitFailed.Error()+" for HTML templates", err))
	}
	logger.Info("Successfully initialized HTML templates")

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

	if tlsConn {
		logger.Info("Running on port 443")
		err = http.ListenAndServeTLS(":443", cert, key, mux)
	} else {
		logger.Warn("Running on port 8080 (without TLS). DO NOT USE THIS IN PRODUCTION!")
		err = http.ListenAndServe("localhost:8080", mux)
	}

	if err != nil {
		logger.Fatal(err)
	}
}

// Create and manage necessary HTML files from template files.
func initTemplates(respath string) error {
	// Create "./templates/" dir if it does not exist
	tmplPath := path.Join(respath, "templates")
	err := os.MkdirAll(tmplPath, os.ModePerm)
	if err != nil {
		return errors.New("could not make 'templates' directory", err)
	}

	tmplResPath := tmplPath

	var files []string
	err = path.WalkDir(tmplResPath, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			logger.Fatal(err)
		}
		// Skip the directory name itself from being appended, although its children won't be affected
		// Excluding via info.IsDir() will exclude files that are under a subdirectory so it cannot be used
		// For MacOS: The .DS_Store directory interferes so it must be ignored.
		if info.Name() == "body" || info.Name() == "components" || (info.Name() == ".DS_Store") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		logger.Fatal(errors.New("error walking the template/ directory", err))
	}

	files = defs.Remove(files, tmplResPath)

	var requiredFiles []string
	tmplFiles := []string{
		"page", "head",
		"components/header", "components/nav", "components/footer",
		"body/error", "body/login", "body/main",
		"body/grades", "body/resource", "body/resources",
		"body/task", "body/tasks", "body/timetable",
	}
	for _, f := range tmplFiles {
		requiredFiles = append(requiredFiles, path.Join(tmplResPath, f+".tmpl"))
	}

	filesMissing := false
	var missingFiles []string
	for _, f := range requiredFiles {
		if !defs.Has(files, f) {
			filesMissing = true
			missingFiles = append(missingFiles, f)
		}
	}
	if filesMissing {
		errStr := fmt.Errorf("%v:\n%+v", "the following files are missing", missingFiles)
		logger.Fatal(errors.New(errStr.Error(), nil))
	}

	sortedFiles := files

	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
	}

	templates = template.Must(template.New("").Funcs(funcMap).ParseFiles(sortedFiles...))
	return nil
}

type config struct {
	Logging loggingConfig `json:"logging"`
}

type loggingConfig struct {
	UseLogFile bool `json:"useLogFile"`
	//LogFileOptions logFileOptions `json:"logFileOptions"`
}

// Get user configuration options from config.json
func getConfig(cfgPath string) (config, error) {
	// Default config
	cfg := config{
		loggingConfig{
			UseLogFile: false,
		},
	}

	jsonFile, err := os.OpenFile(cfgPath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return cfg, errors.New("failed to open config.json", err)
	}

	b, err := io.ReadAll(jsonFile)
	if err != nil {
		return cfg, errors.New("failed to read config.json", err)
	}

	err = jsonFile.Close()
	if err != nil {
		return cfg, errors.New("failed to close config.json", err)
	}

	jsonFile, err = os.OpenFile(cfgPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0622)
	if err != nil {
		return cfg, errors.New("failed to open config.json", err)
	}
	defer jsonFile.Close()

	if len(b) > 0 {
		err = json.Unmarshal(b, &cfg)
		if err != nil {
			return cfg, errors.New("failed to unmarshal config.json", err)
		}
	} else {
		logger.Info("Using default configuration settings. These can be edited in the config.json file")
	}

	rawJson, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return config{}, errors.New("failed to marshal config.json", err)
	}

	_, err = jsonFile.Write(rawJson)
	if err != nil {
		return cfg, errors.New("failed to write to config.json", err)
	}

	return cfg, nil
}
