package http

import (
	"net"
	"net/http"
	"strings"
)

// converts a list of subnets' string to a list of net.IPNet.
func toMasks(ips []string) (masks []net.IPNet, err error) {
	for _, cidr := range ips {
		var network *net.IPNet
		_, network, err = net.ParseCIDR(cidr)
		if err != nil {
			return
		}
		masks = append(masks, *network)
	}
	return
}

// checks if a net.IP is in a list of net.IPNet.
func ipInMasks(ip net.IP, masks []net.IPNet) bool {
	for _, mask := range masks {
		if mask.Contains(ip) {
			return true
		}
	}
	return false
}

// xffParse parses the value of the X-Forwarded-For Header and returns the IP address.
func xffParse(ipList string, allowed func(string) bool) string {
	ips := strings.Split(ipList, ",")
	if len(ips) == 0 {
		return ""
	}

	// simple case of only 1 proxy
	if len(ips) == 1 {
		ip := strings.TrimSpace(ips[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
		return ""
	}

	// multiple proxies
	// common form of X-F-F is: client, proxy1, proxy2, ... proxyN-1
	// so we verify backwards and return the first unallowed/untrusted proxy
	lastIP := ""
	for i := len(ips) - 1; i >= 0; i-- {
		ip := strings.TrimSpace(ips[i])
		if net.ParseIP(ip) == nil {
			break
		}
		lastIP = ip
		if !allowed(ip) {
			break
		}
	}
	return lastIP
}

// getRemoteAddrIfAllowed parses the given request, resolves the X-Forwarded-For header
// and returns the resolved remote address if allowed.
func getRemoteAddrIfAllowed(r *http.Request, allowed func(sip string) bool) string {
	if xffh := r.Header.Get("X-Forwarded-For"); xffh != "" {
		if sip, sport, err := net.SplitHostPort(r.RemoteAddr); err == nil && sip != "" {
			if allowed(sip) {
				if xip := xffParse(xffh, allowed); xip != "" {
					return net.JoinHostPort(xip, sport)
				}
			}
		}
	}
	return r.RemoteAddr
}

// xffOptions is a configuration container to setup the XFF middleware.
type xffOptions struct {
	// AllowedSubnets is a list of Subnets from which we will accept the
	// X-Forwarded-For header.
	// If this list is empty we will accept every Subnets (default).
	AllowedSubnets []string
	// Debugging flag adds additional output to debug server side XFF issues.
	Debug bool
}

// xff http handler.
type xff struct {
	// Set to true if all IPs or Subnets are allowed.
	allowAll bool
	// List of IP subnets that are allowed.
	allowedMasks []net.IPNet
}

// New creates a new XFF handler with the provided options.
func newXFF(options xffOptions) (*xff, error) {
	allowedMasks, err := toMasks(options.AllowedSubnets)
	if err != nil {
		return nil, err
	}
	xff := &xff{
		allowAll:     len(options.AllowedSubnets) == 0,
		allowedMasks: allowedMasks,
	}

	return xff, nil
}

// handler updates RemoteAdd from X-Fowarded-For Headers.
func (x *xff) handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.RemoteAddr = getRemoteAddrIfAllowed(r, x.allowed)
		h.ServeHTTP(w, r)
	})
}

// checks that the IP is allowed.
func (x *xff) allowed(sip string) bool {
	if x.allowAll {
		return true
	} else if ip := net.ParseIP(sip); ip != nil && ipInMasks(ip, x.allowedMasks) {
		return true
	}
	return false
}
