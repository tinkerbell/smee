# DHCP Modes

Smee's DHCP functionality can operate in one of three modes:

- **DHCP Reservation**: Smee will respond to DHCP requests from clients and provide them with IP and next boot info. This is the default mode. IP info is all reservation based. There must be a corresponding hardware record for the requesting client's MAC address.

- **proxDHCP**: Smee will respond to PXE enabled DHCP requests from clients and provide them with next boot info. In this mode you will need an existing DHCP server that does not serve network boot information. Smee will respond to PXE enabled DHCP requests and provide the client with the next boot info. There must be a corresponding hardware record for the requesting client's MAC address. The `auto.ipxe` script will be served with the MAC address in the URL and the MAC address will be used to lookup the corresponding hardware record. In this mode you will still need layer 2 access to machines or need a DHCP relay agent in your environment that will forward the DHCP requests to Smee.

- **DHCP disabled**: Smee will not respond to DHCP requests from clients. This is useful if you have another DHCP server on your network and you want to use Smee's TFTP and HTTP functionality.

In this mode you most likely want to set `-dhcp-http-ipxe-script-prepend-mac=false`. This will cause Smee to provide the `auto.ipxe` script without a MAC address in the URL and Smee will use the source IP address of the request to lookup the corresponding hardware.
