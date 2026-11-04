# Design: the foundry API (`ailloy serve`)

Status: first cut, shipped behind a new `ailloy serve` command. User docs: [`docs/serve.md`](../serve.md).

## Problem

Getting a mold into a project today means installing the ailloy CLI. Some teams will not install another binary on every laptop, CI runner or devcontainer just to pull a set of rendered markdown files. The engine that does the work (resolve a mold from a foundry, layer flux, render blanks and ingots) does not need to run where the files land.

## Goal

An optional, self-hostable HTTP API that wraps the cast pipeline:

- A client sends a mold reference plus flux overrides and gets the rendered cast back, with nothing installed but `curl`.
- The host chooses public (open, usually limited to some foundries) or private (bearer tokens; the server's own git credentials reach private foundries).
- Output is an archive (tar.gz or zip) or a sequence of individual files.
- Agent targets (PR #275) become a first-class request field once that work lands.

Non-goals for this cut: a hosted multi-tenant service, user accounts, a web UI, server-side merge into a client's existing files, caching rendered output.

## Shape

```
client ── HTTP ──▶ internal/server  (auth, parsing, formats)
                        │ Renderer interface
                        ▼
              internal/commands.serveRenderer
                        │ same building blocks as forge
                        ▼
     foundry.ResolveWithMetadata → ResolveDepsEphemeral → flux layering
       → ResolveFilesWithOreSources → ProcessTemplate (+ ingots)
```

`internal/server` owns only HTTP: request decoding, bearer auth, concurrency limit, format negotiation and encoding, and a final path-safety check on every output path. It knows nothing about molds and is fully unit-tested with a fake renderer.

`serveRenderer` is the forge pipeline without forge's CLI globals, plus the rules a shared host needs (below). It renders in memory and writes nothing except the shared foundry cache.

### Why render, not smelt

`smelt` packages a mold's **source** (templates, `flux.yaml`, ingots) for distribution. A client without the CLI cannot do anything with that, since rendering is the part it lacks. So the API returns the **rendered cast**, the same files `ailloy cast` would write, packed as tar.gz or zip with the same archive conventions smelt uses. Serving smelt's source tarball over HTTP (`GET /v1/smelt`) is a cheap follow-up for hosts that want to act as a mold mirror, but it is not what "no CLI" users need.

## API

- `POST /v1/cast` with JSON `{mold, set, values, targets, with_workflows, format}`.
- `GET /v1/cast?mold=…&set=…&target=…&format=…` for links and one-line `curl`.
- `GET /healthz`, unauthenticated.

Request fields map one-to-one onto `ailloy cast` flags, so a mold's docs work for both. Flux precedence matches cast: ore defaults < `flux.yaml` < `mold.yaml` defaults < `values` (like one `-f` file) < `set` < `targets`.

## Returning a sequence of files

Yes, this is possible, and the first cut ships two ways to do it alongside the archives:

| Format | Why |
| --- | --- |
| `ndjson` | One JSON object per line: `mold` header, one `file` event per file, `done` trailer. Trivial to consume from any language, line by line. The trailer tells a client the stream was not cut short. Binary content is base64. |
| `multipart` | `multipart/mixed`, one part per file. Standard MIME, readable by stdlib parsers in Go, Python and Node. `X-Ailloy-Path` carries the full destination (parsers reduce `filename` to a base name). |
| `json` | A single document with all files, for clients that want one parse. |

Today the server renders the whole cast before writing the first byte. That keeps failures honest: a template error is a `422` with a JSON body, not a half-written archive with a `200`. Molds are small (tens of files), so the latency cost is negligible. If molds grow large, NDJSON can move to true incremental rendering by emitting each file as it renders and an `{"type":"error"}` event in place of `done` on failure; the wire format already allows for it.

## Merge and append strategies

`merge` (JSON/YAML deep-merge) and `append` (sentinel block in markdown) operate on the client's existing file, which the server never sees. The server returns the rendered fragment and reports the strategy in every format (NDJSON/JSON field, multipart header, tar PAX record `AILLOY.strategy`, zip entry comment). Plain `tar xz` therefore replaces those files. A future thin client (a shell script or a tiny npm/pip package) can apply them with the same semantics as `pkg/merge`.

## Auth and exposure

- Bearer tokens from `--token-file` and `AILLOY_SERVE_TOKENS`, compared as SHA-256 digests in constant time.
- The server refuses to start without tokens unless `--public` is passed; both together is an error.
- Private foundries use the server's own git credentials. Token holders can read any mold those credentials reach, so `--allow-source` should name the intended orgs or repos.
- `--allow-source` gates the requested mold, not its declared dependencies: an allowed mold may pull ores and ingots from other foundries the server can reach. Molds under an allowed prefix are trusted to that extent.
- TLS is left to the reverse proxy or platform in front of the server.

Tokens are flat for now. Per-token source scopes (`token → [github.com/acme/*]`) and OIDC/JWT verification are natural next steps if hosts need them.

## Isolation rules

A cast request must never read the server's own files:

1. `mold` must be a remote foundry reference; local paths are a `400`.
2. Mold dependencies declared with local paths are refused (the same rule forge applies to remote molds).
3. Ingots resolve only from the requested mold's filesystem and its declared ingot dependencies. Dependencies are resolved into the foundry cache and mounted in memory at `ingots/<name>/`, where cast would have installed them; the mold's own ingots win on a name clash, as in cast. Forge and cast also search cwd, `.ailloy/` and `~/.ailloy/`; the server does not.
4. Persisted flux (`~/.ailloy/flux/`) and `ailloy.lock` are not consulted, and the process runs in a fresh temp directory so cwd-relative lookups find nothing.
5. Every output path is checked again at the HTTP layer; absolute paths or `..` escapes are a `422`.

Resource limits: 1 MiB request bodies, `--max-concurrent` in-flight casts (`503` + `Retry-After` beyond that), and read/write timeouts on the HTTP server.

## Agent targets (#275)

`targets: ["claude", "codex"]` (or `target=claude,codex` on GET) is validated and appended as the last `set` override, `agent.targets=[claude,codex]`. That is exactly how #275 selects harnesses from the CLI, so once it merges the API fans out to every requested harness with no API change. Until then the value is an ordinary flux variable that molds can already read.

When #275 lands, its `mold.OutputFor(flux)` expansion must also be applied at the serve render site (`serveRenderer.Render`, before `ResolveFilesWithOreSources`), like every other render site. Whichever PR merges second should add that line. The cleaner follow-up is to fold forge, `renderMoldFiles` (plugin) and serve into one shared in-memory render function, which this PR avoids doing so it does not collide with #275's edits to those files.

## Follow-ups

- Apply `mold.OutputFor` at the serve render site after #275, then unify the three render paths.
- `GET /v1/molds` backed by foundry indexes, so clients can discover what the server can cast.
- `GET /v1/smelt` for source tarballs (mirror use case).
- A thin client that extracts a cast and applies merge/append strategies.
- Per-token source scopes; optional OIDC.
- A container image and a deploy example.
- Rendered-output caching keyed by commit SHA and request flux.
