// A reimplementation of a portion of Go's standard logging library (log)
// that better suits the needs of TaskCollect

package logger

import (
	"fmt"
	"io"
	"time"
)

type logWriter struct {
	out    io.Writer
	prefix string
}

// Printf prints to the standard logger. Arguments are handled in the manner of fmt.Printf.
func (lw logWriter) logWrite(format string, v ...any) {
	timeFormat := time.Now().Format("2006-01-02 15:04:05")
	str := fmt.Sprintf("%v %v%v\n", timeFormat, lw.prefix, fmt.Sprintf(format, v...))
	lw.out.Write([]byte(str))

}

// Create a new logger.
func newLogger(out io.Writer, prefix string) *logWriter {
	lw := &logWriter{out: out, prefix: prefix}
	return lw
}
