package packet

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"runtime"

	"github.com/packethost/pkg/env"
	"github.com/packethost/pkg/log"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/httplog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var _ client.Reporter = &Reporter{}

// client has all the fields corresponding to connection
type Reporter struct {
	http          *http.Client
	baseURL       *url.URL
	consumerToken string
	authToken     string
	logger        log.Logger
}

func NewReporter(logger log.Logger, baseURL *url.URL, consumerToken, authToken string) (*Reporter, error) {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("unexpected type for http.DefaultTransport")
	}

	// copy the default transport with all the default options
	transport := t.Clone()
	transport.MaxIdleConnsPerHost = env.Int("BOOTS_HTTP_HOST_CONNECTIONS", runtime.GOMAXPROCS(0)/2)

	// wrap the default http transport with otelhttp which will generate traces
	// and inject headers
	otelRt := otelhttp.NewTransport(transport)

	c := &http.Client{
		Transport: &httplog.Transport{
			RoundTripper: otelRt,
		},
	}

	return &Reporter{
		http:          c,
		baseURL:       baseURL,
		authToken:     authToken,
		consumerToken: consumerToken,
		logger:        logger,
	}, nil
}

func (c *Reporter) Do(ctx context.Context, req *http.Request, v interface{}) error {
	req = req.WithContext(ctx)
	req.URL = c.baseURL.ResolveReference(req.URL)
	c.addHeaders(req)

	res, err := c.http.Do(req)
	if err != nil {
		return errors.Wrap(err, "submit http request")
	}

	return unmarshalResponse(res, v)
}

func (c *Reporter) Get(ctx context.Context, ref string, v interface{}) error {
	req, err := http.NewRequest("GET", ref, nil)
	if err != nil {
		return errors.Wrap(err, "setup GET request")
	}

	return c.Do(ctx, req, v)
}

func (c *Reporter) Patch(ctx context.Context, ref, mime string, body io.Reader, v interface{}) error {
	req, err := http.NewRequest("PATCH", ref, body)
	if err != nil {
		return errors.Wrap(err, "setup PATCH request")
	}
	if mime != "" {
		req.Header.Set("Content-Type", mime)
	}

	return c.Do(ctx, req, v)
}

func (c *Reporter) Post(ctx context.Context, ref, mime string, body io.Reader, v interface{}) error {
	req, err := http.NewRequest("POST", ref, body)
	if err != nil {
		return errors.Wrap(err, "setup POST request")
	}
	if mime != "" {
		req.Header.Set("Content-Type", mime)
	}

	return c.Do(ctx, req, v)
}

func (c *Reporter) addHeaders(req *http.Request) {
	h := req.Header
	h.Set("X-Packet-Staff", "1")

	if c.consumerToken != "" {
		h.Set("X-Consumer-Token", c.consumerToken)
	}

	if c.authToken != "" {
		h.Set("X-Auth-Token", c.authToken)
	}
}

func unmarshalResponse(res *http.Response, result interface{}) error {
	defer res.Body.Close()
	defer io.Copy(ioutil.Discard, res.Body) // ensure all of the body is read so we can quickly reuse connection

	if res.StatusCode < 200 || res.StatusCode > 399 {
		e := &httpError{
			StatusCode: res.StatusCode,
		}
		e.unmarshalErrors(res.Body)

		return errors.Wrap(e, "unmarshalling response")
	}

	var err error
	if result == nil {
		return nil
	}

	err = errors.Wrap(json.NewDecoder(res.Body).Decode(result), "decode json body")
	if err == nil {
		return nil
	}

	return errors.Wrap(&httpError{
		StatusCode: res.StatusCode,
		Errors:     []error{err},
	}, "unmarshalling response")
}
