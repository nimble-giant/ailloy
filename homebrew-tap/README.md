# homebrew-tap

Homebrew tap for [nimble-giant](https://github.com/nimble-giant) projects.

## Usage

```sh
brew install nimble-giant/tap/ailloy
```

Or, equivalently:

```sh
brew tap nimble-giant/tap
brew install ailloy
```

## Formulas

| Formula | Description |
| --- | --- |
| [ailloy](Formula/ailloy.rb) | Package manager for AI instructions |

## Maintenance

Formulas are updated automatically by `.github/workflows/update-formulas.yml`, which runs hourly and on manual dispatch. The workflow polls each upstream project's latest GitHub release, recomputes SHA256s from the release's `checksums.txt`, and commits the updated formula using the workflow's built-in `GITHUB_TOKEN`.

To force an immediate refresh:

```sh
gh workflow run "Update formulas" --repo nimble-giant/homebrew-tap
```

## Bootstrapping (this repo)

This tap was bootstrapped from `homebrew-tap/` in [nimble-giant/ailloy](https://github.com/nimble-giant/ailloy). To re-bootstrap or seed a new tap, copy the contents of that directory into the root of an empty `homebrew-tap` repo and push.
