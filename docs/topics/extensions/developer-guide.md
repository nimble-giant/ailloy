# Building Ailloy Extensions

Extensions are standalone binaries that ailloy downloads on demand and
execs. They're not Go plugins — there's no in-process loading, no
shared memory protocol, no FFI. Just `os/exec` with stdin/stdout and a
small set of well-known environment variables.

This guide walks through what an extension is, how the contract
works, and how to publish one.

## Anatomy

```
my-ailloy-thing/
  cmd/ailloy-mything/main.go    # binary entrypoint
  go.mod                         # imports the SDK
  .github/workflows/release.yml  # release-please + cross-platform build
  README.md
  LICENSE
```

The binary name convention is `ailloy-<name>`. Releases publish an
asset matrix `ailloy-<name>_<os>_<arch>` (with `.exe` on Windows) plus
a `checksums.txt` listing SHA-256 sums.

## Minimal extension

```go
package main

import (
    "fmt"

    "github.com/nimble-giant/ailloy-extensions-sdk/pkg/extension"
)

var (
    version           = "v0.0.0-dev"
    ailloyDocsVersion = ""
)

func main() {
    info := extension.Info{
        Name:              "mything",
        Version:           version,
        AilloyDocsVersion: ailloyDocsVersion,
    }
    extension.HandleVersionFlags(info)

    env := extension.ReadHostEnv()
    if env.AilloyVersion == "" {
        fmt.Println("running standalone")
    } else {
        fmt.Printf("running under ailloy %s\n", env.AilloyVersion)
    }

    // Your extension's actual command logic here. argv is yours from
    // here on; stdin/stdout/stderr are inherited from ailloy.
}
```

`HandleVersionFlags` intercepts `--ailloy-protocol-version` and
`--ailloy-extension-info` and exits before your code sees them. This
is how the host introspects extensions.

## Contract

### Environment

The host sets these before exec'ing your binary:

| Variable | Purpose |
|---|---|
| `AILLOY_VERSION` | Semver of the ailloy host (e.g. `v0.43.0`) |
| `AILLOY_CONFIG_DIR` | Absolute path to `~/.ailloy` (or the user's override) |
| `AILLOY_PROTOCOL_VERSION` | The protocol version the host expects |
| `AILLOY_NO_COLOR` | Set when the user wants uncolored output |
| `AILLOY_DOCS_SOURCE` | Optional dev-mode pointer to a docs/ on disk |

Read them via `extension.ReadHostEnv()`.

### Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Generic error |
| 2 | Protocol mismatch — the host should fall back |
| 3 | Consent required — the user must approve a step before continuing |

### Argv

Everything the user passed after `ailloy <subcommand>` is forwarded to
the extension verbatim. Implement your own flag parsing as you would
for any standalone CLI.

## Versioning

Extensions follow semver on their own cadence. If your extension
mirrors ailloy content, bake the source ailloy version into your
binary at build time:

```
go build -ldflags "-X main.version=v1.2.3 -X main.ailloyDocsVersion=v0.43.0"
```

The host uses `ailloy_docs_version` to pick the right release for the
user's installed ailloy: it prefers the highest semver whose
`ailloy_docs_version` is ≤ the host's, falling back to the latest
available with a one-line warning if no exact match exists.

## Releases

We recommend [release-please](https://github.com/googleapis/release-please)
+ a matrix workflow that:

1. Cuts the release on conventional-commit merges to `main`
2. Cross-compiles the binary for `{linux,darwin,windows} × {amd64,arm64}`
3. Generates `checksums.txt`
4. Uploads everything to the GitHub release

See [`nimble-giant/ailloy-embedded-docs/.github/workflows/release.yml`](https://github.com/nimble-giant/ailloy-embedded-docs/blob/main/.github/workflows/release.yml)
for a complete example.

## Submitting an official extension

Publishing your extension as `github.com/owner/repo` is enough — users
can install from anywhere with:

```bash
ailloy ext install github.com/owner/repo
```

To get a friendly registry name like `docs` (so users can run
`ailloy ext install your-name`), open a PR to
[`ailloy-extensions-sdk`](https://github.com/nimble-giant/ailloy-extensions-sdk)
adding an entry to `pkg/registry/registry.go`. Include:

- A short stable name
- The `github.com/owner/repo` source
- A one-line description shown in `ext list`
- The binary name your release publishes
- Whether your release notes carry a `Built from ailloy docs vX.Y.Z`
  line (used for version-mirroring extensions)

Ailloy's next minor will pick up the new entry.

## Examples

- **`examples/hello`** in the SDK — minimal extension demonstrating the
  protocol, useful as a copy-paste starting point.
- **`nimble-giant/ailloy-embedded-docs`** — the docs extension. A
  realistic example of build-time pre-rendering, multiple workflows
  (release + scheduled poller), and an extension that mirrors upstream
  content.
