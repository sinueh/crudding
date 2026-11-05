package main

import (
	"os"

	"github.com/sinueh/crudding/internal/cli"
	"github.com/sinueh/crudding/internal/generator/custom"
)

func main() {
	_, _ = custom.EnsureTemplatesDir()
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
