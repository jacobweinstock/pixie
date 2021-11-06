package cmd

import (
	"context"
	"flag"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/go-playground/validator/v10"
	"github.com/jacobweinstock/proxydhcp/cli"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/rs/zerolog"
)

const name = "pixie"

type config struct {
	LogLevel         string `validate:"oneof=debug info"`
	IPXEAddr         string `validate:"required,ip"`
	IPXEScriptAddr   string `validate:"required,url"`
	IPXEScript       string `validate:"required"`
	ProxyDHCPAddr    string `validate:"required,ip"`
	CustomUserClass  string
	DisableIPXE      bool
	DisableProxyDHCP bool
	Log              logr.Logger
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
	fs.StringVar(&c.IPXEAddr, "ipxe-addr", "", "address for servering tftp (port 69) and http (port 80) ipxe files")
	fs.StringVar(&c.IPXEScriptAddr, "ipxe-script-addr", "", "address that serves the ipxe script (http://192.168.2.2)")
	fs.StringVar(&c.IPXEScript, "ipxe-script-name", "auto.ipxe", "ipxe script name. used with ipxe-script-addr (http://192.168.2.2/<mac-addr>/auto.ipxe)")
	fs.StringVar(&c.ProxyDHCPAddr, "proxy-dhcp-addr", "", "address to listen on for proxy dhcp")
	fs.StringVar(&c.CustomUserClass, "custom-user-class", "iPXE", "custom user class")
	fs.BoolVar(&c.DisableIPXE, "disable-ipxe", false, "disable ipxe")
	fs.BoolVar(&c.DisableProxyDHCP, "disable-proxy-dhcp", false, "disable proxy dhcp")

	return fs
}

func validate(c *config) error {
	return validator.New().Struct(c)
}

// defaultLogger is a zerolog logr implementation.
func defaultLogger(level string) logr.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerologr.NameFieldName = "logger"
	zerologr.NameSeparator = "/"

	zl := zerolog.New(os.Stdout)
	zl = zl.With().Caller().Timestamp().Logger()
	var log logr.Logger = zerologr.New(&zl)

	return log
}
