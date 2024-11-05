package iso

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"github.com/tinkerbell/smee/internal/dhcp/handler"
)

const (
	defaultConsoles = "console=ttyS1 console=ttyS1 console=ttyS0 console=ttyAMA0 console=ttyS1 console=tty0"
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

func (h *Handler) RoundTrip(req *http.Request) (*http.Response, error) {
	h.Logger.V(1).Info("entered the roundtrip func")
	if req.Method != http.MethodHead && req.Method != http.MethodGet {
		return &http.Response{
			Status:     fmt.Sprintf("%d %s", http.StatusNotImplemented, http.StatusText(http.StatusNotImplemented)),
			StatusCode: http.StatusNotImplemented,
			Body:       http.NoBody,
			Request:    req,
		}, nil
	}

	if filepath.Ext(req.URL.Path) != ".iso" {
		h.Logger.Info("Extension not supported, only supported type is '.iso'", "path", req.URL.Path)
		return &http.Response{
			Status:     fmt.Sprintf("%d %s", http.StatusNotFound, http.StatusText(http.StatusNotFound)),
			StatusCode: http.StatusNotFound,
			Body:       http.NoBody,
			Request:    req,
		}, nil
	}

	ctx := req.Context()
	// The incoming request url is expected to have the mac address present.
	// Fetch the mac and validate if there's a hardware object
	// associated with the mac.
	//
	// We serve the iso only if this validation passes.
	ha, err := getMAC(req.URL.Path)
	if err != nil {
		h.Logger.Info("unable to get the mac address", "error", err)
		return &http.Response{
			Status:     "400 BAD REQUEST",
			StatusCode: http.StatusBadRequest,
			Body:       http.NoBody,
			Request:    req,
		}, nil
	}

	f, err := getFacility(ctx, ha, h.Backend)
	if err != nil {
		h.Logger.V(1).Info("unable to get facility", "mac", ha, "error", err)
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

	// The hardware object doesn't contain a specific field for consoles
	// right now facility is used instead.
	var consoles string
	switch {
	case f != "" && strings.Contains(f, "console="):
		consoles = f
	case f != "":
		consoles = fmt.Sprintf("%s %s", f, defaultConsoles)
	default:
		consoles = defaultConsoles
	}

	// Reverse Proxy modifies the request url to
	// the same path it received the incoming request.
	// mac-id is added to the url path to do hardware lookups using the backend reader
	// and is not used when making http calls to the source url.
	req.URL.Path = h.parsedURL.Path

	// RoundTripper needs a Transport to execute a HTTP transaction
	// For our use case the default transport will suffice.
	resp, err := http.DefaultTransport.RoundTrip(req)
	// resp, err := h.RoundTripper.RoundTrip(req)
	if err != nil {
		h.Logger.Info("HTTP request didn't receive a response", "sourceIso", h.SourceISO, "error", err)
		return nil, err
	}

	if req.Method == http.MethodHead {
		// Fuse client typically make a HEAD request before they start requesting content.
		h.Logger.V(1).Info("HTTP HEAD request received, patching only occurs on 206 requests")
		return resp, nil
	}

	// roundtripper should only return error when no response from the server
	// for any other case just log the error and return 404 response
	if resp.StatusCode == http.StatusPartialContent {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			h.Logger.Error(err, "reading response bytes", "response", resp.Body)
			return &http.Response{
				Status:     fmt.Sprintf("%d %s", http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)),
				StatusCode: http.StatusInternalServerError,
				Body:       http.NoBody,
				Request:    req,
			}, nil
		}
		if err := resp.Body.Close(); err != nil {
			h.Logger.Error(err, "closing response body", "response", resp.Body)
			return &http.Response{
				Status:     fmt.Sprintf("%d %s", http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)),
				StatusCode: http.StatusInternalServerError,
				Body:       http.NoBody,
				Request:    req,
			}, nil
		}

		magicStringPadding := bytes.Repeat([]byte{' '}, len(h.MagicString))

		// TODO: revisit later to handle the magic string potentially being spread across two chunks.
		// In current implementation we will never patch the above case. Add logic to patch the case of
		// magic string spread across multiple response bodies in the future.
		i := bytes.Index(b, []byte(h.MagicString))
		if i != -1 {
			h.Logger.Info("Magic string found, patching the iso at runtime")
			dup := make([]byte, len(b))
			copy(dup, b)
			copy(dup[i:], magicStringPadding)
			copy(dup[i:], []byte(h.constructPatch(fmt.Sprintf("facility=%s", consoles), ha.String())))
			b = dup
		}

		resp.Body = io.NopCloser(bytes.NewReader(b))
	}

	h.Logger.Info("roundtrip complete")
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
