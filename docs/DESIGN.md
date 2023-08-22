# Smee Design Details

## Table of Contents

- [Smee Flow](#Smee-Flow)
- [Smee Installers](#Smee-Installers)
- [IPXE](#IPXE)

---

## Smee Flow

High-level traffic flow for Smee.

![smee-flow](smee-flow.png)

<details>
  <summary>Smee Flow Code</summary>

Copy and paste the code below into [https://www.websequencediagrams.com](https://www.websequencediagrams.com) to modify

```flow
title Smee Flow
# DHCP
note over Machine: DHCP start
Machine->Smee: 1. DHCP Discover
Smee->Tink: 2. Get Hardware data from MAC
Tink->Smee: 3. Send Hardware data
Smee->Machine: 4. DHCP Offer
Machine->Smee: 5. DHCP Request
Smee->Tink: 6. Get Hardware data from MAC
Tink->Smee: 7. Send Hardware data
Smee->Machine: 8. DHCP Ack
note over Machine: DHCP end

# TFTP
note over Machine: TFTP start
Machine->Smee: 9. TFTP Get ipxe binary
Smee->Tink: 10. Get Hardware data from IP
Tink->Smee: 11. Send Hardware data
Smee->Machine: 12. Send ipxe binary
note over Machine: TFTP end

# DHCP
note over Machine: DHCP start
Machine->Smee: 13. DHCP Discover
Smee->Tink: 14. Get Hardware data from MAC
Tink->Smee: 15. Send Hardware data
Smee->Machine: 16. DHCP Offer
Machine->Smee: 17. DHCP Request
Smee->Tink: 18. Get Hardware data from MAC
Tink->Smee: 19. Send Hardware data
Smee->Machine: 20. DHCP Ack
note over Machine: DHCP end

# HTTP
note over Machine: HTTP start
Machine->Smee: 21. HTTP Get ipxe script
Smee->Tink: 22. Get Hardware data from IP
Tink->Smee: 23. Send Hardware data
Smee->Machine: 24. Send ipxe script
note over Machine: HTTP start

```

</details>

## Smee Installers

A Smee Installer is a custom iPXE script.
The code for each Installer lives in `installers/`
The idea of iPXE Installers that live in-tree here is an idea that doesn't follow the existing template/workflow paradigm.
Installers should eventually be deprecated.
The deprecation process is forthcoming.

### How an Installers is requested

During a PXE boot request, an iPXE script is provided to a PXE-ing machine through a dynamically generated endpoint (http://smee.addr/auto.ipxe).
The contents of the auto.ipxe script is determined through the following steps:

1. A hardware record is retrieved based on the PXE-ing machines mac address.
2. The following are tried, in order, to determine the content of the iPXE script ([code ref](https://github.com/tinkerbell/smee/blob/b2f4d15f9b55806f4636003948ed95975e1d475e/job/ipxe.go#L71))
   1. If the `metadata.instance.operating_system.slug` matches a registered Installer, the iPXE script from that Installer is returned
   2. If the `metadata.instance.operating_system.distro` matches a registered Installer, the iPXE script from that Installer
   3. If neither of the first 2 is matched, then the default (OSIE) iPXE script is used

### Registering an Installer

To register an Installer, at a minimum, the following is required

1. A [blank import](https://github.com/golang/go/wiki/CodeReviewComments#import-blank) for your Installer should be added to `main.go`
2. Your Installer pkg needs an `func init()` that calls `job.RegisterSlug("InstallerName", funcThatReturnsAnIPXEScript)`

### Testing Installers

Unit tests should be created to validate that your registered func returns the iPXE script you're expecting.
Functional tests would be great but depending on what is in your iPXE script might be difficult because of external dependencies.
At a minimum try to create documentation that details these dependencies so that others can make them available for testing changes.

## IPXE

Smee serves the upstream IPXE binaries built from [https://github.com/ipxe/ipxe](https://github.com/ipxe/ipxe).
The IPXE binaries are built from source and then embedded into the Smee Go binary to be served via TFTP.

### Building the IPXE binary

The IPXE binaries from [https://github.com/ipxe/ipxe](https://github.com/ipxe/ipxe) are built via a Make target.

```make
make bindata
```
