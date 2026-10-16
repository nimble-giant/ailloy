package commands

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"dario.cat/mergo"
	"github.com/nimble-giant/ailloy/internal/server"
	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// serveRenderer is the server.Renderer behind `ailloy serve`. It runs the
// same ephemeral pipeline as `forge` (resolve, layer flux, render in memory)
// with the server-side rules a shared host needs:
//
//   - only remote foundry references are accepted, so a request can never
//     read the server's own filesystem;
//   - ingots resolve from the requested mold's tree only, never from the
//     server's cwd, .ailloy/ or ~/.ailloy/;
//   - persisted flux files and ailloy.lock are not consulted;
//   - nothing is written to disk except the shared foundry cache.
type serveRenderer struct {
	// allowSources restricts which foundries may be cast (host/owner[/repo]
	// prefixes). Empty means any remote reference.
	allowSources []string
	// allowLocal accepts local mold paths. Tests only; never set by the CLI.
	allowLocal bool
	logger     *log.Logger
}

func (r *serveRenderer) Render(_ context.Context, req server.CastRequest) (*server.CastResult, error) {
	reader, res, remote, err := r.open(req.Mold)
	if err != nil {
		return nil, err
	}

	if err := checkAilloyRequirement(reader); err != nil {
		return nil, server.Errorf(http.StatusUnprocessableEntity, "%v", err)
	}
	manifest, err := reader.LoadManifest()
	if err != nil {
		return nil, server.Errorf(http.StatusUnprocessableEntity, "loading mold manifest: %v", err)
	}
	if manifest != nil {
		res.Mold = manifest.Name
	}

	// Ore deps resolve into the foundry cache only. Local-path deps are
	// refused for remote molds, exactly as in forge.
	oreResolver, err := ResolveDepsEphemeral(manifest, !remote)
	if err != nil {
		return nil, server.Errorf(http.StatusBadGateway, "resolving mold dependencies: %v", err)
	}

	flux, err := layerServeFlux(reader, manifest, oreResolver, req.Values, req.Set)
	if err != nil {
		return nil, server.Errorf(http.StatusBadRequest, "%v", err)
	}

	schema, _ := reader.LoadFluxSchema()
	if schema == nil && manifest != nil && len(manifest.Flux) > 0 {
		schema = manifest.Flux
	}
	mergedSchema, _, _, err := oreResolver.MergeInto(schema, nil)
	if err != nil {
		return nil, server.Errorf(http.StatusUnprocessableEntity, "merging ore schema overlays: %v", err)
	}
	if verr := mold.ValidateFlux(mergedSchema, flux); verr != nil {
		r.logger.Printf("cast %s: warning: %v", req.Mold, verr)
	}

	// The mold's own tree is the only ingot source: no disk search paths.
	tplOpts := []mold.TemplateOption{
		mold.WithIngotResolver(mold.NewIngotResolverWithFS(reader.FS(), nil, flux)),
		mold.WithLogger(r.logger),
	}

	var resolveOpts []mold.ResolveOption
	if ignore := mold.LoadIgnorePatterns(reader.FS(), manifest); len(ignore) > 0 {
		resolveOpts = append(resolveOpts, mold.WithIgnorePatterns(ignore))
	}
	resolved, err := mold.ResolveFilesWithOreSources(flux["output"], reader.FS(), oreResolver.OreSources(), resolveOpts...)
	if err != nil {
		return nil, server.Errorf(http.StatusUnprocessableEntity, "resolving output files: %v", err)
	}

	for _, rf := range resolved {
		if !req.WithWorkflows && strings.HasPrefix(rf.DestPath, ".github/") {
			continue
		}
		content, err := fs.ReadFile(chooseFS(rf, reader.FS()), rf.SrcPath)
		if err != nil {
			return nil, server.Errorf(http.StatusUnprocessableEntity, "reading %s: %v", rf.SrcPath, err)
		}
		if rf.Process {
			fluxForFile := flux
			if len(rf.Set) > 0 {
				fluxForFile = mold.MergeSet(flux, rf.Set)
			}
			rendered, err := renderFile(rf.SrcPath, content, fluxForFile, tplOpts...)
			if err != nil {
				return nil, server.Errorf(http.StatusUnprocessableEntity, "%v", err)
			}
			// Skip files that render to empty content, as cast and forge do (#130).
			if strings.TrimSpace(rendered) == "" {
				continue
			}
			content = []byte(rendered)
		}
		res.Files = append(res.Files, server.File{Path: rf.DestPath, Content: content, Strategy: rf.Strategy})
	}
	return res, nil
}

// open resolves the requested reference to a mold reader. remote reports
// whether the mold came from a foundry.
func (r *serveRenderer) open(ref string) (*blanks.MoldReader, *server.CastResult, bool, error) {
	res := &server.CastResult{}
	if !foundry.IsRemoteReference(ref) {
		if !r.allowLocal {
			return nil, nil, false, server.Errorf(http.StatusBadRequest, "mold must be a remote foundry reference (e.g. github.com/owner/repo@v1), got %q", ref)
		}
		reader, err := blanks.NewMoldReaderFromPath(ref)
		if err != nil {
			return nil, nil, false, server.Errorf(http.StatusBadRequest, "%v", err)
		}
		return reader, res, false, nil
	}

	ok, err := sourceAllowed(ref, r.allowSources)
	if err != nil {
		return nil, nil, true, server.Errorf(http.StatusBadRequest, "%v", err)
	}
	if !ok {
		return nil, nil, true, server.Errorf(http.StatusForbidden, "mold source %q is not allowed on this server", ref)
	}

	fsys, meta, err := foundry.ResolveWithMetadata(ref, foundry.WithLogger(r.logger))
	if err != nil {
		return nil, nil, true, server.Errorf(http.StatusBadGateway, "resolving %s: %v", ref, err)
	}
	res.Version = meta.Resolved.Tag
	res.Commit = meta.Resolved.Commit
	return blanks.NewMoldReaderFromFS(fsys, meta.Root), res, true, nil
}

// layerServeFlux layers flux like forge does — ore defaults < flux.yaml <
// mold.yaml schema defaults < request values < request set — minus the
// server-local persisted flux files.
func layerServeFlux(reader *blanks.MoldReader, manifest *mold.Mold, ores *EphemeralOreResolver, values map[string]any, set []string) (map[string]any, error) {
	flux := map[string]any{}
	if err := mergo.Merge(&flux, ores.Defaults()); err != nil {
		return nil, fmt.Errorf("layering ore defaults: %w", err)
	}
	defaults, err := reader.LoadFluxDefaults()
	if err != nil {
		defaults = map[string]any{}
	}
	if manifest != nil && len(manifest.Flux) > 0 {
		defaults = mold.ApplyFluxDefaults(manifest.Flux, defaults)
	}
	if err := mergo.Merge(&flux, defaults, mergo.WithOverride); err != nil {
		return nil, fmt.Errorf("merging mold defaults: %w", err)
	}
	mold.ApplyManifestOutputDefault(flux, manifest)

	// Request values behave like one -f file: top-level keys replace.
	for k, v := range values {
		flux[k] = v
	}
	if err := mold.ApplySetOverrides(flux, set); err != nil {
		return nil, err
	}
	return flux, nil
}

// sourceAllowed reports whether a remote reference matches one of the
// allowed host/owner[/repo[/subpath]] prefixes. Matching is case-insensitive
// and on whole path segments, so "github.com/acme" does not admit
// "github.com/acme-evil". An empty pattern list allows everything.
func sourceAllowed(ref string, patterns []string) (bool, error) {
	if len(patterns) == 0 {
		return true, nil
	}
	parsed, err := foundry.ParseReference(ref)
	if err != nil {
		return false, err
	}
	key := strings.ToLower(parsed.Host + "/" + parsed.Owner + "/" + parsed.Repo)
	if parsed.Subpath != "" {
		key += "/" + strings.ToLower(strings.Trim(parsed.Subpath, "/"))
	}
	for _, p := range patterns {
		p = strings.ToLower(strings.TrimSpace(p))
		p = strings.TrimPrefix(strings.TrimPrefix(p, "https://"), "http://")
		p = strings.TrimSuffix(p, "/")
		if p == "" {
			continue
		}
		if key == p || strings.HasPrefix(key, p+"/") {
			return true, nil
		}
	}
	return false, nil
}
