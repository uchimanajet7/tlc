package cmd

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/pkg/errors"
)

// Client use HTTP access
type Client struct {
	EndpointURL *url.URL
	HTTPClient  *http.Client
}

func createClient(endpointURL string, httpClient *http.Client) (*Client, error) {
	parsedURL, err := url.Parse(endpointURL)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse url: %s", endpointURL)
	}

	// use HTTPS access
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient.Transport = tr

	client := &Client{
		EndpointURL: parsedURL,
		HTTPClient:  httpClient,
	}
	return client, nil
}

func (client *Client) createRequest(ctx context.Context, method string, subPath string, body io.Reader) (*http.Request, error) {
	endpointURL := *client.EndpointURL
	endpointURL.Path = path.Join(client.EndpointURL.Path, subPath)

	req, err := http.NewRequest(method, endpointURL.String(), body)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	// header imitates Mac Chrome
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Charset", "utf8")
	req.Header.Set("Content-Type", "text/html")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/57.0.2987.133 Safari/537.36")

	return req, nil
}
