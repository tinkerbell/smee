# Contributor Guide

Welcome to Boots!
We are really excited to have you.
Please use the following guide on your contributing journey.
Thanks for contributing!

## Table of Contents

- [Context](#Context)
- [Architecture](#Architecture)
  - [Design Docs](#Design-Docs)
  - [Code Structure](#Code-Structure)
- [Prerequisites](#Prerequisites)
  - [DCO Sign Off](#DCO-Sign-Off)
  - [Code of Conduct](#Code-of-Conduct)
  - [Setting up your development environment](#Setting-up-your-development-environment)
- [Development](#Development)
  - [Building](#Building)
  - [Unit testing](#Unit-testing)
  - [Linting](#Linting)
  - [Functional testing](#Functional-testing)
  - [Running Boots locally](#Running-Boots-locally)
- [Pull Requests](#Pull-Requests)
  - [Branching strategy](#Branching-strategy)
  - [Quality](#Quality)
    - [CI](#CI)
    - [Code coverage](#Code-coverage)
  - [Pre PR Checklist](#Pre-PR-Checklist)

---

## Context

Boots is a DHCP and PXE (TFTP & HTTP) service.
It is part of the [Tinkerbell stack](https://tinkerbell.org) and provides the first interaction for any machines being provisioned through Tinkerbell.

## Architecture

### Design Docs

Details and diagrams for Boots are found [here](docs/DESIGN.md).

### Code Structure

Details on Boots's code structure is found [here](docs/CODE_STRUCTURE.md) (WIP)

## Prerequisites

### DCO Sign Off

Please read and understand the DCO found [here](docs/DCO.md).

### Code of Conduct

Please read and understand the code of conduct found [here](https://github.com/tinkerbell/.github/blob/main/CODE_OF_CONDUCT.md).

### Setting up your development environment

---

### Dependencies

#### Build time dependencies

#### Nix

This repo's build environment can be reproduced using `nix` (except for gcc cross compiler on mac).

##### Install Nix

Follow the [Nix installation](https://nixos.org/download.html) guide to setup Nix on your box.

##### Load Dependencies

Loading build dependencies is as simple as running `nix-shell` or using [lorri](https://github.com/nix-community/lorri).
If you have `direnv` installed the included `.envrc` will make that step automatic.

#### Runtime dependencies

At runtime Boots needs to communicate with a Tink server.
Follow this [guide](https://docs.tinkerbell.org/setup/local-vagrant/) for running Tink server.

## Development

### Building

> At the moment, these instructions are only stable on Linux environments

To build Boots, run:

```bash
# drop into a shell with all build dependencies
nix-shell

# build all ipxe files, embed them, and build the Go binary
# Built binary can be found here ./cmd/boots/boots
make boots

```

To build the amd64 Boots container image, run:

```bash
# make the amd64 container image
# Built image will be named boots:latest
make image

```

To build the IPXE binaries and embed them into Go, run:

```bash
# Note, this will not build the Boots binary
make bindata
```

To build Boots binaries for all distro

### Unit testing

To execute the unit tests, run:

```bash
make test

# to get code coverage numbers, run:
make coverage
```

### Linting

To execute linting, run:

```bash
# runs golangci-lint
make golangci-lint

# runs goimports
make goimports

# runs go vet
make vet
```

## Linting of Non Go files

```bash
# lints non Go files like shell scripts, markdown files, etc
# this script is used in CI run, so be sure it passes before submitting a PR
./.github/workflows/ci-non-go.sh
```

### Functional testing

1. Create a hardware record in Tink server - follow the guide [here](https://docs.tinkerbell.org/hardware-data/)
2. boot the machine

### Running Boots

1. Be sure all documented runtime dependencies are satisfied.
2. Define all environment variables.

   ```bash
   # MIRROR_HOST is for downloading kernel, initrd
   export MIRROR_HOST=192.168.2.3
   # PUBLIC_FQDN is for phone home endpoint
   export PUBLIC_FQDN=192.168.2.4
   # DOCKER_REGISTRY, REGISTRY_USERNAME, REGISTRY_PASSWORD, TINKERBELL_GRPC_AUTHORITY, TINKERBELL_CERT_URL are needed for auto.ipxe file generation
   # TINKERBELL_GRPC_AUTHORITY, TINKERBELL_CERT_URL are needed for getting hardware data
   export DOCKER_REGISTRY=192.168.2.1:5000
   export REGISTRY_USERNAME=admin
   export REGISTRY_PASSWORD=secret
   export TINKERBELL_GRPC_AUTHORITY=tinkerbell.tinkerbell:42113
   export TINKERBELL_CERT_URL=http://tinkerbell.tinkerbell:42114/cert
   # FACILITY_CODE is needed for ?
   export FACILITY_CODE=onprem
   # DATA_MODEL_VERSION is need to set "tinkerbell" mode instead of "cacher" mode
   export DATA_MODEL_VERSION=1
   # API_AUTH_TOKEN, API_CONSUMER_TOKEN are needed to by pass panicking in cmd/boots/main.go main func
   export API_AUTH_TOKEN=none
   export API_CONSUMER_TOKEN=none
   ```

3. Run Boots

   ```bash
   # Run the compiled boots
   sudo ./cmd/boots/boots -http-addr 192.168.2.225:80 -tftp-addr 192.168.2.225:69 -dhcp-addr 192.168.2.225:67
   ```

4. Faster iterating via `go run`

   ```bash
   # after the ipxe binaries have been compiled you can use `go run` to iterate a little more quickly than building the binary every time
   sudo go run ./cmd/boots -http-addr 192.168.2.225:80 -tftp-addr 192.168.2.225:69 -dhcp-addr 192.168.2.225:67
   ```

## Pull Requests

### Branching strategy

Boots uses a fork and pull request model.
See this [doc](https://guides.github.com/activities/forking/) for more details.

### Quality

#### CI

Boots uses GitHub Actions for CI.
The workflow is found in [.github/workflows/ci.yaml](.github/workflows/ci.yaml).
It is run for each commit and PR.

#### Code coverage

Boots does run code coverage with each PR.
Coverage thresholds are not currently enforced.
It is always nice and very welcomed to add tests and keep or increase the code coverage percentage.

### Pre PR Checklist

This checklist is a helper to make sure there's no gotchas that come up when you submit a PR.

- [ ] You've reviewed the [code of conduct](#Code-of-Conduct)
- [ ] All commits are DCO signed off
- [ ] Code is [formatted and linted](#Linting)
- [ ] Code [builds](#Building) successfully
- [ ] All tests are [passing](#Unit-testing)
- [ ] Code coverage [percentage](#Code-coverage). (main line is the base with which to compare)
