package packet

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"

	"github.com/tinkerbell/tink/client"
	tink "github.com/tinkerbell/tink/protos/hardware"
	"github.com/packethost/pkg/env"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/httplog"
	"google.golang.org/grpc"
)

type hardwareGetter interface {
	ByMAC(context.Context, *tink.GetRequest, ...grpc.CallOption) (*tink.Hardware, error)
	ByIP(context.Context, *tink.GetRequest, ...grpc.CallOption) (*tink.Hardware, error)
}

type Client struct {
	http          *http.Client
	baseURL       *url.URL
	consumerToken string
	authToken     string
	tink        hardwareGetter
}

func NewClient(consumerToken, authToken string, baseURL *url.URL) (*Client, error) {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("unexpected type for http.DefaultTransport")

	}

	transport := t.Clone()
	transport.MaxIdleConnsPerHost = env.Int("BOOTS_HTTP_HOST_CONNECTIONS", runtime.GOMAXPROCS(0)/2)

	c := &http.Client{
		Transport: &httplog.Transport{
			RoundTripper: transport,
		},
	}

	facility := os.Getenv("FACILITY_CODE")
	if facility == "" {
		return nil, errors.New("FACILITY_CODE env must be set")
	}

	tink, err := client.NewTinkerbellClient()
	if err != nil {
		return nil, errors.Wrap(err, "connect to cacher")
	}

	return &Client{
		http:          c,
		baseURL:       baseURL,
		consumerToken: consumerToken,
		authToken:     authToken,
		tink:        tink,
	}, nil
}

func NewMockClient(baseURL *url.URL) *Client {
	t := &httplog.Transport{
		RoundTripper: http.DefaultTransport,
	}
	c := &http.Client{
		Transport: t,
	}
	return &Client{
		http:    c,
		baseURL: baseURL,
	}
}

func (c *Client) Do(req *http.Request, v interface{}) error {
	req.URL = c.baseURL.ResolveReference(req.URL)
	c.addHeaders(req)

	res, err := c.http.Do(req)
	if err != nil {
		return errors.Wrap(err, "submit http request")
	}
	return unmarshalResponse(res, v)
}

func (c *Client) Get(ref string, v interface{}) error {
	req, err := http.NewRequest("GET", ref, nil)
	if err != nil {
		return errors.Wrap(err, "setup GET request")
	}
	return c.Do(req, v)
}

func (c *Client) Patch(ref, mime string, body io.Reader, v interface{}) error {
	req, err := http.NewRequest("PATCH", ref, body)
	if err != nil {
		return errors.Wrap(err, "setup PATCH request")
	}
	if mime != "" {
		req.Header.Set("Content-Type", mime)
	}
	return c.Do(req, v)
}

func (c *Client) Post(ref, mime string, body io.Reader, v interface{}) error {
	req, err := http.NewRequest("POST", ref, body)
	if err != nil {
		return errors.Wrap(err, "setup POST request")
	}
	if mime != "" {
		req.Header.Set("Content-Type", mime)
	}
	return c.Do(req, v)
}

func (c *Client) addHeaders(req *http.Request) {
	h := req.Header
	h.Set("X-Packet-Staff", "1")

	if c.consumerToken != "" {
		h.Set("X-Consumer-Token", c.consumerToken)
	}

	if c.authToken != "" {
		h.Set("X-Auth-Token", c.authToken)
	}
}

// golangci-lint: unused
//func (c *Client) do(fn func() (*http.Request, error), v interface{}) error {
//	req, err := fn()
//	if err != nil {
//		return err
//	}
//	return c.Do(req, v)
//}

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
