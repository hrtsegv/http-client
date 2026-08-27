package httpmethods

import (
	"crypto/tls"
	"fmt"
	"net/http"
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
	Body       string        `arg:"-b,--body" help:"Request body: raw text/JSON, @path to read from a file, or - to read from stdin. Mutually exclusive with --data"`
	Data       []string      `arg:"-d,--data,separate" help:"Body field as key=value, repeatable; builds a JSON object. Mutually exclusive with --body"`
	Header     []string      `arg:"-H,--header,separate"`
	Output     string        `arg:"-o"`
	Timeout    time.Duration `arg:"--timeout" default:"30s" help:"Request timeout"`
	Insecure   bool          `arg:"-k,--insecure" help:"Skip TLS certificate verification"`
	NoRedirect bool          `arg:"--no-redirect" help:"Do not follow redirects"`
	Auth       string        `arg:"-a,--auth" help:"Basic auth credentials as user:pass"`
	Fail       bool          `arg:"-f,--fail" help:"Exit with a non-zero status code on HTTP error responses (status >= 400)"`
	Verbose    bool          `arg:"-v,--verbose" help:"Print the outgoing request and response headers to stderr"`
	NoColor    bool          `arg:"--no-color" help:"Disable colored output"`
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
// timeout/TLS/redirect flags.
func newHTTPClient(input Input) *http.Client {
	client := &http.Client{Timeout: input.Timeout}

	if input.Insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	if input.NoRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return client
}
