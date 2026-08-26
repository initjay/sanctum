// Command sanctum locks a terminal session to a specific Claude Code
// credential profile.
package main

import (
	"fmt"
	"os"

	"github.com/initjay/sanctum/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
