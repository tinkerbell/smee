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
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"github.com/tinkerbell/smee/internal/dhcp/handler"
)

const (
	defaultConsoles = "console=ttyAMA0 console=ttyS0 console=tty0 console=tty1 console=ttyS1"
)

type Handler struct {
	Logger  logr.Logger
	Backend handler.BackendReader
	// SourceISO is the source url where the unmodified iso lives
	// patch this at runtime, should be a HTTP(S) url.
	SourceISO          string
	ExtraKernelParams  []string
	Syslog             string
	TinkServerTLS      bool
	TinkServerGRPCAddr string
	// parsedURL derives a url.URL from the SourceISO
	// It helps accessing different parts of URL
	parsedURL *url.URL
	// MagicString is the string pattern that will be matched
	// in the source iso before patching. The field can be set
	// during build time by setting this field.
	// Ref: https://github.com/tinkerbell/hook/blob/main/linuxkit-templates/hook.template.yaml
	MagicString string
}

func randomPercentage(precision int64) float64 {
	random, err := rand.Int(rand.Reader, big.NewInt(precision))
	if err != nil {
		return 0
	}

	return float64(random.Int64()) / float64(precision)
}

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

	f, err := getFacility(req.Context(), ha, h.Backend)
	if err != nil {
		log.Error(err, "unable to get the hardware object", "mac", ha)
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

	// Reverse Proxy modifies the request url to
	// the same path it received the incoming request.
	// mac-id is added to the url path to do hardware lookups using the backend reader
	// and is not used when making http calls to the source url.
	req.URL.Path = h.parsedURL.Path

	// RoundTripper needs a Transport to execute a HTTP transaction
	// For our use case the default transport will suffice.
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Info("issue getting the source ISO", "sourceIso", h.SourceISO, "error", err)
		return nil, err
	}
	// by setting this header we are telling the logging middleware to not log its default log message.
	// we do this because there are a lot of partial content requests.
	resp.Header.Set("X-Global-Logging", "false")

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		log.Info("the request to get the source ISO returned a status other than ok (200) or partial content (206)", "sourceIso", h.SourceISO, "status", resp.Status)
		return resp, nil
	}

	if req.Method == http.MethodHead {
		// Fuse clients typically make a HEAD request before they start requesting content.
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
		log.Info("HTTP GET method received with a status code other than 206, source iso will be unpatched", "status", resp.Status)
		return resp, nil
	}
	// If the request is a partial content request, we need to validate the Content-Range header.
	// Because we read the entire response body into memory for patching, we need to ensure that the
	// Content-Range is within a reasonable size. For now, we are limiting the size to 500Kb.

	// Content range RFC: https://tools.ietf.org/html/rfc7233#section-4.2
	// https://datatracker.ietf.org/doc/html/rfc7233#section-4.4

	// Get the content range from the response header
	contentRange := resp.Header.Get("Content-Range")
	l := strings.Split(contentRange, "/")
	if len(l) == 2 {
		sp := strings.Split(l[0], " ")
		if len(sp) == 2 {
			contentRange = fmt.Sprintf("%s=%s", sp[0], sp[1])
		}
	}
	ln := len(l) - 1
	size := l[ln]
	sizeInt, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		log.Error(err, "unable to get the size of the content from the range header", "respContentRange", contentRange, "reqContentRange", resp.Request.Header.Get("Range"))
		return &http.Response{
			Status:     fmt.Sprintf("%d %s", http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)),
			StatusCode: http.StatusInternalServerError,
			Body:       http.NoBody,
			Request:    req,
			Header:     resp.Header,
		}, nil
	}

	crp, err := ParseRange(contentRange, sizeInt)
	if err != nil {
		log.Error(err, "issue parsing the content range header", "respContentRange", contentRange, "reqContentRange", resp.Request.Header.Get("Range"))
		resp.Header.Set("Content-Range", fmt.Sprintf("bytes */%d", sizeInt))
		return &http.Response{
			Status:     fmt.Sprintf("%d %s", http.StatusRequestedRangeNotSatisfiable, http.StatusText(http.StatusRequestedRangeNotSatisfiable)),
			StatusCode: http.StatusRequestedRangeNotSatisfiable,
			Body:       http.NoBody,
			Request:    req,
			Header:     resp.Header,
		}, nil
	}

	for _, r := range crp {
		if r.Length > 500*1024 {
			log.Info("content range length is greater than 512Kb", "contentRange", contentRange)
			resp.Header.Set("Content-Range", fmt.Sprintf("bytes */%d", sizeInt))
			return &http.Response{
				Status:     fmt.Sprintf("%d %s", http.StatusRequestedRangeNotSatisfiable, http.StatusText(http.StatusRequestedRangeNotSatisfiable)),
				StatusCode: http.StatusRequestedRangeNotSatisfiable,
				Body:       http.NoBody,
				Request:    req,
				Header:     resp.Header,
			}, nil
		}
	}
	// Check that it is less than 500Kb
	//http.StatusRequestedRangeNotSatisfiable

	// 0.002% of the time we log a 206 request message.
	// In testing, it was observed that about 3000 HTTP 206 requests are made per ISO mount.
	// 0.002% gives us about 5, +/- 3, log messages per ISO mount.
	// We're optimizing for showing "enough" log messages so that progress can be observed.
	if p := randomPercentage(10000); p < 0.002 {
		log.Info("HTTP GET method received with a 206 status code")
	}

	// this roundtripper func should only return error when there is no response from the server.
	// for any other case we log the error and return a 500 response. See the http.RoundTripper interface code
	// comments for more details.
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		//var b []byte
		//if _, err := io.LimitReader(resp.Body, 500*1024).Read(b); err != nil {
		log.Error(err, "issue reading response bytes")
		return &http.Response{
			Status:     fmt.Sprintf("%d %s", http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)),
			StatusCode: http.StatusInternalServerError,
			Body:       http.NoBody,
			Request:    req,
			Header:     resp.Header,
		}, nil
	}
	if err := resp.Body.Close(); err != nil {
		log.Error(err, "issue closing response body")
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
	case f != "" && strings.Contains(f, "console="):
		consoles = fmt.Sprintf("facility=%s", f)
	case f != "":
		consoles = fmt.Sprintf("facility=%s %s", f, defaultConsoles)
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
		copy(dup[i:], []byte(h.constructPatch(consoles, ha.String())))
		b = dup
	}

	resp.Body = io.NopCloser(bytes.NewReader(b))
	log.V(1).Info("roundtrip complete")
	return resp, nil
}

func (h *Handler) Reverse() (http.HandlerFunc, error) {
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

func (h *Handler) constructPatch(console, mac string) string {
	syslogHost := fmt.Sprintf("syslog_host=%s", h.Syslog)
	grpcAuthority := fmt.Sprintf("grpc_authority=%s", h.TinkServerGRPCAddr)
	tinkerbellTLS := fmt.Sprintf("tinkerbell_tls=%v", h.TinkServerTLS)
	workerID := fmt.Sprintf("worker_id=%s", mac)

	return strings.Join([]string{strings.Join(h.ExtraKernelParams, " "), console, syslogHost, grpcAuthority, tinkerbellTLS, workerID}, " ")
}

func getMAC(urlPath string) (net.HardwareAddr, error) {
	mac := path.Base(path.Dir(urlPath))
	hw, err := net.ParseMAC(mac)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL path: %s , the second to last element in the URL path must be a valid mac address, err: %w", urlPath, err)
	}

	return hw, nil
}

func getFacility(ctx context.Context, mac net.HardwareAddr, br handler.BackendReader) (string, error) {
	if br == nil {
		return "", errors.New("backend is nil")
	}

	_, n, err := br.GetByMac(ctx, mac)
	if err != nil {
		return "", err
	}

	return n.Facility, nil
}
