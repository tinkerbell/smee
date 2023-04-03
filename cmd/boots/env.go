package main

import (
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// getStringEnv retrieves the value of the environment variable named by the key.
// If the value is empty or unset it will return the first value of def or "" if none is given.
func getStringEnv(name string, def ...string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	if len(def) > 0 {
		return def[0]
	}
	return ""
}

// getBoolEnv parses given environment variables as a boolean, or returns the default if the environment variable is empty/unset.
// If the value is empty or unset it will return the first value of def or false if none is given.
// Evaluates true if the value case-insensitive matches 1|t|true|y|yes.
func getBoolEnv(name string, def ...bool) bool {
	if v := os.Getenv(name); v != "" {
		v = strings.ToLower(v)
		if v == "1" || v == "t" || v == "true" || v == "y" || v == "yes" {
			return true
		}
		return false
	}
	if len(def) > 0 {
		return def[0]
	}
	return false
}

// getIntEnv parses given environment variable as an int, or returns the default if the environment variable is empty/unset.
// Int will panic if it fails to parse the value.
func getIntEnv(name string, def ...int) int {
	if v := os.Getenv(name); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			err = errors.Wrap(err, "failed to parse int from env var")
			panic(err)
		}
		return i
	}
	if len(def) > 0 {
		return def[0]
	}
	return 0
}

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

	return publicIPv4
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
