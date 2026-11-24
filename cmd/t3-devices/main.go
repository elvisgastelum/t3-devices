// Command t3-devices is the entry point for the t3-devices CLI.
package main

import (
	"os"

	"github.com/elvisgastelum/t3-devices/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
