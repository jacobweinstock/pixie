package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/go-playground/validator/v10"
	"github.com/jacobweinstock/proxydhcp/cli"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const name = "pixie"

type config struct {
	LogLevel        string `validate:"oneof=debug info"`
	TftpAddr        string `validate:"required"`
	HttpAddr        string `validate:"required"`
	IPXEURL         string `validate:"required"`
	Addr            string `validate:"hostname_port"`
	CustomUserClass string
	Log             logr.Logger
}

func Execute(ctx context.Context) error {
	c := &config{}
	root := &ffcli.Command{
		Name:    name,
		FlagSet: registerFlags(c, name, flag.ExitOnError),
		Options: []ff.Option{
			ff.WithEnvVarPrefix(strings.ToUpper(name)),
		},
		Subcommands: []*ffcli.Command{
			cli.SupportedBins(ctx),
			file(&fileCfg{config: c}),
			tink(&tinkCfg{config: c}),
		},
		Exec: func(ctx context.Context, args []string) error {
			if err := validate(c); err != nil {
				return err
			}
			return flag.ErrHelp
		},
	}
	return root.ParseAndRun(ctx, os.Args[1:])
}

func registerFlags(c *config, name string, errHandler flag.ErrorHandling) *flag.FlagSet {
	fs := flag.NewFlagSet(name, errHandler)
	fs.StringVar(&c.LogLevel, "log-level", "info", "log level")
	fs.StringVar(&c.TftpAddr, "tftp-addr", "", "tftp server address")
	fs.StringVar(&c.HttpAddr, "http-addr", "", "http server address")
	fs.StringVar(&c.IPXEURL, "ipxe-url", "", "ipxe url")
	fs.StringVar(&c.Addr, "addr", ":8080", "address")
	fs.StringVar(&c.CustomUserClass, "custom-user-class", "", "custom user class")

	return fs
}

func validate(c *config) error {
	return validator.New().Struct(c)
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
