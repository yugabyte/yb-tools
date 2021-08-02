# yugatool

`yugatool` is a Yugabyte DB supportability utility.

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

#### Build the binary

```bash
direnv allow
make
```

## Roadmap

TBD