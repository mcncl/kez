package main

import (
	"github.com/alecthomas/kong"
	"github.com/mcncl/kuberneasy/cmd"
)

type Context struct {
    Debug bool
}

var cli struct {
    Debug bool `help:"Enable debug logging"`
    Configure cmd.ConfigureCmd `cmd:"" help:"Configure Buildkite API token"`
}

func main () {
    ctx := kong.Parse(&cli, kong.UsageOnError())
    err := ctx.Run(&Context{Debug: cli.Debug})
    ctx.FatalIfErrorf(err)
}
