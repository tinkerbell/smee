# Contributor Guide

Welcome to Smee!
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
  - [Running Smee locally](#Running-Smee-locally)
- [Pull Requests](#Pull-Requests)
  - [Branching strategy](#Branching-strategy)
  - [Quality](#Quality)
    - [CI](#CI)
    - [Code coverage](#Code-coverage)
  - [Pre PR Checklist](#Pre-PR-Checklist)

---

## Context

Smee is a DHCP and PXE (TFTP & HTTP) service.
It is part of the [Tinkerbell stack](https://tinkerbell.org) and provides the first interaction for any machines being provisioned through Tinkerbell.

## Architecture

### Design Docs

Details and diagrams for Smee are found [here](docs/DESIGN.md).

### Code Structure

Details on Smee's code structure is found [here](docs/CODE_STRUCTURE.md) (WIP)

## Prerequisites

### DCO Sign Off

Please read and understand the DCO found [here](docs/DCO.md).

### Code of Conduct

Please read and understand the code of conduct found [here](https://github.com/tinkerbell/.github/blob/main/CODE_OF_CONDUCT.md).

### Setting up your development environment

---

### Dependencies

#### Build time dependencies

#### Runtime dependencies

At runtime Smee needs to communicate with a Tink server.
Follow this [guide](https://tinkerbell.org/docs/setup/getting_started/) for running Tink server.

## Development

### Building

> At the moment, these instructions are only stable on Linux environments

To build Smee, run:

```bash
# build all ipxe files, embed them, and build the Go binary
# Built binary can be found in the top level directory.
make build

```

To build the amd64 Smee container image, run:

```bash
# make the amd64 container image
# Built image will be named smee:latest
make image

```

To build the IPXE binaries and embed them into Go, run:

```bash
# Note, this will not build the Smee binary
make bindata
```

To build Smee binaries for all distro

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
make lint

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

1. Create a hardware record in Tink server - follow the guide [here](https://tinkerbell.org/docs/concepts/hardware/)
2. boot the machine

### Running Smee

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
   export DATA_MODEL_VERSION=1
   # API_AUTH_TOKEN, API_CONSUMER_TOKEN are needed to by pass panicking in main.go main func
   export API_AUTH_TOKEN=none
   export API_CONSUMER_TOKEN=none
   ```

3. Run Smee

   ```bash
   # Run the compiled smee
   sudo ./smee -http-addr 192.168.2.225:80 -tftp-addr 192.168.2.225:69 -dhcp-addr 192.168.2.225:67
   ```

4. Faster iterating via `go run`

   ```bash
   # after the ipxe binaries have been compiled you can use `go run` to iterate a little more quickly than building the binary every time
   sudo go run ./smee -http-addr 192.168.2.225:80 -tftp-addr 192.168.2.225:69 -dhcp-addr 192.168.2.225:67
   ```

## Pull Requests

### Branching strategy

Smee uses a fork and pull request model.
See this [doc](https://guides.github.com/activities/forking/) for more details.

### Quality

#### CI

Smee uses GitHub Actions for CI.
The workflow is found in [.github/workflows/ci.yaml](.github/workflows/ci.yaml).
It is run for each commit and PR.

#### Code coverage

Smee does run code coverage with each PR.
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
