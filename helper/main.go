// herdr-poke is the helper behind the Herdr poke UI: Claude statusline,
// Codex wrapper, usage aggregator and asset fetcher in one binary.
package main

import (
	"fmt"
	"io"
	"os"
)

const usageText = `usage: herdr-poke <command> [args]

commands:
  statusline   Claude Code statusLine.command (reads stdin JSON, prints the line)
  codex        run Codex under herdr-poke: herdr-poke codex -- <codex args>
  usage        aggregate 14-day token usage into usage.json
  assets       fetch Pokemon and companion assets
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run dispatches one subcommand and returns the process exit code.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usageText)
		return 2
	}
	switch args[0] {
	case "statusline", "codex", "usage", "assets":
		fmt.Fprintf(stderr, "herdr-poke %s: not implemented yet\n", args[0])
		return 1
	default:
		fmt.Fprint(stderr, usageText)
		return 2
	}
}
