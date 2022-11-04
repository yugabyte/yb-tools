# yugatool

`yugatool` is a YugabyteDB supportability utility.

## Building

#### Requirements

* make
* direnv (or source .envrc)
* golang v1.16+
* protoc v3.14.0

#### Ubuntu 18.04

```bash
apt install direnv make
snap install protobuf
snap install golang
```

#### macOS

1. Manually install golang .pkg via download from https://go.dev/doc/install
1. Set $PATH to contain $GOPATH/bin which defaults to $HOME/go/bin (for the linter)

```bash
brew install direnv
brew install protobuf
find . -name Makefile -exec perl -p -i -e 's/GOOS=linux/GOOS=darwin/' '{}' \;
```

#### Build the binary

```bash
direnv allow
make
```

## Roadmap

TBD
