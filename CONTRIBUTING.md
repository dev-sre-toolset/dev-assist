# Contributing to dev-assist

Thank you for your interest in contributing to **dev-assist** — the SRE Utility Belt!
All contributions are welcome: bug reports, feature requests, documentation improvements, and code changes.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [License](#license)
- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Features](#suggesting-features)
  - [Submitting a Pull Request](#submitting-a-pull-request)
- [Development Setup](#development-setup)
- [Adding a New Tool](#adding-a-new-tool)
- [Coding Guidelines](#coding-guidelines)
- [Commit Message Format](#commit-message-format)

---

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/1/code_of_conduct/).
By participating, you are expected to uphold this standard. Please report unacceptable behaviour to the maintainers via a GitHub issue.

---

## License

By contributing to dev-assist, you agree that your contributions will be licensed under the **Apache License 2.0**.

```
Copyright 2024 dev-assist contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

A full copy of the license is available in the [LICENSE](LICENSE) file.

---

## Getting Started

1. **Fork** the repository on GitHub.
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/<your-username>/dev-assist.git
   cd dev-assist
   ```
3. Add the upstream remote so you can keep your fork in sync:
   ```bash
   git remote add upstream https://github.com/dev-sre-toolset/dev-assist.git
   ```
4. Install dependencies and verify the build:
   ```bash
   go version          # requires Go 1.21+
   make tidy
   make build
   ./bin/dev-assist --help
   ```

---

## How to Contribute

### Reporting Bugs

Before opening a new issue, please search existing issues to avoid duplicates.
When filing a bug, include:

- **dev-assist version** (`dev-assist version`)
- **OS and architecture** (e.g. macOS arm64, Linux amd64)
- **Go version** (`go version`)
- **Steps to reproduce** — minimal, copy-pasteable commands
- **Expected vs actual behaviour**
- Any relevant output or error messages

### Suggesting Features

Open a GitHub issue with the `enhancement` label. Describe:

- The problem you are trying to solve
- Your proposed solution or the desired behaviour
- Any alternatives you considered

Feature requests are discussed openly before implementation begins — this avoids wasted effort and keeps the scope manageable.

### Submitting a Pull Request

1. **Sync your fork** before starting work:
   ```bash
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```
2. **Create a feature branch** with a descriptive name:
   ```bash
   git checkout -b feat/whois-ipv6-support
   # or
   git checkout -b fix/jwt-exp-overflow
   ```
3. **Make your changes** following the [Coding Guidelines](#coding-guidelines).
4. **Build and test** locally before pushing:
   ```bash
   make build
   make build-all        # verify cross-compilation still works
   go vet ./...
   ```
5. **Push** your branch and open a Pull Request against `main`:
   ```bash
   git push origin feat/whois-ipv6-support
   ```
6. Fill in the PR template with a clear description, screenshots or terminal output if relevant, and reference any related issues (`Closes #123`).

PRs are reviewed by maintainers. Please be responsive to feedback — stale PRs without activity for 30 days may be closed.

---

## Development Setup

| Command | Description |
|---------|-------------|
| `make build` | Build binary for the current platform → `bin/dev-assist` |
| `make build-all` | Cross-compile for macOS (amd64/arm64) and Linux (amd64/arm64) |
| `make install` | Install binary to `$GOPATH/bin` |
| `make tidy` | Run `go mod tidy` |
| `make clean` | Remove the `bin/` directory |
| `make docker` | Build a Docker image locally |

**Requirements:** Go 1.21+, GNU Make, Docker (optional).

---

## Adding a New Tool

dev-assist uses a central registry (`internal/tools/registry.go`). Adding a tool requires two steps:

1. **Create a new file** in `internal/tools/` (e.g. `internal/tools/mytool.go`):

   ```go
   package tools

   import "github.com/dev-sre-toolset/dev-assist/internal/tools"

   func init() {
       tools.Register(tools.Tool{
           Name:        "my-tool",
           Category:    "Data",            // SSL & Certificates | Auth & Tokens | Network | Data
           Description: "Short description shown in the TUI menu",
           Inputs: []tools.InputDef{
               {Name: "input", Label: "Input", Kind: "text", Required: true},
           },
           Run: func(args map[string]string) (string, error) {
               // implement tool logic here
               return result, nil
           },
       })
   }
   ```

2. **Blank-import the file** in `cmd/tools.go` (or rely on `init()` auto-registration if already wired).

The tool will automatically appear in the TUI menu, as a CLI subcommand, and in the Web UI with no additional plumbing.

---

## Coding Guidelines

- **Go conventions** — follow [Effective Go](https://go.dev/doc/effective_go) and `gofmt` formatting. Run `gofmt -w .` before committing.
- **No new dependencies** without discussion. The binary stays small and offline-capable; prefer standard library solutions.
- **Error handling** — return errors; do not use `log.Fatal` or `os.Exit` inside library code.
- **No global state** outside the tool registry.
- **Comments** — add a comment only where the logic is non-obvious; avoid restating what the code already says.
- **Backwards compatibility** — CLI flags and tool names are part of the public interface. Do not rename or remove them without a deprecation notice.

---

## Commit Message Format

Use the [Conventional Commits](https://www.conventionalcommits.org/) style:

```
<type>(<scope>): <short summary>

[optional body]

[optional footer: Closes #123]
```

| Type | When to use |
|------|-------------|
| `feat` | New tool or feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `chore` | Build, dependency, or tooling changes |
| `test` | Adding or updating tests |

**Examples:**
```
feat(tools): add SSH key fingerprint tool
fix(jwt): handle HS256 tokens with padding-stripped base64
docs: add nginx reverse-proxy example to README
chore: bump bubbletea to v0.27
```

---

Thank you for helping make dev-assist better!
