package server

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Response formats.
const (
	FormatTar       = "tar"
	FormatZip       = "zip"
	FormatNDJSON    = "ndjson"
	FormatMultipart = "multipart"
	FormatJSON      = "json"
)

// paxStrategyKey records a file's output strategy on its tar header so a
// client that cares can apply merge/append itself. Ordinary `tar` ignores it.
const paxStrategyKey = "AILLOY.strategy"

var formatAliases = map[string]string{
	"tar": FormatTar, "tgz": FormatTar, "tar.gz": FormatTar, "gzip": FormatTar,
	"zip":       FormatZip,
	"ndjson":    FormatNDJSON,
	"jsonl":     FormatNDJSON,
	"multipart": FormatMultipart,
	"json":      FormatJSON,
}

var acceptTypes = map[string]string{
	"application/gzip":     FormatTar,
	"application/x-gzip":   FormatTar,
	"application/x-tar":    FormatTar,
	"application/zip":      FormatZip,
	"application/x-ndjson": FormatNDJSON,
	"application/jsonl":    FormatNDJSON,
	"multipart/mixed":      FormatMultipart,
	"application/json":     FormatJSON,
}

// negotiateFormat picks the response format: an explicit format wins, then
// the first recognised Accept media type, then tar.
func negotiateFormat(explicit, accept string) (string, error) {
	if explicit != "" {
		if f, ok := formatAliases[strings.ToLower(explicit)]; ok {
			return f, nil
		}
		return "", fmt.Errorf("unknown format %q (supported: tar, zip, ndjson, multipart, json)", explicit)
	}
	for _, part := range strings.Split(accept, ",") {
		mt, _, err := mime.ParseMediaType(strings.TrimSpace(part))
		if err != nil {
			continue
		}
		if f, ok := acceptTypes[mt]; ok {
			return f, nil
		}
	}
	return FormatTar, nil
}

var unsafeFilename = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func setProvenanceHeaders(w http.ResponseWriter, res *CastResult) {
	h := w.Header()
	if res.Mold != "" {
		h.Set("X-Ailloy-Mold", res.Mold)
	}
	if res.Version != "" {
		h.Set("X-Ailloy-Mold-Version", res.Version)
	}
	if res.Commit != "" {
		h.Set("X-Ailloy-Mold-Commit", res.Commit)
	}
	h.Set("X-Ailloy-File-Count", fmt.Sprint(len(res.Files)))
}

func attachment(w http.ResponseWriter, res *CastResult, ext string) {
	name := unsafeFilename.ReplaceAllString(res.Mold, "-")
	if name == "" {
		name = "mold"
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name + "-cast" + ext}))
}

func writeResult(w http.ResponseWriter, format string, res *CastResult) error {
	setProvenanceHeaders(w, res)
	switch format {
	case FormatZip:
		return writeZip(w, res)
	case FormatNDJSON:
		return writeNDJSON(w, res)
	case FormatMultipart:
		return writeMultipart(w, res)
	case FormatJSON:
		return writeJSONResult(w, res)
	default:
		return writeTarGz(w, res)
	}
}

func writeTarGz(w http.ResponseWriter, res *CastResult) error {
	w.Header().Set("Content-Type", "application/gzip")
	attachment(w, res, ".tar.gz")
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)
	now := time.Now().UTC()
	for _, f := range res.Files {
		hdr := &tar.Header{
			Name:    f.Path,
			Mode:    0o644,
			Size:    int64(len(f.Content)),
			ModTime: now,
			Format:  tar.FormatPAX,
		}
		if f.Strategy != "" {
			hdr.PAXRecords = map[string]string{paxStrategyKey: f.Strategy}
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(f.Content); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

func writeZip(w http.ResponseWriter, res *CastResult) error {
	w.Header().Set("Content-Type", "application/zip")
	attachment(w, res, ".zip")
	zw := zip.NewWriter(w)
	now := time.Now().UTC()
	for _, f := range res.Files {
		fh := &zip.FileHeader{Name: f.Path, Method: zip.Deflate, Modified: now}
		fh.SetMode(0o644)
		if f.Strategy != "" {
			fh.Comment = "strategy=" + f.Strategy
		}
		fw, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}
		if _, err := fw.Write(f.Content); err != nil {
			return err
		}
	}
	return zw.Close()
}

type moldEvent struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Commit  string `json:"commit,omitempty"`
	Files   int    `json:"files"`
}

type fileEvent struct {
	Type     string `json:"type,omitempty"`
	Path     string `json:"path"`
	Strategy string `json:"strategy,omitempty"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
}

type doneEvent struct {
	Type  string `json:"type"`
	Files int    `json:"files"`
}

func encodeFile(f File) fileEvent {
	ev := fileEvent{Path: f.Path, Strategy: f.Strategy}
	if utf8.Valid(f.Content) {
		ev.Encoding = "utf-8"
		ev.Content = string(f.Content)
	} else {
		ev.Encoding = "base64"
		ev.Content = base64.StdEncoding.EncodeToString(f.Content)
	}
	return ev
}

// writeNDJSON emits one JSON object per line: a mold header, one event per
// file, then a done trailer. A client that sees no trailer knows the stream
// was cut short.
func writeNDJSON(w http.ResponseWriter, res *CastResult) error {
	w.Header().Set("Content-Type", "application/x-ndjson")
	enc := json.NewEncoder(w)
	flusher, _ := w.(http.Flusher)
	if err := enc.Encode(moldEvent{Type: "mold", Name: res.Mold, Version: res.Version, Commit: res.Commit, Files: len(res.Files)}); err != nil {
		return err
	}
	for _, f := range res.Files {
		ev := encodeFile(f)
		ev.Type = "file"
		if err := enc.Encode(ev); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
	return enc.Encode(doneEvent{Type: "done", Files: len(res.Files)})
}

// writeMultipart emits one multipart/mixed part per file, each carrying its
// destination in X-Ailloy-Path. The Content-Disposition filename carries the
// same path, but Go's (and most) multipart readers reduce it to a base name.
func writeMultipart(w http.ResponseWriter, res *CastResult) error {
	mw := multipart.NewWriter(w)
	w.Header().Set("Content-Type", "multipart/mixed; boundary="+mw.Boundary())
	for _, f := range res.Files {
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": f.Path}))
		h.Set("Content-Type", "application/octet-stream")
		h.Set("X-Ailloy-Path", f.Path)
		if f.Strategy != "" {
			h.Set("X-Ailloy-Strategy", f.Strategy)
		}
		pw, err := mw.CreatePart(h)
		if err != nil {
			return err
		}
		if _, err := pw.Write(f.Content); err != nil {
			return err
		}
	}
	return mw.Close()
}

func writeJSONResult(w http.ResponseWriter, res *CastResult) error {
	files := make([]fileEvent, 0, len(res.Files))
	for _, f := range res.Files {
		files = append(files, encodeFile(f))
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]any{
		"mold":  map[string]string{"name": res.Mold, "version": res.Version, "commit": res.Commit},
		"files": files,
	})
}
