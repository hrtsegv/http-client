package httpmethods

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
)

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
	HTTPMethod string   `arg:"-m,--http-method,required" help:"HTTP method: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS"`
	URL        string   `arg:"-u,--url,required" help:"Request URL"`
	Body       string   `arg:"-b,--body" help:"Request body: raw text/JSON, @path to read from a file, or - to read from stdin. Mutually exclusive with --data"`
	Data       []string `arg:"-d,--data,separate" help:"Body field as key=value, repeatable; builds a JSON object. Mutually exclusive with --body"`
	Header     []string `arg:"-H,--header,separate"`
	Output     string   `arg:"-o"`
}

var httpClient *http.Client = &http.Client{
	Timeout: 30 * time.Second,
}

func RunHttpMethod(input Input) (*Result, error) {

	if ok := slices.Contains(AvailableHttpMethods, strings.ToUpper(input.HTTPMethod)); !ok {
		return nil, fmt.Errorf("unknown http method: %v", input.HTTPMethod)
	}

	return exec(input)
}
