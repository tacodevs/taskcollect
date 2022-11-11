package logger

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"main/errors"
)

// TODO: set up log file functionality
var useLogFile = false

var buf bytes.Buffer

var errInvalidInterfaceType = errors.NewError("logger", nil, errors.ErrInvalidInterfaceType.Error())

var (
	infoLogger  = log.New(&buf, "INFO: ", log.Ldate|log.Ltime)
	debugLogger = log.New(&buf, "DEBUG: ", log.Ldate|log.Ltime)
	warnLogger  = log.New(&buf, "WARN: ", log.Ldate|log.Ltime)
	errorLogger = log.New(&buf, "ERROR: ", log.Ldate|log.Ltime)
	fatalLogger = log.New(&buf, "FATAL: ", log.Ldate|log.Ltime)
)

// Set up the logger with preferences - such as logging to a file
func UseConfig(cfgFile string) error {
	// open file

	// read JSON

	// set useLogFile to true if user requests it
	return nil
}

// NOTE: error case will always match before errors.ErrorWrapper since ErrorWrapper has its own
// Error() method (which error also has)

func Info(format any, v ...any) {
	switch a := format.(type) {
	case string:
		infoLogger.Printf(a, v...)
	case error:
		log.Println("error case")
		err := fmt.Errorf("%w", a)
		infoLogger.Printf(err.Error(), v...)
	//case errors.ErrorWrapper:
	//	err := a.AsString()
	//	infoLogger.Printf(err, v...)
	default:
		Fatal(errInvalidInterfaceType)
	}
	fmt.Print(buf.String())
	buf.Reset()
}

// TODO: provide more diagnostic info for debug outputs, such as traceback abilities
func Debug(format any, v ...any) {
	switch a := format.(type) {
	case string:
		debugLogger.Printf(a, v...)
	case error:
		err := fmt.Errorf("%w", a)
		debugLogger.Printf(err.Error(), v...)
	//case errors.ErrorWrapper:
	//	err := a.AsString()
	//	debugLogger.Printf(err, v...)
	default:
		Fatal(errInvalidInterfaceType)
	}
	fmt.Print(buf.String())
	buf.Reset()
}

func Warn(format any, v ...any) {
	switch a := format.(type) {
	case string:
		warnLogger.Printf(a, v...)
	case error:
		err := fmt.Errorf("%w", a)
		warnLogger.Printf(err.Error(), v...)
	//case errors.ErrorWrapper:
	//	err := a.AsString()
	//	warnLogger.Printf(err, v...)
	default:
		Fatal(errInvalidInterfaceType)
	}
	fmt.Print(buf.String())
	buf.Reset()
}

func Error(format any, v ...any) {
	switch a := format.(type) {
	case string:
		errorLogger.Printf(a, v...)
	case error:
		err := fmt.Errorf("%w", a)
		errorLogger.Printf(err.Error(), v...)
	//case errors.ErrorWrapper:
	//	err := a.AsString()
	//	errorLogger.Printf(err, v...)
	default:
		Fatal(errInvalidInterfaceType)
	}
	fmt.Print(buf.String())
	buf.Reset()
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
		err := fmt.Errorf("%w", a)
		fatalLogger.Printf(err.Error(), v...)

	default:
		fatalLogger.Printf(errInvalidInterfaceType.Error())
	}

	fmt.Print(buf.String())
	os.Exit(1)
}
