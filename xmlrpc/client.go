package xmlrpc

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type rpc interface {
	Call(req io.Reader) (io.ReadCloser, error)
}

type httpRPC struct {
	addr       string
	httpClient *http.Client
}

func (c *httpRPC) Call(req io.Reader) (io.ReadCloser, error) {
	resp, err := c.httpClient.Post(c.addr, "text/xml", req)
	if err != nil {
		return nil, errors.Wrap(err, "POST failed")
	}
	return resp.Body, nil
}

// Client implements a basic XMLRPC client
type Client struct {
	addr string
	rpc  rpc
}

// NewClient returns a new instance of Client
// Pass in a true value for `insecure` to turn off certificate verification
func NewClient(addr string, insecure bool) *Client {
	url, err := url.Parse(addr)
	if err != nil {
		panic(err)
	}

	if url.Scheme == "scgi" {
		return &Client{
			addr: addr,
			rpc:  nil,
		}
	}

	transport := &http.Transport{}
	if insecure {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	httpClient := &http.Client{Transport: transport}

	return &Client{
		addr: addr,
		rpc: &httpRPC{
			addr:       addr,
			httpClient: httpClient,
		},
	}
}

// NewClientWithHTTPClient returns a new instance of Client.
// This allows you to use a custom http.Client setup for your needs.
func NewClientWithHTTPClient(addr string, client *http.Client) *Client {
	return &Client{
		addr: addr,
		rpc: &httpRPC{
			addr:       addr,
			httpClient: client,
		},
	}
}

// Call calls the method with "name" with the given args
// Returns the result, and an error for communication errors
func (c *Client) Call(name string, args ...interface{}) (interface{}, error) {
	req := bytes.NewBuffer(nil)
	if err := Marshal(req, name, args...); err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	body, err := c.rpc.Call(req)
	if err != nil {
		return nil, errors.Wrap(err, "xml-rpc call failed")
	}

	defer body.Close()
	_, val, fault, err := Unmarshal(body)
	if fault != nil {
		err = errors.Errorf("Error: %v: %v", err, fault)
	}
	return val, err
}
