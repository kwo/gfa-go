# Build and Release Maintenance

This document covers local builds and the existing GitHub release tooling for the Go `gfa` replacement.

## Local build

Build the local binary:

```sh
make build
```

Without Make, build directly with Go:

```sh
go build -trimpath -o dist/gfa ./cmd/gfa
```

Build with an explicit version string:

```sh
make build VERSION=v1.2.3
```

The binary is written to:

```text
dist/gfa
```

Run tests:

```sh
make test
```

With golangci-lint v2 installed, run linting:

```sh
make lint
```

Clean build artifacts:

```sh
make clean
```

## Manual cross-compilation

Examples:

```sh
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w -X main.buildVersion=v1.2.3" -o dist/gfa_darwin_arm64 ./cmd/gfa
GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.buildVersion=v1.2.3" -o dist/gfa_darwin_amd64 ./cmd/gfa
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.buildVersion=v1.2.3" -o dist/gfa_linux_amd64 ./cmd/gfa
GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w -X main.buildVersion=v1.2.3" -o dist/gfa_linux_arm64 ./cmd/gfa
```

## Release workflow

CAUTION: This checkout uses Forgejo, but the existing release workflow and formula generator target GitHub. Do not publish until you select the release destination.

The `.github/workflows/release.yml` workflow publishes platform archives and checksums through GitHub Actions. A tag pushed to Forgejo does not configure that GitHub workflow.

Example:

```sh
git tag v1.2.3
git push origin v1.2.3
```

The release workflow publishes:

- `gfa_<version>_darwin_arm64.tar.gz`
- `gfa_<version>_darwin_amd64.tar.gz`
- `gfa_<version>_linux_amd64.tar.gz`
- `gfa_<version>_linux_arm64.tar.gz`
- `SHA256SUMS`

Each archive contains a single executable named `gfa`.

## Generate the Homebrew formula

From this repository, generate a ready-to-commit `gfa.rb` formula on stdout.

Latest stable release:

```sh
scripts/homebrew-formula
```

Specific tag:

```sh
scripts/homebrew-formula --tag v1.2.3
```

The script:

- uses `gh release list` to select the latest published non-draft, non-prerelease release by default
- downloads and parses the release `SHA256SUMS`
- generates a formula with platform-specific URLs and checksums
- fails if `gh` is missing or required checksums are unavailable

## Homebrew tap

The existing TypeScript `gfa` release uses the `kwo/homebrew-tools` tap. The Go replacement should update the same `gfa.rb` formula once the first Go release is published.

Example:

```sh
scripts/homebrew-formula --tag v1.2.3 > ../homebrew-tools/gfa.rb
```
