package iso

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"github.com/tinkerbell/smee/internal/dhcp/data"
)

const (
	defaultConsoles        = "console=ttyAMA0 console=ttyS0 console=tty0 console=tty1 console=ttyS1"
	maxContentLength int64 = 500 * 1024 // 500Kb
)

// BackendReader is an interface that defines the method to read data from a backend.
type BackendReader interface {
	// Read data (from a backend) based on a mac address
	// and return DHCP headers and options, including netboot info.
	GetByMac(context.Context, net.HardwareAddr) (*data.DHCP, *data.Netboot, error)
}

// Handler is a struct that contains the necessary fields to patch an ISO file with
// relevant information for the Tink worker.
type Handler struct {
	Backend           BackendReader
	ExtraKernelParams []string
	Logger            logr.Logger
	// MagicString is the string pattern that will be matched
	// in the source iso before patching. The field can be set
	// during build time by setting this field.
	// Ref: https://github.com/tinkerbell/hook/blob/main/linuxkit-templates/hook.template.yaml
	MagicString string
	// SourceISO is the source url where the unmodified iso lives.
	// It must be a valid url.URL{} object and must have a url.URL{}.Scheme of HTTP or HTTPS.
	SourceISO          string
	Syslog             string
	TinkServerTLS      bool
	TinkServerGRPCAddr string
	StaticIPAMEnabled  bool
	// parsedURL derives a url.URL from the SourceISO field.
	// It needed for validation of SourceISO and easier modification.
	parsedURL *url.URL
}

// HandlerFunc returns a reverse proxy HTTP handler function that performs ISO patching.
func (h *Handler) HandlerFunc() (http.HandlerFunc, error) {
	target, err := url.Parse(h.SourceISO)
	if err != nil {
		return nil, err
	}
	h.parsedURL = target
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Transport = h
	proxy.FlushInterval = -1

	return proxy.ServeHTTP, nil
}

// RoundTrip is a method on the Handler struct that implements the http.RoundTripper interface.
// This method is called by the httputil.NewSingleHostReverseProxy to handle the incoming request.
// The method is responsible for validating the incoming request, reading the source ISO, patching the ISO.
func (h *Handler) RoundTrip(req *http.Request) (*http.Response, error) {
	log := h.Logger.WithValues("method", req.Method, "urlPath", req.URL.Path, "remoteAddr", req.RemoteAddr, "fullURL", req.URL.String())
	log.V(1).Info("starting the ISO patching HTTP handler")
	if req.Method != http.MethodHead && req.Method != http.MethodGet {
		return &http.Response{
			Status:     fmt.Sprintf("%d %s", http.StatusNotImplemented, http.StatusText(http.StatusNotImplemented)),
			StatusCode: http.StatusNotImplemented,
			Body:       http.NoBody,
			Request:    req,
		}, nil
	}

	if filepath.Ext(req.URL.Path) != ".iso" {
		log.Info("extension not supported, only supported extension is '.iso'")
		return &http.Response{
			Status:     fmt.Sprintf("%d %s", http.StatusNotFound, http.StatusText(http.StatusNotFound)),
			StatusCode: http.StatusNotFound,
			Body:       http.NoBody,
			Request:    req,
		}, nil
	}

	// The incoming request url is expected to have the mac address present.
	// Fetch the mac and validate if there's a hardware object
	// associated with the mac.
	//
	// We serve the iso only if this validation passes.
	ha, err := getMAC(req.URL.Path)
	if err != nil {
		log.Info("unable to parse mac address in the URL path", "error", err)
		return &http.Response{
			Status:     fmt.Sprintf("%d %s", http.StatusBadRequest, http.StatusText(http.StatusBadRequest)),
			StatusCode: http.StatusBadRequest,
			Body:       http.NoBody,
			Request:    req,
		}, nil
	}

	fac, dhcpData, err := h.getFacility(req.Context(), ha, h.Backend)
	if err != nil {
		log.Info("unable to get the hardware object", "error", err, "mac", ha)
		if apierrors.IsNotFound(err) {
			return &http.Response{
				Status:     fmt.Sprintf("%d %s", http.StatusNotFound, http.StatusText(http.StatusNotFound)),
				StatusCode: http.StatusNotFound,
				Body:       http.NoBody,
				Request:    req,
			}, nil
		}
		return &http.Response{
			Status:     fmt.Sprintf("%d %s", http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)),
			StatusCode: http.StatusInternalServerError,
			Body:       http.NoBody,
			Request:    req,
		}, nil
	}

	// The httputil.NewSingleHostReverseProxy takes the incoming request url and adds the path to the target (h.SourceISO).
	// This function is more than a pass through proxy. The MAC address in the url path is required to do hardware lookups using the backend reader
	// and is not used when making http calls to the target (h.SourceISO). All valid requests are passed through to the target.
	req.URL.Path = h.parsedURL.Path

	// RoundTripper needs a Transport to execute a HTTP transaction
	// For our use case the default transport will suffice.
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Error(err, "issue getting the source ISO", "sourceIso", h.SourceISO)
		return nil, err
	}
	// by setting this header we are telling the logging middleware to not log its default log message.
	// we do this because there are a lot of partial content requests and it allow this handler to take care of logging.
	resp.Header.Set("X-Global-Logging", "false")

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		// This log line is not rate limited as we don't anticipate this to be a common occurrence or happen frequently when it does.
		log.Info("the request to get the source ISO returned a status other than ok (200) or partial content (206)", "sourceIso", h.SourceISO, "status", resp.Status)
		return resp, nil
	}

	if req.Method == http.MethodHead {
		// Fuse clients typically make a HEAD request before they start requesting content. This is not rate limited as the occurrence is expected to be low.
		// This allows provides us some info on the progress of the client.
		log.Info("HTTP HEAD method received", "status", resp.Status)
		return resp, nil
	}

	// At this point we only allow HTTP GET method with a 206 status code.
	// Otherwise we are potentially reading the entire ISO file and patching it.
	// This is not the intended use case for this handler.
	// And this can cause memory issues, like OOM, if the ISO file is too large.
	// By returning the `resp` here we allow clients to download the ISO file but without any patching.
	// This is done so that there can be a minimal amount of troubleshooting for SourceISO issues.
	if resp.StatusCode != http.StatusPartialContent {
		log.Info("HTTP GET method received with a status code other than 206, source iso will be unpatched", "status", resp.Status, "respHeader", resp.Header, "reqHeaders", resp.Request.Header)
		return resp, nil
	}
	// If the request is a partial content request, we need to validate the Content-Range header.
	// Because we read the entire response body into memory for patching, we need to ensure that the
	// Content-Range is within a reasonable size. For now, we are limiting the size to 500Kb (partialContentMax).

	// Content range RFC: https://tools.ietf.org/html/rfc7233#section-4.2
	// https://datatracker.ietf.org/doc/html/rfc7233#section-4.4

	// Get the content range from the response header
	if resp.ContentLength > maxContentLength {
		log.Info("content length is greater than max", "contentLengthBytes", resp.ContentLength, "maxAllowedBytes", maxContentLength)
		return resp, nil
	}

	// 0.002% of the time we log a 206 request message.
	// In testing, it was observed that about 3000 HTTP 206 requests are made per ISO mount.
	// 0.002% gives us about 5 - 10, log messages per ISO mount.
	// We're optimizing for showing "enough" log messages so that progress can be observed.
	if p := randomPercentage(100000); p < 0.002 {
		log.Info("HTTP GET method received with a 206 status code")
	}

	// this roundtripper func should only return error when there is no response from the server.
	// for any other case we log the error and return a 500 response. See the http.RoundTripper interface code
	// comments for more details.
	var b []byte
	respBuf := new(bytes.Buffer)
	if _, err := io.CopyN(respBuf, resp.Body, resp.ContentLength); err != nil {
		log.Info("unable to read response bytes", "error", err)
		return &http.Response{
			Status:     fmt.Sprintf("%d %s", http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)),
			StatusCode: http.StatusInternalServerError,
			Body:       http.NoBody,
			Request:    req,
			Header:     resp.Header,
		}, nil
	}
	b = respBuf.Bytes()
	if err := resp.Body.Close(); err != nil {
		log.Info("unable to close response body", "error", err)
		return &http.Response{
			Status:     fmt.Sprintf("%d %s", http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)),
			StatusCode: http.StatusInternalServerError,
			Body:       http.NoBody,
			Request:    req,
			Header:     resp.Header,
		}, nil
	}

	// The hardware object doesn't contain a dedicated field for consoles right now and
	// historically the facility is used as a way to define consoles on a per Hardware basis.
	var consoles string
	switch {
	case fac != "" && strings.Contains(fac, "console="):
		consoles = fmt.Sprintf("facility=%s", fac)
	case fac != "":
		consoles = fmt.Sprintf("facility=%s %s", fac, defaultConsoles)
	default:
		consoles = defaultConsoles
	}
	magicStringPadding := bytes.Repeat([]byte{' '}, len(h.MagicString))

	// TODO: revisit later to handle the magic string potentially being spread across two chunks.
	// In current implementation we will never patch the above case. Add logic to patch the case of
	// magic string spread across multiple response bodies in the future.
	i := bytes.Index(b, []byte(h.MagicString))
	if i != -1 {
		log.Info("magic string found, patching the content", "contentRange", resp.Header.Get("Content-Range"))
		dup := make([]byte, len(b))
		copy(dup, b)
		copy(dup[i:], magicStringPadding)
		copy(dup[i:], []byte(h.constructPatch(consoles, ha.String(), dhcpData)))
		b = dup
	}

	resp.Body = io.NopCloser(bytes.NewReader(b))
	log.V(1).Info("roundtrip complete")

	return resp, nil
}

func (h *Handler) constructPatch(console, mac string, d *data.DHCP) string {
	syslogHost := fmt.Sprintf("syslog_host=%s", h.Syslog)
	grpcAuthority := fmt.Sprintf("grpc_authority=%s", h.TinkServerGRPCAddr)
	tinkerbellTLS := fmt.Sprintf("tinkerbell_tls=%v", h.TinkServerTLS)
	workerID := fmt.Sprintf("worker_id=%s", mac)
	vlanID := func() string {
		if d != nil {
			return fmt.Sprintf("vlan_id=%s", d.VLANID)
		}
		return ""
	}()
	hwAddr := fmt.Sprintf("hw_addr=%s", mac)
	ipam := parseIPAM(d)

	return strings.Join([]string{strings.Join(h.ExtraKernelParams, " "), console, vlanID, hwAddr, syslogHost, grpcAuthority, tinkerbellTLS, workerID, ipam}, " ")
}

func getMAC(urlPath string) (net.HardwareAddr, error) {
	mac := path.Base(path.Dir(urlPath))
	hw, err := net.ParseMAC(mac)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL path: %s , the second to last element in the URL path must be a valid mac address, err: %w", urlPath, err)
	}

	return hw, nil
}

func (h *Handler) getFacility(ctx context.Context, mac net.HardwareAddr, br BackendReader) (string, *data.DHCP, error) {
	if br == nil {
		return "", nil, errors.New("backend is nil")
	}

	d, n, err := br.GetByMac(ctx, mac)
	if err != nil {
		return "", nil, err
	}
	if !h.StaticIPAMEnabled {
		d = nil
	}

	return n.Facility, d, nil
}

func randomPercentage(precision int64) float64 {
	random, err := rand.Int(rand.Reader, big.NewInt(precision))
	if err != nil {
		return 0
	}

	return float64(random.Int64()) / float64(precision)
}

func parseIPAM(d *data.DHCP) string {
	if d == nil {
		return ""
	}
	// return format is ipam=<mac-address>:<vlan-id>:<ip-address>:<netmask>:<gateway>:<hostname>:<dns>:<search-domains>:<ntp>
	ipam := make([]string, 9)
	ipam[0] = func() string {
		m := d.MACAddress.String()

		return strings.ReplaceAll(m, ":", "-")
	}()
	ipam[1] = func() string {
		if d.VLANID != "" {
			return d.VLANID
		}
		return ""
	}()
	ipam[2] = func() string {
		if d.IPAddress.Compare(netip.Addr{}) != 0 {
			return d.IPAddress.String()
		}
		return ""
	}()
	ipam[3] = func() string {
		if d.SubnetMask != nil {
			return net.IP(d.SubnetMask).String()
		}
		return ""
	}()
	ipam[4] = func() string {
		if d.DefaultGateway.Compare(netip.Addr{}) != 0 {
			return d.DefaultGateway.String()
		}
		return ""
	}()
	ipam[5] = d.Hostname
	ipam[6] = func() string {
		var nameservers []string
		for _, e := range d.NameServers {
			nameservers = append(nameservers, e.String())
		}
		if len(nameservers) > 0 {
			return strings.Join(nameservers, ",")
		}

		return ""
	}()
	ipam[8] = func() string {
		var ntp []string
		for _, e := range d.NTPServers {
			ntp = append(ntp, e.String())
		}
		if len(ntp) > 0 {
			return strings.Join(ntp, ",")
		}

		return ""
	}()

	return fmt.Sprintf("ipam=%s", strings.Join(ipam, ":"))
}
