# Contributors Guide

## How to build Boots

(Linux only)

1. Install dependencies

    ```bash
    # install nix
    curl -L https://nixos.org/nix/install | sh
    . /home/ubuntu/.nix-profile/etc/profile.d/nix.sh
    ```

2. Clone the repo

    ```bash
    git clone https://github.com/tinkerbell/boots.git
    ```

3. Drop into nix-shell

    ```bash
    cd boots
    nix-shell
    ```

4. Build Boots

    ```bash
    # this will build the ipxe binaries from github.com/ipxe/ipxe and embed them into a Go file and then build Boots
    make boots
    # this will create the boots binary at cmd/boots/boots
    ```

## How to run Boots

(Linux only)

1. Dependencies  
    a. Running Tink server - follow the guide [here](https://docs.tinkerbell.org/setup/local-vagrant/)  

2. Define Boots environment variables

    ```bash
    # MIRROR_HOST is for downloading kernel, initrd
    MIRROR_HOST=192.168.2.3
    # PUBLIC_FQDN is for phone home endpoint
    PUBLIC_FQDN=192.168.2.4
    # DOCKER_REGISTRY, REGISTRY_USERNAME, REGISTRY_PASSWORD, TINKERBELL_GRPC_AUTHORITY, TINKERBELL_CERT_URL are needed for auto.ipxe file generation
    # TINKERBELL_GRPC_AUTHORITY, TINKERBELL_CERT_URL are needed for getting hardware data
    DOCKER_REGISTRY=192.168.2.1:5000
    REGISTRY_USERNAME=admin
    REGISTRY_PASSWORD=secret
    TINKERBELL_GRPC_AUTHORITY=tinkerbell.tinkerbell:42113
    TINKERBELL_CERT_URL=http://tinkerbell.tinkerbell:42114/cert
    # FACILITY_CODE is needed for ?
    FACILITY_CODE=onprem
    # DATA_MODEL_VERSION is need to set "tinkerbell" mode instead of "cacher" mode
    DATA_MODEL_VERSION=1
    # API_AUTH_TOKEN, API_CONSUMER_TOKEN are needed to by pass panicking in cmd/boots/main.go main func
    API_AUTH_TOKEN=none
    API_CONSUMER_TOKEN=none
    ```

3. Run Boots

    ```bash
    # Run the compiled boots
    sudo BOOTS_PUBLIC_FQDN=192.168.2.225 MIRROR_HOST=192.168.2.225:9090 PUBLIC_FQDN=192.168.2.225 DOCKER_REGISTRY=docker.io REGISTRY_USERNAME=none REGISTRY_PASSWORD=none TINKERBELL_GRPC_AUTHORITY=localhost:42113 TINKERBELL_CERT_URL=http://localhost:42114/cert DATA_MODEL_VERSION=1 FACILITY_CODE=onprem API_AUTH_TOKEN=empty API_CONSUMER_TOKEN=empty ./cmd/boots/boots -http-addr 192.168.2.225:80 -tftp-addr 192.168.2.225:69 -dhcp-addr 192.168.2.225:67
    ```

4. Faster iterating via `go run`

    ```bash
    # after the ipxe binaries have been compiled you can use `go run` to iterate a little more quickly than building the binary every time
    sudo BOOTS_PUBLIC_FQDN=192.168.2.225 MIRROR_HOST=192.168.2.225:9090 PUBLIC_FQDN=192.168.2.225 DOCKER_REGISTRY=docker.io REGISTRY_USERNAME=none REGISTRY_PASSWORD=none TINKERBELL_GRPC_AUTHORITY=localhost:42113 TINKERBELL_CERT_URL=http://localhost:42114/cert DATA_MODEL_VERSION=1 FACILITY_CODE=onprem API_AUTH_TOKEN=empty API_CONSUMER_TOKEN=empty go run ./cmd/boots -http-addr 192.168.2.225:80 -tftp-addr 192.168.2.225:69 -dhcp-addr 192.168.2.225:67
    ```

5. Testing

    A. Unit testing

    ```bash
    go test ./... -gcflags=-l
    ```

    B. Functional testing

    1. Create a hardware record in Tink server - follow the guide [here](https://docs.tinkerbell.org/hardware-data/)
    2. boot the machine

## Boots Flow

![boots-flow](docs/boots-flow.png)

Copy and paste the code below into [https://www.websequencediagrams.com](https://www.websequencediagrams.com) to modify

```flow
title Boots - HTTP Flow
# DHCP
Machine->Boots: 1. DHCP Discover
Boots->Tink: 2. Get Hardware data from MAC
Tink->Boots: 3. Send Hardware data
Boots->Machine: 4. DHCP Offer
Machine->Boots: 5. DHCP Request
Boots->Tink: 6. Get Hardware data from MAC
Tink->Boots: 7. Send Hardware data
Boots->Machine: 8. DHCP Ack

# TFTP
Machine->Boots: 9. TFTP Get ipxe binary
Boots->Tink: 10. Get Hardware data from IP
Tink->Boots: 11. Send Hardware data
Boots->Machine: 12. Send ipxe binary

# HTTP
Machine->Boots: 13. HTTP Get ipxe file
Boots->Tink: 14. Get Hardware data from IP
Tink->Boots: 15. Send Hardware data
Boots->Machine: 16. Send ipxe file
```

## Boots Installers

A Boots Installer is a custom iPXE script.
The code for each Installer lives in `installers/`
The idea of iPXE Installers that live in-tree here is an idea that doesn't follow the existing template/workflow paradigm.
Installers should eventually be deprecated.
The deprecation process is forthcoming.

### How an Installers is requested

During a PXE boot request an iPXE script is provided to a PXE-ing machine through a dynamically generated endpoint (http://boots.addr/auto.ipxe).
The contents of the auto.ipxe script is determined through the following steps:

1. A hardware record is retrieved based off the PXE-ing machines mac addr.
2. One of the following is used to determine the content of the iPXE script ([code ref](https://github.com/tinkerbell/boots/blob/b2f4d15f9b55806f4636003948ed95975e1d475e/job/ipxe.go#L71))
    1. If the `metadata.instance.operating_system.slug` matches a registered Installer, the iPXE script from that Installer is returned
    2. If the `metadata.instance.operating_system.distro` matches a registered Installer, the iPXE script from that Installer
    3. If neither of the first 2 are matches, then the default (OSIE) iPXE script is used

### Registering an Installer

To register an Installer, at a minimum, the following are required

1. Add a [blank import](https://github.com/golang/go/wiki/CodeReviewComments#import-blank) should be added to `cmd/boots/main.go`
2. Your installer needs an `func init()` that calls `job.RegisterSlug("InstallerName", funcThatReturnsAnIPXEScript)`

### Testing

Unit tests should be created to validate that your registered func returns the iPXE script you're expecting.
Functional tests would be great but depending on what is in your iPXE script might be difficult because of external dependencies.
At a minimum try to create documentation that details these dependencies so that other can make them available for testing changes.

