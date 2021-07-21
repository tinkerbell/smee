package packet

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"

	cacherClient "github.com/packethost/cacher/client"
	"github.com/packethost/pkg/env"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/httplog"
	tinkClient "github.com/tinkerbell/tink/client"
	tw "github.com/tinkerbell/tink/protos/workflow"
	"gopkg.in/yaml.v3"
)

type hardwareGetter interface {
}

// Client has all the fields corresponding to connection
type Client struct {
	http           *http.Client
	baseURL        *url.URL
	consumerToken  string
	authToken      string
	hardwareClient hardwareGetter
	workflowClient tw.WorkflowServiceClient
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
	// standalone, use a yaml file for all hardware data
	case "standalone":
		saYamlFile := os.Getenv("BOOTS_STANDALONE_YAML")
		if saYamlFile == "" {
			return nil, errors.New("BOOTS_STANDALONE_YAML env must be set")
		}
		// set the baseURL from here so it gets returned in the client
		// TODO(@tobert): maybe there's a way to pass a file:// in the first place?
		baseURL, err = url.Parse("file://" + saYamlFile)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to convert path %q to a URL as 'file://%s'", saYamlFile)
		}
		saData, err := ioutil.ReadFile(saYamlFile)
		if err != nil {
			return nil, errors.Wrapf(err, "could not read file %q", saYamlFile)
		}
		dsDb := StandaloneYaml{}
		err = yaml.Unmarshal(saData, &dsDb)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to parse configuration file %q", saYamlFile)
		}

		// the "client" part is done - reading the yaml, now return a struct client
		// that is just the filename and parsed data structure
		hg = StandaloneClient{
			filename: saYamlFile,
			db:       dsDb.Discovery,
		}
	default:
		return nil, errors.Errorf("invalid DATA_MODEL_VERSION: %q", dataModelVersion)
	}

	return &Client{
		http:           c,
		baseURL:        baseURL,
		consumerToken:  consumerToken,
		authToken:      authToken,
		hardwareClient: hg,
		workflowClient: wg,
	}, nil
}

func NewMockClient(baseURL *url.URL, workflowClient tw.WorkflowServiceClient) *Client {
	t := &httplog.Transport{
		RoundTripper: http.DefaultTransport,
	}
	c := &http.Client{
		Transport: t,
	}
	return &Client{
		http:           c,
		workflowClient: workflowClient,
		baseURL:        baseURL,
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
