package merge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
)

func tomlFixedPoint(t *testing.T, name, input string) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		n1, err := loadTOML([]byte(input))
		if err != nil {
			t.Fatalf("load1: %v\ninput: %s", err, input)
		}
		out1, err := dumpTOML(n1)
		if err != nil {
			t.Fatalf("dump1: %v", err)
		}
		n2, err := loadTOML(out1)
		if err != nil {
			t.Fatalf("load2: %v\nout1:\n%s", err, out1)
		}
		if !nodeEqual(n1, n2) {
			t.Errorf("dump changed the value.\ninput:\n%s\nout1:\n%s", input, out1)
		}
		out2, err := dumpTOML(n2)
		if err != nil {
			t.Fatalf("dump2: %v", err)
		}
		if string(out1) != string(out2) {
			t.Errorf("not a fixed point.\nout1:\n%s\nout2:\n%s", out1, out2)
		}
	})
}

func TestTOML_RoundTripFixedPoint(t *testing.T) {
	tomlFixedPoint(t, "empty", ``)
	tomlFixedPoint(t, "scalars", "s = \"hi\"\ni = 42\nf = 1.0\nb = true\nneg = -7\nbig = 9007199254740993\n")
	tomlFixedPoint(t, "special-floats", "a = inf\nb = -inf\nc = 1e+06\nd = 0.5\n")
	tomlFixedPoint(t, "datetimes", "odt = 1979-05-27T07:32:00Z\nldt = 1979-05-27T07:32:00\nld = 1979-05-27\nlt = 07:32:00\n")
	tomlFixedPoint(t, "strings", "nl = \"a\\nb\"\nq = 'it\"s'\ntab = \"x\\ty\"\nml = \"\"\"\nline1\nline2\"\"\"\nuni = \"héllo ✓\"\nctl = \"\\u0001\"\n")
	tomlFixedPoint(t, "quoted-keys", "\"a.b\" = 1\n\"has space\" = 2\n\"\" = 3\n[\"dotted.table\"]\nx = 1\n")
	tomlFixedPoint(t, "dotted-keys", "a.b.c = 1\na.b.d = 2\n")
	tomlFixedPoint(t, "nested-tables", "[a]\nx = 1\n[a.b]\ny = 2\n[a.b.c]\nz = 3\n")
	tomlFixedPoint(t, "implicit-parent", "[a.b.c]\nz = 3\n")
	tomlFixedPoint(t, "empty-table", "[a]\n[b]\nx = 1\n")
	tomlFixedPoint(t, "arrays", "a = [1, 2, 3]\nb = []\nc = [[1, 2], [\"x\"]]\nd = [\"a\", \"b\"]\n")
	tomlFixedPoint(t, "inline-table", "p = { x = 1, y = { z = \"q\" } }\n")
	tomlFixedPoint(t, "mixed-array", "m = [1, { a = 1 }, [2]]\n")
	tomlFixedPoint(t, "array-of-tables", "[[p]]\nname = \"a\"\n[[p]]\nname = \"b\"\nlist = [1]\n")
	tomlFixedPoint(t, "aot-with-subtables", "[[p]]\nname = \"a\"\n[p.meta]\nk = 1\n[[p.sub]]\nv = 1\n[[p.sub]]\nv = 2\n[[p]]\nname = \"b\"\n")
	tomlFixedPoint(t, "empty-aot-element", "[[p]]\n[[p]]\nx = 1\n")
	tomlFixedPoint(t, "nested-aot", "[a]\n[[a.b]]\nx = 1\n[[a.b]]\nx = 2\n")
}

func TestLoadTOML_PreservesKeyOrderAndTypes(t *testing.T) {
	n, err := loadTOML([]byte("zeta = 1\nalpha = 2.5\nmid = \"m\"\n[tbl]\nb = true\na = 1979-05-27T07:32:00Z\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(n.keys, ","); got != "zeta,alpha,mid,tbl" {
		t.Errorf("top-level order: got %s", got)
	}
	if got := strings.Join(n.fields["tbl"].keys, ","); got != "b,a" {
		t.Errorf("table order: got %s", got)
	}
	if _, ok := n.fields["zeta"].scalar.(int64); !ok {
		t.Errorf("int should stay int64, got %T", n.fields["zeta"].scalar)
	}
	if _, ok := n.fields["alpha"].scalar.(float64); !ok {
		t.Errorf("float should stay float64, got %T", n.fields["alpha"].scalar)
	}
	if _, ok := n.fields["tbl"].fields["a"].scalar.(time.Time); !ok {
		t.Errorf("datetime should stay time.Time, got %T", n.fields["tbl"].fields["a"].scalar)
	}
}

func TestDumpTOML_IntegralFloatStaysFloat(t *testing.T) {
	n, err := loadTOML([]byte("f = 1.0\n"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := dumpTOML(n)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "f = 1.0" {
		t.Errorf("got %q", out)
	}
}

func TestDumpTOML_OutputIsValidTOML(t *testing.T) {
	n, err := loadTOML([]byte("[[github.pr_monitor]]\nname = \"a\"\nbase_branches = [\"main\"]\n[mayor]\nappend_fragments = [\"x\"]\n"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := dumpTOML(n)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if _, err := toml.Decode(string(out), &v); err != nil {
		t.Fatalf("dump is not valid TOML: %v\n%s", err, out)
	}
}

// cityTOML mirrors the shape of a Gas City city.toml that the
// con-voyage-gascity mold needs to contribute to.
const cityTOML = `[workspace]
name = "repl_city"
provider = "claude"

[providers.cv-review-light]
base = "builtin:opencode"
model = "fireworks/custom"

[[github.pr_monitor]]
name = "ailloy-prs"
owner = "nimble-giant"
repo = "ailloy"
base_branches = ["main"]
merge_queue = "observe"

[mayor]
append_fragments = ["house-style"]
`

const conVoyageOverlay = `[[github.pr_monitor]]
name = "foundry-prs"
owner = "kriscoleman"
repo = "foundry"
base_branches = ["main"]
merge_queue = "observe"

[mayor]
append_fragments = ["con-voyage-orchestration"]
`

func TestMergeFile_TOML_CityConfig(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "city.toml")
	if err := os.WriteFile(dest, []byte(cityTOML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := MergeFile(dest, []byte(conVoyageOverlay), Options{}); err != nil {
		t.Fatalf("MergeFile: %v", err)
	}
	got, _ := os.ReadFile(dest)

	var v struct {
		Workspace struct{ Name, Provider string }
		Providers map[string]struct{ Base, Model string }
		GitHub    struct {
			PRMonitor []struct{ Name, Repo string } `toml:"pr_monitor"`
		}
		Mayor struct {
			AppendFragments []string `toml:"append_fragments"`
		}
	}
	if _, err := toml.Decode(string(got), &v); err != nil {
		t.Fatalf("merged city.toml does not parse: %v\n%s", err, got)
	}
	if v.Workspace.Name != "repl_city" || v.Workspace.Provider != "claude" {
		t.Errorf("workspace lost: %+v", v.Workspace)
	}
	if v.Providers["cv-review-light"].Model != "fireworks/custom" {
		t.Errorf("user provider override lost: %+v", v.Providers)
	}
	if len(v.GitHub.PRMonitor) != 2 || v.GitHub.PRMonitor[0].Name != "ailloy-prs" || v.GitHub.PRMonitor[1].Name != "foundry-prs" {
		t.Errorf("pr monitors: %+v", v.GitHub.PRMonitor)
	}
	if strings.Join(v.Mayor.AppendFragments, ",") != "house-style,con-voyage-orchestration" {
		t.Errorf("fragments: %v", v.Mayor.AppendFragments)
	}
	// Existing top-level tables keep their position.
	gs := string(got)
	headers := []string{"[workspace]", "[providers.cv-review-light]", "[[github.pr_monitor]]", "[mayor]"}
	for i := 1; i < len(headers); i++ {
		if strings.Index(gs, headers[i-1]) > strings.Index(gs, headers[i]) {
			t.Errorf("%s should precede %s:\n%s", headers[i-1], headers[i], gs)
		}
	}
}

func TestMergeFile_TOML_RecastIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "city.toml")
	if err := os.WriteFile(dest, []byte(cityTOML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := MergeFile(dest, []byte(conVoyageOverlay), Options{}); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(dest)
	if err := MergeFile(dest, []byte(conVoyageOverlay), Options{}); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(dest)
	if string(first) != string(second) {
		t.Errorf("recast changed file.\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestMergeFile_TOML_Malformed_ReturnsParseError(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "city.toml")
	if err := os.WriteFile(dest, []byte("[workspace\nname = "), 0644); err != nil {
		t.Fatal(err)
	}
	err := MergeFile(dest, []byte("a = 1\n"), Options{})
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("want *ParseError, got %v", err)
	}
	if pe.Format != "toml" {
		t.Errorf("format: got %q", pe.Format)
	}
	if err := MergeFile(dest, []byte("a = 1\n"), Options{ForceReplaceOnParseError: true}); err != nil {
		t.Fatalf("force replace: %v", err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "a = 1\n" {
		t.Errorf("force replace should write verbatim, got %q", got)
	}
}

func TestMergeFile_TOML_ScalarOverlayWins(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "x.TOML")
	if err := os.WriteFile(dest, []byte("a = 1\nb = \"keep\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := MergeFile(dest, []byte("a = 2\nc = 3.5\n"), Options{}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "a = 2\nb = \"keep\"\nc = 3.5\n" {
		t.Errorf("got:\n%s", got)
	}
}

func TestMergeFile_TOML_RecastDedupesOffsetDatetimes(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "c.toml")
	overlay := []byte("[[w]]\nat = 1979-05-27T07:32:00+07:00\nlocal = 1979-05-27T07:32:00\n")
	for range 2 {
		if err := MergeFile(dest, overlay, Options{}); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := os.ReadFile(dest)
	if c := strings.Count(string(got), "[[w]]"); c != 1 {
		t.Errorf("want 1 [[w]] after recast, got %d:\n%s", c, got)
	}
}
