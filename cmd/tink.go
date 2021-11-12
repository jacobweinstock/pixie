package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const tinkCLI = "tink"

type tinkCfg struct {
	*config
	// TLS can be one of the following
	// 1. location on disk of a cert
	// example: /location/on/disk/of/cert
	// 2. URL from which to GET a cert
	// example: http://weburl:8080/cert
	// 3. boolean; true if the tink server (specified by the Tink key/value) has a cert from a known CA
	// false if the tink server does not have TLS enabled
	// example: true
	TLS string
	// Tink is the URL:Port for the tink server
	Tink string `validate:"required"`
}

func tink(c *tinkCfg) *ffcli.Command {
	return &ffcli.Command{
		Name:       tinkCLI,
		ShortUsage: tinkCLI,
		FlagSet:    c.registerFlags(tinkCLI, flag.ExitOnError),
		Exec: func(ctx context.Context, _ []string) error {
			return c.exec(ctx)
		},
	}
}

func (c *tinkCfg) exec(_ context.Context) error {
	if err := validator.New().Struct(c); err != nil {
		return err
	}
	fmt.Printf("tink: %+v\n", c.config)
	fmt.Printf("tink: %+v\n", c)
	return nil
}

func (c *tinkCfg) registerFlags(name string, errHandler flag.ErrorHandling) *flag.FlagSet {
	fs := registerFlags(c.config, name, errHandler)
	fs.StringVar(&c.TLS, "tls", "", "tls")
	fs.StringVar(&c.Tink, "tink", "", "tink")
	return fs
}
