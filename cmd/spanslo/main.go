// Command spanslo is the offline scoring CLI.
package main

import (
	"fmt"
	"io"
	"os"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// run dispatches the subcommand and returns the exit code (R-068).
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "score" {
		return score(args[1:], stdout, stderr)
	}
	fmt.Fprintln(stderr, usage)
	return 2
}
