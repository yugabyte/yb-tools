# yugatool

`yugatool` is a Yugabyte DB supportability utility.

## Building

### Requirements

* make
* direnv (or source .envrc)
* golang v1.16+
* protoc v3.14.0
* golangci-lint (for `make`)

### Install Requirements
#### -- *_Tested on Ubuntu 18.04_* --

#### `apt` packages ####

```bash
apt install direnv make
snap install protobuf --classic
```

### Install golang

#### Option 1: snap
```
snap install golang
```

#### Option 2: manual

https://go.dev/doc/install


### Install golangci-lint

https://golangci-lint.run/usage/install/


#### Build the binary

```bash
direnv allow
make
```

## Roadmap

TBD
