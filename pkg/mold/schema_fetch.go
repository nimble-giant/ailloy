package mold

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// FetchSchemaFromSource resolves a mold by source string (local path or remote
// ref) and returns its flux schema and default values, without doing a full
// cast. For local paths it reads from the filesystem directly; for remote refs
// it delegates to ResolveSchemaFunc (set by callers that have access to the
// foundry resolver to avoid an import cycle).
//
// When neither flux.schema.yaml nor flux.yaml exist, returns an empty schema
// and empty defaults with no error.
func FetchSchemaFromSource(ctx context.Context, source string) ([]FluxVar, map[string]any, error) {
	if source == "" {
		return nil, nil, errors.New("FetchSchemaFromSource: empty source")
	}
	// Local path?
	if info, err := os.Stat(source); err == nil && info.IsDir() {
		return loadSchemaFromFS(os.DirFS(source))
	}
	if ResolveSchemaFunc == nil {
		return nil, nil, fmt.Errorf("FetchSchemaFromSource: no remote resolver registered for source %q", source)
	}
	fsys, cleanup, err := ResolveSchemaFunc(ctx, source)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return nil, nil, err
	}
	return loadSchemaFromFS(fsys)
}

// ResolveSchemaFunc is set by callers that can resolve a remote mold to an
// fs.FS without creating an import cycle into pkg/mold. The returned cleanup
// function (if non-nil) is called after schema files are read.
var ResolveSchemaFunc func(ctx context.Context, source string) (fs.FS, func(), error)

func loadSchemaFromFS(fsys fs.FS) ([]FluxVar, map[string]any, error) {
	schema, err := LoadFluxSchema(fsys, "flux.schema.yaml")
	if err != nil {
		return nil, nil, fmt.Errorf("load schema: %w", err)
	}
	defaults, err := LoadFluxFile(fsys, "flux.yaml")
	if err != nil {
		return nil, nil, fmt.Errorf("load defaults: %w", err)
	}
	if defaults == nil {
		defaults = map[string]any{}
	}
	return schema, defaults, nil
}
