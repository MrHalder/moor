package main

import (
	"os"

	"github.com/MrHalder/moor/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
