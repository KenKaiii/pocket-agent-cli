package main

import (
	"os"

	"github.com/unstablemind/pocket/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
