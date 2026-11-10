# Build and Release Maintenance

This document covers local builds and the existing GitHub release tooling for the Go `gfa` replacement.

## Requirements

Use Git and Go 1.27.0 or later. The root `go.mod` pins Task `v3.53.1` and golangci-lint `v2.14.0` through tool directives.

Go downloads and builds these tools on first use. A separate Task or golangci-lint installation is not required.

## Local build

Run the checked build:

```sh
go tool task build
```

The default `go tool task` command and the `all` alias run the same build. The task runs lint, tests, and compilation in order. A failed check stops the build.

The executable is `dist/gfa`. Builds use `CGO_ENABLED=0`, `-trimpath`, and stripped linker flags. No task installs the executable.

Build without checks, with an explicit version string:

```sh
go tool task build-go VERSION=v1.2.3
./dist/gfa --version
```

The default version is `dev`. The `build` task also accepts the `VERSION` override.

List the tasks:

```sh
go tool task --list
```

Run the test suite twice:

```sh
go tool task test
```

Run the pinned linter:

```sh
go tool task lint
```

Apply linter fixes:

```sh
go tool golangci-lint run --fix
```

Remove generated executables and release artifacts under `dist`:

```sh
go tool task clean
```

For direct compilation without Task, run:

```sh
CGO_ENABLED=0 go build -trimpath -o dist/gfa ./cmd/gfa
```

## Manual cross-compilation

These direct Go commands build application binaries without running host-side checks:

```sh
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w -X main.buildVersion=v1.2.3" -o dist/gfa_darwin_arm64 ./cmd/gfa
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.buildVersion=v1.2.3" -o dist/gfa_darwin_amd64 ./cmd/gfa
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.buildVersion=v1.2.3" -o dist/gfa_linux_amd64 ./cmd/gfa
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w -X main.buildVersion=v1.2.3" -o dist/gfa_linux_arm64 ./cmd/gfa
```

## Release workflow

The `.github/workflows/release.yml` workflow publishes releases to `kwo/gfa-go` through GitHub Actions. It runs when a tag that starts with lowercase `v` reaches GitHub. Tags without this prefix do not trigger a release.

The workflow runs lint, tests, and compilation before it builds the four platform archives. It embeds the tag in `gfa --version` and publishes the archives with SHA-256 checksums. Publication requires an existing remote tag.

The `origin` remote targets Forgejo. The `github` remote targets GitHub. A tag pushed only to Forgejo does not trigger this workflow.

Push the release commit to GitHub before you push its tag:

```sh
git push github main
git tag v1.2.3
git push github v1.2.3
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

The script reads releases from `kwo/gfa-go` on GitHub.

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
