package main

import (
	"context"
	"os"

	"skillview/internal/cli"
)

// Set by script/build-cli.sh from the shared release version file.
var version = "dev"

func main() {
	os.Exit(cli.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, cli.Options{Version: version}))
}
