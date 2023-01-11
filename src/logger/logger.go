package logger

import (
	"bufio"
	"bytes"
	"fmt"
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

var (
	infoLogger  = newLogger(&buf, "INFO: ")
	debugLogger = newLogger(&buf, "DEBUG: ")
	warnLogger  = newLogger(&buf, "WARN: ")
	errorLogger = newLogger(&buf, "ERROR: ")
	fatalLogger = newLogger(&buf, "FATAL: ")
)

// Set up the logger to use a config file. Invoking it will start logging to file as well as console.
// Must provide the path to where the log files should go.
func UseConfigFile(logPath string) error {
	useLogFile = true

	err := os.MkdirAll(logPath, os.ModePerm)
	if err != nil {
		newErr := errors.NewError("logger.UseConfigFile", "failed to create directory", err)
		return newErr
	}

	logFileName = filepath.Join(logPath, time.Now().Format("2006-01-02_150405")+".log")
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		newErr := errors.NewError("logger.UseConfigFile", "could not open log file", err)
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
			newErr := errors.NewError("logger.write", "could not open log file", err)
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

func Info(format any, v ...any) {
	switch a := format.(type) {
	case string:
		infoLogger.logWrite(a, v...)

	case error:
		err := fmt.Errorf("%v", a)
		infoLogger.logWrite(err.Error(), v...)
	default:
		Fatal(errInvalidInterfaceType)
	}
	write()
}

// TODO: provide more diagnostic info for debug outputs, such as traceback abilities
func Debug(format any, v ...any) {
	switch a := format.(type) {
	case string:
		debugLogger.logWrite(a, v...)
	case error:
		err := fmt.Errorf("%v", a)
		debugLogger.logWrite(err.Error(), v...)
	default:
		Fatal(errInvalidInterfaceType)
	}
	write()
}

func Warn(format any, v ...any) {
	switch a := format.(type) {
	case string:
		warnLogger.logWrite(a, v...)
	case error:
		err := fmt.Errorf("%v", a)
		warnLogger.logWrite(err.Error(), v...)
	default:
		Fatal(errInvalidInterfaceType)
	}
	write()
}

func Error(format any, v ...any) {
	switch a := format.(type) {
	case string:
		errorLogger.logWrite(a, v...)
	case error:
		err := fmt.Errorf("%v", a)
		errorLogger.logWrite(err.Error(), v...)
	default:
		Fatal(errInvalidInterfaceType)
	}
	write()
}

// This will log the error, then call os.Exit(1).
func Fatal(format any, v ...any) {
	switch a := format.(type) {
	case string:
		fatalLogger.logWrite(a, v...)
	case errors.ErrorWrapper:
		err := a.AsString()
		fatalLogger.logWrite(err, v...)
	case error:
		err := fmt.Errorf("%v", a)
		fatalLogger.logWrite(err.Error(), v...)

	default:
		fatalLogger.logWrite(errInvalidInterfaceType.Error())
	}
	write()
	os.Exit(1)
}
