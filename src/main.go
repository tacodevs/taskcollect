package main

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

	"codeberg.org/kvo/builtin"
	"golang.org/x/term"

	"main/errors"
	"main/logger"
)

var (
	errAuthFailed = errors.NewError("main", errors.ErrAuthFailed.Error(), nil)
	//errCorruptMIME = errors.NewError("main", errors.ErrCorruptMIME.Error(), nil)
	//errIncompleteCreds = errors.NewError("main", errors.ErrIncompleteCreds.Error(), nil)
	errInvalidAuth = errors.NewError("main", errors.ErrInvalidAuth.Error(), nil)
	errNoPlatform  = errors.NewError("main", errors.ErrNoPlatform.Error(), nil)
	errNotFound    = errors.NewError("main", errors.ErrNotFound.Error(), nil)
	errNeedsGAuth  = errors.NewError("main", errors.ErrNeedsGAuth.Error(), nil)
)

// Create and manage necessary HTML files from template files.
func initTemplates(resPath string) (*template.Template, error) {
	// Create "./templates/" dir if it does not exist
	tmplPath := fp.Join(resPath, "templates")
	err := os.MkdirAll(tmplPath, os.ModePerm)
	if err != nil {
		newErr := errors.NewError("main.initTemplates", "could not make 'templates' directory", err)
		return nil, newErr
	}

	tmplResPath := tmplPath

	var files []string
	err = fp.WalkDir(tmplResPath, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			logger.Fatal(err)
		}
		// Skip the directory name itself from being appended, although its children won't be affected
		// Excluding via info.IsDir() will exclude files that are under a subdirectory so it cannot be used
		if info.Name() == "components" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		logger.Fatal(errors.NewError("main", "error walking the template/ directory", err))
	}

	files = builtin.Remove(files, tmplResPath)

	var requiredFiles []string
	rf := []string{
		"page", "head", "error", "login",
		"components/nav", "components/footer",
		"main", "res", "task", "tasks", "timetable",
	}
	for _, f := range rf {
		requiredFiles = append(requiredFiles, fp.Join(tmplResPath, f+".tmpl"))
	}

	filesMissing := false
	var missingFiles []string
	for _, f := range requiredFiles {
		if !builtin.Contains(files, f) {
			filesMissing = true
			missingFiles = append(missingFiles, f)
		}
	}
	if filesMissing {
		errStr := fmt.Errorf("%v:\n%+v", errors.ErrMissingFiles.Error(), missingFiles)
		logger.Fatal(errors.NewError("main", errStr.Error(), nil))
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

func getConfig(cfgPath string) (config, error) {
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

	jsonFile, err := os.OpenFile(cfgPath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		newErr := errors.NewError("main.getConfig", "failed to open config.json", err)
		return result, newErr
	}

	b, err := io.ReadAll(jsonFile)
	if err != nil {
		newErr := errors.NewError("main.getConfig", "failed to read config.json", err)
		return result, newErr
	}

	err = jsonFile.Close()
	if err != nil {
		newErr := errors.NewError("main.getConfig", "failed to close config.json", err)
		return result, newErr
	}

	jsonFile, err = os.OpenFile(cfgPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0622)
	if err != nil {
		newErr := errors.NewError("main.getConfig", "failed to open config.json", err)
		return result, newErr
	}
	defer jsonFile.Close()

	if len(b) > 0 {
		err = json.Unmarshal(b, &result)
		if err != nil {
			newErr := errors.NewError("main.getConfig", "failed to unmarshal config.json", err)
			return result, newErr
		}
	} else {
		logger.Info("Using default configuration settings. These can be edited in the config.json file")
	}

	rawJson, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		newErr := errors.NewError("main.getConfig", "failed to marshal config.json", err)
		return config{}, newErr
	}

	_, err = jsonFile.Write(rawJson)
	if err != nil {
		newErr := errors.NewError("main.getConfig", "failed to write to config.json", err)
		return result, newErr
	}

	return result, nil
}

func main() {
	tlsConn := true

	if len(os.Args) > 2 || len(os.Args) == 2 && os.Args[1] != "-w" {
		logger.Info(errors.ErrBadCommandUsage)
		os.Exit(1)
	}

	if builtin.Contains(os.Args, "-w") {
		tlsConn = false
	}

	fmt.Println(tcVersion)
	fmt.Println("")

	curUser, err := user.Current()
	if err != nil {
		logger.Fatal("taskcollect: Cannot determine current user's home folder")
	}

	home := curUser.HomeDir
	resPath := fp.Join(home, "res/taskcollect")
	configFile := fp.Join(resPath, "config.json")
	certFile := fp.Join(resPath, "cert.pem")
	keyFile := fp.Join(resPath, "key.pem")

	result, err := getConfig(configFile)
	if err != nil {
		newErr := errors.NewError("main", "unable to read config file", err)
		logger.Error(newErr)
		logger.Warn("Resorting to default configuration settings")
	}

	// TODO: Implement logging to file with further options
	if result.Logging.UseLogFile {
		logPath := fp.Join(resPath, "logs")
		err = logger.UseConfigFile(logPath)
		if err != nil {
			newErr := errors.NewError("main", "Log file was not set up successfully", err)
			logger.Error(newErr)
		} else {
			logger.Info("Log file set up successfully")
		}
	}

	logger.Info("Running %v", tcVersion)

	var password string
	fmt.Print("Password to Redis database: ")
	dbPwdInput, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		newErr := errors.NewError("main", "could not get password input", err)
		logger.Fatal(newErr)
	}
	fmt.Println()

	dbAddr := result.Database.Address
	dbIdx := result.Database.Index
	password = string(dbPwdInput)
	newRedisDB := initDB(dbAddr, password, dbIdx)
	logger.Info("Connected to Redis on %s with database index of %d", dbAddr, dbIdx)

	gcid, err := os.ReadFile(fp.Join(resPath, "gauth.json"))
	if err != nil {
		newErr := errors.NewError("main", "Google client ID "+errors.ErrFileRead.Error(), err)
		logger.Fatal(newErr)
	}

	templates, err := initTemplates(resPath)
	if err != nil {
		newErr := errors.NewError("main", errors.ErrInitFailed.Error()+" for HTML templates", err)
		logger.Fatal(newErr)
	}
	logger.Info("Successfully initialized HTML templates")

	db := authDB{
		path:   resPath,
		client: newRedisDB,
		gAuth:  gcid,
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
	mux.HandleFunc("/login", h.loginHandler)
	mux.HandleFunc("/logout", h.logoutHandler)
	mux.HandleFunc("/auth", h.authHandler)
	mux.HandleFunc("/", h.rootHandler)

	if tlsConn {
		logger.Info("Running on port 443")
		err = http.ListenAndServeTLS(":443", certFile, keyFile, mux)
	} else {
		logger.Warn("Running on port 8080 (without TLS). DO NOT USE THIS IN PRODUCTION!")
		err = http.ListenAndServe(":8080", mux)
	}

	if err != nil {
		logger.Fatal(err)
	}
}
