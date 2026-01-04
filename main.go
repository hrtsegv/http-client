package main

import (
	"io"
	"log"
	"strings"

	"github.com/alexflint/go-arg"

	"github.com/knbr13/http-client/internal/httpmethods"
	"github.com/knbr13/http-client/internal/output"
)

func main() {
	input := httpmethods.Input{}
	arg.MustParse(&input)

	httpResponse, err := httpmethods.RunHttpMethod(input)
	if err != nil {
		log.Fatal(err)
	}
	defer httpResponse.Body.Close()

	switch strings.ToLower(input.HTTPMethod) {
	case "head":
		// For HEAD request, only print the response status
		output.PrintColoredHeaders(httpResponse.Header)
	default:
		// For other requests, print the response body
		body, err := io.ReadAll(httpResponse.Body)
		if err != nil {
			log.Fatal(err)
		}
		output.PrintColoredBody(body)
		if input.Output != "" {
			err = output.WriteToFile(input.Output, body)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
