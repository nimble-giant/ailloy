package commands

import (
	"strings"
	"testing"
)

func TestRunOreGet_NonRemoteRef_Errors(t *testing.T) {
	err := runOreGet(nil, []string{"./local/path"})
	if err == nil || !strings.Contains(err.Error(), "remote reference") {
		t.Errorf("expected remote-reference error, got %v", err)
	}
}
