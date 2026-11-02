package main

import (
	"fmt"
	"os"

	"github.com/benoit-pereira-da-silva/gogrammalecte/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.NewCommand(version).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
