package smelt

import (
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/knadh/stuffbin"
)

// StuffFS wraps a stuffbin.FileSystem to implement fs.FS and fs.ReadFileFS.
// This allows MoldReader to work with mold files embedded in a stuffed binary.
type StuffFS struct {
	sfs stuffbin.FileSystem
}

// NewStuffFS creates a new StuffFS from a stuffbin.FileSystem.
func NewStuffFS(sfs stuffbin.FileSystem) *StuffFS {
	return &StuffFS{sfs: sfs}
}

// stuffPath converts an fs.FS-style path to a stuffbin path (adds leading /).
func stuffPath(name string) string {
	if name == "." || name == "" {
		return "/"
	}
	return "/" + name
}

// Open implements fs.FS.
func (s *StuffFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	// Try as regular file first.
	sp := stuffPath(name)
	f, err := s.sfs.Get(sp)
	if err == nil {
		data := f.ReadBytes()
		return &stuffFile{
			name: path.Base(name),
			data: data,
			r:    strings.NewReader(string(data)),
		}, nil
	}

	// Try as directory — check if any files have this prefix.
	return s.openDir(name)
}

// ReadFile implements fs.ReadFileFS for efficient reads without Open overhead.
func (s *StuffFS) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrInvalid}
	}
	data, err := s.sfs.Read(stuffPath(name))
	if err != nil {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
	}
	return data, nil
}

// openDir synthesizes a directory from the flat file list.
func (s *StuffFS) openDir(name string) (fs.File, error) {
	prefix := stuffPath(name)
	if prefix != "/" {
		prefix += "/"
	}

	seen := make(map[string]bool)
	var entries []fs.DirEntry

	for _, fp := range s.sfs.List() {
		if !strings.HasPrefix(fp, prefix) {
			continue
		}
		rest := strings.TrimPrefix(fp, prefix)
		parts := strings.SplitN(rest, "/", 2)
		child := parts[0]
		if child == "" || seen[child] {
			continue
		}
		seen[child] = true

		isDir := len(parts) > 1
		entries = append(entries, &stuffDirEntry{
			name:  child,
			isDir: isDir,
		})
	}

	if len(entries) == 0 {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	dirName := path.Base(name)
	if name == "." {
		dirName = "."
	}
	return &stuffDir{name: dirName, entries: entries}, nil
}

// Stat implements fs.StatFS.
func (s *StuffFS) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrInvalid}
	}

	// Try as file.
	sp := stuffPath(name)
	f, err := s.sfs.Get(sp)
	if err == nil {
		return &stuffFileInfo{
			name: path.Base(name),
			size: int64(len(f.ReadBytes())),
		}, nil
	}

	// Try as directory.
	prefix := sp
	if prefix != "/" {
		prefix += "/"
	}
	for _, fp := range s.sfs.List() {
		if strings.HasPrefix(fp, prefix) {
			return &stuffFileInfo{
				name:  path.Base(name),
				isDir: true,
			}, nil
		}
	}

	return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
}

// stuffFile implements fs.File for a regular file.
type stuffFile struct {
	name string
	data []byte
	r    *strings.Reader
}

func (f *stuffFile) Stat() (fs.FileInfo, error) {
	return &stuffFileInfo{name: f.name, size: int64(len(f.data))}, nil
}

func (f *stuffFile) Read(b []byte) (int, error) {
	return f.r.Read(b)
}

func (f *stuffFile) Close() error {
	return nil
}

// stuffDir implements fs.ReadDirFile for synthesized directories.
type stuffDir struct {
	name    string
	entries []fs.DirEntry
	offset  int
}

func (d *stuffDir) Stat() (fs.FileInfo, error) {
	return &stuffFileInfo{name: d.name, isDir: true}, nil
}

func (d *stuffDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.name, Err: fs.ErrInvalid}
}

func (d *stuffDir) Close() error {
	return nil
}

func (d *stuffDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if n <= 0 {
		entries := d.entries[d.offset:]
		d.offset = len(d.entries)
		return entries, nil
	}

	if d.offset >= len(d.entries) {
		return nil, io.EOF
	}

	end := min(d.offset+n, len(d.entries))
	entries := d.entries[d.offset:end]
	d.offset = end

	if d.offset >= len(d.entries) {
		return entries, io.EOF
	}
	return entries, nil
}

// stuffFileInfo implements fs.FileInfo.
type stuffFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (i *stuffFileInfo) Name() string      { return i.name }
func (i *stuffFileInfo) Size() int64       { return i.size }
func (i *stuffFileInfo) Mode() fs.FileMode { return 0444 }
func (i *stuffFileInfo) ModTime() time.Time {
	return time.Time{}
}
func (i *stuffFileInfo) IsDir() bool                { return i.isDir }
func (i *stuffFileInfo) Sys() any                   { return nil }
func (i *stuffFileInfo) Type() fs.FileMode          { return i.Mode().Type() }
func (i *stuffFileInfo) Info() (fs.FileInfo, error) { return i, nil }

// stuffDirEntry implements fs.DirEntry.
type stuffDirEntry struct {
	name  string
	isDir bool
}

func (e *stuffDirEntry) Name() string      { return e.name }
func (e *stuffDirEntry) IsDir() bool       { return e.isDir }
func (e *stuffDirEntry) Type() fs.FileMode { return e.mode().Type() }
func (e *stuffDirEntry) Info() (fs.FileInfo, error) {
	return &stuffFileInfo{name: e.name, isDir: e.isDir}, nil
}
func (e *stuffDirEntry) mode() fs.FileMode {
	if e.isDir {
		return fs.ModeDir | 0555
	}
	return 0444
}
