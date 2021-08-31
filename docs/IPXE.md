# IPXE

In Boots we build custom IPXE binaries for different architectures.
We also enable some custom IPXE features.

## Explicitly enabled iPXE features

Below is a list of the enabled IPXE features we use.
See the `.h` files in the [ipxe/ipxe/](../ipxe/ipxe/) directory for the exact details of the features enabled.

- image trust commands like [imgtrust](https://ipxe.org/cmd/imgtrust), [imgverify](https://ipxe.org/cmd/imgverify)
- boot from a web server via [HTTP/HTTPS](https://ipxe.org/cmd)
- [san](https://ipxe.org/cmd) commands support
- IPv6 support
- [ntp](https://ipxe.org/cmd/ntp) command support
- VLAN([vcreate](https://ipxe.org/cmd/vcreate),[vdestroy](https://ipxe.org/cmd/vdestroy)) command support
- [params](https://ipxe.org/cmd/params) command support
- [param](https://ipxe.org/cmd/param) command support
- Image crypto digest commands
- [reboot](https://ipxe.org/cmd/reboot) command support
- [nslookup](https://ipxe.org/cmd/nslookup) command support
- console syslog support
- ISA_PROBE_ONLY (bios only)
- Non-volatile option storage commands
- [ping](https://ipxe.org/cmd/ping) command support
