package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/user"
	fp "path/filepath"
	_ "time/tzdata"

	"git.sr.ht/~kvo/libgo/defs"
	"git.sr.ht/~kvo/libgo/errors"
	"golang.org/x/term"

	"main/logger"
	"main/plat"
)

// Run the TaskCollect server.
func Run(version string, tlsConn bool) {
	curUser, e := user.Current()
	if e != nil {
		logger.Fatal("cannot determine current user's home folder")
	}

	home := curUser.HomeDir
	resPath := fp.Join(home, "res/taskcollect")
	configFile := fp.Join(resPath, "config.json")
	certFile := fp.Join(resPath, "cert.pem")
	keyFile := fp.Join(resPath, "key.pem")

	result, err := getConfig(configFile)
	if err != nil {
		logger.Error(errors.New("unable to read config file", err))
		logger.Warn("Resorting to default configuration settings")
	}

	// TODO: Implement logging to file with further options
	if result.Logging.UseLogFile {
		logPath := fp.Join(resPath, "logs")
		err = logger.UseConfigFile(logPath)
		if err != nil {
			logger.Error(errors.New("Log file was not set up successfully", err))
		} else {
			logger.Info("Log file set up successfully")
		}
	}

	logger.Info("Running %v", version)
	configMux()

	var password string
	fmt.Print("Password to Redis database: ")
	dbPwdInput, e := term.ReadPassword(int(os.Stdin.Fd()))
	if e != nil {
		err = errors.New(e.Error(), nil)
		logger.Fatal(errors.New("could not get password input", err))
	}
	fmt.Println()

	dbAddr := result.Database.Address
	dbIdx := result.Database.Index
	password = string(dbPwdInput)
	newRedisDB := initDB(dbAddr, password, dbIdx)
	logger.Info("Connected to Redis on %s with database index of %d", dbAddr, dbIdx)

	templates, err := initTemplates(resPath)
	if err != nil {
		logger.Fatal(errors.New(plat.ErrInitFailed.Error()+" for HTML templates", err))
	}
	logger.Info("Successfully initialized HTML templates")

	db := authDB{
		path:   resPath,
		client: newRedisDB,
	}

	h := handler{
		templates: templates,
		database:  &db,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/assets/", h.assetHandler)
	mux.HandleFunc("/res", h.resHandler)
	mux.HandleFunc("/res/", h.resourceHandler)
	mux.HandleFunc("/tasks", h.tasksHandler)
	mux.HandleFunc("/tasks/", h.taskHandler)
	mux.HandleFunc("/timetable", h.timetableHandler)
	mux.HandleFunc("/grades", h.gradesHandler)

	mux.HandleFunc("/login", h.loginHandler)
	mux.HandleFunc("/logout", h.logoutHandler)
	mux.HandleFunc("/auth", h.authHandler)
	mux.HandleFunc("/", h.rootHandler)

	if tlsConn {
		logger.Info("Running on port 443")
		e = http.ListenAndServeTLS(":443", certFile, keyFile, mux)
	} else {
		logger.Warn("Running on port 8080 (without TLS). DO NOT USE THIS IN PRODUCTION!")
		e = http.ListenAndServe(":8080", mux)
	}

	if e != nil {
		logger.Fatal(errors.New(e.Error(), nil))
	}
}

// Create and manage necessary HTML files from template files.
func initTemplates(resPath string) (*template.Template, errors.Error) {
	// Create "./templates/" dir if it does not exist
	tmplPath := fp.Join(resPath, "templates")
	e := os.MkdirAll(tmplPath, os.ModePerm)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return nil, errors.New("could not make 'templates' directory", err)
	}

	tmplResPath := tmplPath

	var files []string
	e = fp.WalkDir(tmplResPath, func(path string, info fs.DirEntry, err error) error {
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
	if e != nil {
		err := errors.New(e.Error(), nil)
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
		requiredFiles = append(requiredFiles, fp.Join(tmplResPath, f+".tmpl"))
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
		errStr := fmt.Errorf("%v:\n%+v", plat.ErrMissingFiles.Error(), missingFiles)
		logger.Fatal(errors.New(errStr.Error(), nil))
	}

	// Find page.tmpl and put it at the front; NOTE: (only needed when not using ExecuteTemplate)
	//var sortedFiles []string
	//pageTmpl := fp.Join(tmplResPath, "page.tmpl")
	//for _, f := range files {
	//	if f == pageTmpl {
	//		sortedFiles = append([]string{f}, sortedFiles...)
	//	} else {
	//		sortedFiles = append(sortedFiles, f)
	//	}
	//}
	sortedFiles := files

	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
	}

	templates := template.Must(template.New("").Funcs(funcMap).ParseFiles(sortedFiles...))
	return templates, nil
}

type config struct {
	Logging  loggingConfig  `json:"logging"`
	Database databaseConfig `json:"database"`
}

type loggingConfig struct {
	UseLogFile bool `json:"useLogFile"`
	//LogFileOptions logFileOptions `json:"logFileOptions"`
}

// NOTE: Not implemented
//type logFileOptions struct {
//	LogInfo bool `json:"logInfo"`
//}

type databaseConfig struct {
	Address string `json:"address"`
	Index   int    `json:"index"`
}

// Get user configuration options from config.json
func getConfig(cfgPath string) (config, errors.Error) {
	// Default config
	result := config{
		loggingConfig{
			UseLogFile: false,
		},
		databaseConfig{
			Address: "localhost:6379",
			Index:   0,
		},
	}

	jsonFile, e := os.OpenFile(cfgPath, os.O_RDONLY|os.O_CREATE, 0644)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return result, errors.New("failed to open config.json", err)
	}

	b, e := io.ReadAll(jsonFile)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return result, errors.New("failed to read config.json", err)
	}

	e = jsonFile.Close()
	if e != nil {
		err := errors.New(e.Error(), nil)
		return result, errors.New("failed to close config.json", err)
	}

	jsonFile, e = os.OpenFile(cfgPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0622)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return result, errors.New("failed to open config.json", err)
	}
	defer jsonFile.Close()

	if len(b) > 0 {
		e = json.Unmarshal(b, &result)
		if e != nil {
			err := errors.New(e.Error(), nil)
			return result, errors.New("failed to unmarshal config.json", err)
		}
	} else {
		logger.Info("Using default configuration settings. These can be edited in the config.json file")
	}

	rawJson, e := json.MarshalIndent(result, "", "    ")
	if e != nil {
		err := errors.New(e.Error(), nil)
		return config{}, errors.New("failed to marshal config.json", err)
	}

	_, e = jsonFile.Write(rawJson)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return result, errors.New("failed to write to config.json", err)
	}

	return result, nil
}
