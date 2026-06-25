package main

import (
	"os"

	"github.com/finetension/toss-openapi-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
