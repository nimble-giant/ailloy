// Package server exposes ailloy's cast pipeline over HTTP so clients can
// fetch a rendered mold without installing the ailloy CLI.
//
// The package owns the HTTP surface only: request parsing, bearer-token auth,
// format negotiation and response encoding (tar.gz, zip, NDJSON,
// multipart/mixed, JSON). Resolving and rendering a mold is delegated to a
// Renderer, which the `ailloy serve` command wires to the real ephemeral
// render pipeline. Nothing here touches the server's filesystem.
package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// CastRequest is the body of POST /v1/cast (and the decoded query of
// GET /v1/cast). Fields mirror the `ailloy cast` flags.
type CastRequest struct {
	// Mold is a remote foundry reference, e.g. github.com/acme/molds@^1.0.
	Mold string `json:"mold"`
	// Set holds --set style key=value overrides, applied in order.
	Set []string `json:"set,omitempty"`
	// Values is an inline equivalent of a -f/--values flux file.
	Values map[string]any `json:"values,omitempty"`
	// Targets selects agent harnesses (claude, codex, opencode, ...). It is
	// forwarded to the renderer as the highest-precedence
	// `agent.targets=[...]` override.
	Targets []string `json:"targets,omitempty"`
	// WithWorkflows includes .github/ workflow blanks (off by default, like cast).
	WithWorkflows bool `json:"with_workflows,omitempty"`
	// Format selects the response encoding: tar (default), zip, ndjson,
	// multipart or json. Overrides the Accept header when set.
	Format string `json:"format,omitempty"`
}

// File is one rendered output file, at its cast destination path.
type File struct {
	Path     string // slash-separated, relative to the project root
	Content  []byte
	Strategy string // "", "replace", "merge" or "append"
}

// CastResult is what a Renderer produces for one request.
type CastResult struct {
	Mold    string // mold.yaml name
	Version string // resolved tag (or branch)
	Commit  string // resolved commit SHA
	Files   []File
}

// Renderer resolves and renders a mold. Implementations return an *Error to
// choose the HTTP status; any other error maps to 500.
type Renderer interface {
	Render(ctx context.Context, req CastRequest) (*CastResult, error)
}

// Error carries an HTTP status alongside a client-safe message.
type Error struct {
	Status int
	Err    error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

// Errorf builds an *Error with the given status.
func Errorf(status int, format string, args ...any) error {
	return &Error{Status: status, Err: fmt.Errorf(format, args...)}
}

// Config configures New.
type Config struct {
	Renderer Renderer
	// Tokens are accepted bearer tokens. At least one is required unless
	// Public is true.
	Tokens []string
	// Public serves /v1/cast without authentication.
	Public bool
	// MaxConcurrent caps in-flight casts; extra requests get 503. Default 8.
	MaxConcurrent int
	// MaxBodyBytes caps the request body. Default 1 MiB.
	MaxBodyBytes int64
	// Version is reported by /healthz.
	Version string
	// Logger receives one line per cast request. Defaults to log.Default().
	Logger *log.Logger
}

type server struct {
	cfg    Config
	tokens [][sha256.Size]byte
	slots  chan struct{}
}

// New returns the API handler. It refuses to build an open server by
// accident: either Tokens or Public must be set.
func New(cfg Config) (http.Handler, error) {
	if cfg.Renderer == nil {
		return nil, errors.New("server: a Renderer is required")
	}
	var tokens [][sha256.Size]byte
	for _, t := range cfg.Tokens {
		if t = strings.TrimSpace(t); t != "" {
			tokens = append(tokens, sha256.Sum256([]byte(t)))
		}
	}
	if len(tokens) == 0 && !cfg.Public {
		return nil, errors.New("server: refusing to start without auth; configure at least one token or opt in to public access")
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 8
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = 1 << 20
	}
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	s := &server{cfg: cfg, tokens: tokens, slots: make(chan struct{}, cfg.MaxConcurrent)}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/v1/cast", s.handleCast)
	return mux, nil
}

func (s *server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": s.cfg.Version})
}

func (s *server) authorized(r *http.Request) bool {
	if len(s.tokens) == 0 {
		return s.cfg.Public
	}
	scheme, tok, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return false
	}
	// Compare fixed-size digests so neither token length nor content leaks
	// through timing.
	sum := sha256.Sum256([]byte(strings.TrimSpace(tok)))
	match := 0
	for _, t := range s.tokens {
		match |= subtle.ConstantTimeCompare(sum[:], t[:])
	}
	return match == 1
}

func (s *server) handleCast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "use GET or POST")
		return
	}
	if !s.authorized(r) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ailloy"`)
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		w.Header().Set("Retry-After", "5")
		writeError(w, http.StatusServiceUnavailable, "server is busy; retry shortly")
		return
	}

	req, err := s.decodeRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	format, err := negotiateFormat(req.Format, r.Header.Get("Accept"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := s.cfg.Renderer.Render(r.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		var he *Error
		if errors.As(err, &he) {
			status = he.Status
		}
		s.cfg.Logger.Printf("cast %s: %d %v", req.Mold, status, err)
		writeError(w, status, err.Error())
		return
	}
	for i, f := range res.Files {
		clean, ok := safeArchivePath(f.Path)
		if !ok {
			s.cfg.Logger.Printf("cast %s: refusing unsafe output path %q", req.Mold, f.Path)
			writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("mold produced an unsafe output path %q", f.Path))
			return
		}
		res.Files[i].Path = clean
	}

	s.cfg.Logger.Printf("cast %s: %d files as %s", req.Mold, len(res.Files), format)
	if err := writeResult(w, format, res); err != nil {
		// Headers are already sent; all we can do is log and drop the connection.
		s.cfg.Logger.Printf("cast %s: writing response: %v", req.Mold, err)
	}
}

// targetName keeps target names to a shape that is safe to splice into a
// YAML flow sequence for the agent.targets override.
var targetName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

func (s *server) decodeRequest(r *http.Request) (CastRequest, error) {
	var req CastRequest
	if r.Method == http.MethodPost {
		dec := json.NewDecoder(io.LimitReader(r.Body, s.cfg.MaxBodyBytes))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			return req, fmt.Errorf("invalid JSON body: %w", err)
		}
	} else {
		q := r.URL.Query()
		req.Mold = q.Get("mold")
		req.Set = q["set"]
		for _, t := range q["target"] {
			req.Targets = append(req.Targets, strings.Split(t, ",")...)
		}
		req.Format = q.Get("format")
		if v := q.Get("with_workflows"); v != "" {
			b, err := strconv.ParseBool(v)
			if err != nil {
				return req, fmt.Errorf("with_workflows: %w", err)
			}
			req.WithWorkflows = b
		}
	}

	req.Mold = strings.TrimSpace(req.Mold)
	if req.Mold == "" {
		return req, errors.New("mold is required")
	}
	for _, kv := range req.Set {
		if k, _, ok := strings.Cut(kv, "="); !ok || strings.TrimSpace(k) == "" {
			return req, fmt.Errorf("invalid set %q (expected key=value)", kv)
		}
	}
	var targets []string
	for _, t := range req.Targets {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if !targetName.MatchString(t) {
			return req, fmt.Errorf("invalid target %q", t)
		}
		targets = append(targets, t)
	}
	req.Targets = targets
	if len(targets) > 0 {
		req.Set = append(req.Set, "agent.targets=["+strings.Join(targets, ",")+"]")
	}
	return req, nil
}

// safeArchivePath normalizes a destination path and rejects anything that
// could escape the extraction directory.
func safeArchivePath(p string) (string, bool) {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") {
		return "", false
	}
	clean := path.Clean(p)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	return clean, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
