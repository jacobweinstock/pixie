package cmd

import (
	"context"
	"flag"
	"fmt"

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
	fmt.Printf("file: %+v\n", c.config)
	c.Log.Info("debugging", "file", fmt.Sprintf("%+v", c))
	cf := ipxe.FileCfg{
		Config: ipxe.Config{
			TFTPAddr: c.TftpAddr,
			HTTPAddr: c.HttpAddr,
			LogLevel: c.LogLevel,
			Log:      c.Log,
		},
		Filename: c.Filename,
	}
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return cf.Exec(ctx, nil)
	})
	g.Go(func() error {
		pd := &proxydhcp.Config{
			LogLevel:        c.LogLevel,
			TFTPAddr:        "tftp://" + "192.168.2.225",
			HTTPAddr:        "http://" + "192.168.2.225",
			IPXEURL:         c.IPXEURL,
			Addr:            c.Addr,
			CustomUserClass: c.CustomUserClass,
			Log:             c.Log,
		}
		c.Log.Info("debugging", "pd", fmt.Sprintf("%+v", pd))
		return pd.Run(ctx, nil)
	})
	return g.Wait()
}

func (c *fileCfg) registerFlags(name string, errHandler flag.ErrorHandling) *flag.FlagSet {
	fs := registerFlags(c.config, name, errHandler)
	fs.StringVar(&c.Filename, "filename", "", "filename")
	return fs
}
