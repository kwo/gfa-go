# Build and Release Maintenance

This document covers local builds and the planned release workflow for the Go `gfa` replacement.

## Local build

Build the local binary:

```sh
make build
```

The binary is written to:

```text
dist/gfa
```

Run tests:

```sh
make test
```

Run linting, if `golangci-lint` is installed:

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
GOOS=darwin GOARCH=arm64 go build -o dist/gfa_darwin_arm64 ./cmd/gfa
GOOS=darwin GOARCH=amd64 go build -o dist/gfa_darwin_amd64 ./cmd/gfa
GOOS=linux GOARCH=amd64 go build -o dist/gfa_linux_amd64 ./cmd/gfa
GOOS=linux GOARCH=arm64 go build -o dist/gfa_linux_arm64 ./cmd/gfa
```

## Planned release workflow

Tagged releases should eventually publish platform archives and checksums, for example:

- `gfa_<version>_darwin_arm64.tar.gz`
- `gfa_<version>_darwin_amd64.tar.gz`
- `gfa_<version>_linux_amd64.tar.gz`
- `gfa_<version>_linux_arm64.tar.gz`
- `SHA256SUMS`

Each archive should extract a single executable named `gfa`.

## Homebrew tap

The existing TypeScript `gfa` release uses the `kwo/homebrew-tools` tap. The Go replacement should update the same `gfa.rb` formula once the first Go release is published.
