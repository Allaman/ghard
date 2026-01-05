package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/alecthomas/kong"
	"github.com/allaman/ghard/internal/app"
	"github.com/allaman/ghard/internal/config"
)

func main() {
	var cli app.CLI
	ctx := kong.Parse(&cli)

	logLevel := slog.LevelInfo
	if cli.Debug {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		config.HandleError(err)
		os.Exit(1)
	}

	application := app.New(cfg)
	if err := application.Run(ctx.Command(), &cli); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
