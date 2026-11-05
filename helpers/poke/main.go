// herdr-poke is the helper behind the Herdr UI: Codex wrapper, usage aggregator and
// asset fetcher in one binary. (Claude limits come from the global statusline.)
package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"herdr-poke/internal/assets"
	"herdr-poke/internal/codex"
	"herdr-poke/internal/usage"
)

const usageText = `usage: herdr-poke <command> [args]

commands:
  codex        run Codex under herdr-poke: herdr-poke codex -- <codex args>
               or mirror one started elsewhere: herdr-poke codex --attach <terminal>
  usage        aggregate 14-day token usage into daily.json (the usage data contract folder)
  assets       fetch Pokemon and companion assets: assets <id> | --walk <id> | --companion | --list | --dex | --normalize
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
	case "codex":
		// herdr-poke codex -- codex --yolo --no-daemon: what follows "--" is the command.
		cmd := args[1:]
		if len(cmd) == 2 && cmd[0] == "--attach" {
			return codex.Attach(cmd[1])
		}
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
		return assets.Run(args[1:], assets.ConfigFromEnv(), stdout, stderr)
	default:
		fmt.Fprint(stderr, usageText)
		return 2
	}
}
