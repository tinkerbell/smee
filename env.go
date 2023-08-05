package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
)

func detectPublicIPv4(extra string) string {
	ip, err := autoDetectPublicIPv4()
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%v%v", ip.String(), extra)
}

func autoDetectPublicIPv4() (net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("unable to auto-detect public IPv4: %w", err)
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

		return v4, nil
	}

	return nil, errors.New("unable to auto-detect public IPv4")
}

func parseTrustedProxies(trustedProxies string) (result []string) {
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
					cidr += "/32"
				} else {
					cidr += "/128"
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
