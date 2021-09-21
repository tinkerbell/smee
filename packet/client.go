package packet

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"

	cacherClient "github.com/packethost/cacher/client"
	"github.com/packethost/pkg/env"
	"github.com/packethost/pkg/log"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/httplog"
	tinkClient "github.com/tinkerbell/tink/client"
	tw "github.com/tinkerbell/tink/protos/workflow"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type hardwareGetter interface {
}

type Client interface {
	GetWorkflowsFromTink(context.Context, HardwareID) (*tw.WorkflowContextList, error)
	DiscoverHardwareFromDHCP(ctx context.Context, mac net.HardwareAddr, giaddr net.IP, circuitID string) (Discovery, error)
	ReportDiscovery(ctx context.Context, mac net.HardwareAddr, giaddr net.IP, circuitID string) (Discovery, error)

	DiscoverHardwareFromIP(ctx context.Context, ip net.IP) (Discovery, error)
	PostHardwareComponent(ctx context.Context, hardwareID HardwareID, body io.Reader) (*ComponentsResponse, error)
	PostHardwareEvent(ctx context.Context, id string, body io.Reader) (string, error)
	PostHardwarePhoneHome(ctx context.Context, id string) error
	PostHardwareFail(ctx context.Context, id string, body io.Reader) error
	PostHardwareProblem(ctx context.Context, id HardwareID, body io.Reader) (string, error)

	GetInstanceIDFromIP(ctx context.Context, dip net.IP) (string, error)
	PostInstancePhoneHome(context.Context, string) error
	PostInstanceEvent(ctx context.Context, id string, body io.Reader) (string, error)
	PostInstanceFail(ctx context.Context, id string, body io.Reader) error
	PostInstancePassword(ctx context.Context, id, pass string) error
	UpdateInstance(ctx context.Context, id string, body io.Reader) error
}

var _ Client = &client{}

// client has all the fields corresponding to connection
type client struct {
	http           *http.Client
	baseURL        *url.URL
	consumerToken  string
	authToken      string
	hardwareClient hardwareGetter
	workflowClient tw.WorkflowServiceClient
	logger         log.Logger
}

func NewClient(logger log.Logger, consumerToken, authToken string, baseURL *url.URL) (Client, error) {
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

	var hg hardwareGetter
	var wg tw.WorkflowServiceClient
	var err error
	dataModelVersion := os.Getenv("DATA_MODEL_VERSION")
	switch dataModelVersion {
	// Tinkerbell V1 backend
	case "1":
		hg, err = tinkClient.TinkHardwareClient()
		if err != nil {
			return nil, errors.Wrap(err, "connect to tink")
		}

		wg, err = tinkClient.TinkWorkflowClient()
		if err != nil {
			return nil, errors.Wrap(err, "connect to tink")
		}
	// classic Packet API / Cacher backend (default for empty envvar)
	case "":
		facility := os.Getenv("FACILITY_CODE")
		if facility == "" {
			return nil, errors.New("FACILITY_CODE env must be set")
		}

		hg, err = cacherClient.New(facility)
		if err != nil {
			return nil, errors.Wrap(err, "connect to cacher")
		}
	// standalone, use a json file for all hardware data
	case "standalone":
		saFile := os.Getenv("BOOTS_STANDALONE_JSON")
		if saFile == "" {
			return nil, errors.New("BOOTS_STANDALONE_JSON env must be set")
		}
		// set the baseURL from here so it gets returned in the client
		// TODO(@tobert): maybe there's a way to pass a file:// in the first place?
		baseURL, err = url.Parse("file://" + saFile)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to convert path %q to a URL as 'file://%s'", saFile, saFile)
		}
		saData, err := ioutil.ReadFile(saFile)
		if err != nil {
			return nil, errors.Wrapf(err, "could not read file %q", saFile)
		}
		dsDb := []DiscoverStandalone{}
		err = json.Unmarshal(saData, &dsDb)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to parse configuration file %q", saFile)
		}

		// the "client" part is done - reading the json, now return a struct client
		// that is just the filename and parsed data structure
		hg = StandaloneClient{
			filename: saFile,
			db:       dsDb,
		}
	default:
		return nil, errors.Errorf("invalid DATA_MODEL_VERSION: %q", dataModelVersion)
	}

	return &client{
		http:           c,
		baseURL:        baseURL,
		consumerToken:  consumerToken,
		authToken:      authToken,
		hardwareClient: hg,
		workflowClient: wg,
		logger:         logger,
	}, nil
}

func NewMockClient(baseURL *url.URL, workflowClient tw.WorkflowServiceClient) *client {
	t := &httplog.Transport{
		RoundTripper: http.DefaultTransport,
	}
	c := &http.Client{
		Transport: t,
	}

	return &client{
		http:           c,
		workflowClient: workflowClient,
		baseURL:        baseURL,
	}
}

func (c *client) Do(ctx context.Context, req *http.Request, v interface{}) error {
	req = req.WithContext(ctx)
	req.URL = c.baseURL.ResolveReference(req.URL)
	c.addHeaders(req)

	res, err := c.http.Do(req)
	if err != nil {
		return errors.Wrap(err, "submit http request")
	}

	return unmarshalResponse(res, v)
}

func (c *client) Get(ctx context.Context, ref string, v interface{}) error {
	req, err := http.NewRequest("GET", ref, nil)
	if err != nil {
		return errors.Wrap(err, "setup GET request")
	}

	return c.Do(ctx, req, v)
}

func (c *client) Patch(ctx context.Context, ref, mime string, body io.Reader, v interface{}) error {
	req, err := http.NewRequest("PATCH", ref, body)
	if err != nil {
		return errors.Wrap(err, "setup PATCH request")
	}
	if mime != "" {
		req.Header.Set("Content-Type", mime)
	}

	return c.Do(ctx, req, v)
}

func (c *client) Post(ctx context.Context, ref, mime string, body io.Reader, v interface{}) error {
	req, err := http.NewRequest("POST", ref, body)
	if err != nil {
		return errors.Wrap(err, "setup POST request")
	}
	if mime != "" {
		req.Header.Set("Content-Type", mime)
	}

	return c.Do(ctx, req, v)
}

func (c *client) addHeaders(req *http.Request) {
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
