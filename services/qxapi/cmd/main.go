package main

import (
	"os"
)

var exit = os.Exit

func main() {
	if err := run(); err != nil {
		exit(1)
	}
}
