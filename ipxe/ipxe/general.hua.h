#define DIGEST_CMD            /* Image crypto digest commands */
#define DOWNLOAD_PROTO_HTTPS  /* Secure Hypertext Transfer Protocol */
#define IMAGE_TRUST_CMD	      /* Image trust management commands */
#define NET_PROTO_IPV6        /* IPv6 protocol */
#define NSLOOKUP_CMD          /* DNS resolving command */
#define NTP_CMD               /* NTP commands */
#define PARAM_CMD             /* params and param commands, for POSTing to tink */
#define REBOOT_CMD            /* Reboot command */
#define VLAN_CMD              /* VLAN commands */
#undef IMAGE_COMBOOT          /* COMBOOT */
#undef NET_PROTO_LACP
#undef NET_PROTO_STP

//#define BANNER_TIMEOUT          1 // 20
//#define ROM_BANNER_TIMEOUT      ( 2 * BANNER_TIMEOUT )

#define MAX_MODULES 16
