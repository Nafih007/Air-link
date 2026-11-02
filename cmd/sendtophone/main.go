package main

import (
	"os"

	"sendtophone/internal/cli"
)

// version is set at link time: -ldflags "-X main.version=1.2.3"
var version = "dev"

func main() {
	if version != "" {
		cli.Version = version
	}
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
