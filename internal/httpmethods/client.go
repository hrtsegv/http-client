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
	HTTPMethod string   `arg:"-m,--http-method,required"`
	URL        string   `arg:"-u,--url,required"`
	Body       string   `arg:"-b,--body"`
	Header     []string `arg:"-H,--header,separate"`
	Output     string   `arg:"-o"`
}

var httpClient *http.Client = &http.Client{
	Timeout: 30 * time.Second,
}

func RunHttpMethod(input Input) (*http.Response, error) {

	if ok := slices.Contains(AvailableHttpMethods, strings.ToUpper(input.HTTPMethod)); !ok {
		return nil, fmt.Errorf("unknown http method: %v", input.HTTPMethod)
	}

	return exec(input)
}
