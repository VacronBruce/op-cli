package main

import (
	"os"

	"github.com/chenhuijun/op-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
