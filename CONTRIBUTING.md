# Contributing to Ailloy

Thank you for your interest in contributing to Ailloy! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Commit Message Conventions](#commit-message-conventions)
- [Pull Request Process](#pull-request-process)
- [Reporting Issues](#reporting-issues)
- [Community](#community)

## Getting Started

### Prerequisites

- **Go 1.24+**: Ailloy is written in Go. Install from [go.dev](https://go.dev/dl/)
- **Git**: For version control
- **GitHub CLI** (`gh`): Required for testing GitHub integration features
- **golangci-lint**: For code linting (optional but recommended)
- **lefthook**: For git hooks (`brew install lefthook`)

### Development Setup

1. **Fork the repository** on GitHub

2. **Clone your fork**:
   ```bash
   git clone https://github.com/YOUR_USERNAME/ailloy.git
   cd ailloy
   ```

3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/nimble-giant/ailloy.git
   ```

4. **Install git hooks**:
   ```bash
   make hooks
   ```

   This installs lefthook-managed hooks that run graduated checks:
   - **pre-commit**: `go vet` + `gofmt` on staged files
   - **commit-msg**: conform (conventional commits)
   - **pre-push**: `golangci-lint` + `go build` + `go test -race`

5. **Build the project**:
   ```bash
   make build
   ```

6. **Run tests**:
   ```bash
   make test
   ```

7. **Run linter** (requires golangci-lint):
   ```bash
   make lint
   ```

### Project Structure

```
ailloy/
├── cmd/ailloy/          # CLI entry point
├── internal/            # Private Go packages
│   └── commands/        # CLI command implementations (cast, forge, smelt, etc.)
├── pkg/                 # Public Go packages
│   ├── blanks/          # MoldReader abstraction
│   ├── foundry/         # SCM-native mold resolution, caching, version management
│   ├── github/          # GitHub ProjectV2 discovery via gh API GraphQL
│   ├── mold/            # Template engine, flux loading, ingot resolution
│   ├── plugin/          # Plugin generation pipeline
│   ├── safepath/        # Safe path utilities
│   ├── smelt/           # Mold packaging (tarball/binary)
│   └── styles/          # Terminal styling
├── docs/                # Documentation
└── Makefile             # Build targets
```

## How to Contribute

### Types of Contributions

We welcome contributions in many forms:

- **Bug fixes**: Found a bug? Submit a fix!
- **New features**: Have an idea? Propose it first via an issue
- **Documentation**: Improve docs, add examples, fix typos
- **Blanks**: Create or improve AI workflow blanks
- **Tests**: Increase test coverage

### Before You Start

1. **Check existing issues** to avoid duplicate work
2. **Open an issue** for significant changes to discuss the approach
3. **For small fixes** (typos, minor bugs), you can submit a PR directly

## Development Workflow

### Branch Naming

Use descriptive branch names with a prefix:

- `feat/add-blank-validation` - New features
- `fix/config-loading-error` - Bug fixes
- `docs/improve-readme` - Documentation
- `chore/update-dependencies` - Maintenance tasks
- `refactor/simplify-plugin-system` - Code refactoring

### Making Changes

1. **Create a branch** from `main`:
   ```bash
   git checkout main
   git pull upstream main
   git checkout -b feat/your-feature-name
   ```

2. **Make your changes** with clear, focused commits

3. **Test your changes**:
   ```bash
   make test
   make lint
   ```

4. **Build and verify**:
   ```bash
   make build
   ./bin/ailloy --help
   ```

### Keeping Your Fork Updated

```bash
git fetch upstream
git checkout main
git merge upstream/main
```

## Code Standards

### Go Style

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `gofmt` for formatting (automatic with most editors)
- Run `golangci-lint` before submitting

### Code Guidelines

- **Keep functions focused**: Each function should do one thing well
- **Write clear variable names**: Prefer clarity over brevity
- **Handle errors properly**: Don't ignore errors; handle or propagate them
- **Add comments for exported items**: Document public functions, types, and packages
- **Avoid global state**: Pass dependencies explicitly

### Example Code Style

```go
// ProcessTemplate reads a template file and applies variable substitution.
// It returns the processed content or an error if the template is invalid.
func ProcessTemplate(path string, vars map[string]string) (string, error) {
    content, err := os.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("reading template %s: %w", path, err)
    }

    result := string(content)
    for key, value := range vars {
        placeholder := fmt.Sprintf("{{%s}}", key)
        result = strings.ReplaceAll(result, placeholder, value)
    }

    return result, nil
}
```

## Commit Message Conventions

We follow [Conventional Commits](https://www.conventionalcommits.org/) for clear, structured commit history.

### Format

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Formatting (no code change)
- `refactor`: Code restructuring (no feature change)
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `ci`: CI/CD changes
- `perf`: Performance improvements
- `build`: Build system changes
- `revert`: Reverting a previous commit
- `release`: Release automation (used by release-please)

### Scopes

- `cli`: CLI commands and flags
- `config`: Configuration system
- `blanks`: Blank system
- `plugin`: Plugin functionality
- `docs`: Documentation

### Examples

```
feat(cli): add blank validation command

fix(config): handle missing configuration file gracefully

docs(readme): update installation instructions

chore(deps): update cobra to v1.9.1
```

## Pull Request Process

### Before Submitting

1. **Ensure tests pass**: `make test`
2. **Run the linter**: `make lint`
3. **Update documentation** if needed
4. **Add tests** for new functionality

### Submitting a PR

1. **Push your branch** to your fork:
   ```bash
   git push origin feat/your-feature-name
   ```

2. **Open a Pull Request** against `nimble-giant/ailloy:main`

3. **Fill out the PR template** with:
   - Description of changes
   - Related issue number (if applicable)
   - Testing performed

### PR Review Process

- Maintainers will review your PR
- Address feedback by pushing additional commits
- Once approved, a maintainer will merge your PR

### What We Look For

- Code quality and style adherence
- Test coverage for new functionality
- Clear commit messages
- Documentation updates where appropriate

## Reporting Issues

### Bug Reports

When reporting a bug, include:

- **Ailloy version**: `ailloy --version`
- **Go version**: `go version`
- **Operating system**
- **Steps to reproduce**
- **Expected behavior**
- **Actual behavior**
- **Error messages** (if any)

### Feature Requests

For feature requests, describe:

- **The problem** you're trying to solve
- **Your proposed solution**
- **Alternative approaches** you've considered
- **Additional context** or examples

### Where to Report

- [GitHub Issues](https://github.com/nimble-giant/ailloy/issues) for bugs and features
- Security issues: See [SECURITY.md](SECURITY.md) for responsible disclosure

## Community

### Getting Help

- Check existing [documentation](docs/)
- Search [existing issues](https://github.com/nimble-giant/ailloy/issues)
- Open a new issue with your question

### Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md) to help maintain a welcoming community.

---

Thank you for contributing to Ailloy! Your efforts help make AI-assisted development better for everyone.
