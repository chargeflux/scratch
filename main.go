package main

import (
	"log/slog"
	"os"

	"github.com/alecthomas/kong"
)

func main() {
	ctx := kong.Parse(&CLI)
	cliCtx := &CLIContext{}
	ctx.Bind(cliCtx)

	if CLI.Verbose {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	}

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
