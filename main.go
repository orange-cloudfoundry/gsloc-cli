package main

import (
	"github.com/orange-cloudfoundry/gsloc-cli/cli"
	"os"

	msg "github.com/ArthurHlt/messages"
)

var (
	version = "0.0.1-dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	err := cli.Start(version, commit, date)
	if err != nil {
		msg.Error(err.Error())
		os.Exit(1)
	}
}
