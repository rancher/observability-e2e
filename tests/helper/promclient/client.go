package promclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

// Client is a wrapper for the Prometheus v1 API client.
type Client struct {
	v1api v1.API
}

// NewClient creates and returns a new Prometheus client.
// It accepts a bearerToken for authorization and skips TLS verification if needed.
func NewClient(prometheusURL, bearerToken string) (*Client, error) {
	// Create the base transport with optional insecure TLS
	baseTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // skip TLS verification if true
		},
	}

	// Wrap the transport to add the Authorization header
	authTransport := &authRoundTripper{
		originalTransport: baseTransport,
		bearerToken:       bearerToken,
	}

	// Create HTTP client
	httpClient := &http.Client{
		Transport: authTransport,
		Timeout:   10 * time.Second,
	}

	// Create Prometheus client
	client, err := api.NewClient(api.Config{
		Address: prometheusURL,
		Client:  httpClient,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating Prometheus client: %w", err)
	}

	return &Client{v1api: v1.NewAPI(client)}, nil
}

// authRoundTripper adds an Authorization header to every request
type authRoundTripper struct {
	originalTransport http.RoundTripper
	bearerToken       string
}

func (art *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if art.bearerToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", art.bearerToken))
	}
	return art.originalTransport.RoundTrip(req)
}

// Query executes a PromQL query and returns the result as a model.Vector
func (c *Client) Query(query string) (*model.Vector, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, warnings, err := c.v1api.Query(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("error querying Prometheus: %w", err)
	}

	if len(warnings) > 0 {
		e2e.Logf("Warnings: %v\n", warnings)
	}

	if result.Type() != model.ValVector {
		return nil, fmt.Errorf("unexpected result type: %s", result.Type().String())
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("failed to cast result to vector")
	}

	return &vector, nil
}
