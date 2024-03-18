// Package log implements a basic logging package. Each log record includes the
// date, time, severity level, and description of the reported incident by
// default.
//
// The package defines a type, Logger, which provides several methods (such as
// Logger.Info and Logger.Error) for reporting incidents at different severity
// levels. Top-level functions sharing the same names use the default logger's
// methods, unless a custom logger is specified with SetDefault. These default
// methods behave like fmt.Print but log to os.Stderr, while associated methods
// suffixed by an f (i.e. Errorf) behave like fmt.Printf). The only exception is
// Debug, which has no associated counterpart and only accepts a single error as
// an argument.
//
// A number of severity levels are defined by default. Incidents at levels INFO,
// WARN, and ERROR represent important notices, warnings, and errors,
// respectively. Reporting an incident at level FATAL will result in the
// subsequent termination of the program with os.Exit(1). Reporting an error
// at level DEBUG will cause the logger to print the error's traceback. Any of
// these methods can be redefined by a custom logger; the above information
// pertains to the standard Logger type.
//
// Variable names and values can also be logged at level DEBUG with the Values
// function:
//	2006-01-02 15:04:05 DEBUG: key1=value1 key2=value2
//
// If the default logger does not suffice, a custom logger may be implemented.
// This can be done by either implementing an alternative logger from scratch,
// creating a new instance of the Logger struct, or creating a custom logger
// struct which extends the standard logger through struct embedding.
//
// Structured logging is not yet implemented. This may be added in a future
// version of the package, with support for encoding and decoding structured
// logs.
package logger

import (
	"fmt"
	"io"
	"time"
)

func Fatal(v ...any) {
}

func Fatalf(format string, v ...any) {
}

func Debug(err error) {
}

func Debugf(format string, v ...any) {
}

func Error(v ...any) {
}

func Errorf(format string, v ...any) {
}

func Warn(v ...any) {
}

func Warnf(format string, v ...any) {
}

func Info(v ...any) {
}

func Infof(format string, v ...any) {
}

func Prefix() string {
	return ""
}

func SetPrefix(format string) {
}

func SetWriter(w io.Writer) {
}

func Values(v ...Vars) {
}

func Writer() io.Writer {
	return nil
}

type Logger struct {
}

func Default() *Logger {
	return nil
}

func New(out io.Writer, prefix string) *Logger {
	return nil
}

func (l *Logger) Fatal(v ...any) {
}

func (l *Logger) Fatalf(format string, v ...any) {
}

func (l *Logger) Debug(err error) {
}

func (l *Logger) Debugf(format string, v ...any) {
}

func (l *Logger) Error(v ...any) {
}

func (l *Logger) Errorf(format string, v ...any) {
}

func (l *Logger) Warn(v ...any) {
}

func (l *Logger) Warnf(format string, v ...any) {
}

func (l *Logger) Info(v ...any) {
}

func (l *Logger) Infof(format string, v ...any) {
}

func (l *Logger) Prefix() string {
	return ""
}

func (l *Logger) SetPrefix(format string) {
}

func (l *Logger) SetWriter(w io.Writer) {
}

func (l *Logger) Values(v ...Vars) {
}

func (l *Logger) Writer() io.Writer {
	return nil
}
