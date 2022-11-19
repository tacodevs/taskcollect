package logger

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"main/errors"
)

var errInvalidInterfaceType = errors.NewError("logger", errors.ErrInvalidInterfaceType.Error(), nil)

var buf bytes.Buffer

var useLogFile = false
var logFileOpenFailCount = 0
var logFileOpenFailLimit = 20
var logFileName string

// TODO: Implement more customized logging formatting, such as the date formatting

var (
	infoLogger  = log.New(&buf, "INFO: ", log.Ldate|log.Ltime)
	debugLogger = log.New(&buf, "DEBUG: ", log.Ldate|log.Ltime)
	warnLogger  = log.New(&buf, "WARN: ", log.Ldate|log.Ltime)
	errorLogger = log.New(&buf, "ERROR: ", log.Ldate|log.Ltime)
	fatalLogger = log.New(&buf, "FATAL: ", log.Ldate|log.Ltime)
)

// Set up the logger to use a config file. Invoking it will start logging to file as well as console.
// Must provide the path to where the log files should go.
func UseConfigFile(logPath string) error {
	useLogFile = true

	err := os.MkdirAll(logPath, os.ModePerm)
	if err != nil {
		return err
	}

	logFileName = filepath.Join(logPath, time.Now().Format("2006-01-02_150405")+".log")
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		newErr := errors.NewError("logger", "could not open log file", err)
		return newErr
	}
	defer logFile.Close()

	return nil
}

// Write the log to console, or console and log file. Buffer is reset automatically.
func write() {
	if logFileOpenFailCount > logFileOpenFailLimit {
		useLogFile = false
		Warn("Log file failed to open too many times. Logging to file has been disabled to prevent further errors")
	}

	if useLogFile {
		logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
		if err != nil {
			newErr := errors.NewError("logger", "could not open log file", err)
			Error(newErr)
			logFileOpenFailCount += 1
		}
		defer logFile.Close()

		f := bufio.NewWriter(logFile)
		f.WriteString(buf.String())
		f.Flush()

		fmt.Print(buf.String())
	} else {
		fmt.Print(buf.String())
	}

	buf.Reset()
}

// NOTE: error case will always match before errors.ErrorWrapper since ErrorWrapper has its own
// Error() method (which error also has)

func Info(format any, v ...any) {
	switch a := format.(type) {
	case string:
		infoLogger.Printf(a, v...)
	case error:
		err := fmt.Errorf("%v", a)
		infoLogger.Printf(err.Error(), v...)
	//case errors.ErrorWrapper:
	//	err := a.AsString()
	//	infoLogger.Printf(err, v...)
	default:
		Fatal(errInvalidInterfaceType)
	}
	write()
}

// TODO: provide more diagnostic info for debug outputs, such as traceback abilities
func Debug(format any, v ...any) {
	switch a := format.(type) {
	case string:
		debugLogger.Printf(a, v...)
	case error:
		err := fmt.Errorf("%v", a)
		debugLogger.Printf(err.Error(), v...)
	//case errors.ErrorWrapper:
	//	err := a.AsString()
	//	debugLogger.Printf(err, v...)
	default:
		Fatal(errInvalidInterfaceType)
	}
	write()
}

func Warn(format any, v ...any) {
	switch a := format.(type) {
	case string:
		warnLogger.Printf(a, v...)
	case error:
		err := fmt.Errorf("%v", a)
		warnLogger.Printf(err.Error(), v...)
	//case errors.ErrorWrapper:
	//	err := a.AsString()
	//	warnLogger.Printf(err, v...)
	default:
		Fatal(errInvalidInterfaceType)
	}
	write()
}

func Error(format any, v ...any) {
	switch a := format.(type) {
	case string:
		errorLogger.Printf(a, v...)
	case error:
		err := fmt.Errorf("%v", a)
		errorLogger.Printf(err.Error(), v...)
	//case errors.ErrorWrapper:
	//	err := a.AsString()
	//	errorLogger.Printf(err, v...)
	default:
		Fatal(errInvalidInterfaceType)
	}
	write()
}

// This will log the error, then call os.Exit(1)
func Fatal(format any, v ...any) {
	switch a := format.(type) {
	case string:
		fatalLogger.Printf(a, v...)
	case errors.ErrorWrapper:
		err := a.AsString()
		fatalLogger.Printf(err, v...)
	case error:
		err := fmt.Errorf("%v", a)
		fatalLogger.Printf(err.Error(), v...)

	default:
		fatalLogger.Printf(errInvalidInterfaceType.Error())
	}
	write()
	os.Exit(1)
}
