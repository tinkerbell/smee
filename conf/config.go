package conf

import (
	"net"
	"os"
	"strings"
	"time"

	"github.com/packethost/pkg/env"
	"github.com/pkg/errors"
)

var (
	PublicIPv4 = mustPublicIPv4()
	PublicFQDN = env.Get("PUBLIC_FQDN", PublicIPv4.String())

	PublicSyslogIPv4 = mustPublicSyslogIPv4()
	PublicSyslogFQDN = env.Get("PUBLIC_SYSLOG_FQDN", PublicSyslogIPv4.String())

	SyslogBind = env.Get("SYSLOG_BIND", PublicIPv4.String()+":514")
	HTTPBind   = env.Get("HTTP_BIND", PublicIPv4.String()+":80")
	TFTPBind   = env.Get("TFTP_BIND", PublicIPv4.String()+":69")
	BOOTPBind  = env.Get("BOOTP_BIND", PublicIPv4.String()+":67")

	// Default to Google Public DNS
	DHCPLeaseTime = env.Duration("DHCP_LEASE_TIME", (2 * 24 * time.Hour))
	DNSServers    = ParseIPv4s(env.Get("DNS_SERVERS", "8.8.8.8,8.8.4.4"))

	ignoredOUIs = getIgnoredMACs()
	ignoredGIs  = getIgnoredGIs()

	TrustedProxies = parseTrustedProxies()

	// Eclypsium registration token, passed into osie
	EclypsiumToken = env.Get("ECLYPSIUM_TOKEN")
	// Hollow auth secrets, passed into osie
	HollowClientId            = env.Get("HOLLOW_CLIENT_ID")
	HollowClientRequestSecret = env.Get("HOLLOW_CLIENT_REQUEST_SECRET")
)

func mustPublicIPv4() net.IP {
	if s, ok := os.LookupEnv("PUBLIC_IP"); ok {
		if a := net.ParseIP(s).To4(); a != nil {
			return a
		}
		err := errors.New("PUBLIC_IP must be an IPv4 address")
		panic(err)
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		err = errors.Wrap(err, "unable to auto-detect public IPv4")
		panic(err)
	}
	for _, addr := range addrs {
		ip, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		v4 := ip.IP.To4()
		if v4 == nil || !v4.IsGlobalUnicast() {
			continue
		}

		return v4
	}
	err = errors.New("unable to auto-detect public IPv4")
	panic(err)
}

func mustPublicSyslogIPv4() net.IP {
	if s, ok := os.LookupEnv("PUBLIC_SYSLOG_IP"); ok {
		if a := net.ParseIP(s).To4(); a != nil {
			return a
		}
		err := errors.New("PUBLIC_SYSLOG_IP must be an IPv4 address")
		panic(err)
	}

	return PublicIPv4
}

func ParseIPv4s(str string) (ips []net.IP) {
	for _, s := range strings.Split(str, ",") {
		ip := net.ParseIP(s).To4()
		if ip == nil {
			envlog.With("address", s).Info("value is not a valid IPv4 address")
		}
		ips = append(ips, ip)
	}

	return
}

func getIgnoredMACs() map[string]struct{} {
	macs := os.Getenv("TINK_IGNORED_OUIS")
	if macs == "" {
		return nil
	}

	slice := strings.Split(macs, ",")
	if len(slice) == 0 {
		return nil
	}

	ignore := map[string]struct{}{}
	for _, oui := range slice {
		_, err := net.ParseMAC(oui + ":00:00:00")
		if err != nil {
			panic(errors.Errorf("invalid oui in TINK_IGNORED_OUIS oui=%s", oui))
		}
		oui = strings.ToLower(oui)
		ignore[oui] = struct{}{}

	}

	return ignore
}

func ShouldIgnoreOUI(mac string) bool {
	if ignoredOUIs == nil {
		return false
	}
	oui := strings.ToLower(mac[:8])
	_, ok := ignoredOUIs[oui]

	return ok
}

func getIgnoredGIs() map[string]struct{} {
	ips := os.Getenv("TINK_IGNORED_GIS")
	if ips == "" {
		return nil
	}

	slice := strings.Split(ips, ",")
	if len(slice) == 0 {
		return nil
	}

	ignore := map[string]struct{}{}
	for _, ip := range slice {
		if net.ParseIP(ip) == nil {
			panic(errors.Errorf("invalid ip address in TINK_IGNORED_GIS ip=%s", ip))
		}
		ignore[ip] = struct{}{}

	}

	return ignore
}

func ShouldIgnoreGI(ip string) bool {
	if ignoredGIs == nil {
		return false
	}
	_, ok := ignoredGIs[ip]

	return ok
}

func parseTrustedProxies() (result []string) {
	trustedProxies := os.Getenv("TRUSTED_PROXIES")
	for _, cidr := range strings.Split(trustedProxies, ",") {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		_, _, err := net.ParseCIDR(cidr)
		if err != nil {
			// Its not a cidr, but maybe its an IP
			if ip := net.ParseIP(cidr); ip != nil {
				if ip.To4() != nil {
					cidr = cidr + "/32"
				} else {
					cidr = cidr + "/128"
				}
			} else {
				// not an IP, panic
				panic("invalid ip cidr in TRUSTED_PROXIES cidr=" + cidr)
			}
		}
		result = append(result, cidr)
	}

	return result
}
