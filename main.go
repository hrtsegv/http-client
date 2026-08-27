package main

import (
	"os"

	"github.com/alexflint/go-arg"

	"github.com/hrtsegv/http-client/internal/cli"
	"github.com/hrtsegv/http-client/internal/httpmethods"
)

func main() {
	input := httpmethods.Input{}
	arg.MustParse(&input)

	os.Exit(cli.Run(input))
}
