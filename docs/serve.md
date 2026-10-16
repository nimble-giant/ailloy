# Serving Casts over HTTP with `ailloy serve`

`ailloy serve` runs the **foundry API**: an HTTP server that resolves a mold from a foundry, renders it just in time, and returns the cast. Teams can host it once and let everyone else fetch casts with `curl`, a CI step, or an editor integration, with no ailloy CLI installed.

```bash
# On the host
echo "$(openssl rand -hex 32)" > tokens.txt
ailloy serve --addr :8080 --token-file tokens.txt --allow-source github.com/acme

# Anywhere else
curl -fsS -H "Authorization: Bearer $TOKEN" \
  "https://ailloy.acme.dev/v1/cast?mold=github.com/acme/molds@^1&set=project.organization=acme" \
  | tar xz
```

The response is exactly what `ailloy cast` would write into the project, rendered on the server.

## Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `--addr` | `127.0.0.1:8080` | Listen address. Use `:8080` to listen on all interfaces. |
| `--token-file` | none | Accepted bearer tokens, one per line (`#` comments allowed). |
| `--public` | `false` | Serve casts with no authentication. |
| `--allow-source` | any | Only cast molds under this `host/owner[/repo]` prefix. Repeatable. |
| `--max-concurrent` | `8` | Casts rendered at once. Extra requests get `503` with `Retry-After`. |
| `--workdir` | temp dir | Working directory for the server process. |

Tokens can also come from `AILLOY_SERVE_TOKENS` (comma-separated). The server refuses to start without at least one token unless `--public` is passed, so it is never open by accident.

## Public or private

- **Private**: give the server git credentials for your private foundries (an SSH key or a credential helper, exactly as for `ailloy cast`) and hand out bearer tokens. Only token holders can read those molds through the API.
- `--allow-source` gates the requested mold. Its declared dependencies are fetched from wherever the mold says.
- **Public**: pass `--public` and usually `--allow-source` so the server only renders the foundries you intend to share.

## `POST /v1/cast`

```json
{
  "mold": "github.com/acme/molds@^1.0",
  "set": ["project.organization=acme"],
  "values": {"project": {"board": "Engineering"}},
  "targets": ["claude", "codex"],
  "with_workflows": false,
  "format": "zip"
}
```

| Field | Same as | Notes |
| --- | --- | --- |
| `mold` | `cast <ref>` | Required. Remote foundry references only. |
| `set` | `--set` | `key=value` strings, applied in order. |
| `values` | `-f`/`--values` | Inline flux values, layered below `set`. |
| `targets` | `--set agent.targets=[...]` | Agent harnesses to cast for. Applied last. |
| `with_workflows` | `--with-workflows` | Include `.github/` blanks. |
| `format` | | `tar` (default), `zip`, `ndjson`, `multipart` or `json`. |

## `GET /v1/cast`

The same request as query parameters, handy for `curl` and download links: `mold`, `set` (repeatable), `target` (repeatable or comma-separated), `with_workflows=true`, `format`.

## Response formats

The `format` field wins; otherwise the `Accept` header picks one.

| Format | Accept | Body |
| --- | --- | --- |
| `tar` | `application/gzip` | `.tar.gz` of the cast, paths relative to the project root. |
| `zip` | `application/zip` | `.zip` of the cast. |
| `ndjson` | `application/x-ndjson` | One JSON object per line: a `mold` header, one `file` event per file, then a `done` trailer. |
| `multipart` | `multipart/mixed` | One part per file. `X-Ailloy-Path` holds the destination path. |
| `json` | `application/json` | `{"mold": {...}, "files": [...]}` in one document. |

Every response carries `X-Ailloy-Mold`, `X-Ailloy-Mold-Version`, `X-Ailloy-Mold-Commit` and `X-Ailloy-File-Count`.

File events in `ndjson` and `json` look like this. Content that is not valid UTF-8 is sent as base64.

```json
{"type":"file","path":"CLAUDE.md","strategy":"append","encoding":"utf-8","content":"..."}
```

### Merge and append destinations

The server has no copy of your project, so it cannot merge into an existing `settings.json` or append into an existing `CLAUDE.md`. Those files are delivered as the rendered fragment, and their strategy is reported (`strategy` in NDJSON and JSON, `X-Ailloy-Strategy` on multipart parts, an `AILLOY.strategy` PAX record in tar, the entry comment in zip) so a client can apply it. Extracting the archive over a project replaces those files.

## Errors

Errors are JSON: `{"error": "..."}`.

| Status | When |
| --- | --- |
| `400` | Bad body, unknown format, a local path as `mold`, bad `set` or target. |
| `401` | Missing or wrong bearer token. |
| `403` | `mold` is outside `--allow-source`. |
| `422` | The mold failed to render (template error, missing ingot, `requires.ailloy` unmet). |
| `502` | The foundry or a dependency could not be resolved. |
| `503` | `--max-concurrent` casts already running. |

## `GET /healthz`

Unauthenticated liveness check: `{"status":"ok","version":"..."}`.

## What the server will not do

- Read local paths. `mold` must be a foundry reference, so a request can never read the server's own files.
- Resolve ingots from its own `.ailloy/`, `~/.ailloy/` or working directory. Ingots come from the requested mold.
- Use persisted flux (`~/.ailloy/flux/`) or an `ailloy.lock`. Each request carries its own flux.
- Resolve ingots a mold declares as dependencies, yet. Bundled ingots in the mold's own `ingots/` work.
- Write anything except the shared foundry cache (`~/.ailloy/cache/`).
