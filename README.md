# gfa

Go implementation of `gfa`, a command-line tool that discovers Git repositories in the current directory, fetches all remotes, reports repository state, and pulls when safe.

## Development

```sh
go run ./cmd/gfa
```

Show version information:

```sh
go run ./cmd/gfa --version
# or
go run ./cmd/gfa -v
```

## Build

```sh
make build
```

See [BUILD.md](BUILD.md) for build and release maintenance notes.

## Current status

This repository is being ported from the TypeScript/Ink `gfa` implementation. See [PORTING.md](PORTING.md) for the parity plan and remaining release work.
