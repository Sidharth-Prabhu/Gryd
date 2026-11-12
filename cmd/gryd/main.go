package main

import (
	"os"

	"github.com/loom-cli/gryd/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
