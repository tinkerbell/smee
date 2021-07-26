#define DIGEST_CMD            /* Image crypto digest commands */
#define DOWNLOAD_PROTO_HTTPS  /* Secure Hypertext Transfer Protocol */
#define IMAGE_TRUST_CMD       /* Image trust management commands */
#define NET_PROTO_IPV6        /* IPv6 protocol */
#define NSLOOKUP_CMD          /* DNS resolving command */
#define NTP_CMD               /* NTP commands */
#define NVO_CMD               /* Non-volatile option storage commands */
#define PARAM_CMD             /* params and param commands, for POSTing to tink */
#define PING_CMD              /* Ping command */
#define REBOOT_CMD            /* Reboot command */
#define SANBOOT_PROTO_HTTP    /* HTTP SAN protocol */
#define VLAN_CMD              /* VLAN commands */

#undef CRYPTO_80211_WEP       /* WEP encryption (deprecated and insecure!) */
#undef CRYPTO_80211_WPA2      /* Add support for stronger WPA cryptography */
#undef CRYPTO_80211_WPA       /* WPA Personal, authenticating with passphrase */
#undef FCMGMT_CMD             /* Fibre Channel management commands */
#undef IBMGMT_CMD             /* Infiniband management commands */
#undef IMAGE_PNG              /* PNG image support */
#undef IMAGE_PNM              /* PNM image support */
#undef IWMGMT_CMD             /* Wireless interface management commands */
#undef NET_PROTO_LACP         /* Link Aggregation control protocol */
#undef NET_PROTO_STP          /* Spanning Tree protocol */
#undef ROUTE_CMD              /* Routing table management commands */
#undef VNIC_IPOIB             /* Infiniband IPoIB virtual NICs */

//defined in config/defaults/{efi,pcbios}.h and we don't want
#undef SANBOOT_PROTO_AOE      /* AoE protocol */
#undef SANBOOT_PROTO_FCP      /* Fibre Channel protocol */
#undef SANBOOT_PROTO_IB_SRP   /* Infiniband SCSI RDMA protocol */
#undef SANBOOT_PROTO_ISCSI    /* iSCSI protocol */
#undef USB_EFI                /* Provide EFI_USB_IO_PROTOCOL interface */
#undef USB_HCD_EHCI           /* EHCI USB host controller */
#undef USB_HCD_UHCI           /* UHCI USB host controller */
#undef USB_HCD_XHCI           /* xHCI USB host controller */
#undef USB_KEYBOARD           /* USB keyboards */

#define MAX_MODULES 16
