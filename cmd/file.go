package cmd

import (
	"context"
	"flag"

	"github.com/go-playground/validator/v10"
	ipxe "github.com/jacobweinstock/ipxe/cli"
	proxydhcp "github.com/jacobweinstock/proxydhcp/cli"
	"github.com/peterbourgon/ff/v3/ffcli"
	"golang.org/x/sync/errgroup"
)

const fileCLI = "file"

type fileCfg struct {
	*config
	Filename string `validate:"required"`
}

func file(c *fileCfg) *ffcli.Command {
	return &ffcli.Command{
		Name:       fileCLI,
		ShortUsage: fileCLI,
		FlagSet:    c.registerFlags(fileCLI, flag.ExitOnError),
		Exec: func(ctx context.Context, _ []string) error {
			return c.exec(ctx)
		},
	}
}

func (c *fileCfg) exec(ctx context.Context) error {
	if err := validator.New().Struct(c); err != nil {
		return err
	}
	c.Log = defaultLogger(c.LogLevel)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		cf := ipxe.NewFile(
			ipxe.WithLogger(c.Log),
			ipxe.WithFilename(c.Filename),
			ipxe.WithHTTP(c.IPXEAddr+":80"),
			ipxe.WithTFTPAddr(c.IPXEAddr+":69"),
			ipxe.WithLogLevel(c.LogLevel),
		)
		return cf.Exec(ctx, nil)
	})
	g.Go(func() error {
		pd := proxydhcp.NewConfig(
			proxydhcp.WithLogger(c.Log),
			proxydhcp.WithLogLevel(c.LogLevel),
			proxydhcp.WithHTTPAddr("http://"+c.IPXEAddr),
			proxydhcp.WithTFTPAddr("tftp://"+c.IPXEAddr),
			proxydhcp.WithCustomUserClass(c.CustomUserClass),
			proxydhcp.WithAddr(c.ProxyDHCPAddr),
			proxydhcp.WithIPXEURL(c.IPXEURL),
		)
		if err := pd.ValidateConfig(); err != nil {
			return err
		}
		return pd.Run(ctx, nil)
	})
	return g.Wait()
}

func (c *fileCfg) registerFlags(name string, errHandler flag.ErrorHandling) *flag.FlagSet {
	fs := registerFlags(c.config, name, errHandler)
	fs.StringVar(&c.Filename, "filename", "", "filename")
	return fs
}
