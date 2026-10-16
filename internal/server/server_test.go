package server

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeRenderer records the last request and returns a canned result.
type fakeRenderer struct {
	got    CastRequest
	result *CastResult
	err    error
}

func (f *fakeRenderer) Render(_ context.Context, req CastRequest) (*CastResult, error) {
	f.got = req
	return f.result, f.err
}

func cannedResult() *CastResult {
	return &CastResult{
		Mold:    "demo",
		Version: "v1.2.3",
		Commit:  "abc123",
		Files: []File{
			{Path: ".claude/commands/hello.md", Content: []byte("# Hello acme\n")},
			{Path: "CLAUDE.md", Content: []byte("root\n"), Strategy: "append"},
			{Path: "bin/blob", Content: []byte{0xff, 0x00, 0xfe}},
		},
	}
}

func newTestServer(t *testing.T, r Renderer, cfg Config) *httptest.Server {
	t.Helper()
	cfg.Renderer = r
	if cfg.Logger == nil {
		cfg.Logger = log.New(io.Discard, "", 0)
	}
	h, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return ts
}

func postCast(t *testing.T, ts *httptest.Server, body string, header map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/cast", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func readAll(t *testing.T, r io.Reader) []byte {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestNew_RequiresTokensOrPublic(t *testing.T) {
	if _, err := New(Config{Renderer: &fakeRenderer{}}); err == nil {
		t.Fatal("expected error when neither tokens nor Public are configured")
	}
	if _, err := New(Config{Renderer: &fakeRenderer{}, Public: true}); err != nil {
		t.Fatalf("public server: %v", err)
	}
	if _, err := New(Config{Renderer: &fakeRenderer{}, Tokens: []string{"s3cret"}}); err != nil {
		t.Fatalf("token server: %v", err)
	}
	if _, err := New(Config{Public: true}); err == nil {
		t.Fatal("expected error without a renderer")
	}
}

func TestHealthz_IsUnauthenticated(t *testing.T) {
	ts := newTestServer(t, &fakeRenderer{}, Config{Tokens: []string{"s3cret"}, Version: "v9"})
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" || body["version"] != "v9" {
		t.Fatalf("body = %v", body)
	}
}

func TestCast_RejectsMissingOrWrongToken(t *testing.T) {
	ts := newTestServer(t, &fakeRenderer{result: cannedResult()}, Config{Tokens: []string{"s3cret"}})
	for name, hdr := range map[string]map[string]string{
		"missing": nil,
		"wrong":   {"Authorization": "Bearer nope"},
		"scheme":  {"Authorization": "Basic s3cret"},
	} {
		t.Run(name, func(t *testing.T) {
			resp := postCast(t, ts, `{"mold":"github.com/acme/molds"}`, hdr)
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", resp.StatusCode)
			}
			if resp.Header.Get("WWW-Authenticate") == "" {
				t.Fatal("missing WWW-Authenticate header")
			}
		})
	}
}

func TestCast_AcceptsValidToken(t *testing.T) {
	ts := newTestServer(t, &fakeRenderer{result: cannedResult()}, Config{Tokens: []string{"other", "s3cret"}})
	resp := postCast(t, ts, `{"mold":"github.com/acme/molds"}`, map[string]string{"Authorization": "Bearer s3cret"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.StatusCode, readAll(t, resp.Body))
	}
}

func TestCast_DefaultsToGzipTarball(t *testing.T) {
	ts := newTestServer(t, &fakeRenderer{result: cannedResult()}, Config{Public: true})
	resp := postCast(t, ts, `{"mold":"github.com/acme/molds"}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", resp.StatusCode, readAll(t, resp.Body))
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/gzip" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "demo-cast.tar.gz") {
		t.Fatalf("Content-Disposition = %q", cd)
	}
	if resp.Header.Get("X-Ailloy-Mold-Version") != "v1.2.3" || resp.Header.Get("X-Ailloy-Mold-Commit") != "abc123" {
		t.Fatalf("provenance headers missing: %v", resp.Header)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	got := map[string]string{}
	strategies := map[string]string{}
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		got[h.Name] = string(readAll(t, tr))
		strategies[h.Name] = h.PAXRecords[paxStrategyKey]
	}
	if got[".claude/commands/hello.md"] != "# Hello acme\n" || got["CLAUDE.md"] != "root\n" {
		t.Fatalf("tar contents = %v", got)
	}
	if strategies["CLAUDE.md"] != "append" {
		t.Fatalf("CLAUDE.md strategy PAX record = %q, want append", strategies["CLAUDE.md"])
	}
}

func TestCast_Zip(t *testing.T) {
	ts := newTestServer(t, &fakeRenderer{result: cannedResult()}, Config{Public: true})
	resp := postCast(t, ts, `{"mold":"github.com/acme/molds","format":"zip"}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", resp.StatusCode, readAll(t, resp.Body))
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/zip" {
		t.Fatalf("Content-Type = %q", ct)
	}
	data := readAll(t, resp.Body)
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		got[f.Name] = string(readAll(t, rc))
		_ = rc.Close()
		if f.Name == "CLAUDE.md" && f.Comment != "strategy=append" {
			t.Fatalf("CLAUDE.md comment = %q", f.Comment)
		}
	}
	if got[".claude/commands/hello.md"] != "# Hello acme\n" || len(got) != 3 {
		t.Fatalf("zip contents = %v", got)
	}
}

func TestCast_NDJSONStreamsOneFilePerLine(t *testing.T) {
	ts := newTestServer(t, &fakeRenderer{result: cannedResult()}, Config{Public: true})
	resp := postCast(t, ts, `{"mold":"github.com/acme/molds"}`, map[string]string{"Accept": "application/x-ndjson"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", resp.StatusCode, readAll(t, resp.Body))
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/x-ndjson" {
		t.Fatalf("Content-Type = %q", ct)
	}
	var events []map[string]any
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		var ev map[string]any
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			t.Fatalf("line %q: %v", sc.Text(), err)
		}
		events = append(events, ev)
	}
	// mold header, 3 files, done trailer
	if len(events) != 5 {
		t.Fatalf("got %d events: %v", len(events), events)
	}
	if events[0]["type"] != "mold" || events[0]["name"] != "demo" {
		t.Fatalf("first event = %v", events[0])
	}
	if events[1]["type"] != "file" || events[1]["path"] != ".claude/commands/hello.md" || events[1]["content"] != "# Hello acme\n" {
		t.Fatalf("file event = %v", events[1])
	}
	if events[2]["strategy"] != "append" {
		t.Fatalf("strategy not carried: %v", events[2])
	}
	if events[3]["encoding"] != "base64" {
		t.Fatalf("binary file not base64-encoded: %v", events[3])
	}
	if raw, _ := base64.StdEncoding.DecodeString(events[3]["content"].(string)); !bytes.Equal(raw, []byte{0xff, 0x00, 0xfe}) {
		t.Fatalf("binary round-trip failed: %v", raw)
	}
	if events[4]["type"] != "done" || events[4]["files"] != float64(3) {
		t.Fatalf("trailer = %v", events[4])
	}
}

func TestCast_MultipartMixed(t *testing.T) {
	ts := newTestServer(t, &fakeRenderer{result: cannedResult()}, Config{Public: true})
	resp := postCast(t, ts, `{"mold":"github.com/acme/molds","format":"multipart"}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", resp.StatusCode, readAll(t, resp.Body))
	}
	mt, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || mt != "multipart/mixed" {
		t.Fatalf("Content-Type = %q (%v)", resp.Header.Get("Content-Type"), err)
	}
	mr := multipart.NewReader(resp.Body, params["boundary"])
	var names []string
	for {
		p, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, p.Header.Get("X-Ailloy-Path"))
		if p.FileName() == "CLAUDE.md" && p.Header.Get("X-Ailloy-Strategy") != "append" {
			t.Fatalf("strategy header missing on part: %v", p.Header)
		}
	}
	if strings.Join(names, ",") != ".claude/commands/hello.md,CLAUDE.md,bin/blob" {
		t.Fatalf("parts = %v", names)
	}
}

func TestCast_JSONListsFiles(t *testing.T) {
	ts := newTestServer(t, &fakeRenderer{result: cannedResult()}, Config{Public: true})
	resp := postCast(t, ts, `{"mold":"github.com/acme/molds","format":"json"}`, nil)
	var body struct {
		Mold  map[string]string `json:"mold"`
		Files []map[string]any  `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Mold["name"] != "demo" || len(body.Files) != 3 {
		t.Fatalf("body = %+v", body)
	}
}

func TestCast_GETQueryParams(t *testing.T) {
	fr := &fakeRenderer{result: cannedResult()}
	ts := newTestServer(t, fr, Config{Public: true})
	resp, err := http.Get(ts.URL + "/v1/cast?mold=github.com/acme/molds@v1&set=project.organization=acme&set=x=y&target=claude,codex&target=opencode&with_workflows=true&format=zip")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", resp.StatusCode, readAll(t, resp.Body))
	}
	if resp.Header.Get("Content-Type") != "application/zip" {
		t.Fatalf("Content-Type = %q", resp.Header.Get("Content-Type"))
	}
	if fr.got.Mold != "github.com/acme/molds@v1" || !fr.got.WithWorkflows {
		t.Fatalf("request = %+v", fr.got)
	}
	if strings.Join(fr.got.Targets, ",") != "claude,codex,opencode" {
		t.Fatalf("targets = %v", fr.got.Targets)
	}
	// Targets become the highest-precedence agent.targets override.
	want := "project.organization=acme,x=y,agent.targets=[claude,codex,opencode]"
	if strings.Join(fr.got.Set, ",") != want {
		t.Fatalf("set = %v, want %s", fr.got.Set, want)
	}
}

func TestCast_ValuesPassThrough(t *testing.T) {
	fr := &fakeRenderer{result: cannedResult()}
	ts := newTestServer(t, fr, Config{Public: true})
	resp := postCast(t, ts, `{"mold":"github.com/acme/molds","values":{"project":{"organization":"acme"}},"set":["a=b"]}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	proj, _ := fr.got.Values["project"].(map[string]any)
	if proj["organization"] != "acme" || strings.Join(fr.got.Set, ",") != "a=b" {
		t.Fatalf("request = %+v", fr.got)
	}
}

func TestCast_BadRequests(t *testing.T) {
	ts := newTestServer(t, &fakeRenderer{result: cannedResult()}, Config{Public: true})
	cases := map[string]string{
		"missing mold":   `{}`,
		"bad json":       `{"mold":`,
		"bad format":     `{"mold":"github.com/a/b","format":"rar"}`,
		"bad target":     `{"mold":"github.com/a/b","targets":["claude]"]}`,
		"bad set":        `{"mold":"github.com/a/b","set":["novalue"]}`,
		"unknown fields": `{"mold":"github.com/a/b","nope":1}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			resp := postCast(t, ts, body, nil)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", resp.StatusCode, readAll(t, resp.Body))
			}
		})
	}
}

func TestCast_RendererErrorStatus(t *testing.T) {
	fr := &fakeRenderer{err: Errorf(http.StatusForbidden, "source %q is not allowed", "x")}
	ts := newTestServer(t, fr, Config{Public: true})
	resp := postCast(t, ts, `{"mold":"github.com/acme/molds"}`, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if !strings.Contains(body["error"], "not allowed") {
		t.Fatalf("body = %v", body)
	}

	fr.err = errors.New("boom")
	resp = postCast(t, ts, `{"mold":"github.com/acme/molds"}`, nil)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("plain error status = %d, want 500", resp.StatusCode)
	}
}

func TestCast_RefusesUnsafeOutputPaths(t *testing.T) {
	for _, p := range []string{"../escape.md", "/etc/passwd", "a/../../b", ""} {
		t.Run(p, func(t *testing.T) {
			fr := &fakeRenderer{result: &CastResult{Mold: "evil", Files: []File{{Path: p, Content: []byte("x")}}}}
			ts := newTestServer(t, fr, Config{Public: true})
			resp := postCast(t, ts, `{"mold":"github.com/acme/molds"}`, nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422", resp.StatusCode)
			}
		})
	}
}

func TestCast_MethodNotAllowed(t *testing.T) {
	ts := newTestServer(t, &fakeRenderer{}, Config{Public: true})
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/v1/cast", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func TestCast_ConcurrencyLimit(t *testing.T) {
	block := make(chan struct{})
	started := make(chan struct{})
	br := &blockingRenderer{started: started, release: block}
	ts := newTestServer(t, br, Config{Public: true, MaxConcurrent: 1})

	done := make(chan int)
	go func() {
		resp, err := http.Post(ts.URL+"/v1/cast", "application/json", strings.NewReader(`{"mold":"github.com/a/b"}`))
		if err != nil {
			done <- 0
			return
		}
		_ = resp.Body.Close()
		done <- resp.StatusCode
	}()
	<-started

	resp := postCast(t, ts, `{"mold":"github.com/a/b"}`, nil)
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 while saturated", resp.StatusCode)
	}
	if resp.Header.Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
	close(block)
	if code := <-done; code != http.StatusOK {
		t.Fatalf("first request status = %d", code)
	}
}

type blockingRenderer struct {
	started chan struct{}
	release chan struct{}
}

func (b *blockingRenderer) Render(_ context.Context, _ CastRequest) (*CastResult, error) {
	close(b.started)
	<-b.release
	return cannedResult(), nil
}
