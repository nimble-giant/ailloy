package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestPrintForgeDebugProvenance_FormatsRowsByOrigin(t *testing.T) {
	resolved := []mold.ResolvedFile{
		{SrcPath: "commands/hello.md", DestPath: ".claude/commands/hello.md", Origin: ""},
		{SrcPath: "blanks/AGENTS.md", DestPath: "AGENTS.md", Origin: "agent_targets"},
	}

	var buf bytes.Buffer
	printForgeDebugProvenance(&buf, resolved)

	out := buf.String()
	if !strings.Contains(out, "resolved output") {
		t.Errorf("missing header: %s", out)
	}
	if !strings.Contains(out, "@ mold") {
		t.Errorf("missing mold origin: %s", out)
	}
	if !strings.Contains(out, "@ ore:agent_targets") {
		t.Errorf("missing ore origin: %s", out)
	}
	if !strings.Contains(out, "AGENTS.md") || !strings.Contains(out, "blanks/AGENTS.md") {
		t.Errorf("missing dest/src paths: %s", out)
	}
}

