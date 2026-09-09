package httpmethods

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

// Version is the tool's version, injected at build time via
// -ldflags "-X .../httpmethods.Version=...". Used for the default
// User-Agent header and the --version flag.
var Version = "dev"

var AvailableHttpMethods = []string{
	"GET",
	"POST",
	"PUT",
	"PATCH",
	"DELETE",
	"HEAD",
	"OPTIONS",
}

type Input struct {
	HTTPMethod string        `arg:"-m,--http-method,required" help:"HTTP method: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS"`
	URL        string        `arg:"-u,--url,required" help:"Request URL"`
	Body       string        `arg:"-b,--body" help:"Request body: raw text/JSON, @path to read from a file, or - to read from stdin"`
	Data       []string      `arg:"-d,--data,separate" help:"Body field as key=value, repeatable; builds a JSON object"`
	Form       []string      `arg:"--form,separate" help:"Body field as key=value, repeatable; sends application/x-www-form-urlencoded"`
	Multipart  []string      `arg:"-F,--multipart,separate" help:"Multipart field as key=value or key=@path to upload a file, repeatable; sends multipart/form-data"`
	Query      []string      `arg:"--query,separate" help:"Query parameter as key=value, repeatable; merged into the URL and encoded"`
	Header     []string      `arg:"-H,--header,separate"`
	Output     string        `arg:"-o"`
	Timeout    time.Duration `arg:"--timeout" default:"30s" help:"Request timeout"`
	Insecure   bool          `arg:"-k,--insecure" help:"Skip TLS certificate verification"`
	Proxy      string        `arg:"--proxy" help:"Send the request through this proxy, e.g. http://127.0.0.1:8080 (overrides HTTP_PROXY/HTTPS_PROXY)"`
	NoRedirect bool          `arg:"--no-redirect" help:"Do not follow redirects"`
	Auth       string        `arg:"-a,--auth" help:"Basic auth credentials as user:pass"`
	Fail       bool          `arg:"-f,--fail" help:"Exit with a non-zero status code on HTTP error responses (status >= 400)"`
	Verbose    bool          `arg:"-v,--verbose" help:"Print the outgoing request and response headers to stderr"`
	Include    bool          `arg:"-i,--include" help:"Include the response status line and headers in the output, before the body"`
	NoColor    bool          `arg:"--no-color" help:"Disable colored output"`
	Raw        bool          `arg:"--raw" help:"Write the response body verbatim: no JSON formatting and no binary-output guard (implied when stdout is not a terminal)"`
}

// Version implements go-arg's Versioned interface, enabling an automatic
// --version flag.
func (Input) Version() string {
	return "http-client " + Version
}

// Description implements go-arg's Described interface, shown at the top
// of --help output.
func (Input) Description() string {
	return "http-client sends HTTP requests from the command line."
}

func RunHttpMethod(input Input) (*Result, error) {

	if ok := slices.Contains(AvailableHttpMethods, strings.ToUpper(input.HTTPMethod)); !ok {
		return nil, fmt.Errorf("unknown http method: %v", input.HTTPMethod)
	}

	return exec(input)
}

// newHTTPClient builds an *http.Client configured from input's
// timeout/TLS/redirect/proxy flags.
func newHTTPClient(input Input) (*http.Client, error) {
	client := &http.Client{Timeout: input.Timeout}

	if input.Insecure || input.Proxy != "" {
		// Clone the default transport rather than building a bare one, so
		// HTTP_PROXY/HTTPS_PROXY, connection pooling and HTTP/2 keep
		// working when only TLS verification or the proxy is customized.
		transport := http.DefaultTransport.(*http.Transport).Clone()

		if input.Insecure {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}

		if input.Proxy != "" {
			proxyURL, err := url.Parse(input.Proxy)
			if err != nil {
				return nil, fmt.Errorf("invalid --proxy value: %w", err)
			}
			if proxyURL.Scheme == "" || proxyURL.Host == "" {
				return nil, fmt.Errorf("invalid --proxy value: %q, expected something like http://127.0.0.1:8080", input.Proxy)
			}
			transport.Proxy = http.ProxyURL(proxyURL)
		}

		client.Transport = transport
	}

	if input.NoRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return client, nil
}
