# dev-assist

**SRE Utility Belt** — a fast, offline-capable terminal tool for day-to-day SRE and platform-engineering tasks.
Works as an interactive TUI, a scriptable CLI subcommand, and a **web UI** you can self-host or expose publicly.

---

## Quick Start

### Download a pre-built binary

```bash
# macOS (Apple Silicon)
curl -Lo dev-assist https://github.com/dev-sre-toolset/dev-assist/releases/latest/download/dev-assist-darwin-arm64
chmod +x dev-assist && mv dev-assist /usr/local/bin/

# macOS (Intel)
curl -Lo dev-assist https://github.com/dev-sre-toolset/dev-assist/releases/latest/download/dev-assist-darwin-amd64
chmod +x dev-assist && mv dev-assist /usr/local/bin/

# Linux (amd64)
curl -Lo dev-assist https://github.com/dev-sre-toolset/dev-assist/releases/latest/download/dev-assist-linux-amd64
chmod +x dev-assist && mv dev-assist /usr/local/bin/

# Linux (arm64)
curl -Lo dev-assist https://github.com/dev-sre-toolset/dev-assist/releases/latest/download/dev-assist-linux-arm64
chmod +x dev-assist && mv dev-assist /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/dev-sre-toolset/dev-assist.git
cd dev-assist
make build          # current platform  → bin/dev-assist
make build-all      # all platforms     → bin/
make install        # installs to $GOPATH/bin
```

**Requirements:** Go 1.21+

### Launch

```bash
dev-assist           # opens interactive TUI
dev-assist --help    # shows all available subcommands
dev-assist version   # prints the version
dev-assist web       # starts the web UI at http://localhost:8080
```

---

## Web UI

```bash
dev-assist web                            # serve at http://localhost:8080
dev-assist web --port 9000                # custom port
dev-assist web --host 0.0.0.0 --port 80  # expose on all interfaces (public)
```

The web UI is embedded directly in the binary — no Node.js, no separate build step, no external CDN.
Open `http://localhost:8080` in a browser after starting.

**Features:**
- Categorised sidebar with live search / filter
- All 15 tools available with the same inputs as the TUI
- Option toggles rendered as pill buttons
- Result pane with Copy-to-clipboard
- Neon dark theme matching the TUI
- Responsive layout (desktop & tablet)

### Self-hosting with a reverse proxy (nginx example)

```nginx
location / {
    proxy_pass         http://127.0.0.1:8080;
    proxy_set_header   Host $host;
    proxy_set_header   X-Real-IP $remote_addr;
}
```

### Docker (single-file)

```dockerfile
FROM golang:1.21-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /dev-assist .

FROM alpine:3.19
COPY --from=build /dev-assist /usr/local/bin/dev-assist
EXPOSE 8080
CMD ["dev-assist", "web", "--host", "0.0.0.0", "--port", "8080"]
```

```bash
docker build -t dev-assist .
docker run -p 8080:8080 dev-assist
```

---

## What can it do?

### SSL & Certificates
- **Parse a PEM certificate** — subject, issuer, validity, SANs, key type, SHA-1/SHA-256 fingerprints; accepts full PEM or bare base64 DER (headers added automatically)
- **Verify a certificate chain** — validates against a supplied CA bundle or system roots; shows the full verified chain
- **Parse any PEM block** — certificates, CSRs, RSA/ECDSA/PKCS8 private keys, public keys; verifies CSR signature
- **Generate a CSR + private key** — equivalent to `openssl req -new -newkey rsa:2048 -nodes`; supports RSA-2048/4096 and ECDSA P-256/P-384

### Auth & Tokens
- **Decode a JWT** — header, payload (pretty-printed), claims analysis (exp/nbf/iat age, sub, iss, aud), signature algorithm; no secret required
- **Decode a SAML request/response** — accepts a full SSO redirect URL or a bare base64 value; auto-detects SAMLRequest (base64 + DEFLATE) vs SAMLResponse (base64 only); outputs syntax-highlighted XML with namespace prefixes
- **Base64 encode/decode** — all four variants: standard, URL-safe, raw-standard, raw-URL-safe; auto-detect on decode
- **URL encode/decode** — query-encode, path-encode, full URL parse with per-parameter breakdown

### Network
- **DNS lookup** — A, AAAA, MX, TXT, CNAME, NS, PTR for any hostname or IP; defaults to ALL record types
- **WHOIS lookup** — queries `whois.iana.org` by default; auto-detects structured formats and renders each section as a coloured subnet hierarchy tree with the most-specific entry highlighted
- **CIDR calculator** — network address, broadcast, host range, netmask, wildcard, host count, membership check
- **HTTP headers** — performs a real GET and displays response headers categorised by Security / Cache / Content / Other

### Data
- **Date / time diff** — given two dates calculates the difference in days, weeks, and a calendar breakdown (years/months/days); given two datetimes calculates total seconds, minutes, hours, and a `Xd Xh Xm Xs` breakdown; second input defaults to "now"
- **JSON ↔ YAML converter** — auto-detects input format, pretty-prints, and converts in either direction
- **Unix timestamp** — converts Unix epoch (seconds or milliseconds) or a date string to a human-readable time across 8 timezones; also shows relative age

---

## Interactive TUI

```
dev-assist
```

- Type to **filter** tools by name, category, or description
- `↑ ↓` or `j k` — navigate the list
- `Enter` — select a tool
- `q` — quit

### Input screen

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Next / previous field |
| `Enter` | Run (single-line fields) |
| `Ctrl+R` | Run (always — use in multiline text areas) |
| `Ctrl+F` | Toggle between **raw** input and **file path** mode |
| `← →` | Cycle through option toggles |
| `Esc` | Back to menu |

### Result screen

| Key | Action |
|-----|--------|
| `↑ ↓` / `j k` | Scroll output |
| `g` / `G` | Jump to top / bottom |
| `Esc` | Back to input |
| `m` | Back to main menu |
| `q` | Quit |

---

## CLI — Non-interactive / Scriptable Mode

Every tool is also a subcommand. Pass `--help` to any subcommand for its full flag list.

```bash
# SSL & Certificates
dev-assist ssl-decode --cert server.pem
dev-assist ssl-decode --cert "$(cat server.pem)"   # inline PEM or bare base64
dev-assist ssl-verify --cert chain.pem --ca ca-bundle.pem
dev-assist ssl-verify --cert leaf.pem              # uses system roots
dev-assist pem-parse --input bundle.pem
dev-assist csr-gen   --cn api.example.com --san "api.example.com,api-staging.example.com" \
                    --org "Example Corp" --algo ecdsa-p256 --out-dir ./certs

# Auth & Tokens
dev-assist jwt   --input "eyJhbGciOiJSUzI1NiJ9..."
dev-assist saml  --input "https://idp.example.com/SSO?SAMLRequest=jVJdb..."
dev-assist saml  --input jVJdb6MwEPwr...           # bare base64 value
dev-assist base64 --input "hello world" --mode encode
dev-assist base64 --input "aGVsbG8gd29ybGQ=" --mode decode
dev-assist url   --input "search=hello world&page=2"

# Network
dev-assist dns   --host example.com
dev-assist dns   --host example.com --type MX
dev-assist whois --host example.com
dev-assist whois --host 8.8.8.8
dev-assist whois --host example.com --server whois.verisign-grs.com
dev-assist cidr  --cidr 10.128.0.0/12
dev-assist cidr  --cidr 10.128.0.0/12 --check 10.130.5.1
dev-assist http-headers --url https://api.example.com

# Data
dev-assist date-diff --from 2024-01-15 --to 2025-03-19
dev-assist date-diff --from "2025-03-19 09:00:00" --to "2025-03-19 17:45:30"
dev-assist date-diff --from 2024-01-15             # "to" defaults to now
dev-assist json-yaml --input data.json
dev-assist timestamp --input 1710864000
dev-assist timestamp --input "2024-03-19T12:00:00Z"
```

---

## Releasing

### One-command release (uses `gh` CLI + `make`)

```bash
# Ensure you are on the correct branch and everything is committed
git checkout main
git pull

# Build all platform binaries, tag, and publish to GitHub
make release TAG=v0.1
```

This will:
1. Cross-compile binaries for 4 platforms into `bin/` (macOS arm64/amd64, Linux arm64/amd64)
2. Create and push an annotated git tag `v0.1`
3. Create a GitHub release on `github.com/dev-sre-toolset/dev-assist` with all binaries attached

### Manual step-by-step

```bash
# 1. Build all platform binaries
make build-all

# 2. Tag the release
git tag -a v0.1 -m "Release v0.1"
git push origin v0.1

# 3. Create the GitHub release and upload binaries
gh release create v0.1 \
  bin/dev-assist-darwin-amd64 \
  bin/dev-assist-darwin-arm64 \
  bin/dev-assist-linux-amd64 \
  bin/dev-assist-linux-arm64 \
  --title "dev-assist v0.1" \
  --notes "Initial release of dev-assist SRE Utility Belt."
```

### Using a custom release message

```bash
gh release create v0.1 \
  bin/dev-assist-darwin-amd64 \
  bin/dev-assist-darwin-arm64 \
  bin/dev-assist-linux-amd64 \
  bin/dev-assist-linux-arm64 \
  --title "dev-assist v0.1" \
  --notes-file RELEASE_NOTES.md
```

### Listing and deleting releases

```bash
# List all releases
gh release list

# View a specific release
gh release view v0.1

# Delete a release (keeps the tag)
gh release delete v0.1

# Delete the tag as well
git push --delete origin v0.1
git tag --delete v0.1
```

---

## Project Layout

```
dev-assist/
├── main.go                     # entry point — injects version at build time
├── cmd/
│   ├── root.go                 # cobra root; launches TUI when no subcommand
│   ├── tools.go                # auto-generates a subcommand for every registered tool
│   └── web.go                  # `dev-assist web` subcommand
├── internal/
│   ├── tools/
│   │   ├── registry.go         # Tool + InputDef types; global registry
│   │   ├── ssl.go              # ssl-decode, ssl-verify
│   │   ├── pem.go              # pem-parse
│   │   ├── csrgen.go           # csr-gen
│   │   ├── jwt.go              # jwt
│   │   ├── saml.go             # saml (with XML syntax highlighting)
│   │   ├── base64.go           # base64
│   │   ├── url.go              # url
│   │   ├── dns.go              # dns, whois (with structured hierarchy renderer)
│   │   ├── cidr.go             # cidr
│   │   ├── httpheader.go       # http-headers
│   │   ├── datediff.go         # date-diff
│   │   ├── json.go             # json-yaml
│   │   └── timestamp.go        # timestamp
│   ├── ui/
│   │   ├── app.go              # BubbleTea root model + state machine
│   │   ├── menu.go             # tool-selection list with live filter
│   │   ├── input.go            # per-tool input form
│   │   ├── result.go           # scrollable result viewport
│   │   └── styles.go           # neon colour palette + shared helpers
│   └── web/
│       ├── server.go           # HTTP server, API handlers, ANSI stripping
│       └── static/
│           ├── index.html      # SPA shell
│           ├── style.css       # neon dark theme
│           └── app.js          # vanilla JS SPA logic
├── deployment/                 # Kubernetes manifests (deployment, service, ingress, networkpolicy)
├── Makefile                    # build, build-all, release, install, tidy, clean (macOS + Linux)
└── go.mod
```

---

## Tech Stack

| Component | Library |
|-----------|---------|
| TUI framework | [Bubble Tea](https://github.com/charmbracelet/bubbletea) v0.26 |
| TUI components | [Bubbles](https://github.com/charmbracelet/bubbles) v0.18 (textarea, textinput, viewport) |
| Styling | [Lip Gloss](https://github.com/charmbracelet/lipgloss) v0.11 |
| CLI flags | [Cobra](https://github.com/spf13/cobra) v1.8 |
| YAML | [go-yaml](https://gopkg.in/yaml.v3) v3 |
| Language | Go 1.21+ |
