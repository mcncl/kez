package main

import (
	"github.com/alecthomas/kong"
	"github.com/mcncl/kez/cmd"
	"github.com/mcncl/kez/cmd/stack"
	"github.com/mcncl/kez/internal/logger"
)

type Context struct {
	Debug bool
}

var cli struct {
	Debug     bool             `help:"Enable debug logging"`
	Configure cmd.ConfigureCmd `cmd:"" help:"Configure Buildkite API token"`
	Stack     struct {
		Create stack.CreateCmd `cmd:"" help:"Create a Buildkite agent stack in Kubernetes"`
		Status stack.StatusCmd `cmd:"" help:"Check the status of a Buildkite agent stack"`
		Delete stack.DeleteCmd `cmd:"" help:"Delete a Buildkite agent stack from Kubernetes"`
	} `cmd:"" help:"Manage Buildkite agent stacks"`
}

func main() {
	ctx := kong.Parse(&cli, kong.UsageOnError())

	logLevel := logger.LevelWarn
	if cli.Debug {
		logLevel = logger.LevelDebug
	}

	logger.Setup(logger.Config{
		Level: logLevel,
	})

	err := ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}
