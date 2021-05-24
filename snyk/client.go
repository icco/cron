package snyk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Client struct {
	APIToken string
	OrgID    string
	http     *http.Client
}

// AddHeaderTransport is a http transport for adding auth headers to a request.
type addHeaderTransport struct {
	T   http.RoundTripper
	Key string
}

// RoundTrip actually adds the headers.
func (adt *addHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if adt.Key == "" {
		return nil, fmt.Errorf("no key provided")
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("token %s", adt.Key))
	req.Header.Add("User-Agent", "cron.natwelch/1.0")

	return adt.T.RoundTrip(req)
}

// New returns a new client for Snyk.io.
func New(apiKey string, org string) *Client {
	httpclient := &http.Client{
		Transport: &addHeaderTransport{T: http.DefaultTransport, Key: apiKey},
	}

	return &Client{
		APIToken: apiKey,
		OrgID:    org,
		http:     httpclient,
	}
}

func (c *Client) Do(ctx context.Context, path string, reqBody interface{}) (int, []byte, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("https://snyk.io/api/v1/org/%s%s", c.OrgID, path)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return 0, nil, fmt.Errorf("building request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("http request: %w", err)
	}

	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("reading response: %w", err)
	}

	return resp.Status, resp_body, nil
}
