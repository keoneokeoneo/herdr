// herdr-poke is the helper behind the Herdr poke UI: Claude statusline,
// Codex wrapper, usage aggregator and asset fetcher in one binary.
package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"herdr-poke/internal/claude"
	"herdr-poke/internal/codex"
	"herdr-poke/internal/usage"
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
	case "statusline":
		return claude.Run(stdin, stdout)
	case "codex":
		// herdr-poke codex -- codex --yolo --no-daemon: what follows "--" is the command.
		cmd := args[1:]
		if len(cmd) > 0 && cmd[0] == "--" {
			cmd = cmd[1:]
		}
		return codex.Wrap(cmd)
	case "usage":
		if err := usage.Run(time.Now(), usage.DefaultRoots()); err != nil {
			fmt.Fprintf(stderr, "herdr-poke usage: %v\n", err)
			return 1
		}
		return 0
	case "assets":
		fmt.Fprintf(stderr, "herdr-poke %s: not implemented yet\n", args[0])
		return 1
	default:
		fmt.Fprint(stderr, usageText)
		return 2
	}
}
