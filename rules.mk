# Only use the recipes defined in these makefiles
MAKEFLAGS += --no-builtin-rules
.SUFFIXES:
# Delete target files if there's an error
# This avoids a failure to then skip building on next run if the output is created by shell redirection for example
# Not really necessary for now, but just good to have already if it becomes necessary later.
.DELETE_ON_ERROR:
# Treat the whole recipe as a one shell script/invocation instead of one-per-line
.ONESHELL:
# Use bash instead of plain sh
SHELL := bash
.SHELLFLAGS := -o pipefail -euc

.PHONY: all boots crosscompile dc image gen run test

CGO_ENABLED := 0
export CGO_ENABLED

GitRev := $(shell git rev-parse --short HEAD)
crossbinaries := cmd/boots/boots-linux-386 cmd/boots/boots-linux-amd64 cmd/boots/boots-linux-arm64 cmd/boots/boots-linux-armv6 cmd/boots/boots-linux-armv7
cmd/boots/boots-linux-386:   FLAGS=GOARCH=386
cmd/boots/boots-linux-amd64: FLAGS=GOARCH=amd64
cmd/boots/boots-linux-arm64: FLAGS=GOARCH=arm64
cmd/boots/boots-linux-armv6: FLAGS=GOARCH=arm GOARM=6
cmd/boots/boots-linux-armv7: FLAGS=GOARCH=arm GOARM=7
cmd/boots/boots-linux-386 cmd/boots/boots-linux-amd64 cmd/boots/boots-linux-arm64 cmd/boots/boots-linux-armv6 cmd/boots/boots-linux-armv7: boots
	${FLAGS} GOOS=linux go build -v -ldflags="-X main.GitRev=${GitRev}" -o $@ ./cmd/boots/

generated_files := packet/mock_cacher/cacher_mock.go packet/mock_hardware/hardware_mock.go packet/mock_workflow/workflow_mock.go syslog/facility_string.go syslog/severity_string.go
.PHONY: $(generated_files)
$(generated_files):
	go generate -run="$(@F)" ./...
	goimports -w $@

# this is quick and its really only for rebuilding when dev'ing, I wish go would
# output deps in make syntax like gcc does... oh well this is good enough
cmd/boots/boots: $(shell git ls-files | grep -v -e vendor -e '_test.go' | grep '.go$$' ) ipxe/bindata.go syslog/facility_string.go syslog/severity_string.go
	go build -v -ldflags="-X main.GitRev=${GitRev}" -o $@ ./cmd/boots/


ifeq ($(origin GOBIN), undefined)
GOBIN := ${PWD}/bin
export GOBIN
endif
ipxe/bindata.go: ipxe/bin/ipxe.efi ipxe/bin/snp-hua.efi ipxe/bin/snp-nolacp.efi ipxe/bin/undionly.kpxe
	go-bindata -pkg ipxe -prefix ipxe -o $@ $^
	gofmt -w $@

include ipxev.mk
ipxeconfigs := $(wildcard ipxe/ipxe/*.h)

ipxe/bin/ipxe.efi: ipxe/ipxe/build/bin-x86_64-efi/ipxe.efi
ipxe/bin/snp-nolacp.efi: ipxe/ipxe/build/bin-arm64-efi/snp.efi
ipxe/bin/undionly.kpxe: ipxe/ipxe/build/bin/undionly.kpxe
ipxe/bin/ipxe.efi ipxe/bin/snp-nolacp.efi ipxe/bin/undionly.kpxe:
	cp $^ $@

ipxe/ipxe/build/${ipxev}.tar.gz: ipxev.mk ## Download iPXE source tarball
	mkdir -p $(@D)
	curl -fL https://github.com/ipxe/ipxe/archive/${ipxev}.tar.gz > $@
	echo "${ipxeh}  $@" | sha512sum -c

# given  t=$(patsubst ipxe/ipxe/build/%,%,$@)
# and   $@=ipxe/ipxe/build/*/*
# t       =                */*
ipxe/ipxe/build/bin-arm64-efi/snp.efi ipxe/ipxe/build/bin-x86_64-efi/ipxe.efi ipxe/ipxe/build/bin/undionly.kpxe: ipxe/ipxe/build/${ipxev}.tar.gz ipxe/ipxe/build.sh ${ipxeconfigs}
	+t=$(patsubst ipxe/ipxe/build/%,%,$@)
	rm -rf $(@D)
	mkdir -p $(@D)
	tar -xzf $< -C $(@D)
	cp ${ipxeconfigs} $(@D)
	cd $(@D) && ../../build.sh $$t ${ipxev}
