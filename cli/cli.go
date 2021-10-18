package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/jacobweinstock/proxydhcp/cli"
	"github.com/peterbourgon/ff/v3/ffcli"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const name = "pixie"

func Pixie(ctx context.Context) *ffcli.Command {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	_, cfg := cli.ProxyDHCP(ctx)
	cli.RegisterFlags(cfg, fs)
	cfg.Log = defaultLogger("info")
	return &ffcli.Command{
		Name:       name,
		FlagSet:    fs,
		ShortUsage: fmt.Sprintf("%v", name),
		Subcommands: []*ffcli.Command{
			cli.SupportedBins(ctx),
		},
		Exec: func(ctx context.Context, args []string) error {
			return cfg.Exec(ctx, args)
		},
	}
}

// defaultLogger is zap logr implementation.
func defaultLogger(level string) logr.Logger {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
	zapLogger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}

	return zapr.NewLogger(zapLogger)
}
