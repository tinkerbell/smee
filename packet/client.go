package packet

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/tinkerbell/boots/httplog"
	"github.com/packethost/cacher/client"
	"github.com/packethost/cacher/protos/cacher"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type hardwareGetter interface {
	ByMAC(context.Context, *cacher.GetRequest, ...grpc.CallOption) (*cacher.Hardware, error)
	ByIP(context.Context, *cacher.GetRequest, ...grpc.CallOption) (*cacher.Hardware, error)
}

type Client struct {
	http          *http.Client
	baseURL       *url.URL
	consumerToken string
	authToken     string
	cacher        hardwareGetter
}

func NewClient(consumerToken, authToken string, baseURL *url.URL) (*Client, error) {
	t := &httplog.Transport{
		RoundTripper: http.DefaultTransport,
	}
	c := &http.Client{
		Transport: t,
	}

	facility := os.Getenv("FACILITY_CODE")
	if facility == "" {
		return nil, errors.New("FACILITY_CODE env must be set")
	}

	cacher, err := client.New(facility)
	if err != nil {
		return nil, errors.Wrap(err, "connect to cacher")
	}

	return &Client{
		http:          c,
		baseURL:       baseURL,
		consumerToken: consumerToken,
		authToken:     authToken,
		cacher:        cacher,
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

//func (c *Client) do(fn func() (*http.Request, error), v interface{}) error {
//	req, err := fn()
//	if err != nil {
//		return err
//	}
//	return c.Do(req, v)
//}

func unmarshalResponse(res *http.Response, result interface{}) error {
	defer res.Body.Close()

	if res.StatusCode > 399 || res.StatusCode < 200 {
		e := &httpError{
			StatusCode: res.StatusCode,
		}
		e.unmarshalErrors(res.Body)
		return errors.Wrap(e, "unmarshalling response")
	}

	var err error
	if result == nil {
		_, err = io.Copy(ioutil.Discard, res.Body)
		err = errors.Wrap(err, "discard errors response body")
	} else {
		err = errors.Wrap(json.NewDecoder(res.Body).Decode(result), "decode json body")
	}

	if err == nil {
		return nil
	}

	return errors.Wrap(&httpError{
		StatusCode: res.StatusCode,
		Errors:     []error{err},
	}, "unmarshalling response")
}
