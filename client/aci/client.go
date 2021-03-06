package aci

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.opencensus.io/plugin/ochttp/propagation/b3"

	"go.opencensus.io/plugin/ochttp"

	azure "github.com/virtual-kubelet/azure-aci/client"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

const (
	defaultUserAgent = "virtual-kubelet/azure-arm-aci/2018-10-01"
	apiVersion       = "2018-10-01"

	containerGroupURLPath                    = "subscriptions/{{.subscriptionId}}/resourceGroups/{{.resourceGroup}}/providers/Microsoft.ContainerInstance/containerGroups/{{.containerGroupName}}"
	containerGroupListURLPath                = "subscriptions/{{.subscriptionId}}/providers/Microsoft.ContainerInstance/containerGroups"
	containerGroupListByResourceGroupURLPath = "subscriptions/{{.subscriptionId}}/resourceGroups/{{.resourceGroup}}/providers/Microsoft.ContainerInstance/containerGroups"
	containerLogsURLPath                     = containerGroupURLPath + "/containers/{{.containerName}}/logs"
	containerExecURLPath                     = containerGroupURLPath + "/containers/{{.containerName}}/exec"
	containerGroupMetricsURLPath             = containerGroupURLPath + "/providers/microsoft.Insights/metrics"
)

// Client is a client for interacting with Azure Container Instances.
//
// Clients should be reused instead of created as needed.
// The methods of Client are safe for concurrent use by multiple goroutines.
type Client struct {
	hc   *http.Client
	auth *azure.Authentication
	rc   retryablehttp.Client
}

// NewClient creates a new Azure Container Instances client with extra user agent.
func NewClient(auth *azure.Authentication, extraUserAgent string) (*Client, error) {
	if auth == nil {
		return nil, fmt.Errorf("Authentication is not supplied for the Azure client")
	}

	userAgent := []string{defaultUserAgent}
	if extraUserAgent != "" {
		userAgent = append(userAgent, extraUserAgent)
	}

	client, err := azure.NewClient(auth, userAgent)
	if err != nil {
		return nil, fmt.Errorf("Creating Azure client failed: %v", err)
	}
	hc := client.HTTPClient
	hc.Transport = &ochttp.Transport{
		Base:           hc.Transport,
		Propagation:    &b3.HTTPFormat{},
		NewClientTrace: ochttp.NewSpanAnnotatingClientTrace,
	}

	return &Client{
		hc:   client.HTTPClient,
		auth: auth,
		rc: retryablehttp.Client{
			HTTPClient:   hc,
			Logger:       defaultLogger,
			RetryWaitMin: defaultRetryWaitMin,
			RetryWaitMax: defaultRetryWaitMax,
			RetryMax:     defaultRetryMax,
			CheckRetry:   retryablehttp.DefaultRetryPolicy,
			Backoff:      retryablehttp.DefaultBackoff,
		}}, nil
}

var (
	defaultRetryWaitMin = 1 * time.Second
	defaultRetryWaitMax = 30 * time.Second
	defaultRetryMax     = 4

	// defaultLogger is the logger provided with defaultClient
	defaultLogger = log.New(os.Stderr, "", log.LstdFlags)
)

const (
	// DefaultPollingDelay is a reasonable delay between polling requests.
	DefaultPollingDelay = 60 * time.Second

	// DefaultPollingDuration is a reasonable total polling duration.
	DefaultPollingDuration = 15 * time.Minute

	// DefaultRetryAttempts is number of attempts for retry status codes (5xx).
	DefaultRetryAttempts = 3

	// DefaultRetryDuration is the duration to wait between retries.
	DefaultRetryDuration = 30 * time.Second
)

var (
	// StatusCodesForRetry are a defined group of status code for which the client will retry
	StatusCodesForRetry = []int{
		http.StatusRequestTimeout,      // 408
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout,      // 504
	}
)
