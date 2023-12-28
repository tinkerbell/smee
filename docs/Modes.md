# DHCP Modes

Smee's DHCP functionality can operate in one of the following modes:

1. **DHCP Reservation**  
Smee will respond to DHCP requests from clients and provide them with IP and next boot info when netbooting. This is the default mode. IP info is all reservation based. There must be a corresponding hardware record for the requesting client's MAC address.  
To enable this mode set `-dhcp-mode=reservation`.

1. **Proxy DHCP**  
Smee will respond to PXE enabled DHCP requests from clients and provide them with next boot info when netbooting. In this mode you will need an existing DHCP server that does not serve network boot information. Smee will respond to PXE enabled DHCP requests and provide the client with the next boot info. There must be a corresponding hardware record for the requesting client's MAC address. The `auto.ipxe` script will be served with the MAC address in the URL and the MAC address will be used to lookup the corresponding hardware record. In this mode you will still need layer 2 access to machines or need a DHCP relay agent in your environment that will forward the DHCP requests to Smee.  
To enable this mode set `-dhcp-mode=proxy`.

1. **DHCP disabled**  
Smee will not respond to DHCP requests from clients. This is useful if you have another DHCP server on your network that will provide both IP and next boot info and you want to use Smee's TFTP and HTTP functionality. In this mode you most likely want to set `-dhcp-http-ipxe-script-prepend-mac=false`. This will cause Smee to provide the `auto.ipxe` script without a MAC address in the URL and Smee will use the source IP address of the client request to lookup the corresponding hardware. There must be a corresponding hardware record for the requesting client's IP address. The IP address in the hardware record must be the same as the IP address of the client requesting the `auto.ipxe` script.  
To enable this mode set `-dhcp-enabled=false`.
