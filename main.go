package main

import (
	"os"

	"github.com/stephenwilliams/mcp-helper/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
