package logger

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"main/plat"

	"git.sr.ht/~kvo/libgo/errors"
)

var buf bytes.Buffer

var useLogFile = false
var logFileFailCount = 0
var logFileFailLimit = 20
var logFileName string

var (
	fatalLogger = newLogger(&buf, "FATAL: ")
	errorLogger = newLogger(&buf, "ERROR: ")
	warnLogger  = newLogger(&buf, "WARN: ")
	infoLogger  = newLogger(&buf, "INFO: ")
	debugLogger = newLogger(&buf, "DEBUG: ")
)

// Set up the logger to use a config file. Invoking it will start logging to file as well as console.
// Must provide the path to where the log files should go.
func UseConfigFile(logPath string) errors.Error {
	useLogFile = true

	err := os.MkdirAll(logPath, os.ModePerm)
	if err != nil {
		return errors.New(
			"failed to create directory",
			errors.New(err.Error(), nil),
		)
	}

	logFileName = filepath.Join(logPath, time.Now().Format("2006-01-02_150405")+".log")
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		return errors.New(
			"could not open log file",
			errors.New(err.Error(), nil),
		)
	}
	defer logFile.Close()

	return nil
}

// Write the log to console, or console and log file. Buffer is reset automatically.
func write() {
	if useLogFile && logFileFailCount > logFileFailLimit {
		useLogFile = false
		Warn("Operations with the log file failed too many times. Logging to file has been disabled to prevent further errors")
	}

	if useLogFile {
		logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
		if err != nil {
			logFileFailCount += 1
			Error(errors.New(
				"could not open log file",
				errors.New(err.Error(), nil),
			))
		}
		defer logFile.Close()

		f := bufio.NewWriter(logFile)
		_, err = f.WriteString(buf.String())
		if err != nil {
			logFileFailCount += 1
			Error(errors.New(
				"could not write to log file",
				errors.New(err.Error(), nil),
			))
		}
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
	case errors.Error:
		err := fmt.Sprintf("%s: %s", a.Func(), a.Error())
		i := strings.Index(a.File(), "/src/")
		if i != -1 {
			err = fmt.Sprintf("%s:%d: %s", a.File()[i+1:], a.Line(), err)
		}
		infoLogger.logWrite(err, v...)
	case error:
		err := fmt.Errorf("%v", a)
		infoLogger.logWrite(err.Error(), v...)
	default:
		Fatal(plat.ErrInvalidInterfaceType)
	}
	write()
}

func Debug(format any, v ...any) {
	switch a := format.(type) {
	case string:
		debugLogger.logWrite(a, v...)
	case errors.Error:
		err := fmt.Sprintf("%s: %s", a.Func(), a.Error())
		i := strings.Index(a.File(), "/src/")
		if i != -1 {
			err = fmt.Sprintf("%s:%d: %s", a.File()[i+1:], a.Line(), err)
		}
		debugLogger.logWrite(err, v...)
		errors.Trace(debugLogger.out, a)
	case error:
		err := fmt.Errorf("%v", a)
		debugLogger.logWrite(err.Error(), v...)
	default:
		Fatal(plat.ErrInvalidInterfaceType)
	}
	write()
}

func Warn(format any, v ...any) {
	switch a := format.(type) {
	case string:
		warnLogger.logWrite(a, v...)
	case errors.Error:
		err := fmt.Sprintf("%s: %s", a.Func(), a.Error())
		i := strings.Index(a.File(), "/src/")
		if i != -1 {
			err = fmt.Sprintf("%s:%d: %s", a.File()[i+1:], a.Line(), err)
		}
		warnLogger.logWrite(err, v...)
	case error:
		err := fmt.Errorf("%v", a)
		warnLogger.logWrite(err.Error(), v...)
	default:
		Fatal(plat.ErrInvalidInterfaceType)
	}
	write()
}

func Error(format any, v ...any) {
	switch a := format.(type) {
	case string:
		errorLogger.logWrite(a, v...)
	case errors.Error:
		err := fmt.Sprintf("%s: %s", a.Func(), a.Error())
		i := strings.Index(a.File(), "/src/")
		if i != -1 {
			err = fmt.Sprintf("%s:%d: %s", a.File()[i+1:], a.Line(), err)
		}
		errorLogger.logWrite(err, v...)
	case error:
		err := fmt.Errorf("%v", a)
		errorLogger.logWrite(err.Error(), v...)
	default:
		Fatal(plat.ErrInvalidInterfaceType)
	}
	write()
}

// This will log the error, then call os.Exit(1).
func Fatal(format any, v ...any) {
	switch a := format.(type) {
	case string:
		fatalLogger.logWrite(a, v...)
	case errors.Error:
		err := fmt.Sprintf("%s: %s", a.Func(), a.Error())
		i := strings.Index(a.File(), "/src/")
		if i != -1 {
			err = fmt.Sprintf("%s:%d: %s", a.File()[i+1:], a.Line(), err)
		}
		fatalLogger.logWrite(err, v...)
	case error:
		err := fmt.Errorf("%v", a)
		fatalLogger.logWrite(err.Error(), v...)
	default:
		fatalLogger.logWrite(plat.ErrInvalidInterfaceType.Error())
	}
	write()
	os.Exit(1)
}
