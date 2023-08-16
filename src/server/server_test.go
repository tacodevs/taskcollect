package server

import "testing"

func TestGetConfig(t *testing.T) {
	// The test config.json has useLogFile set to true
	cfgPath := "../tests/data/config.json"
	if _, err := getConfig(cfgPath); err != nil {
		t.Fail()
	}
}

func TestInitTemplates(t *testing.T) {
	// Not implemented
}
