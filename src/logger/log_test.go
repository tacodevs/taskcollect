package logger

import (
	"bytes"
	"testing"
)

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	if lw := newLogger(&buf, "LOG_LEVEL: "); lw == nil {
		t.Fail()
	}
}
