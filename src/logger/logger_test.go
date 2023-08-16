package logger

import (
	"main/tests"
	"testing"
)

func TestUseConfigFile(t *testing.T) {
	logPath := tests.GetResPath("logs")
	if err := UseConfigFile(logPath); err != nil {
		t.Fail()
	}
}
